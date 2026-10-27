package database

import (
	"errors"
	"os"
	"strings"

	"golang.org/x/crypto/bcrypt"
	"gorm.io/datatypes"
	"gorm.io/gorm"
)

const (
	DefaultAdminUsername = "admin"
	DefaultAdminPassword = "Admin@123456"
)

var legacyBrokenAdminPasswordHashes = map[string]struct{}{
	"$2a$10$CUGqJ5ksLKdbw18nGX7one88/ApRlKVmoi6JrnZSM8ciJ2WF30Sfy": {},
}

type seedRole struct {
	Code        string
	Name        string
	Description string
	Permissions []seedPermission
}

type seedUser struct {
	Username string
	Password string
	RealName string
	Email    string
	RoleCode string
}

type seedPermission struct {
	Code     string
	Name     string
	Resource string
	Action   string
}

var baseRoles = []seedRole{
	{Code: "SYS_ADMIN", Name: "系统管理员", Description: "用户管理、角色权限配置、系统配置、审计日志", Permissions: []seedPermission{{Code: "*", Name: "全部权限", Resource: "*", Action: "*"}}},
	{Code: "FUND_MANAGER", Name: "基金管理人", Description: "基金设立、LP管理、募资管理、合规审查", Permissions: []seedPermission{{Code: "fund:manage", Name: "募资基金管理", Resource: "fund", Action: "manage"}, {Code: "lp:manage", Name: "LP管理", Resource: "lp", Action: "manage"}, {Code: "audit:read", Name: "读取审计日志", Resource: "audit", Action: "read"}}},
	{Code: "INVEST_MANAGER", Name: "投资经理", Description: "项目筛选、尽职调查、投决发起、投资执行", Permissions: []seedPermission{{Code: "invest:manage", Name: "投资项目管理", Resource: "invest", Action: "manage"}, {Code: "meeting:read", Name: "会议查看", Resource: "meeting", Action: "read"}, {Code: "audit:read", Name: "读取审计日志", Resource: "audit", Action: "read"}}},
	{Code: "POST_INVEST_MANAGER", Name: "投后经理", Description: "投后监控、风险预警、报告管理、退出管理", Permissions: []seedPermission{{Code: "project:manage", Name: "投后项目管理", Resource: "project", Action: "manage"}, {Code: "exit:manage", Name: "退出管理", Resource: "exit", Action: "manage"}, {Code: "audit:read", Name: "读取审计日志", Resource: "audit", Action: "read"}}},
	{Code: "RISK_MANAGER", Name: "风控经理", Description: "风控审核、风险评级、缓释措施", Permissions: []seedPermission{{Code: "risk:manage", Name: "风控管理", Resource: "risk", Action: "manage"}, {Code: "invest:read", Name: "投资数据查看", Resource: "invest", Action: "read"}, {Code: "audit:read", Name: "读取审计日志", Resource: "audit", Action: "read"}}},
	{Code: "LEGAL_MANAGER", Name: "法务经理", Description: "合同审查、协议起草、条款对比", Permissions: []seedPermission{{Code: "legal:manage", Name: "法务管理", Resource: "legal", Action: "manage"}, {Code: "contract:manage", Name: "合同管理", Resource: "contract", Action: "manage"}, {Code: "audit:read", Name: "读取审计日志", Resource: "audit", Action: "read"}}},
	{Code: "FINANCE_MANAGER", Name: "财务经理", Description: "财务报表、估值计算、收益分配", Permissions: []seedPermission{{Code: "finance:manage", Name: "财务管理", Resource: "finance", Action: "manage"}, {Code: "valuation:manage", Name: "估值管理", Resource: "valuation", Action: "manage"}, {Code: "audit:read", Name: "读取审计日志", Resource: "audit", Action: "read"}}},
	{Code: "OPS_MANAGER", Name: "运营经理", Description: "会议管理、知识库维护、内部协同", Permissions: []seedPermission{{Code: "meeting:manage", Name: "会议管理", Resource: "meeting", Action: "manage"}, {Code: "knowledge:manage", Name: "知识库管理", Resource: "knowledge", Action: "manage"}, {Code: "audit:read", Name: "读取审计日志", Resource: "audit", Action: "read"}}},
	{Code: "REVIEWER", Name: "复核人员", Description: "AI生成内容复核、投资建议复核、估值复核", Permissions: []seedPermission{{Code: "review:manage", Name: "复核管理", Resource: "review", Action: "manage"}, {Code: "audit:read", Name: "读取审计日志", Resource: "audit", Action: "read"}}},
	{Code: "LP", Name: "LP（投资者）", Description: "查看基金报告、接收信披通知、参与合伙人会议", Permissions: []seedPermission{{Code: "lp_portal:read", Name: "LP门户查看", Resource: "lp_portal", Action: "read"}, {Code: "audit:read", Name: "读取审计日志", Resource: "audit", Action: "read"}}},
	{Code: "GUEST", Name: "访客", Description: "查看公开信息（需授权）", Permissions: []seedPermission{{Code: "public:read", Name: "公开信息查看", Resource: "public", Action: "read"}}},
}

var legacyRoleCodes = []string{
	"ORG_ADMIN",
	"COMPLIANCE_MANAGER",
	"PROJECT_MANAGER",
	"MEETING_MANAGER",
	"EXIT_MANAGER",
	"AGENT_MANAGER",
	"VIEWER",
}

var baseUsers = []seedUser{
	{Username: DefaultAdminUsername, Password: DefaultAdminPassword, RealName: "系统管理员", Email: "admin@example.com", RoleCode: "SYS_ADMIN"},
	{Username: "fund_manager", Password: "Fund@123456", RealName: "基金管理人", Email: "fund_manager@example.com", RoleCode: "FUND_MANAGER"},
	{Username: "invest_manager", Password: "Invest@123456", RealName: "投资经理", Email: "invest_manager@example.com", RoleCode: "INVEST_MANAGER"},
	{Username: "post_invest_manager", Password: "PostInvest@123456", RealName: "投后经理", Email: "post_invest_manager@example.com", RoleCode: "POST_INVEST_MANAGER"},
	{Username: "risk_manager", Password: "Risk@123456", RealName: "风控经理", Email: "risk_manager@example.com", RoleCode: "RISK_MANAGER"},
	{Username: "legal_manager", Password: "Legal@123456", RealName: "法务经理", Email: "legal_manager@example.com", RoleCode: "LEGAL_MANAGER"},
	{Username: "finance_manager", Password: "Finance@123456", RealName: "财务经理", Email: "finance_manager@example.com", RoleCode: "FINANCE_MANAGER"},
	{Username: "ops_manager", Password: "Ops@123456", RealName: "运营经理", Email: "ops_manager@example.com", RoleCode: "OPS_MANAGER"},
	{Username: "reviewer", Password: "Reviewer@123456", RealName: "复核人员", Email: "reviewer@example.com", RoleCode: "REVIEWER"},
	{Username: "lp_user", Password: "LP@123456", RealName: "LP投资者", Email: "lp_user@example.com", RoleCode: "LP"},
	{Username: "guest", Password: "Guest@123456", RealName: "访客", Email: "guest@example.com", RoleCode: "GUEST"},
}

// SeedInitialAdmin creates the bootstrap admin account and the base role matrix.
func SeedInitialAdmin(db *gorm.DB) error {
	if db == nil {
		return nil
	}

	return db.Transaction(func(tx *gorm.DB) error {
		dept, err := firstOrCreateDepartment(tx)
		if err != nil {
			return err
		}

		roles, err := seedRolesAndPermissions(tx)
		if err != nil {
			return err
		}
		if err := retireLegacyRoles(tx); err != nil {
			return err
		}

		for _, seed := range baseUsers {
			user, err := firstOrCreateSeedUser(tx, seed, dept.ID)
			if err != nil {
				return err
			}
			role, ok := roles[seed.RoleCode]
			if !ok {
				return errors.New("seed role is missing: " + seed.RoleCode)
			}
			if err := firstOrCreateUserRole(tx, user.ID, role.ID); err != nil {
				return err
			}
		}
		return nil
	})
}

func firstOrCreateDepartment(tx *gorm.DB) (SysDepartment, error) {
	dept := SysDepartment{}
	err := tx.Where("dept_code = ?", "ROOT").First(&dept).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		dept = SysDepartment{DeptCode: "ROOT", DeptName: "System Administration", SortOrder: 0, Status: "active"}
		return dept, tx.Create(&dept).Error
	}
	return dept, err
}

func seedRolesAndPermissions(tx *gorm.DB) (map[string]SysRole, error) {
	roles := make(map[string]SysRole, len(baseRoles))
	for _, item := range baseRoles {
		role, err := firstOrCreateRole(tx, item)
		if err != nil {
			return nil, err
		}
		roles[item.Code] = role
		for _, permItem := range item.Permissions {
			permission, err := firstOrCreatePermission(tx, permItem)
			if err != nil {
				return nil, err
			}
			if err := firstOrCreateRolePermission(tx, role.ID, permission.ID); err != nil {
				return nil, err
			}
		}
	}
	return roles, nil
}

func firstOrCreateRole(tx *gorm.DB, item seedRole) (SysRole, error) {
	role := SysRole{}
	err := tx.Where("role_code = ?", item.Code).First(&role).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		role = SysRole{RoleCode: item.Code, RoleName: item.Name, Description: item.Description, Status: "active"}
		return role, tx.Create(&role).Error
	}
	if err == nil {
		if err := tx.Model(&SysRole{}).Where("id = ?", role.ID).Updates(map[string]any{
			"role_name":   item.Name,
			"description": item.Description,
			"status":      "active",
			"is_deleted":  false,
		}).Error; err != nil {
			return role, err
		}
		role.RoleName = item.Name
		role.Description = item.Description
		role.Status = "active"
		role.IsDeleted = false
	}
	return role, err
}

func retireLegacyRoles(tx *gorm.DB) error {
	if len(legacyRoleCodes) == 0 {
		return nil
	}
	return tx.Model(&SysRole{}).
		Where("role_code IN ?", legacyRoleCodes).
		Updates(map[string]any{"status": "inactive", "is_deleted": true}).Error
}

func firstOrCreatePermission(tx *gorm.DB, item seedPermission) (SysPermission, error) {
	permission := SysPermission{}
	err := tx.Where("perm_code = ?", item.Code).First(&permission).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		permission = SysPermission{PermCode: item.Code, PermName: item.Name, Resource: item.Resource, Action: item.Action, Description: item.Name}
		return permission, tx.Create(&permission).Error
	}
	if err == nil {
		if err := tx.Model(&SysPermission{}).Where("id = ?", permission.ID).Updates(map[string]any{
			"perm_name":   item.Name,
			"resource":    item.Resource,
			"action":      item.Action,
			"description": item.Name,
			"is_deleted":  false,
		}).Error; err != nil {
			return permission, err
		}
		permission.PermName = item.Name
		permission.Resource = item.Resource
		permission.Action = item.Action
		permission.Description = item.Name
		permission.IsDeleted = false
	}
	return permission, err
}

func firstOrCreateSeedUser(tx *gorm.DB, seed seedUser, deptID uint64) (SysUser, error) {
	user := SysUser{}
	err := tx.Where("username = ?", seed.Username).First(&user).Error
	if err == nil {
		updates := map[string]any{
			"real_name":     seed.RealName,
			"email":         seed.Email,
			"status":        "active",
			"is_deleted":    false,
			"department_id": deptID,
		}
		if shouldResetSeedPassword(seed.Username) || shouldRepairSeedPassword(seed.Username, user.Password) {
			hashedPassword, err := hashPassword(seed.Password)
			if err != nil {
				return user, err
			}
			updates["password"] = hashedPassword
			user.Password = hashedPassword
		}
		if err := tx.Model(&SysUser{}).Where("id = ?", user.ID).Updates(updates).Error; err != nil {
			return user, err
		}
		return user, nil
	}
	if !errors.Is(err, gorm.ErrRecordNotFound) {
		return user, err
	}

	hashedPassword, err := hashPassword(seed.Password)
	if err != nil {
		return user, err
	}

	user = SysUser{
		Username:     seed.Username,
		Password:     hashedPassword,
		RealName:     seed.RealName,
		Email:        seed.Email,
		Status:       "active",
		DepartmentID: &deptID,
	}
	return user, tx.Create(&user).Error
}

func hashDefaultAdminPassword() (string, error) {
	return hashPassword(DefaultAdminPassword)
}

func hashPassword(password string) (string, error) {
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	return string(hashedPassword), err
}

func shouldResetAdminPassword() bool {
	value := strings.ToLower(strings.TrimSpace(os.Getenv("BOOTSTRAP_ADMIN_RESET_PASSWORD")))
	return value == "true" || value == "1" || value == "yes"
}

func shouldResetSeedPassword(username string) bool {
	if username == DefaultAdminUsername {
		return shouldResetAdminPassword()
	}
	value := strings.ToLower(strings.TrimSpace(os.Getenv("BOOTSTRAP_DEMO_USERS_RESET_PASSWORD")))
	return value == "true" || value == "1" || value == "yes"
}

func shouldRepairSeedPassword(username string, passwordHash string) bool {
	password := ""
	for _, seed := range baseUsers {
		if seed.Username == username {
			password = seed.Password
			break
		}
	}
	if password == "" {
		return false
	}
	if username == DefaultAdminUsername {
		return shouldRepairAdminPassword(passwordHash)
	}
	if err := bcrypt.CompareHashAndPassword([]byte(passwordHash), []byte(password)); err == nil {
		return false
	}
	return strings.TrimSpace(passwordHash) == ""
}

func shouldRepairAdminPassword(passwordHash string) bool {
	if _, ok := legacyBrokenAdminPasswordHashes[passwordHash]; ok {
		return true
	}
	if err := bcrypt.CompareHashAndPassword([]byte(passwordHash), []byte(DefaultAdminPassword)); err == nil {
		return false
	}
	return strings.TrimSpace(passwordHash) == ""
}

func firstOrCreateRolePermission(tx *gorm.DB, roleID uint64, permissionID uint64) error {
	var rolePerm SysRolePerm
	err := tx.Where("role_id = ? AND permission_id = ?", roleID, permissionID).First(&rolePerm).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return tx.Create(&SysRolePerm{RoleID: roleID, PermissionID: permissionID}).Error
	}
	return err
}

func firstOrCreateUserRole(tx *gorm.DB, userID uint64, roleID uint64) error {
	var userRole SysUserRole
	err := tx.Where("user_id = ? AND role_id = ?", userID, roleID).First(&userRole).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return tx.Create(&SysUserRole{
			UserID:       userID,
			RoleID:       roleID,
			FundScope:    datatypes.JSON([]byte("{}")),
			ProjectScope: datatypes.JSON([]byte("{}")),
		}).Error
	}
	return err
}
