package dto

// Meeting DTOs follow MEETING-001~MEETING-006 in the interface design document.
type CreateMeetingRequest struct {
	ProjectID    string   `json:"project_id" binding:"required"`
	MeetingTitle string   `json:"meeting_title" binding:"required"`
	Agenda       string   `json:"agenda"`
	MeetingDate  string   `json:"meeting_date"`
	Participants []string `json:"participants"`
}

type CreateMeetingResponse struct {
	MeetingID string `json:"meeting_id"`
	Status    string `json:"status"`
}

type UploadMeetingAudioResponse struct {
	AudioID  string `json:"audio_id"`
	FileSize int64  `json:"file_size"`
	Duration int    `json:"duration"`
	Format   string `json:"format"`
}

type TranscribeMeetingRequest struct {
	EnableCorrection bool   `json:"enable_correction"`
	ASRBackend       string `json:"asr_backend"`
}

type MeetingSummary struct {
	Overview    JSONMap   `json:"overview"`
	KeyPoints   []JSONMap `json:"key_points"`
	ActionItems []JSONMap `json:"action_items"`
}

type TranscribeMeetingResponse struct {
	TranscriptID   string         `json:"transcript_id"`
	RawText        string         `json:"raw_text"`
	CorrectedText  string         `json:"corrected_text"`
	AISummary      MeetingSummary `json:"ai_summary"`
	ProcessingTime int            `json:"processing_time"`
	ASRBackend     string         `json:"asr_backend"`
	HasCorrection  bool           `json:"has_correction"`
}

type MeetingDetailResponse struct {
	MeetingID    string         `json:"meeting_id"`
	MeetingTitle string         `json:"meeting_title"`
	Agenda       string         `json:"agenda"`
	MeetingDate  string         `json:"meeting_date"`
	Participants []string       `json:"participants"`
	AudioURL     string         `json:"audio_url"`
	Duration     int            `json:"duration"`
	RawText      string         `json:"raw_text"`
	AISummary    MeetingSummary `json:"ai_summary"`
	Status       string         `json:"status"`
}

type MeetingActionItemsResponse struct {
	Items []MeetingActionItem `json:"items"`
}

type MeetingActionItem struct {
	Item         string `json:"item"`
	Owner        string `json:"owner"`
	Deadline     string `json:"deadline"`
	Synchronized bool   `json:"synchronized"`
}
