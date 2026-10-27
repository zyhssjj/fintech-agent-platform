package approval

import (
	"time"

	"gorm.io/datatypes"
	"gorm.io/gorm"

	"wfagent/internal/base/database"
)

type ApprovalFlow struct {
	database.BaseModel
	FlowCode     string `gorm:"column:flow_code;type:varchar(64);not null;uniqueIndex" json:"flow_code"`
	FlowName     string `gorm:"column:flow_name;type:varchar(128);not null" json:"flow_name"`
	BusinessType string `gorm:"column:business_type;type:varchar(64);not null;index" json:"business_type"`
	Description  string `gorm:"column:description;type:varchar(255);not null;default:''" json:"description"`
	Status       string `gorm:"column:status;type:varchar(32);not null;default:'active';index" json:"status"`
	Steps        []ApprovalStep `gorm:"foreignKey:FlowID" json:"steps,omitempty"`
}

func (ApprovalFlow) TableName() string { return "wf_approval_flow" }

type ApprovalStep struct {
	database.BaseModel
	FlowID        uint64         `gorm:"column:flow_id;not null;index" json:"flow_id"`
	StepNo        int            `gorm:"column:step_no;not null" json:"step_no"`
	StepName      string         `gorm:"column:step_name;type:varchar(128);not null" json:"step_name"`
	ApproverRoles datatypes.JSON `gorm:"column:approver_roles;type:jsonb;not null;default:'[]'" json:"approver_roles"`
	TimeoutHours  int            `gorm:"column:timeout_hours;not null;default:48" json:"timeout_hours"`
}

func (ApprovalStep) TableName() string { return "wf_approval_step" }

type ApprovalRecord struct {
	database.BaseModel
	ApprovalCode        string         `gorm:"column:approval_code;type:varchar(64);not null;uniqueIndex" json:"approval_code"`
	FlowID              uint64         `gorm:"column:flow_id;not null;index" json:"flow_id"`
	BusinessType        string         `gorm:"column:business_type;type:varchar(64);not null;index" json:"business_type"`
	BusinessID          string         `gorm:"column:business_id;type:varchar(64);not null;index" json:"business_id"`
	Title               string         `gorm:"column:title;type:varchar(255);not null" json:"title"`
	Status              string         `gorm:"column:status;type:varchar(32);not null;default:'pending';index" json:"status"`
	CurrentStepNo       int            `gorm:"column:current_step_no;not null;default:1" json:"current_step_no"`
	CurrentApproverRole string         `gorm:"column:current_approver_role;type:varchar(64);not null;default:''" json:"current_approver_role"`
	ApplicantID         string         `gorm:"column:applicant_id;type:varchar(64);not null;index" json:"applicant_id"`
	ApplicantName       string         `gorm:"column:applicant_name;type:varchar(128);not null;default:''" json:"applicant_name"`
	Payload             datatypes.JSON `gorm:"column:payload;type:jsonb;not null;default:'{}'" json:"payload"`
	ProgressLog         datatypes.JSON `gorm:"column:progress_log;type:jsonb;not null;default:'[]'" json:"progress_log"`
	DeadlineAt          *time.Time     `gorm:"column:deadline_at" json:"deadline_at"`
	FinishedAt          *time.Time     `gorm:"column:finished_at" json:"finished_at"`
	Flow                ApprovalFlow   `gorm:"foreignKey:FlowID;constraint:OnUpdate:CASCADE,OnDelete:RESTRICT" json:"flow,omitempty"`
}

func (ApprovalRecord) TableName() string { return "wf_approval_record" }

func AutoMigrate(db *gorm.DB) error {
	if db == nil {
		return nil
	}
	return db.AutoMigrate(&ApprovalFlow{}, &ApprovalStep{}, &ApprovalRecord{})
}
