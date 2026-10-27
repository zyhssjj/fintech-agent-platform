package approval

import (
	"context"
	"errors"
	"fmt"
	"slices"
	"time"

	"github.com/google/uuid"
	"gorm.io/datatypes"
	"gorm.io/gorm"

	basedto "wfagent/internal/base/dto"
	"wfagent/internal/base/statemachine"
)

var (
	ErrApprovalNotFound = errors.New("approval not found")
	ErrFlowNotFound     = errors.New("approval flow not found")
	ErrInvalidAction    = errors.New("invalid approval action")
)

type Service struct {
	db *gorm.DB
}

func NewService(db *gorm.DB) *Service { return &Service{db: db} }

type CreateApprovalRequest struct {
	BusinessType string          `json:"business_type" binding:"required"`
	BusinessID   string          `json:"business_id" binding:"required"`
	Title        string          `json:"title" binding:"required"`
	Payload      basedto.JSONMap `json:"payload"`
}

type ApprovalQuery struct {
	Status       string `form:"status"`
	BusinessType string `form:"business_type"`
	Page         int    `form:"page"`
	PageSize     int    `form:"page_size"`
}

type ApprovalActionRequest struct {
	Action  string `json:"action" binding:"required"`
	Comment string `json:"comment"`
}

type ApprovalView struct {
	ApprovalID          string            `json:"approval_id"`
	BusinessType        string            `json:"business_type"`
	BusinessID          string            `json:"business_id"`
	Title               string            `json:"title"`
	Status              string            `json:"status"`
	CurrentStepNo       int               `json:"current_step_no"`
	CurrentApproverRole string            `json:"current_approver_role"`
	ApplicantID         string            `json:"applicant_id"`
	ApplicantName       string            `json:"applicant_name"`
	DeadlineAt          string            `json:"deadline_at,omitempty"`
	FinishedAt          string            `json:"finished_at,omitempty"`
	IsOverdue           bool              `json:"is_overdue"`
	Payload             basedto.JSONMap   `json:"payload"`
	ProgressLog         []basedto.JSONMap `json:"progress_log"`
}

func SeedDefaults(db *gorm.DB) error {
	if db == nil {
		return nil
	}
	return db.Transaction(func(tx *gorm.DB) error {
		defaults := []struct {
			flow  ApprovalFlow
			steps []ApprovalStep
		}{
			{
				flow: ApprovalFlow{
					FlowCode:     "communication_template_publish",
					FlowName:     "通讯模板发布审批",
					BusinessType: "template_publish",
					Description:  "模板草稿提交发布审批",
					Status:       "active",
				},
				steps: []ApprovalStep{
					{StepNo: 1, StepName: "运营复核", ApproverRoles: mustJSON([]string{"OPS_MANAGER", "REVIEWER"}), TimeoutHours: 24},
					{StepNo: 2, StepName: "法务审批", ApproverRoles: mustJSON([]string{"LEGAL_MANAGER"}), TimeoutHours: 48},
				},
			},
			{
				flow: ApprovalFlow{
					FlowCode:     "communication_document_send",
					FlowName:     "函件通知发送审批",
					BusinessType: "document_send",
					Description:  "函件、通知正式发送审批",
					Status:       "active",
				},
				steps: []ApprovalStep{
					{StepNo: 1, StepName: "业务复核", ApproverRoles: mustJSON([]string{"OPS_MANAGER", "FUND_MANAGER", "POST_INVEST_MANAGER"}), TimeoutHours: 24},
					{StepNo: 2, StepName: "合规审批", ApproverRoles: mustJSON([]string{"LEGAL_MANAGER", "RISK_MANAGER"}), TimeoutHours: 48},
				},
			},
		}

		for _, item := range defaults {
			var flow ApprovalFlow
			err := tx.Where("flow_code = ?", item.flow.FlowCode).First(&flow).Error
			if errors.Is(err, gorm.ErrRecordNotFound) {
				flow = item.flow
				if err := tx.Create(&flow).Error; err != nil {
					return err
				}
			} else if err != nil {
				return err
			} else if err := tx.Model(&flow).Updates(map[string]any{
				"flow_name":     item.flow.FlowName,
				"business_type": item.flow.BusinessType,
				"description":   item.flow.Description,
				"status":        item.flow.Status,
				"is_deleted":    false,
			}).Error; err != nil {
				return err
			}

			if err := tx.Where("flow_id = ?", flow.ID).Delete(&ApprovalStep{}).Error; err != nil {
				return err
			}
			for _, step := range item.steps {
				step.FlowID = flow.ID
				if err := tx.Create(&step).Error; err != nil {
					return err
				}
			}
		}
		return nil
	})
}

func (s *Service) CreateApproval(ctx context.Context, req CreateApprovalRequest, applicantID, applicantName string) (ApprovalView, error) {
	flow, steps, err := s.findFlow(ctx, req.BusinessType)
	if err != nil {
		return ApprovalView{}, err
	}
	deadline := calcDeadline(steps[0].TimeoutHours)
	record := ApprovalRecord{
		ApprovalCode:        "APR-" + time.Now().Format("20060102") + "-" + uuid.NewString()[:8],
		FlowID:              flow.ID,
		BusinessType:        req.BusinessType,
		BusinessID:          req.BusinessID,
		Title:               req.Title,
		Status:              "pending",
		CurrentStepNo:       steps[0].StepNo,
		CurrentApproverRole: firstRole(steps[0].ApproverRoles),
		ApplicantID:         applicantID,
		ApplicantName:       applicantName,
		Payload:             mustJSON(req.Payload),
		ProgressLog: mustJSON([]basedto.JSONMap{{
			"action":        "submit",
			"comment":       "发起审批",
			"operator_id":   applicantID,
			"operator_name": applicantName,
			"step_no":       steps[0].StepNo,
			"time":          time.Now().Format(time.RFC3339),
		}}),
		DeadlineAt: &deadline,
	}
	if err := s.db.WithContext(ctx).Create(&record).Error; err != nil {
		return ApprovalView{}, err
	}
	return s.GetApproval(ctx, record.ApprovalCode)
}

func (s *Service) ListApprovals(ctx context.Context, query ApprovalQuery, userID string, roles []string) (basedto.PageResult[ApprovalView], error) {
	if err := s.autoRemindOverdue(ctx); err != nil {
		return basedto.PageResult[ApprovalView]{}, err
	}
	if query.Page <= 0 {
		query.Page = 1
	}
	if query.PageSize <= 0 || query.PageSize > 100 {
		query.PageSize = 10
	}
	db := s.db.WithContext(ctx).Model(&ApprovalRecord{}).Where("is_deleted = false")
	if query.Status != "" {
		db = db.Where("status = ?", query.Status)
	}
	if query.BusinessType != "" {
		db = db.Where("business_type = ?", query.BusinessType)
	}
	if !slices.Contains(roles, "SYS_ADMIN") {
		db = db.Where("applicant_id = ? OR current_approver_role IN ?", userID, roles)
	}
	var total int64
	if err := db.Count(&total).Error; err != nil {
		return basedto.PageResult[ApprovalView]{}, err
	}
	var records []ApprovalRecord
	if err := db.Order("create_time desc").Offset((query.Page - 1) * query.PageSize).Limit(query.PageSize).Find(&records).Error; err != nil {
		return basedto.PageResult[ApprovalView]{}, err
	}
	items := make([]ApprovalView, 0, len(records))
	for _, item := range records {
		items = append(items, toApprovalView(item))
	}
	return basedto.PageResult[ApprovalView]{
		List: items,
		Pagination: basedto.PaginationInfo{
			Page: query.Page, PageSize: query.PageSize, Total: total,
		},
	}, nil
}

func (s *Service) GetApproval(ctx context.Context, approvalID string) (ApprovalView, error) {
	if err := s.autoRemindOverdue(ctx); err != nil {
		return ApprovalView{}, err
	}
	var record ApprovalRecord
	if err := s.db.WithContext(ctx).Where("approval_code = ? AND is_deleted = false", approvalID).First(&record).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return ApprovalView{}, ErrApprovalNotFound
		}
		return ApprovalView{}, err
	}
	return toApprovalView(record), nil
}

func (s *Service) ActionApproval(ctx context.Context, approvalID string, req ApprovalActionRequest, actorID, actorName string, actorRoles []string) (ApprovalView, error) {
	var result ApprovalView
	err := s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var record ApprovalRecord
		if err := tx.Where("approval_code = ? AND is_deleted = false", approvalID).First(&record).Error; err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return ErrApprovalNotFound
			}
			return err
		}
		if record.Status != "pending" {
			return fmt.Errorf("%w: approval is already %s", ErrInvalidAction, record.Status)
		}
		if !canAct(record, actorID, actorRoles) {
			return fmt.Errorf("%w: actor lacks permission for current step", ErrInvalidAction)
		}
		_, steps, err := s.findFlow(ctx, record.BusinessType)
		if err != nil {
			return err
		}
		logs := decodeLog(record.ProgressLog)
		action := req.Action
		switch action {
		case "approve":
			nextStepNo := record.CurrentStepNo + 1
			nextStep, ok := findStep(steps, nextStepNo)
			if ok {
				record.CurrentStepNo = nextStep.StepNo
				record.CurrentApproverRole = firstRole(nextStep.ApproverRoles)
				deadline := calcDeadline(nextStep.TimeoutHours)
				record.DeadlineAt = &deadline
			} else {
				if !statemachine.CanTransitionApproval(record.Status, "approved") {
					return fmt.Errorf("%w: cannot approve", ErrInvalidAction)
				}
				record.Status = "approved"
				record.CurrentApproverRole = ""
				now := time.Now()
				record.FinishedAt = &now
				record.DeadlineAt = nil
			}
		case "reject":
			if !statemachine.CanTransitionApproval(record.Status, "rejected") {
				return fmt.Errorf("%w: cannot reject", ErrInvalidAction)
			}
			record.Status = "rejected"
			record.CurrentApproverRole = ""
			now := time.Now()
			record.FinishedAt = &now
			record.DeadlineAt = nil
		case "return":
			if !statemachine.CanTransitionApproval(record.Status, "rejected") {
				return fmt.Errorf("%w: cannot return", ErrInvalidAction)
			}
			record.Status = "returned"
			record.CurrentApproverRole = ""
			now := time.Now()
			record.FinishedAt = &now
			record.DeadlineAt = nil
		case "remind":
			// Keep pending while recording the reminder action.
		default:
			return ErrInvalidAction
		}
		logs = append(logs, basedto.JSONMap{
			"action":        action,
			"comment":       req.Comment,
			"operator_id":   actorID,
			"operator_name": actorName,
			"step_no":       record.CurrentStepNo,
			"time":          time.Now().Format(time.RFC3339),
		})
		record.ProgressLog = mustJSON(logs)
		if err := tx.Model(&ApprovalRecord{}).Where("id = ?", record.ID).Updates(map[string]any{
			"status":                record.Status,
			"current_step_no":       record.CurrentStepNo,
			"current_approver_role": record.CurrentApproverRole,
			"deadline_at":           record.DeadlineAt,
			"finished_at":           record.FinishedAt,
			"progress_log":          record.ProgressLog,
		}).Error; err != nil {
			return err
		}
		result = toApprovalView(record)
		return nil
	})
	return result, err
}

func (s *Service) findFlow(ctx context.Context, businessType string) (ApprovalFlow, []ApprovalStep, error) {
	var flow ApprovalFlow
	if err := s.db.WithContext(ctx).Where("business_type = ? AND status = ? AND is_deleted = false", businessType, "active").First(&flow).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return ApprovalFlow{}, nil, ErrFlowNotFound
		}
		return ApprovalFlow{}, nil, err
	}
	var steps []ApprovalStep
	if err := s.db.WithContext(ctx).Where("flow_id = ? AND is_deleted = false", flow.ID).Order("step_no asc").Find(&steps).Error; err != nil {
		return ApprovalFlow{}, nil, err
	}
	if len(steps) == 0 {
		return ApprovalFlow{}, nil, ErrFlowNotFound
	}
	return flow, steps, nil
}

func (s *Service) autoRemindOverdue(ctx context.Context) error {
	var records []ApprovalRecord
	if err := s.db.WithContext(ctx).
		Where("is_deleted = false AND status = ? AND deadline_at IS NOT NULL AND deadline_at < ?", "pending", time.Now()).
		Find(&records).Error; err != nil {
		return err
	}
	for _, record := range records {
		logs := decodeLog(record.ProgressLog)
		alreadyLogged := false
		for _, item := range logs {
			if item["action"] == "auto_remind" && item["step_no"] == float64(record.CurrentStepNo) {
				alreadyLogged = true
				break
			}
		}
		if alreadyLogged {
			continue
		}
		logs = append(logs, basedto.JSONMap{
			"action":        "auto_remind",
			"comment":       "审批已超时，系统自动催办",
			"operator_id":   "system",
			"operator_name": "system",
			"step_no":       record.CurrentStepNo,
			"time":          time.Now().Format(time.RFC3339),
		})
		if err := s.db.WithContext(ctx).Model(&ApprovalRecord{}).Where("id = ?", record.ID).Update("progress_log", mustJSON(logs)).Error; err != nil {
			return err
		}
	}
	return nil
}

func toApprovalView(record ApprovalRecord) ApprovalView {
	payload := basedto.JSONMap{}
	_ = datatypesJSONTo(record.Payload, &payload)
	return ApprovalView{
		ApprovalID:          record.ApprovalCode,
		BusinessType:        record.BusinessType,
		BusinessID:          record.BusinessID,
		Title:               record.Title,
		Status:              record.Status,
		CurrentStepNo:       record.CurrentStepNo,
		CurrentApproverRole: record.CurrentApproverRole,
		ApplicantID:         record.ApplicantID,
		ApplicantName:       record.ApplicantName,
		DeadlineAt:          formatTime(record.DeadlineAt),
		FinishedAt:          formatTime(record.FinishedAt),
		IsOverdue:           record.DeadlineAt != nil && time.Now().After(*record.DeadlineAt) && record.Status == "pending",
		Payload:             payload,
		ProgressLog:         decodeLog(record.ProgressLog),
	}
}

func canAct(record ApprovalRecord, actorID string, roles []string) bool {
	if actorID == record.ApplicantID {
		return true
	}
	return slices.Contains(roles, "SYS_ADMIN") || slices.Contains(roles, record.CurrentApproverRole)
}

func findStep(steps []ApprovalStep, stepNo int) (ApprovalStep, bool) {
	for _, item := range steps {
		if item.StepNo == stepNo {
			return item, true
		}
	}
	return ApprovalStep{}, false
}

func calcDeadline(timeoutHours int) time.Time {
	if timeoutHours <= 0 {
		timeoutHours = 24
	}
	return time.Now().Add(time.Duration(timeoutHours) * time.Hour)
}

func firstRole(value datatypes.JSON) string {
	var roles []string
	_ = datatypesJSONTo(value, &roles)
	if len(roles) == 0 {
		return ""
	}
	return roles[0]
}

func decodeLog(value datatypes.JSON) []basedto.JSONMap {
	var logs []basedto.JSONMap
	if err := datatypesJSONTo(value, &logs); err != nil || logs == nil {
		return []basedto.JSONMap{}
	}
	return logs
}

func formatTime(value *time.Time) string {
	if value == nil {
		return ""
	}
	return value.Format(time.RFC3339)
}
