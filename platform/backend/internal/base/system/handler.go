package system

import (
	"errors"
	"strconv"

	"github.com/gin-gonic/gin"

	"wfagent/internal/base/response"
)

type replaceCodesRequest struct {
	Codes []string `json:"codes"`
}

type Handler struct{ service *Service }

func NewHandler(service *Service) *Handler { return &Handler{service: service} }

// ListRoles godoc
// @Summary List roles with assigned permissions
// @Tags System
// @Produce json
// @Security BearerAuth
// @Success 200 {object} response.Body{data=[]RoleView}
// @Router /v1/system/roles [get]
func (h *Handler) ListRoles(c *gin.Context) {
	roles, err := h.service.ListRoles(c.Request.Context())
	handleError(c, err)
	if err == nil {
		response.Success(c, roles)
	}
}

// ListPermissions godoc
// @Summary List assignable permissions
// @Tags System
// @Produce json
// @Security BearerAuth
// @Success 200 {object} response.Body{data=[]PermissionView}
// @Router /v1/system/permissions [get]
func (h *Handler) ListPermissions(c *gin.Context) {
	permissions, err := h.service.ListPermissions(c.Request.Context())
	handleError(c, err)
	if err == nil {
		response.Success(c, permissions)
	}
}

// ReplaceUserRoles godoc
// @Summary Replace all roles for a user
// @Tags System
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param user_id path int true "User ID"
// @Param body body replaceCodesRequest true "Role codes"
// @Success 200 {object} response.Body
// @Router /v1/system/users/{user_id}/roles [put]
func (h *Handler) ReplaceUserRoles(c *gin.Context) {
	userID, ok := pathID(c)
	if !ok {
		return
	}
	var req replaceCodesRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BindError(c, err)
		return
	}
	err := h.service.ReplaceUserRoles(c.Request.Context(), userID, unique(req.Codes))
	handleError(c, err)
	if err == nil {
		response.Success(c, gin.H{"user_id": userID, "role_codes": unique(req.Codes)})
	}
}

// ReplaceRolePermissions godoc
// @Summary Replace all permissions for a role
// @Tags System
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param role_id path int true "Role ID"
// @Param body body replaceCodesRequest true "Permission codes"
// @Success 200 {object} response.Body
// @Router /v1/system/roles/{role_id}/permissions [put]
func (h *Handler) ReplaceRolePermissions(c *gin.Context) {
	roleID, ok := pathID(c)
	if !ok {
		return
	}
	var req replaceCodesRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BindError(c, err)
		return
	}
	err := h.service.ReplaceRolePermissions(c.Request.Context(), roleID, unique(req.Codes))
	handleError(c, err)
	if err == nil {
		response.Success(c, gin.H{"role_id": roleID, "permission_codes": unique(req.Codes)})
	}
}

func pathID(c *gin.Context) (uint64, bool) {
	id, err := strconv.ParseUint(c.Param("user_id"), 10, 64)
	if err != nil {
		id, err = strconv.ParseUint(c.Param("role_id"), 10, 64)
	}
	if err != nil || id == 0 {
		response.Error(c, response.ErrValidation, "a positive resource ID is required")
		return 0, false
	}
	return id, true
}

func handleError(c *gin.Context, err error) {
	if err == nil {
		return
	}
	switch {
	case errors.Is(err, ErrNotFound):
		response.Error(c, response.ErrNotFound, err.Error())
	case errors.Is(err, ErrUnavailable):
		response.Error(c, response.ErrUnavailable, err.Error())
	default:
		response.Error(c, response.ErrInternal, err.Error())
	}
}
