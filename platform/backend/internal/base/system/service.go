package system

import (
	"context"
	"errors"
	"fmt"

	"gorm.io/gorm"

	"wfagent/internal/base/database"
)

var (
	ErrUnavailable = errors.New("system authorization storage is unavailable")
	ErrNotFound    = errors.New("system authorization resource not found")
)

type RoleView struct {
	ID          uint64   `json:"id"`
	RoleCode    string   `json:"role_code"`
	RoleName    string   `json:"role_name"`
	Description string   `json:"description"`
	Status      string   `json:"status"`
	Permissions []string `json:"permissions"`
}

type PermissionView struct {
	ID          uint64 `json:"id"`
	PermCode    string `json:"perm_code"`
	PermName    string `json:"perm_name"`
	Resource    string `json:"resource"`
	Action      string `json:"action"`
	Description string `json:"description"`
}

type Service struct{ db *gorm.DB }

func NewService(db *gorm.DB) *Service { return &Service{db: db} }

func (s *Service) ListRoles(ctx context.Context) ([]RoleView, error) {
	if s.db == nil {
		return nil, ErrUnavailable
	}
	var roles []database.SysRole
	if err := s.db.WithContext(ctx).Preload("Permissions", "is_deleted = false").Preload("Permissions.Permission", "is_deleted = false").Where("is_deleted = false").Order("role_code").Find(&roles).Error; err != nil {
		return nil, err
	}
	result := make([]RoleView, 0, len(roles))
	for _, role := range roles {
		view := RoleView{ID: role.ID, RoleCode: role.RoleCode, RoleName: role.RoleName, Description: role.Description, Status: role.Status, Permissions: []string{}}
		for _, assignment := range role.Permissions {
			if assignment.Permission.PermCode != "" {
				view.Permissions = append(view.Permissions, assignment.Permission.PermCode)
			}
		}
		result = append(result, view)
	}
	return result, nil
}

func (s *Service) ListPermissions(ctx context.Context) ([]PermissionView, error) {
	if s.db == nil {
		return nil, ErrUnavailable
	}
	var permissions []database.SysPermission
	if err := s.db.WithContext(ctx).Where("is_deleted = false").Order("perm_code").Find(&permissions).Error; err != nil {
		return nil, err
	}
	result := make([]PermissionView, 0, len(permissions))
	for _, permission := range permissions {
		result = append(result, PermissionView{ID: permission.ID, PermCode: permission.PermCode, PermName: permission.PermName, Resource: permission.Resource, Action: permission.Action, Description: permission.Description})
	}
	return result, nil
}

func (s *Service) ReplaceUserRoles(ctx context.Context, userID uint64, roleCodes []string) error {
	if s.db == nil {
		return ErrUnavailable
	}
	return s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var user database.SysUser
		if err := tx.Where("id = ? AND is_deleted = false", userID).First(&user).Error; err != nil {
			return normalizeNotFound(err)
		}
		roles, err := findRoles(tx, roleCodes)
		if err != nil {
			return err
		}
		if err := tx.Where("user_id = ?", userID).Delete(&database.SysUserRole{}).Error; err != nil {
			return err
		}
		for _, role := range roles {
			if err := tx.Create(&database.SysUserRole{UserID: userID, RoleID: role.ID}).Error; err != nil {
				return err
			}
		}
		return nil
	})
}

func (s *Service) ReplaceRolePermissions(ctx context.Context, roleID uint64, permissionCodes []string) error {
	if s.db == nil {
		return ErrUnavailable
	}
	return s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var role database.SysRole
		if err := tx.Where("id = ? AND is_deleted = false", roleID).First(&role).Error; err != nil {
			return normalizeNotFound(err)
		}
		permissions, err := findPermissions(tx, permissionCodes)
		if err != nil {
			return err
		}
		if err := tx.Where("role_id = ?", roleID).Delete(&database.SysRolePerm{}).Error; err != nil {
			return err
		}
		for _, permission := range permissions {
			if err := tx.Create(&database.SysRolePerm{RoleID: roleID, PermissionID: permission.ID}).Error; err != nil {
				return err
			}
		}
		return nil
	})
}

func findRoles(tx *gorm.DB, codes []string) ([]database.SysRole, error) {
	if len(codes) == 0 {
		return []database.SysRole{}, nil
	}
	var roles []database.SysRole
	if err := tx.Where("role_code IN ? AND is_deleted = false AND status = ?", codes, "active").Find(&roles).Error; err != nil {
		return nil, err
	}
	if len(roles) != len(unique(codes)) {
		return nil, fmt.Errorf("%w: one or more role codes", ErrNotFound)
	}
	return roles, nil
}

func findPermissions(tx *gorm.DB, codes []string) ([]database.SysPermission, error) {
	if len(codes) == 0 {
		return []database.SysPermission{}, nil
	}
	var permissions []database.SysPermission
	if err := tx.Where("perm_code IN ? AND is_deleted = false", codes).Find(&permissions).Error; err != nil {
		return nil, err
	}
	if len(permissions) != len(unique(codes)) {
		return nil, fmt.Errorf("%w: one or more permission codes", ErrNotFound)
	}
	return permissions, nil
}

func unique(values []string) []string {
	seen := make(map[string]struct{}, len(values))
	result := make([]string, 0, len(values))
	for _, value := range values {
		if value == "" {
			continue
		}
		if _, ok := seen[value]; !ok {
			seen[value] = struct{}{}
			result = append(result, value)
		}
	}
	return result
}

func normalizeNotFound(err error) error {
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return ErrNotFound
	}
	return err
}
