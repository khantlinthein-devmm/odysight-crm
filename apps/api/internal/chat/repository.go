package chat

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
)

var (
	ErrNotFound    = errors.New("conversation not found")
	ErrNotAllowed  = errors.New("you cannot message this person")
	ErrUserMissing = errors.New("user not found")
)

type Repository struct {
	pool *pgxpool.Pool
}

func NewRepository(pool *pgxpool.Pool) *Repository {
	return &Repository{pool: pool}
}

const officeRoles = `('SUPER_ADMIN', 'ADMIN', 'MANAGER', 'DISPATCH')`

// Contacts lists everyone the user may message.
func (r *Repository) Contacts(ctx context.Context, me int64, office bool) ([]Contact, error) {
	query := `SELECT u.id, u.name, u.role FROM users u WHERE u.id <> $1`
	if !office {
		query += ` AND (u.role IN ` + officeRoles + ` OR EXISTS (
			SELECT 1 FROM chat_group_members g1 JOIN chat_group_members g2 ON g1.group_id = g2.group_id
			 WHERE g1.user_id = $1 AND g2.user_id = u.id))`
	}
	rows, err := r.pool.Query(ctx, query+` ORDER BY u.name`, me)
	if err != nil {
		return nil, fmt.Errorf("list chat contacts: %w", err)
	}
	defer rows.Close()
	out := []Contact{}
	for rows.Next() {
		var c Contact
		if err := rows.Scan(&c.UserID, &c.Name, &c.Role); err != nil {
			return nil, fmt.Errorf("scan chat contact: %w", err)
		}
		out = append(out, c)
	}
	return out, rows.Err()
}

// UserRole returns a user's name and role.
func (r *Repository) UserRole(ctx context.Context, id int64) (string, string, error) {
	var name, role string
	err := r.pool.QueryRow(ctx, `SELECT name, role FROM users WHERE id = $1`, id).Scan(&name, &role)
	if errors.Is(err, pgx.ErrNoRows) {
		return "", "", ErrUserMissing
	}
	if err != nil {
		return "", "", fmt.Errorf("load user %d: %w", id, err)
	}
	return name, role, nil
}

// ShareGroup reports whether two users are in at least one common group.
func (r *Repository) ShareGroup(ctx context.Context, a, b int64) (bool, error) {
	var ok bool
	err := r.pool.QueryRow(ctx,
		`SELECT EXISTS (SELECT 1 FROM chat_group_members g1 JOIN chat_group_members g2 ON g1.group_id = g2.group_id
		                 WHERE g1.user_id = $1 AND g2.user_id = $2)`, a, b).Scan(&ok)
	if err != nil {
		return false, fmt.Errorf("check shared group: %w", err)
	}
	return ok, nil
}

// OpenConversation returns the pair's conversation, creating it if needed.
func (r *Repository) OpenConversation(ctx context.Context, a, b int64) (int64, error) {
	lo, hi := a, b
	if lo > hi {
		lo, hi = hi, lo
	}
	var id int64
	err := r.pool.QueryRow(ctx,
		`INSERT INTO chat_conversations (user_a, user_b) VALUES ($1, $2)
		 ON CONFLICT (user_a, user_b) DO UPDATE SET user_a = EXCLUDED.user_a
		 RETURNING id`, lo, hi).Scan(&id)
	if err != nil {
		return 0, fmt.Errorf("open conversation: %w", err)
	}
	return id, nil
}

// memberOf is the SQL condition "user p may use conversation c": one of the
// pair in a direct conversation, or a member of the group for a group chat.
func memberOf(p string) string {
	return `((c.group_id IS NULL AND ` + p + ` IN (c.user_a, c.user_b)) OR EXISTS (
		SELECT 1 FROM chat_group_members gm WHERE gm.group_id = c.group_id AND gm.user_id = ` + p + `))`
}

// myRead is user p's read marker in conversation c.
func myRead(p string) string {
	return `CASE WHEN c.group_id IS NOT NULL
		THEN COALESCE((SELECT r.last_read FROM chat_reads r WHERE r.conversation_id = c.id AND r.user_id = ` + p + `), 0)
		WHEN c.user_a = ` + p + ` THEN c.last_read_a ELSE c.last_read_b END`
}

// Thread is who else is in a conversation.
type Thread struct {
	// Other is the other person of a direct conversation.
	Other int64
	// GroupID and GroupName are set for a group conversation; Members are
	// the other members.
	GroupID   int64
	GroupName string
	Members   []int64
}

// IsGroup reports whether this is a group conversation.
func (t Thread) IsGroup() bool { return t.GroupID != 0 }

// Recipients is everyone but the caller.
func (t Thread) Recipients() []int64 {
	if t.IsGroup() {
		return t.Members
	}
	return []int64{t.Other}
}

// Thread loads a conversation the caller belongs to, or ErrNotFound.
func (r *Repository) Thread(ctx context.Context, conversationID, me int64) (Thread, error) {
	var t Thread
	err := r.pool.QueryRow(ctx,
		`SELECT COALESCE(c.group_id, 0), COALESCE(g.name, ''),
		        COALESCE(CASE WHEN c.user_a = $2 THEN c.user_b ELSE c.user_a END, 0)
		   FROM chat_conversations c LEFT JOIN chat_groups g ON g.id = c.group_id
		  WHERE c.id = $1 AND `+memberOf("$2"), conversationID, me).Scan(&t.GroupID, &t.GroupName, &t.Other)
	if errors.Is(err, pgx.ErrNoRows) {
		return Thread{}, ErrNotFound
	}
	if err != nil {
		return Thread{}, fmt.Errorf("load conversation %d: %w", conversationID, err)
	}
	if !t.IsGroup() {
		return t, nil
	}
	t.Other = 0
	rows, err := r.pool.Query(ctx,
		`SELECT user_id FROM chat_group_members WHERE group_id = $1 AND user_id <> $2`, t.GroupID, me)
	if err != nil {
		return Thread{}, fmt.Errorf("load group members: %w", err)
	}
	defer rows.Close()
	for rows.Next() {
		var id int64
		if err := rows.Scan(&id); err != nil {
			return Thread{}, err
		}
		t.Members = append(t.Members, id)
	}
	return t, rows.Err()
}

var conversationSelect = `
	SELECT c.id, COALESCE(o.id, 0), COALESCE(o.name, g.name, ''), COALESCE(o.role, ''), c.last_message_at,
	       COALESCE(m.kind, ''), COALESCE(m.body, ''), COALESCE(m.sender_id = $1, false), COALESCE(mu.name, ''),
	       (SELECT COUNT(*) FROM chat_messages x
	         WHERE x.conversation_id = c.id AND x.sender_id IS DISTINCT FROM $1
	           AND x.id > ` + myRead("$1") + `)::int,
	       CASE WHEN c.group_id IS NOT NULL THEN 0 WHEN c.user_a = $1 THEN c.last_read_b ELSE c.last_read_a END,
	       c.group_id, (SELECT array_agg(gm2.user_id ORDER BY gm2.user_id) FROM chat_group_members gm2 WHERE gm2.group_id = c.group_id)
	  FROM chat_conversations c
	  LEFT JOIN users o ON c.group_id IS NULL AND o.id = CASE WHEN c.user_a = $1 THEN c.user_b ELSE c.user_a END
	  LEFT JOIN chat_groups g ON g.id = c.group_id
	  LEFT JOIN LATERAL (SELECT kind, body, sender_id FROM chat_messages
	                      WHERE conversation_id = c.id ORDER BY id DESC LIMIT 1) m ON true
	  LEFT JOIN users mu ON mu.id = m.sender_id
	 WHERE ` + memberOf("$1")

func scanConversation(row pgx.Row) (Conversation, error) {
	var c Conversation
	var groupID *int64
	var members []int64
	err := row.Scan(&c.ID, &c.Other.UserID, &c.Other.Name, &c.Other.Role, &c.LastMessageAt,
		&c.LastKind, &c.LastBody, &c.LastFromMe, &c.LastSender, &c.Unread, &c.OtherLastRead, &groupID, &members)
	if groupID != nil {
		if members == nil {
			members = []int64{}
		}
		c.Group = &GroupRef{ID: *groupID, Members: len(members), MemberIDs: members}
	} else {
		c.LastSender = ""
	}
	return c, err
}

// Conversations lists the user's group chats and the direct conversations
// that have messages, newest first.
func (r *Repository) Conversations(ctx context.Context, me int64) ([]Conversation, error) {
	rows, err := r.pool.Query(ctx, conversationSelect+
		` AND (c.last_message_at IS NOT NULL OR c.group_id IS NOT NULL)
		  ORDER BY COALESCE(c.last_message_at, c.created_at) DESC LIMIT 200`, me)
	if err != nil {
		return nil, fmt.Errorf("list conversations: %w", err)
	}
	defer rows.Close()
	out := []Conversation{}
	for rows.Next() {
		c, err := scanConversation(rows)
		if err != nil {
			return nil, fmt.Errorf("scan conversation: %w", err)
		}
		out = append(out, c)
	}
	return out, rows.Err()
}

func (r *Repository) Conversation(ctx context.Context, id, me int64) (Conversation, error) {
	c, err := scanConversation(r.pool.QueryRow(ctx, conversationSelect+` AND c.id = $2`, me, id))
	if errors.Is(err, pgx.ErrNoRows) {
		return Conversation{}, ErrNotFound
	}
	if err != nil {
		return Conversation{}, fmt.Errorf("load conversation %d: %w", id, err)
	}
	return c, nil
}

// UnreadTotal counts unread messages across all of the user's conversations.
func (r *Repository) UnreadTotal(ctx context.Context, me int64) (int, error) {
	var n int
	err := r.pool.QueryRow(ctx,
		`SELECT COUNT(*) FROM chat_messages x JOIN chat_conversations c ON c.id = x.conversation_id
		  WHERE `+memberOf("$1")+` AND x.sender_id IS DISTINCT FROM $1
		    AND x.id > `+myRead("$1"), me).Scan(&n)
	if err != nil {
		return 0, fmt.Errorf("count unread: %w", err)
	}
	return n, nil
}

const messageColumns = `id, conversation_id, sender_id,
	COALESCE((SELECT u.name FROM users u WHERE u.id = chat_messages.sender_id), ''),
	kind, body, file_name, mime_type, duration_ms, file_expired, created_at`

func scanMessage(row pgx.Row) (Message, string, error) {
	var m Message
	var file string
	err := row.Scan(&m.ID, &m.ConversationID, &m.SenderID, &m.SenderName, &m.Kind, &m.Body, &file, &m.MimeType, &m.DurationMs, &m.Expired, &m.CreatedAt)
	if file != "" {
		m.FileURL = "/api/v1/chat/files/" + file
	}
	return m, file, err
}

// Messages pages a conversation: after>0 returns newer messages (polling),
// before>0 older ones (scroll back), otherwise the latest page. Always in
// chronological order.
func (r *Repository) Messages(ctx context.Context, conversationID, after, before int64, limit int) ([]Message, error) {
	var rows pgx.Rows
	var err error
	switch {
	case after > 0:
		rows, err = r.pool.Query(ctx, `SELECT `+messageColumns+` FROM chat_messages
			WHERE conversation_id = $1 AND id > $2 ORDER BY id LIMIT $3`, conversationID, after, limit)
	case before > 0:
		rows, err = r.pool.Query(ctx, `SELECT * FROM (SELECT `+messageColumns+` FROM chat_messages
			WHERE conversation_id = $1 AND id < $2 ORDER BY id DESC LIMIT $3) t ORDER BY id`, conversationID, before, limit)
	default:
		rows, err = r.pool.Query(ctx, `SELECT * FROM (SELECT `+messageColumns+` FROM chat_messages
			WHERE conversation_id = $1 ORDER BY id DESC LIMIT $2) t ORDER BY id`, conversationID, limit)
	}
	if err != nil {
		return nil, fmt.Errorf("list messages: %w", err)
	}
	defer rows.Close()
	out := []Message{}
	for rows.Next() {
		m, _, err := scanMessage(rows)
		if err != nil {
			return nil, fmt.Errorf("scan message: %w", err)
		}
		out = append(out, m)
	}
	return out, rows.Err()
}

// NewMessage is a message to store.
type NewMessage struct {
	ConversationID int64
	SenderID       int64
	Kind           string
	Body           string
	FileName       string
	MimeType       string
	DurationMs     int
}

// AddMessage stores a message, bumps the conversation and marks it read for
// the sender, in one transaction.
func (r *Repository) AddMessage(ctx context.Context, m NewMessage) (Message, error) {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return Message{}, fmt.Errorf("begin tx: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()
	msg, _, err := scanMessage(tx.QueryRow(ctx,
		`INSERT INTO chat_messages (conversation_id, sender_id, kind, body, file_name, mime_type, duration_ms)
		 VALUES ($1, $2, $3, $4, $5, $6, $7) RETURNING `+messageColumns,
		m.ConversationID, m.SenderID, m.Kind, m.Body, m.FileName, m.MimeType, m.DurationMs))
	if err != nil {
		return Message{}, fmt.Errorf("insert message: %w", err)
	}
	if _, err := tx.Exec(ctx,
		`UPDATE chat_conversations SET last_message_at = $2,
			last_read_a = CASE WHEN user_a = $3 THEN $4 ELSE last_read_a END,
			last_read_b = CASE WHEN user_b = $3 THEN $4 ELSE last_read_b END
		 WHERE id = $1`, m.ConversationID, msg.CreatedAt, m.SenderID, msg.ID); err != nil {
		return Message{}, fmt.Errorf("bump conversation: %w", err)
	}
	if err := groupRead(ctx, tx, m.ConversationID, m.SenderID, msg.ID); err != nil {
		return Message{}, err
	}
	if err := tx.Commit(ctx); err != nil {
		return Message{}, fmt.Errorf("commit message: %w", err)
	}
	return msg, nil
}

// MarkRead moves the user's read marker forward (never back).
func (r *Repository) MarkRead(ctx context.Context, conversationID, me, messageID int64) error {
	_, err := r.pool.Exec(ctx,
		`UPDATE chat_conversations SET
			last_read_a = CASE WHEN user_a = $2 THEN GREATEST(last_read_a, $3) ELSE last_read_a END,
			last_read_b = CASE WHEN user_b = $2 THEN GREATEST(last_read_b, $3) ELSE last_read_b END
		 WHERE id = $1 AND $2 IN (user_a, user_b)`, conversationID, me, messageID)
	if err != nil {
		return fmt.Errorf("mark read: %w", err)
	}
	return groupRead(ctx, r.pool, conversationID, me, messageID)
}

type execer interface {
	Exec(ctx context.Context, sql string, args ...any) (pgconn.CommandTag, error)
}

// groupRead moves a member's read marker in a group conversation; it does
// nothing for a direct conversation.
func groupRead(ctx context.Context, db execer, conversationID, me, messageID int64) error {
	_, err := db.Exec(ctx,
		`INSERT INTO chat_reads (conversation_id, user_id, last_read)
		 SELECT c.id, $2, $3 FROM chat_conversations c WHERE c.id = $1 AND c.group_id IS NOT NULL
		 ON CONFLICT (conversation_id, user_id) DO UPDATE SET last_read = GREATEST(chat_reads.last_read, EXCLUDED.last_read)`,
		conversationID, me, messageID)
	if err != nil {
		return fmt.Errorf("mark group read: %w", err)
	}
	return nil
}

// FileAccess returns the stored mime type when the user may download the
// file (they are in the conversation it was sent to).
func (r *Repository) FileAccess(ctx context.Context, fileName string, me int64) (string, error) {
	var mime string
	err := r.pool.QueryRow(ctx,
		`SELECT m.mime_type FROM chat_messages m JOIN chat_conversations c ON c.id = m.conversation_id
		  WHERE m.file_name = $1 AND `+memberOf("$2")+` LIMIT 1`, fileName, me).Scan(&mime)
	if errors.Is(err, pgx.ErrNoRows) {
		return "", ErrNotFound
	}
	if err != nil {
		return "", fmt.Errorf("check file access: %w", err)
	}
	return mime, nil
}

// Groups

func (r *Repository) Groups(ctx context.Context) ([]Group, error) {
	rows, err := r.pool.Query(ctx,
		`SELECT g.id, g.name, g.created_at, u.id, u.name, u.role
		   FROM chat_groups g
		   LEFT JOIN chat_group_members m ON m.group_id = g.id
		   LEFT JOIN users u ON u.id = m.user_id
		  ORDER BY g.name, g.id, u.name`)
	if err != nil {
		return nil, fmt.Errorf("list chat groups: %w", err)
	}
	defer rows.Close()
	out := []Group{}
	for rows.Next() {
		var g Group
		var uid *int64
		var uname, urole *string
		if err := rows.Scan(&g.ID, &g.Name, &g.CreatedAt, &uid, &uname, &urole); err != nil {
			return nil, fmt.Errorf("scan chat group: %w", err)
		}
		if len(out) == 0 || out[len(out)-1].ID != g.ID {
			g.Members = []Contact{}
			out = append(out, g)
		}
		if uid != nil {
			last := &out[len(out)-1]
			last.Members = append(last.Members, Contact{UserID: *uid, Name: *uname, Role: *urole})
		}
	}
	return out, rows.Err()
}

// SaveGroup creates (id=0) or renames a group and replaces its members.
func (r *Repository) SaveGroup(ctx context.Context, id int64, name string, members []int64, createdBy int64) (int64, error) {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return 0, fmt.Errorf("begin tx: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()
	if id == 0 {
		if err := tx.QueryRow(ctx, `INSERT INTO chat_groups (name, created_by) VALUES ($1, $2) RETURNING id`,
			strings.TrimSpace(name), createdBy).Scan(&id); err != nil {
			return 0, fmt.Errorf("create chat group: %w", err)
		}
		if _, err := tx.Exec(ctx, `INSERT INTO chat_conversations (group_id) VALUES ($1)`, id); err != nil {
			return 0, fmt.Errorf("create group conversation: %w", err)
		}
	} else {
		tag, err := tx.Exec(ctx, `UPDATE chat_groups SET name = $2 WHERE id = $1`, id, strings.TrimSpace(name))
		if err != nil {
			return 0, fmt.Errorf("rename chat group: %w", err)
		}
		if tag.RowsAffected() == 0 {
			return 0, ErrNotFound
		}
		if _, err := tx.Exec(ctx, `DELETE FROM chat_group_members WHERE group_id = $1`, id); err != nil {
			return 0, fmt.Errorf("clear chat group members: %w", err)
		}
	}
	if len(members) > 0 {
		if _, err := tx.Exec(ctx,
			`INSERT INTO chat_group_members (group_id, user_id)
			 SELECT $1, u.id FROM users u WHERE u.id = ANY($2) ON CONFLICT DO NOTHING`, id, members); err != nil {
			return 0, fmt.Errorf("add chat group members: %w", err)
		}
		// New members start with the history read, so joining a busy group
		// does not light up hundreds of unread messages.
		if _, err := tx.Exec(ctx,
			`INSERT INTO chat_reads (conversation_id, user_id, last_read)
			 SELECT c.id, gm.user_id, COALESCE((SELECT MAX(x.id) FROM chat_messages x WHERE x.conversation_id = c.id), 0)
			   FROM chat_conversations c JOIN chat_group_members gm ON gm.group_id = c.group_id
			  WHERE c.group_id = $1
			 ON CONFLICT (conversation_id, user_id) DO NOTHING`, id); err != nil {
			return 0, fmt.Errorf("start group read markers: %w", err)
		}
	}
	if err := tx.Commit(ctx); err != nil {
		return 0, fmt.Errorf("commit chat group: %w", err)
	}
	return id, nil
}

// DeleteGroup removes a group and its group chat. It returns the voice files
// of the deleted messages so the caller can remove them from disk.
func (r *Repository) DeleteGroup(ctx context.Context, id int64) ([]string, error) {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return nil, fmt.Errorf("begin tx: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()
	rows, err := tx.Query(ctx,
		`SELECT m.file_name FROM chat_messages m JOIN chat_conversations c ON c.id = m.conversation_id
		  WHERE c.group_id = $1 AND m.file_name <> ''`, id)
	if err != nil {
		return nil, fmt.Errorf("list group chat files: %w", err)
	}
	files, err := pgx.CollectRows(rows, pgx.RowTo[string])
	if err != nil {
		return nil, fmt.Errorf("scan group chat files: %w", err)
	}
	tag, err := tx.Exec(ctx, `DELETE FROM chat_groups WHERE id = $1`, id)
	if err != nil {
		return nil, fmt.Errorf("delete chat group: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return nil, ErrNotFound
	}
	if err := tx.Commit(ctx); err != nil {
		return nil, fmt.Errorf("commit group delete: %w", err)
	}
	return files, nil
}

// Push subscriptions

type PushSub struct {
	ID       int64
	Endpoint string
	P256dh   string
	Auth     string
}

func (r *Repository) SaveSubscription(ctx context.Context, userID int64, endpoint, p256dh, authKey string) error {
	_, err := r.pool.Exec(ctx,
		`INSERT INTO push_subscriptions (user_id, endpoint, p256dh, auth) VALUES ($1, $2, $3, $4)
		 ON CONFLICT (endpoint) DO UPDATE SET user_id = EXCLUDED.user_id, p256dh = EXCLUDED.p256dh,
		     auth = EXCLUDED.auth, created_at = now()`, userID, endpoint, p256dh, authKey)
	if err != nil {
		return fmt.Errorf("save push subscription: %w", err)
	}
	return nil
}

func (r *Repository) DeleteSubscription(ctx context.Context, userID int64, endpoint string) error {
	_, err := r.pool.Exec(ctx, `DELETE FROM push_subscriptions WHERE user_id = $1 AND endpoint = $2`, userID, endpoint)
	return err
}

func (r *Repository) DropSubscription(ctx context.Context, id int64) {
	_, _ = r.pool.Exec(ctx, `DELETE FROM push_subscriptions WHERE id = $1`, id)
}

func (r *Repository) Subscriptions(ctx context.Context, userID int64) ([]PushSub, error) {
	rows, err := r.pool.Query(ctx, `SELECT id, endpoint, p256dh, auth FROM push_subscriptions WHERE user_id = $1`, userID)
	if err != nil {
		return nil, fmt.Errorf("list push subscriptions: %w", err)
	}
	defer rows.Close()
	out := []PushSub{}
	for rows.Next() {
		var s PushSub
		if err := rows.Scan(&s.ID, &s.Endpoint, &s.P256dh, &s.Auth); err != nil {
			return nil, err
		}
		out = append(out, s)
	}
	return out, rows.Err()
}

// VAPIDKey returns the stored VAPID private key, creating it with gen on
// first use (concurrent starts agree on one key via the id = 1 row).
func (r *Repository) VAPIDKey(ctx context.Context, gen func() (string, error)) (string, error) {
	var key string
	err := r.pool.QueryRow(ctx, `SELECT private_key FROM push_vapid_keys WHERE id = 1`).Scan(&key)
	if err == nil {
		return key, nil
	}
	if !errors.Is(err, pgx.ErrNoRows) {
		return "", fmt.Errorf("load vapid key: %w", err)
	}
	fresh, err := gen()
	if err != nil {
		return "", err
	}
	if _, err := r.pool.Exec(ctx,
		`INSERT INTO push_vapid_keys (id, private_key) VALUES (1, $1) ON CONFLICT (id) DO NOTHING`, fresh); err != nil {
		return "", fmt.Errorf("store vapid key: %w", err)
	}
	err = r.pool.QueryRow(ctx, `SELECT private_key FROM push_vapid_keys WHERE id = 1`).Scan(&key)
	return key, err
}

// ExpiredFile is a stored chat file due for deletion.
type ExpiredFile struct {
	MessageID int64
	FileName  string
}

// FilesOlderThan lists stored chat files sent before cutoff (a batch).
func (r *Repository) FilesOlderThan(ctx context.Context, cutoff time.Time, limit int) ([]ExpiredFile, error) {
	rows, err := r.pool.Query(ctx,
		`SELECT id, file_name FROM chat_messages
		  WHERE file_name <> '' AND created_at < $1 ORDER BY created_at LIMIT $2`, cutoff, limit)
	if err != nil {
		return nil, fmt.Errorf("list expired chat files: %w", err)
	}
	defer rows.Close()
	out := []ExpiredFile{}
	for rows.Next() {
		var f ExpiredFile
		if err := rows.Scan(&f.MessageID, &f.FileName); err != nil {
			return nil, err
		}
		out = append(out, f)
	}
	return out, rows.Err()
}

// MarkFileExpired clears a message's file after it was deleted from disk.
func (r *Repository) MarkFileExpired(ctx context.Context, messageID int64) error {
	_, err := r.pool.Exec(ctx,
		`UPDATE chat_messages SET file_name = '', file_expired = true WHERE id = $1`, messageID)
	return err
}
