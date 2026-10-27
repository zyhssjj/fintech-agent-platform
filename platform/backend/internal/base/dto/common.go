package dto

// 本文件定义跨模块复用的通用传输结构，如分页、用户、文件和实体引用。
type JSONMap map[string]any

type PaginationQuery struct {
	Page     int `form:"page" json:"page"`
	PageSize int `form:"page_size" json:"page_size"`
}

type PaginationInfo struct {
	Page     int   `json:"page"`
	PageSize int   `json:"page_size"`
	Total    int64 `json:"total"`
}

type PageResult[T any] struct {
	List       []T            `json:"list"`
	Pagination PaginationInfo `json:"pagination"`
}

type UserInfoResponse struct {
	UserID        string   `json:"user_id"`
	Username      string   `json:"username"`
	RealName      string   `json:"real_name"`
	Email         string   `json:"email"`
	Phone         string   `json:"phone"`
	Avatar        string   `json:"avatar"`
	Roles         []string `json:"roles"`
	Permissions   []string `json:"permissions"`
	Departments   []string `json:"departments"`
	LastLoginTime string   `json:"last_login_time"`
	LastLoginIP   string   `json:"last_login_ip"`
}

type EntityRef struct {
	EntityID   string `json:"entity_id"`
	EntityType string `json:"entity_type"`
	Name       string `json:"name,omitempty"`
}

type PersonRef struct {
	UserID   string `json:"user_id,omitempty"`
	Name     string `json:"name"`
	Email    string `json:"email,omitempty"`
	Phone    string `json:"phone,omitempty"`
	Position string `json:"position,omitempty"`
}

type FileRef struct {
	FileID   string `json:"file_id"`
	FileName string `json:"file_name"`
	FileURL  string `json:"file_url"`
	FileSize int64  `json:"file_size"`
	MimeType string `json:"mime_type,omitempty"`
	MD5      string `json:"md5,omitempty"`
}
