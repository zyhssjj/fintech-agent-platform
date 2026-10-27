package response

import "net/http"

// 本文件定义全系统统一错误码，业务模块新增错误码应在此集中登记。
type ErrorCode struct {
	Code       int
	Message    string
	HTTPStatus int
}

var (
	OK                 = ErrorCode{Code: 10000, Message: "成功", HTTPStatus: http.StatusOK}
	ErrValidation      = ErrorCode{Code: 10001, Message: "参数校验失败", HTTPStatus: http.StatusBadRequest}
	ErrUnauthorized    = ErrorCode{Code: 10002, Message: "认证失败", HTTPStatus: http.StatusUnauthorized}
	ErrForbidden       = ErrorCode{Code: 10003, Message: "权限不足", HTTPStatus: http.StatusForbidden}
	ErrNotFound        = ErrorCode{Code: 10004, Message: "资源不存在", HTTPStatus: http.StatusNotFound}
	ErrAlreadyExists   = ErrorCode{Code: 10005, Message: "资源已存在", HTTPStatus: http.StatusConflict}
	ErrBusiness        = ErrorCode{Code: 10006, Message: "业务规则冲突", HTTPStatus: http.StatusUnprocessableEntity}
	ErrTooManyRequests = ErrorCode{Code: 10007, Message: "请求过于频繁", HTTPStatus: http.StatusTooManyRequests}
	ErrInternal        = ErrorCode{Code: 10008, Message: "服务器内部错误", HTTPStatus: http.StatusInternalServerError}
	ErrUnavailable     = ErrorCode{Code: 10009, Message: "服务不可用", HTTPStatus: http.StatusServiceUnavailable}
)
