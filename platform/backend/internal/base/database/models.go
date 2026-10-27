package database

import (
	"time"

	"gorm.io/datatypes"
	"gorm.io/gorm"
)

type BaseModel struct {
	ID         uint64    `gorm:"primaryKey;autoIncrement;column:id" json:"id"`
	CreateTime time.Time `gorm:"column:create_time;not null;autoCreateTime" json:"create_time"`
	UpdateTime time.Time `gorm:"column:update_time;not null;autoUpdateTime" json:"update_time"`
	IsDeleted  bool      `gorm:"column:is_deleted;not null;default:false;index" json:"is_deleted"`
}

type SysUser struct {
	BaseModel
	Username      string     `gorm:"column:username;type:varchar(64);not null;uniqueIndex:idx_sys_user_username" json:"username"`
	Password      string     `gorm:"column:password;type:varchar(255);not null" json:"-"`
	RealName      string     `gorm:"column:real_name;type:varchar(64);not null;default:''" json:"real_name"`
	Email         string     `gorm:"column:email;type:varchar(128);not null;default:'';index" json:"email"`
	Phone         string     `gorm:"column:phone;type:varchar(32);not null;default:'';index" json:"phone"`
	Avatar        string     `gorm:"column:avatar;type:varchar(512);not null;default:''" json:"avatar"`
	Status        string     `gorm:"column:status;type:varchar(32);not null;default:'active';index" json:"status"`
	DepartmentID  *uint64    `gorm:"column:department_id;index" json:"department_id"`
	LastLoginTime *time.Time `gorm:"column:last_login_time" json:"last_login_time"`
	LastLoginIP   string     `gorm:"column:last_login_ip;type:varchar(64);not null;default:''" json:"last_login_ip"`

	Department *SysDepartment `gorm:"foreignKey:DepartmentID;constraint:OnUpdate:CASCADE,OnDelete:SET NULL" json:"department,omitempty"`
	Roles      []SysUserRole  `gorm:"foreignKey:UserID" json:"roles,omitempty"`
}

func (SysUser) TableName() string {
	return "sys_user"
}

type SysRole struct {
	BaseModel
	RoleCode    string `gorm:"column:role_code;type:varchar(64);not null;uniqueIndex:idx_sys_role_code" json:"role_code"`
	RoleName    string `gorm:"column:role_name;type:varchar(64);not null" json:"role_name"`
	Description string `gorm:"column:description;type:varchar(255);not null;default:''" json:"description"`
	Status      string `gorm:"column:status;type:varchar(32);not null;default:'active';index" json:"status"`

	Users       []SysUserRole `gorm:"foreignKey:RoleID" json:"users,omitempty"`
	Permissions []SysRolePerm `gorm:"foreignKey:RoleID" json:"permissions,omitempty"`
}

func (SysRole) TableName() string {
	return "sys_role"
}

type SysUserRole struct {
	BaseModel
	UserID       uint64         `gorm:"column:user_id;not null;uniqueIndex:idx_sys_user_role" json:"user_id"`
	RoleID       uint64         `gorm:"column:role_id;not null;uniqueIndex:idx_sys_user_role" json:"role_id"`
	FundScope    datatypes.JSON `gorm:"column:fund_scope;type:jsonb;not null;default:'{}'" json:"fund_scope"`
	ProjectScope datatypes.JSON `gorm:"column:project_scope;type:jsonb;not null;default:'{}'" json:"project_scope"`

	User SysUser `gorm:"foreignKey:UserID;constraint:OnUpdate:CASCADE,OnDelete:CASCADE" json:"user,omitempty"`
	Role SysRole `gorm:"foreignKey:RoleID;constraint:OnUpdate:CASCADE,OnDelete:CASCADE" json:"role,omitempty"`
}

func (SysUserRole) TableName() string {
	return "sys_user_role"
}

type SysPermission struct {
	BaseModel
	PermCode    string `gorm:"column:perm_code;type:varchar(128);not null;uniqueIndex:idx_sys_permission_code" json:"perm_code"`
	PermName    string `gorm:"column:perm_name;type:varchar(128);not null" json:"perm_name"`
	Resource    string `gorm:"column:resource;type:varchar(64);not null;index" json:"resource"`
	Action      string `gorm:"column:action;type:varchar(64);not null;index" json:"action"`
	Description string `gorm:"column:description;type:varchar(255);not null;default:''" json:"description"`

	Roles []SysRolePerm `gorm:"foreignKey:PermissionID" json:"roles,omitempty"`
}

func (SysPermission) TableName() string {
	return "sys_permission"
}

type SysRolePerm struct {
	BaseModel
	RoleID       uint64 `gorm:"column:role_id;not null;uniqueIndex:idx_sys_role_perm" json:"role_id"`
	PermissionID uint64 `gorm:"column:permission_id;not null;uniqueIndex:idx_sys_role_perm" json:"permission_id"`

	Role       SysRole       `gorm:"foreignKey:RoleID;constraint:OnUpdate:CASCADE,OnDelete:CASCADE" json:"role,omitempty"`
	Permission SysPermission `gorm:"foreignKey:PermissionID;constraint:OnUpdate:CASCADE,OnDelete:CASCADE" json:"permission,omitempty"`
}

func (SysRolePerm) TableName() string {
	return "sys_role_perm"
}

type SysDepartment struct {
	BaseModel
	ParentID     *uint64 `gorm:"column:parent_id;index" json:"parent_id"`
	DeptCode     string  `gorm:"column:dept_code;type:varchar(64);not null;uniqueIndex:idx_sys_department_code" json:"dept_code"`
	DeptName     string  `gorm:"column:dept_name;type:varchar(128);not null" json:"dept_name"`
	LeaderUserID *uint64 `gorm:"column:leader_user_id;index" json:"leader_user_id"`
	SortOrder    int     `gorm:"column:sort_order;not null;default:0" json:"sort_order"`
	Status       string  `gorm:"column:status;type:varchar(32);not null;default:'active';index" json:"status"`

	Parent   *SysDepartment  `gorm:"foreignKey:ParentID;constraint:OnUpdate:CASCADE,OnDelete:SET NULL" json:"parent,omitempty"`
	Children []SysDepartment `gorm:"foreignKey:ParentID" json:"children,omitempty"`
}

func (SysDepartment) TableName() string {
	return "sys_department"
}

type OperationLog struct {
	BaseModel
	RequestID string         `gorm:"column:request_id;type:varchar(64);not null;default:'';index" json:"request_id"`
	UserID    string         `gorm:"column:user_id;type:varchar(64);not null;default:'';index" json:"user_id"`
	Username  string         `gorm:"column:username;type:varchar(64);not null;default:'';index" json:"username"`
	Method    string         `gorm:"column:method;type:varchar(16);not null" json:"method"`
	Path      string         `gorm:"column:path;type:varchar(512);not null;index" json:"path"`
	ClientIP  string         `gorm:"column:client_ip;type:varchar(64);not null;default:'';index" json:"client_ip"`
	UserAgent string         `gorm:"column:user_agent;type:varchar(512);not null;default:''" json:"user_agent"`
	Status    int            `gorm:"column:status;not null;index" json:"status"`
	LatencyMS int64          `gorm:"column:latency_ms;not null;default:0" json:"latency_ms"`
	Params    datatypes.JSON `gorm:"column:params;type:jsonb;not null;default:'{}'" json:"params"`
}

func (OperationLog) TableName() string {
	return "com_operation_log"
}

func AutoMigrateBase(db *gorm.DB) error {
	if db == nil {
		return nil
	}
	return db.AutoMigrate(
		&SysDepartment{},
		&SysUser{},
		&SysRole{},
		&SysPermission{},
		&SysUserRole{},
		&SysRolePerm{},
		&OperationLog{},
	)
}
