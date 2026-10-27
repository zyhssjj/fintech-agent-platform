package task

import (
	"time"

	"gorm.io/datatypes"
	"gorm.io/gorm"
)

type AsyncTask struct {
	ID           uint64         `gorm:"primaryKey;autoIncrement;column:id" json:"id"`
	TaskID       string         `gorm:"column:task_id;type:varchar(64);not null;uniqueIndex" json:"task_id"`
	TaskType     string         `gorm:"column:task_type;type:varchar(64);not null;index" json:"task_type"`
	Status       string         `gorm:"column:status;type:varchar(32);not null;index" json:"status"`
	Progress     int            `gorm:"column:progress;not null;default:0" json:"progress"`
	InputParams  datatypes.JSON `gorm:"column:input_params;type:jsonb;not null;default:'{}'" json:"input_params" swaggertype:"object"`
	OutputResult datatypes.JSON `gorm:"column:output_result;type:jsonb;not null;default:'{}'" json:"output_result" swaggertype:"object"`
	ErrorMessage string         `gorm:"column:error_message;type:text;not null;default:''" json:"error_message"`
	RetryCount   int            `gorm:"column:retry_count;not null;default:0" json:"retry_count"`
	RetryLimit   int            `gorm:"column:retry_limit;not null;default:3" json:"retry_limit"`
	StartedAt    *time.Time     `gorm:"column:started_at" json:"started_at"`
	FinishedAt   *time.Time     `gorm:"column:finished_at" json:"finished_at"`
	CreateTime   time.Time      `gorm:"column:create_time;not null;autoCreateTime" json:"create_time"`
	UpdateTime   time.Time      `gorm:"column:update_time;not null;autoUpdateTime" json:"update_time"`
	IsDeleted    bool           `gorm:"column:is_deleted;not null;default:false;index" json:"is_deleted"`
}

func (AsyncTask) TableName() string {
	return "com_async_task"
}

func AutoMigrate(db *gorm.DB) error {
	if db == nil {
		return nil
	}
	return db.AutoMigrate(&AsyncTask{})
}
