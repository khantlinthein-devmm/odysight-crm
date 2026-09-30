package auth

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"sort"
	"sync"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/odysight/crm/pkg/response"
)

// Editable role permissions. The built-in sets in rbac.go are the defaults;
// Settings → Roles & permissions can replace a role's set, stored in
// role_permission_overrides and cached here. SUPER_ADMIN always keeps every
// permission so an edit can never lock the owner out.

var overrides = struct {
	sync.RWMutex
	m map[Role]map[Permission]struct{}
}{m: map[Role]map[Permission]struct{}{}}

// effectiveSet is the permission set a role has right now.
func effectiveSet(role Role) (map[Permission]struct{}, bool) {
	if role != RoleSuperAdmin {
		overrides.RLock()
		set, ok := overrides.m[role]
		overrides.RUnlock()
		if ok {
			return set, true
		}
	}
	set, ok := rolePermissions[role]
	return set, ok
}

// AllPermissions lists every permission the API knows, sorted.
func AllPermissions() []Permission {
	out := make([]Permission, 0, len(rolePermissions[RoleSuperAdmin]))
	for p := range rolePermissions[RoleSuperAdmin] {
		out = append(out, p)
	}
	sort.Slice(out, func(i, j int) bool { return out[i] < out[j] })
	return out
}

func sortedKeys(set map[Permission]struct{}) []string {
	out := make([]string, 0, len(set))
	for p := range set {
		out = append(out, string(p))
	}
	sort.Strings(out)
	return out
}

// EffectivePermissions is what the role can do now (for the web app).
func EffectivePermissions(role Role) []string {
	set, _ := effectiveSet(role)
	return sortedKeys(set)
}

// EditableRoles are the roles whose permissions can be changed.
var EditableRoles = []Role{RoleAdmin, RoleManager, RoleDispatch, RoleAccountant, RoleCleaner}

func editable(role Role) bool {
	for _, r := range EditableRoles {
		if r == role {
			return true
		}
	}
	return false
}

// PermissionStore loads and saves overrides.
type PermissionStore struct {
	pool *pgxpool.Pool
}

func NewPermissionStore(pool *pgxpool.Pool) *PermissionStore {
	return &PermissionStore{pool: pool}
}

// Load replaces the cache with what the database holds.
func (s *PermissionStore) Load(ctx context.Context) error {
	rows, err := s.pool.Query(ctx, `SELECT role, permissions FROM role_permission_overrides`)
	if err != nil {
		return fmt.Errorf("load role permissions: %w", err)
	}
	defer rows.Close()
	known := rolePermissions[RoleSuperAdmin]
	next := map[Role]map[Permission]struct{}{}
	for rows.Next() {
		var role string
		var list []string
		if err := rows.Scan(&role, &list); err != nil {
			return err
		}
		if !editable(Role(role)) {
			continue
		}
		set := map[Permission]struct{}{}
		for _, p := range list {
			if _, ok := known[Permission(p)]; ok { // ignore permissions that no longer exist
				set[Permission(p)] = struct{}{}
			}
		}
		next[Role(role)] = set
	}
	if err := rows.Err(); err != nil {
		return err
	}
	overrides.Lock()
	overrides.m = next
	overrides.Unlock()
	return nil
}

// Run keeps the cache fresh (other API instances may save changes).
func (s *PermissionStore) Run(ctx context.Context) {
	t := time.NewTicker(30 * time.Second)
	defer t.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-t.C:
			if err := s.Load(ctx); err != nil {
				slog.Warn("reload role permissions", "error", err)
			}
		}
	}
}

func (s *PermissionStore) save(ctx context.Context, role Role, list []string, by int64) error {
	if _, err := s.pool.Exec(ctx,
		`INSERT INTO role_permission_overrides (role, permissions, updated_by, updated_at)
		 VALUES ($1, $2, $3, now())
		 ON CONFLICT (role) DO UPDATE SET permissions = EXCLUDED.permissions,
		     updated_by = EXCLUDED.updated_by, updated_at = now()`, string(role), list, by); err != nil {
		return fmt.Errorf("save role permissions: %w", err)
	}
	return s.Load(ctx)
}

func (s *PermissionStore) reset(ctx context.Context, role Role) error {
	if _, err := s.pool.Exec(ctx, `DELETE FROM role_permission_overrides WHERE role = $1`, string(role)); err != nil {
		return fmt.Errorf("reset role permissions: %w", err)
	}
	return s.Load(ctx)
}

// --- HTTP ---

type roleView struct {
	Role        string   `json:"role"`
	Editable    bool     `json:"editable"`
	Customized  bool     `json:"customized"`
	Permissions []string `json:"permissions"`
	Defaults    []string `json:"defaults"`
}

type rolesResponse struct {
	AllPermissions []string   `json:"allPermissions"`
	Roles          []roleView `json:"roles"`
}

func (s *PermissionStore) view() rolesResponse {
	all := AllPermissions()
	out := rolesResponse{AllPermissions: make([]string, 0, len(all))}
	for _, p := range all {
		out.AllPermissions = append(out.AllPermissions, string(p))
	}
	overrides.RLock()
	defer overrides.RUnlock()
	for _, role := range append([]Role{RoleSuperAdmin}, EditableRoles...) {
		_, custom := overrides.m[role]
		set := rolePermissions[role]
		if custom {
			set = overrides.m[role]
		}
		out.Roles = append(out.Roles, roleView{
			Role: string(role), Editable: editable(role), Customized: custom,
			Permissions: sortedKeys(set), Defaults: sortedKeys(rolePermissions[role]),
		})
	}
	return out
}

// List handles GET /roles
func (s *PermissionStore) List(w http.ResponseWriter, r *http.Request) {
	response.JSON(w, http.StatusOK, s.view())
}

// Save handles PATCH /roles/{role} {"permissions": [...]}
func (s *PermissionStore) Save(w http.ResponseWriter, r *http.Request) {
	role := Role(chi.URLParam(r, "role"))
	if !editable(role) {
		response.Error(w, http.StatusBadRequest, "this role's permissions cannot be changed")
		return
	}
	var req struct {
		Permissions []string `json:"permissions"`
	}
	dec := json.NewDecoder(http.MaxBytesReader(w, r.Body, 64<<10))
	dec.DisallowUnknownFields()
	if err := dec.Decode(&req); err != nil {
		response.Error(w, http.StatusBadRequest, "invalid JSON body")
		return
	}
	known := rolePermissions[RoleSuperAdmin]
	seen := map[string]bool{}
	clean := []string{}
	for _, p := range req.Permissions {
		if _, ok := known[Permission(p)]; !ok {
			response.Error(w, http.StatusBadRequest, "unknown permission: "+p)
			return
		}
		if !seen[p] {
			seen[p] = true
			clean = append(clean, p)
		}
	}
	sort.Strings(clean)
	id, _ := IdentityFromContext(r.Context())
	if err := s.save(r.Context(), role, clean, id.UserID); err != nil {
		response.HandleError(w, r, err)
		return
	}
	response.JSON(w, http.StatusOK, s.view())
}

// Reset handles DELETE /roles/{role} — back to the built-in defaults.
func (s *PermissionStore) Reset(w http.ResponseWriter, r *http.Request) {
	role := Role(chi.URLParam(r, "role"))
	if !editable(role) {
		response.Error(w, http.StatusBadRequest, "this role's permissions cannot be changed")
		return
	}
	if err := s.reset(r.Context(), role); err != nil {
		response.HandleError(w, r, err)
		return
	}
	response.JSON(w, http.StatusOK, s.view())
}

// RoleRoutes serves /roles: anyone who can read settings sees the matrix;
// changing it needs users.manage (Super admin by default, and locked there).
func RoleRoutes(s *PermissionStore, az *Authorizer) chi.Router {
	r := chi.NewRouter()
	r.With(az.Require(PermSettingsRead)).Get("/", s.List)
	r.With(az.Require(PermUsersManage)).Patch("/{role}", s.Save)
	r.With(az.Require(PermUsersManage)).Delete("/{role}", s.Reset)
	return r
}
