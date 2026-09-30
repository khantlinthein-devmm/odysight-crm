package auth

import (
	"regexp"
	"testing"
)

func TestRequiresTwoFactor(t *testing.T) {
	for _, r := range []Role{RoleSuperAdmin, RoleAdmin, RoleManager, RoleAccountant, RoleDispatch} {
		if !RequiresTwoFactor(r) {
			t.Errorf("%s must require 2FA", r)
		}
	}
	if RequiresTwoFactor(RoleCleaner) {
		t.Error("cleaners opt in")
	}
}

func TestSecretSealing(t *testing.T) {
	k := newTwoFactorKeys([]byte("server-secret"))
	sealed, err := k.seal("JBSWY3DPEHPK3PXP")
	if err != nil {
		t.Fatal(err)
	}
	if sealed == "JBSWY3DPEHPK3PXP" {
		t.Fatal("secret stored in clear")
	}
	if got, err := k.open(sealed); err != nil || got != "JBSWY3DPEHPK3PXP" {
		t.Fatalf("open = %q, %v", got, err)
	}
	other := newTwoFactorKeys([]byte("different-secret"))
	if _, err := other.open(sealed); err == nil {
		t.Fatal("secret opened with the wrong key")
	}
}

func TestBackupCodes(t *testing.T) {
	format := regexp.MustCompile(`^[a-hjkmnp-z2-9]{5}-[a-hjkmnp-z2-9]{5}$`)
	seen := map[string]bool{}
	for i := 0; i < 200; i++ {
		c, err := newBackupCode()
		if err != nil {
			t.Fatal(err)
		}
		if !format.MatchString(c) {
			t.Fatalf("bad code format %q", c)
		}
		seen[c] = true
	}
	if len(seen) < 200 {
		t.Fatal("backup codes repeat")
	}
	k := newTwoFactorKeys([]byte("s"))
	if k.codeHash("abcde-fghjk") != k.codeHash(" ABCDEFGHJK ") {
		t.Fatal("backup codes must match regardless of case, dash and spaces")
	}
	if newTwoFactorKeys([]byte("s2")).codeHash("abcde-fghjk") == k.codeHash("abcde-fghjk") {
		t.Fatal("backup code hashes must depend on the server secret")
	}
}
