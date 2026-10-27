package response

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

// 本文件封装统一 JSON 响应格式，所有 handler 禁止直接拼接返回结构。
type Body struct {
	Code      int    `json:"code"`
	Message   string `json:"message"`
	Data      any    `json:"data,omitempty"`
	RequestID string `json:"request_id,omitempty"`
}

type Pagination struct {
	Page     int   `json:"page"`
	PageSize int   `json:"page_size"`
	Total    int64 `json:"total"`
}

type PageData struct {
	List       any        `json:"list"`
	Pagination Pagination `json:"pagination"`
}

func Success(c *gin.Context, data any) {
	c.JSON(http.StatusOK, Body{
		Code:      OK.Code,
		Message:   OK.Message,
		Data:      data,
		RequestID: requestID(c),
	})
}

func Created(c *gin.Context, data any) {
	c.JSON(http.StatusCreated, Body{
		Code:      OK.Code,
		Message:   OK.Message,
		Data:      data,
		RequestID: requestID(c),
	})
}

func NoContent(c *gin.Context) {
	c.Status(http.StatusNoContent)
}

func Page(c *gin.Context, list any, pagination Pagination) {
	Success(c, PageData{List: list, Pagination: pagination})
}

func Error(c *gin.Context, code ErrorCode, message ...string) {
	msg := code.Message
	if len(message) > 0 && message[0] != "" {
		msg = message[0]
	}
	c.AbortWithStatusJSON(code.HTTPStatus, Body{
		Code:      code.Code,
		Message:   msg,
		RequestID: requestID(c),
	})
}

func BindError(c *gin.Context, err error) {
	if err == nil {
		Error(c, ErrValidation)
		return
	}
	Error(c, ErrValidation, err.Error())
}

func requestID(c *gin.Context) string {
	if value, ok := c.Get("request_id"); ok {
		if requestID, ok := value.(string); ok {
			return requestID
		}
	}
	return ""
}
