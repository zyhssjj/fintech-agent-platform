package approval

import (
	"errors"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"

	"wfagent/internal/base/middleware"
	"wfagent/internal/base/response"
)

type Handler struct {
	service *Service
}

func RegisterRoutes(v1 *gin.RouterGroup, db *gorm.DB) {
	handler := &Handler{service: NewService(db)}
	group := v1.Group("/approvals")
	group.GET("", middleware.RequireRole("SYS_ADMIN", "FUND_MANAGER", "INVEST_MANAGER", "POST_INVEST_MANAGER", "RISK_MANAGER", "LEGAL_MANAGER", "FINANCE_MANAGER", "OPS_MANAGER", "REVIEWER"), handler.ListApprovals)
	group.POST("", middleware.RequireRole("SYS_ADMIN", "FUND_MANAGER", "INVEST_MANAGER", "POST_INVEST_MANAGER", "RISK_MANAGER", "LEGAL_MANAGER", "FINANCE_MANAGER", "OPS_MANAGER", "REVIEWER"), handler.CreateApproval)
	group.GET("/:approval_id", middleware.RequireRole("SYS_ADMIN", "FUND_MANAGER", "INVEST_MANAGER", "POST_INVEST_MANAGER", "RISK_MANAGER", "LEGAL_MANAGER", "FINANCE_MANAGER", "OPS_MANAGER", "REVIEWER"), handler.GetApproval)
	group.POST("/:approval_id/action", middleware.RequireRole("SYS_ADMIN", "FUND_MANAGER", "INVEST_MANAGER", "POST_INVEST_MANAGER", "RISK_MANAGER", "LEGAL_MANAGER", "FINANCE_MANAGER", "OPS_MANAGER", "REVIEWER"), handler.ActionApproval)
}

// ListApprovals godoc
// @Summary 获取审批列表
// @Tags Approvals
// @Produce json
// @Security BearerAuth
// @Param status query string false "审批状态"
// @Param business_type query string false "业务类型"
// @Param page query int false "页码"
// @Param page_size query int false "每页条数"
// @Success 200 {object} response.Body
// @Router /v1/approvals [get]
func (h *Handler) ListApprovals(c *gin.Context) {
	var query ApprovalQuery
	if err := c.ShouldBindQuery(&query); err != nil {
		response.BindError(c, err)
		return
	}
	userID, _ := c.Get("user_id")
	roles, _ := c.Get("roles")
	data, err := h.service.ListApprovals(c.Request.Context(), query, toString(userID), toStrings(roles))
	if err != nil {
		writeError(c, err)
		return
	}
	response.Success(c, data)
}

// CreateApproval godoc
// @Summary 发起审批
// @Tags Approvals
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param body body CreateApprovalRequest true "审批申请"
// @Success 200 {object} response.Body
// @Router /v1/approvals [post]
func (h *Handler) CreateApproval(c *gin.Context) {
	var req CreateApprovalRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BindError(c, err)
		return
	}
	userID, _ := c.Get("user_id")
	username, _ := c.Get("username")
	data, err := h.service.CreateApproval(c.Request.Context(), req, toString(userID), toString(username))
	if err != nil {
		writeError(c, err)
		return
	}
	response.Success(c, data)
}

// GetApproval godoc
// @Summary 获取审批详情
// @Tags Approvals
// @Produce json
// @Security BearerAuth
// @Param approval_id path string true "审批编号"
// @Success 200 {object} response.Body
// @Router /v1/approvals/{approval_id} [get]
func (h *Handler) GetApproval(c *gin.Context) {
	data, err := h.service.GetApproval(c.Request.Context(), c.Param("approval_id"))
	if err != nil {
		writeError(c, err)
		return
	}
	response.Success(c, data)
}

// ActionApproval godoc
// @Summary 审批操作
// @Tags Approvals
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param approval_id path string true "审批编号"
// @Param body body ApprovalActionRequest true "审批动作"
// @Success 200 {object} response.Body
// @Router /v1/approvals/{approval_id}/action [post]
func (h *Handler) ActionApproval(c *gin.Context) {
	var req ApprovalActionRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BindError(c, err)
		return
	}
	userID, _ := c.Get("user_id")
	username, _ := c.Get("username")
	roles, _ := c.Get("roles")
	data, err := h.service.ActionApproval(c.Request.Context(), c.Param("approval_id"), req, toString(userID), toString(username), toStrings(roles))
	if err != nil {
		writeError(c, err)
		return
	}
	response.Success(c, data)
}

func writeError(c *gin.Context, err error) {
	switch {
	case errors.Is(err, ErrApprovalNotFound), errors.Is(err, ErrFlowNotFound):
		response.Error(c, response.ErrNotFound, err.Error())
	case errors.Is(err, ErrInvalidAction):
		response.Error(c, response.ErrValidation, err.Error())
	default:
		response.Error(c, response.ErrInternal, err.Error())
	}
}
