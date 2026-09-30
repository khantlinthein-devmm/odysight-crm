package chat

import (
	"testing"

	"github.com/odysight/crm/internal/auth"
)

func TestCanMessage(t *testing.T) {
	cases := []struct {
		a, b  auth.Role
		share bool
		want  bool
	}{
		{auth.RoleCleaner, auth.RoleCleaner, false, false},
		{auth.RoleCleaner, auth.RoleCleaner, true, true},
		{auth.RoleCleaner, auth.RoleDispatch, false, true},
		{auth.RoleAdmin, auth.RoleCleaner, false, true},
		{auth.RoleAccountant, auth.RoleCleaner, false, false},
		{auth.RoleAccountant, auth.RoleManager, false, true},
	}
	for _, c := range cases {
		if got := CanMessage(c.a, c.b, c.share); got != c.want {
			t.Errorf("CanMessage(%s, %s, share=%v) = %v, want %v", c.a, c.b, c.share, got, c.want)
		}
	}
}
