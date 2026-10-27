package dto

// 本文件定义退出管理、收益分配、信披和合伙人会议接口的传输结构。
type ExitOpportunityQuery struct {
	ProjectID   string   `form:"project_id"`
	ExitType    []string `form:"exit_type"`
	Confidence  float64  `form:"confidence"`
	TimeHorizon string   `form:"time_horizon"`
	PaginationQuery
}

type ExitOpportunityItem struct {
	OpportunityID     string    `json:"opportunity_id"`
	ProjectID         string    `json:"project_id"`
	ProjectName       string    `json:"project_name"`
	ExitType          string    `json:"exit_type"`
	ConfidenceScore   float64   `json:"confidence_score"`
	EstimatedTimeline string    `json:"estimated_timeline"`
	EstimatedReturn   float64   `json:"estimated_return"`
	KeyMilestones     []JSONMap `json:"key_milestones"`
	RiskFactors       []JSONMap `json:"risk_factors"`
}

type ExitAnalysisRequest struct {
	AnalysisType     string  `json:"analysis_type" binding:"required"`
	TimeHorizon      string  `json:"time_horizon" binding:"required"`
	MarketConditions JSONMap `json:"market_conditions"`
}

type ExitAnalysisResponse struct {
	AnalysisID          string    `json:"analysis_id"`
	ProjectID           string    `json:"project_id"`
	ExitPaths           []JSONMap `json:"exit_paths"`
	RecommendedPath     JSONMap   `json:"recommended_path"`
	Comparison          JSONMap   `json:"comparison"`
	SensitivityAnalysis JSONMap   `json:"sensitivity_analysis"`
}

type ExitExecutionCreateRequest struct {
	ProjectID   string    `json:"project_id" binding:"required"`
	ExitType    string    `json:"exit_type" binding:"required"`
	TargetDate  string    `json:"target_date" binding:"required"`
	TargetPrice float64   `json:"target_price"`
	TargetBuyer string    `json:"target_buyer"`
	Documents   []FileRef `json:"documents"`
}

type ExitExecutionResponse struct {
	ExecutionID         string    `json:"execution_id"`
	ProjectID           string    `json:"project_id"`
	ExitType            string    `json:"exit_type"`
	Status              string    `json:"status"`
	Progress            float64   `json:"progress"`
	Milestones          []JSONMap `json:"milestones"`
	EstimatedCompletion string    `json:"estimated_completion"`
}

type ExitProgressRequest struct {
	CurrentStep string    `json:"current_step" binding:"required"`
	Progress    float64   `json:"progress" binding:"required"`
	Status      string    `json:"status"`
	Notes       string    `json:"notes"`
	Documents   []FileRef `json:"documents"`
}

type DistributionRequest struct {
	TotalProceeds    float64 `json:"total_proceeds" binding:"required"`
	DistributionDate string  `json:"distribution_date" binding:"required"`
	AllocationRules  JSONMap `json:"allocation_rules"`
}

type DistributionResponse struct {
	DistributionID  string    `json:"distribution_id"`
	TotalProceeds   float64   `json:"total_proceeds"`
	Distributions   []JSONMap `json:"distributions"`
	GPShare         float64   `json:"gp_share"`
	LPShare         float64   `json:"lp_share"`
	Carry           float64   `json:"carry"`
	NetDistribution float64   `json:"net_distribution"`
}

type LPDisclosureQuery struct {
	LPID       string `form:"lp_id"`
	FundID     string `form:"fund_id"`
	ReportType string `form:"report_type"`
	PaginationQuery
}

type LPDisclosureItem struct {
	DisclosureID string `json:"disclosure_id"`
	LPID         string `json:"lp_id"`
	LPName       string `json:"lp_name"`
	FundID       string `json:"fund_id"`
	FundName     string `json:"fund_name"`
	ReportType   string `json:"report_type"`
	ReportDate   string `json:"report_date"`
	Status       string `json:"status"`
	SentAt       string `json:"sent_at"`
}

type GenerateLPDisclosureRequest struct {
	FundID             string   `json:"fund_id" binding:"required"`
	LPIDs              []string `json:"lp_ids"`
	ReportType         string   `json:"report_type" binding:"required"`
	IncludeFinancials  bool     `json:"include_financials"`
	IncludePerformance bool     `json:"include_performance"`
	IncludePortfolio   bool     `json:"include_portfolio"`
	SendNotification   bool     `json:"send_notification"`
}

type GenerateLPDisclosureResponse struct {
	DisclosureIDs    []string `json:"disclosure_ids"`
	TotalRecipients  int      `json:"total_recipients"`
	FailedRecipients []string `json:"failed_recipients"`
	GeneratedAt      string   `json:"generated_at"`
}

type FundMeetingQuery struct {
	FundID      string `form:"fund_id"`
	MeetingType string `form:"meeting_type"`
	Status      string `form:"status"`
	PaginationQuery
}

type FundMeetingItem struct {
	MeetingID      string `json:"meeting_id"`
	FundID         string `json:"fund_id"`
	FundName       string `json:"fund_name"`
	MeetingType    string `json:"meeting_type"`
	MeetingDate    string `json:"meeting_date"`
	Status         string `json:"status"`
	Location       string `json:"location"`
	AttendeesCount int    `json:"attendees_count"`
}

type CreateFundMeetingRequest struct {
	FundID      string    `json:"fund_id" binding:"required"`
	MeetingType string    `json:"meeting_type" binding:"required"`
	MeetingDate string    `json:"meeting_date" binding:"required"`
	Location    string    `json:"location"`
	Agenda      []JSONMap `json:"agenda"`
	Materials   []FileRef `json:"materials"`
	Invitees    []string  `json:"invitees"`
}

type CreateFundMeetingResponse struct {
	MeetingID       string `json:"meeting_id"`
	FundID          string `json:"fund_id"`
	MeetingType     string `json:"meeting_type"`
	MeetingDate     string `json:"meeting_date"`
	Status          string `json:"status"`
	InvitationsSent int    `json:"invitations_sent"`
}

type FundMeetingVoteRequest struct {
	ProposalID        string  `json:"proposal_id" binding:"required"`
	Vote              string  `json:"vote" binding:"required"`
	SharesRepresented float64 `json:"shares_represented" binding:"required"`
	Comments          string  `json:"comments"`
}
