package system

import (
	"context"
	"errors"
	"testing"
)

func TestServiceRequiresPersistentAuthorizationStore(t *testing.T) {
	service := NewService(nil)
	ctx := context.Background()
	if _, err := service.ListRoles(ctx); !errors.Is(err, ErrUnavailable) {
		t.Fatalf("ListRoles error = %v, want ErrUnavailable", err)
	}
	if err := service.ReplaceUserRoles(ctx, 1, []string{"VIEWER"}); !errors.Is(err, ErrUnavailable) {
		t.Fatalf("ReplaceUserRoles error = %v, want ErrUnavailable", err)
	}
	if err := service.ReplaceRolePermissions(ctx, 1, []string{"base:read"}); !errors.Is(err, ErrUnavailable) {
		t.Fatalf("ReplaceRolePermissions error = %v, want ErrUnavailable", err)
	}
}

func TestUniqueCodes(t *testing.T) {
	got := unique([]string{"VIEWER", "", "VIEWER", "MANAGER", "MANAGER"})
	want := []string{"VIEWER", "MANAGER"}
	if len(got) != len(want) {
		t.Fatalf("unique length = %d, want %d", len(got), len(want))
	}
	for i, code := range want {
		if got[i] != code {
			t.Fatalf("unique[%d] = %q, want %q", i, got[i], code)
		}
	}
}
