package auth

import (
	"bytes"
	"context"
	"crypto/aes"
	"crypto/cipher"
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"errors"
	"fmt"
	"image/png"
	"strconv"
	"strings"
	"time"

	"github.com/boombuler/barcode"
	"github.com/boombuler/barcode/qr"
	"github.com/golang-jwt/jwt/v5"
	"github.com/jackc/pgx/v5"

	"github.com/odysight/crm/pkg/response"
	"github.com/odysight/crm/pkg/totp"
)

// Two-factor authentication with an authenticator app (TOTP).
//
// Login: password → (2FA on) a 5-minute challenge token → code → session.
// Office roles must enrol: until they do, their session only reaches the
// /auth endpoints needed to set 2FA up.

const (
	totpIssuer       = "Smile Clean"
	mfaTokenTTL      = 5 * time.Minute
	mfaAudience      = "odysight-mfa"
	trustedDeviceTTL = 30 * 24 * time.Hour
	backupCodeCount  = 10
	// Session claim marking a login that still has to enrol in 2FA.
	claimMFASetup = "mfa_setup"
)

var errMFAFailed = response.NewAPIError(401, "invalid code")

// RequiresTwoFactor reports whether a role must use 2FA. Office roles handle
// money, customer data and settings; cleaners may opt in.
func RequiresTwoFactor(role Role) bool {
	switch role {
	case RoleSuperAdmin, RoleAdmin, RoleManager, RoleAccountant, RoleDispatch:
		return true
	}
	return false
}

// twoFactorKeys derive the secret-encryption and backup-code keys from the
// server secret, so the database alone is not enough to pass 2FA.
type twoFactorKeys struct {
	enc [32]byte
	mac [32]byte
}

func newTwoFactorKeys(serverSecret []byte) twoFactorKeys {
	return twoFactorKeys{
		enc: sha256.Sum256(append([]byte("odysight-totp-enc:"), serverSecret...)),
		mac: sha256.Sum256(append([]byte("odysight-backup-mac:"), serverSecret...)),
	}
}

func (k twoFactorKeys) seal(plain string) (string, error) {
	block, err := aes.NewCipher(k.enc[:])
	if err != nil {
		return "", err
	}
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return "", err
	}
	nonce := make([]byte, gcm.NonceSize())
	if _, err := rand.Read(nonce); err != nil {
		return "", err
	}
	return base64.StdEncoding.EncodeToString(gcm.Seal(nonce, nonce, []byte(plain), nil)), nil
}

func (k twoFactorKeys) open(sealed string) (string, error) {
	raw, err := base64.StdEncoding.DecodeString(sealed)
	if err != nil {
		return "", err
	}
	block, err := aes.NewCipher(k.enc[:])
	if err != nil {
		return "", err
	}
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return "", err
	}
	if len(raw) < gcm.NonceSize() {
		return "", errors.New("sealed secret too short")
	}
	plain, err := gcm.Open(nil, raw[:gcm.NonceSize()], raw[gcm.NonceSize():], nil)
	if err != nil {
		return "", errors.New("cannot decrypt 2FA secret (server secret changed?)")
	}
	return string(plain), nil
}

func (k twoFactorKeys) codeHash(code string) string {
	m := hmac.New(sha256.New, k.mac[:])
	m.Write([]byte(normalizeBackupCode(code)))
	return hex.EncodeToString(m.Sum(nil))
}

func normalizeBackupCode(code string) string {
	return strings.ToLower(strings.NewReplacer("-", "", " ", "").Replace(strings.TrimSpace(code)))
}

// Backup codes: 10 characters from an alphabet without look-alikes, shown
// as xxxxx-xxxxx (about 50 bits each).
const backupAlphabet = "abcdefghjkmnpqrstuvwxyz23456789"

func newBackupCode() (string, error) {
	var b strings.Builder
	var one [1]byte
	limit := 256 - 256%len(backupAlphabet) // reject to avoid modulo bias
	for n := 0; n < 10; {
		if _, err := rand.Read(one[:]); err != nil {
			return "", err
		}
		if int(one[0]) >= limit {
			continue
		}
		if n == 5 {
			b.WriteByte('-')
		}
		b.WriteByte(backupAlphabet[int(one[0])%len(backupAlphabet)])
		n++
	}
	return b.String(), nil
}

// --- repository ---

type twoFactorState struct {
	Enabled       bool
	Secret        string // sealed
	PendingSecret string // sealed
	LastStep      int64
}

func (r *Repository) twoFactor(ctx context.Context, userID int64) (twoFactorState, error) {
	var s twoFactorState
	err := r.pool.QueryRow(ctx,
		`SELECT totp_enabled, totp_secret, totp_pending_secret, totp_last_step FROM users WHERE id = $1`, userID).
		Scan(&s.Enabled, &s.Secret, &s.PendingSecret, &s.LastStep)
	if errors.Is(err, pgx.ErrNoRows) {
		return s, ErrInvalidCredentials
	}
	return s, err
}

// claimStep records a used TOTP step; false means the code was already used.
func (r *Repository) claimStep(ctx context.Context, userID, step int64) (bool, error) {
	tag, err := r.pool.Exec(ctx,
		`UPDATE users SET totp_last_step = $2 WHERE id = $1 AND totp_last_step < $2`, userID, step)
	if err != nil {
		return false, err
	}
	return tag.RowsAffected() == 1, nil
}

func (r *Repository) useBackupCode(ctx context.Context, userID int64, hash string) (bool, error) {
	tag, err := r.pool.Exec(ctx,
		`UPDATE user_backup_codes SET used_at = now()
		  WHERE id = (SELECT id FROM user_backup_codes
		               WHERE user_id = $1 AND code_hash = $2 AND used_at IS NULL LIMIT 1)`, userID, hash)
	if err != nil {
		return false, err
	}
	return tag.RowsAffected() == 1, nil
}

func (r *Repository) replaceBackupCodes(ctx context.Context, tx pgx.Tx, userID int64, hashes []string) error {
	if _, err := tx.Exec(ctx, `DELETE FROM user_backup_codes WHERE user_id = $1`, userID); err != nil {
		return err
	}
	for _, h := range hashes {
		if _, err := tx.Exec(ctx, `INSERT INTO user_backup_codes (user_id, code_hash) VALUES ($1, $2)`, userID, h); err != nil {
			return err
		}
	}
	return nil
}

func (r *Repository) addTrustedDevice(ctx context.Context, userID int64, hash, userAgent string) error {
	if len(userAgent) > 300 {
		userAgent = userAgent[:300]
	}
	_, err := r.pool.Exec(ctx,
		`INSERT INTO user_trusted_devices (user_id, token_hash, user_agent, expires_at) VALUES ($1, $2, $3, $4)`,
		userID, hash, userAgent, time.Now().Add(trustedDeviceTTL))
	return err
}

func (r *Repository) trustedDevice(ctx context.Context, userID int64, hash string) (bool, error) {
	tag, err := r.pool.Exec(ctx,
		`UPDATE user_trusted_devices SET last_used_at = now()
		  WHERE user_id = $1 AND token_hash = $2 AND expires_at > now()`, userID, hash)
	if err != nil {
		return false, err
	}
	return tag.RowsAffected() == 1, nil
}

// --- service ---

// LoginResult is what a password login yields.
type LoginResult struct {
	Token            string
	User             UserDTO
	MFARequired      bool
	MFAToken         string
	MFASetupRequired bool
}

func (s *Service) signMFAToken(userID int64) (string, error) {
	now := time.Now()
	t := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
		"sub": strconv.FormatInt(userID, 10),
		"iss": "odysight-crm",
		"aud": mfaAudience,
		"iat": now.Unix(),
		"exp": now.Add(mfaTokenTTL).Unix(),
	})
	return t.SignedString(s.jwtSecret)
}

func (s *Service) parseMFAToken(raw string) (int64, error) {
	parsed, err := jwt.Parse(raw, func(t *jwt.Token) (any, error) {
		if _, ok := t.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, jwt.ErrSignatureInvalid
		}
		return s.jwtSecret, nil
	}, jwt.WithIssuer("odysight-crm"), jwt.WithAudience(mfaAudience), jwt.WithExpirationRequired())
	if err != nil || !parsed.Valid {
		return 0, response.NewAPIError(401, "sign-in expired; enter your password again")
	}
	sub, err := parsed.Claims.GetSubject()
	if err != nil {
		return 0, response.NewAPIError(401, "sign-in expired; enter your password again")
	}
	return parseID(sub)
}

func deviceHash(token string) string {
	sum := sha256.Sum256([]byte(token))
	return hex.EncodeToString(sum[:])
}

// afterPassword decides what a correct password gets: a session, a 2FA
// challenge, or a setup-only session for office roles without 2FA.
func (s *Service) afterPassword(ctx context.Context, user User, deviceToken string) (LoginResult, error) {
	state, err := s.repo.twoFactor(ctx, user.ID)
	if err != nil {
		return LoginResult{}, err
	}
	if state.Enabled {
		if deviceToken != "" {
			if ok, err := s.repo.trustedDevice(ctx, user.ID, deviceHash(deviceToken)); err == nil && ok {
				token, err := s.signToken(user, false)
				return LoginResult{Token: token, User: toDTO(user)}, err
			}
		}
		mfa, err := s.signMFAToken(user.ID)
		if err != nil {
			return LoginResult{}, err
		}
		return LoginResult{MFARequired: true, MFAToken: mfa, User: toDTO(user)}, nil
	}
	setup := RequiresTwoFactor(user.Role)
	token, err := s.signToken(user, setup)
	return LoginResult{Token: token, User: toDTO(user), MFASetupRequired: setup}, err
}

// VerifyLogin completes a 2FA login with an app code or a backup code.
func (s *Service) VerifyLogin(ctx context.Context, mfaToken, code string) (string, UserDTO, error) {
	userID, err := s.parseMFAToken(mfaToken)
	if err != nil {
		return "", UserDTO{}, err
	}
	lockKey := "mfa:" + strconv.FormatInt(userID, 10)
	if s.isLocked(lockKey) {
		return "", UserDTO{}, response.NewAPIError(429, "too many wrong codes; try again in 15 minutes")
	}
	user, err := s.repo.GetByID(ctx, userID)
	if err != nil {
		return "", UserDTO{}, response.NewAPIError(401, "sign-in expired; enter your password again")
	}
	if err := s.checkSecondFactor(ctx, userID, code); err != nil {
		s.recordFailure(lockKey)
		return "", UserDTO{}, err
	}
	s.clearFailures(lockKey)
	token, err := s.signToken(user, false)
	return token, toDTO(user), err
}

// checkSecondFactor accepts a current app code (once) or an unused backup code.
func (s *Service) checkSecondFactor(ctx context.Context, userID int64, code string) error {
	state, err := s.repo.twoFactor(ctx, userID)
	if err != nil {
		return err
	}
	if !state.Enabled {
		return response.NewAPIError(400, "two-factor authentication is not on")
	}
	clean := strings.ReplaceAll(strings.TrimSpace(code), " ", "")
	if len(clean) == totp.Digits {
		secret, err := s.twoFA.open(state.Secret)
		if err != nil {
			return err
		}
		step, ok := totp.Verify(secret, clean, time.Now())
		if !ok {
			return errMFAFailed
		}
		fresh, err := s.repo.claimStep(ctx, userID, step)
		if err != nil {
			return err
		}
		if !fresh {
			return response.NewAPIError(401, "that code was already used; wait for the next one")
		}
		return nil
	}
	ok, err := s.repo.useBackupCode(ctx, userID, s.twoFA.codeHash(clean))
	if err != nil {
		return err
	}
	if !ok {
		return errMFAFailed
	}
	return nil
}

// NewTrustedDevice registers "remember this device" and returns the cookie value.
func (s *Service) NewTrustedDevice(ctx context.Context, userID int64, userAgent string) (string, error) {
	buf := make([]byte, 32)
	if _, err := rand.Read(buf); err != nil {
		return "", err
	}
	token := base64.RawURLEncoding.EncodeToString(buf)
	if err := s.repo.addTrustedDevice(ctx, userID, deviceHash(token), userAgent); err != nil {
		return "", err
	}
	return token, nil
}

// TwoFactorStatus is shown on the Security settings page.
type TwoFactorStatus struct {
	Enabled         bool `json:"enabled"`
	Required        bool `json:"required"`
	BackupCodesLeft int  `json:"backupCodesLeft"`
	TrustedDevices  int  `json:"trustedDevices"`
}

func (s *Service) TwoFactorStatus(ctx context.Context, id Identity) (TwoFactorStatus, error) {
	st := TwoFactorStatus{Required: RequiresTwoFactor(id.Role)}
	err := s.repo.pool.QueryRow(ctx,
		`SELECT u.totp_enabled,
		        (SELECT COUNT(*) FROM user_backup_codes b WHERE b.user_id = u.id AND b.used_at IS NULL)::int,
		        (SELECT COUNT(*) FROM user_trusted_devices d WHERE d.user_id = u.id AND d.expires_at > now())::int
		   FROM users u WHERE u.id = $1`, id.UserID).Scan(&st.Enabled, &st.BackupCodesLeft, &st.TrustedDevices)
	return st, err
}

// TwoFactorSetup is the enrolment QR code (and the key, for typing in).
type TwoFactorSetup struct {
	Secret string `json:"secret"`
	URI    string `json:"uri"`
	QR     string `json:"qr"` // data:image/png;base64,…
}

// BeginSetup creates a pending secret; it takes effect once confirmed.
func (s *Service) BeginSetup(ctx context.Context, id Identity) (TwoFactorSetup, error) {
	user, err := s.repo.GetByID(ctx, id.UserID)
	if err != nil {
		return TwoFactorSetup{}, err
	}
	secret, err := totp.NewSecret()
	if err != nil {
		return TwoFactorSetup{}, err
	}
	sealed, err := s.twoFA.seal(secret)
	if err != nil {
		return TwoFactorSetup{}, err
	}
	if _, err := s.repo.pool.Exec(ctx, `UPDATE users SET totp_pending_secret = $2 WHERE id = $1`, id.UserID, sealed); err != nil {
		return TwoFactorSetup{}, err
	}
	uri := totp.URI(totpIssuer, user.Email, secret)
	img, err := qrDataURL(uri)
	if err != nil {
		return TwoFactorSetup{}, err
	}
	return TwoFactorSetup{Secret: secret, URI: uri, QR: img}, nil
}

func qrDataURL(content string) (string, error) {
	code, err := qr.Encode(content, qr.M, qr.Auto)
	if err != nil {
		return "", err
	}
	code, err = barcode.Scale(code, 240, 240)
	if err != nil {
		return "", err
	}
	var buf bytes.Buffer
	if err := png.Encode(&buf, code); err != nil {
		return "", err
	}
	return "data:image/png;base64," + base64.StdEncoding.EncodeToString(buf.Bytes()), nil
}

// ConfirmSetup turns 2FA on once the app shows a matching code, and returns
// the backup codes (shown once) plus a full session token.
func (s *Service) ConfirmSetup(ctx context.Context, id Identity, code string) ([]string, string, error) {
	state, err := s.repo.twoFactor(ctx, id.UserID)
	if err != nil {
		return nil, "", err
	}
	if state.PendingSecret == "" {
		return nil, "", response.NewAPIError(400, "start the setup again")
	}
	secret, err := s.twoFA.open(state.PendingSecret)
	if err != nil {
		return nil, "", err
	}
	step, ok := totp.Verify(secret, code, time.Now())
	if !ok {
		return nil, "", response.NewAPIError(400, "code does not match; check the phone's time and try the newest code")
	}
	codes, hashes, err := s.newBackupCodes()
	if err != nil {
		return nil, "", err
	}
	tx, err := s.repo.pool.Begin(ctx)
	if err != nil {
		return nil, "", err
	}
	defer func() { _ = tx.Rollback(ctx) }()
	if _, err := tx.Exec(ctx,
		`UPDATE users SET totp_enabled = true, totp_secret = totp_pending_secret, totp_pending_secret = '',
		        totp_last_step = $2, totp_enabled_at = now()
		  WHERE id = $1`, id.UserID, step); err != nil {
		return nil, "", err
	}
	if err := s.repo.replaceBackupCodes(ctx, tx, id.UserID, hashes); err != nil {
		return nil, "", err
	}
	if _, err := tx.Exec(ctx, `DELETE FROM user_trusted_devices WHERE user_id = $1`, id.UserID); err != nil {
		return nil, "", err
	}
	if err := tx.Commit(ctx); err != nil {
		return nil, "", err
	}
	user, err := s.repo.GetByID(ctx, id.UserID)
	if err != nil {
		return nil, "", err
	}
	token, err := s.signToken(user, false)
	return codes, token, err
}

func (s *Service) newBackupCodes() ([]string, []string, error) {
	codes := make([]string, 0, backupCodeCount)
	hashes := make([]string, 0, backupCodeCount)
	for i := 0; i < backupCodeCount; i++ {
		c, err := newBackupCode()
		if err != nil {
			return nil, nil, err
		}
		codes = append(codes, c)
		hashes = append(hashes, s.twoFA.codeHash(c))
	}
	return codes, hashes, nil
}

// RegenerateBackupCodes replaces all backup codes (needs a current code).
func (s *Service) RegenerateBackupCodes(ctx context.Context, id Identity, code string) ([]string, error) {
	if err := s.checkSecondFactor(ctx, id.UserID, code); err != nil {
		return nil, err
	}
	codes, hashes, err := s.newBackupCodes()
	if err != nil {
		return nil, err
	}
	tx, err := s.repo.pool.Begin(ctx)
	if err != nil {
		return nil, err
	}
	defer func() { _ = tx.Rollback(ctx) }()
	if err := s.repo.replaceBackupCodes(ctx, tx, id.UserID, hashes); err != nil {
		return nil, err
	}
	return codes, tx.Commit(ctx)
}

// DisableTwoFactor turns 2FA off for roles where it is optional.
func (s *Service) DisableTwoFactor(ctx context.Context, id Identity, code string) error {
	if RequiresTwoFactor(id.Role) {
		return response.NewAPIError(403, "two-factor authentication is required for your role")
	}
	if err := s.checkSecondFactor(ctx, id.UserID, code); err != nil {
		return err
	}
	return s.clearTwoFactor(ctx, id.UserID)
}

// ForgetDevices signs every "remembered" device out of the 2FA shortcut.
func (s *Service) ForgetDevices(ctx context.Context, id Identity) error {
	_, err := s.repo.pool.Exec(ctx, `DELETE FROM user_trusted_devices WHERE user_id = $1`, id.UserID)
	return err
}

// ResetTwoFactor is the admin recovery for a lost phone: the user sets 2FA
// up again at next sign-in (office roles are required to).
func (s *Service) ResetTwoFactor(ctx context.Context, admin Identity, targetID int64) error {
	if targetID == admin.UserID {
		return response.NewAPIError(400, "ask another admin to reset your own two-factor authentication")
	}
	if _, err := s.repo.GetByID(ctx, targetID); err != nil {
		return response.NewAPIError(404, "user not found")
	}
	return s.clearTwoFactor(ctx, targetID)
}

func (s *Service) clearTwoFactor(ctx context.Context, userID int64) error {
	tx, err := s.repo.pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback(ctx) }()
	for _, q := range []string{
		`UPDATE users SET totp_enabled = false, totp_secret = '', totp_pending_secret = '', totp_enabled_at = NULL WHERE id = $1`,
		`DELETE FROM user_backup_codes WHERE user_id = $1`,
		`DELETE FROM user_trusted_devices WHERE user_id = $1`,
	} {
		if _, err := tx.Exec(ctx, q, userID); err != nil {
			return fmt.Errorf("clear 2fa: %w", err)
		}
	}
	return tx.Commit(ctx)
}
