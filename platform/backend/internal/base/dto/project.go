package dto

// 本文件定义投后项目管理接口 PROJECT-* 的传输结构。
type PostInvestmentProjectQuery struct {
	Status   string `form:"status"`
	Industry string `form:"industry"`
	PaginationQuery
}

type OperationUpdateRequest struct {
	Period           string  `json:"period" binding:"required"`
	Revenue          float64 `json:"revenue"`
	GrossMargin      float64 `json:"gross_margin"`
	NetProfit        float64 `json:"net_profit"`
	CashFlow         float64 `json:"cash_flow"`
	Headcount        int     `json:"headcount"`
	OperationalNotes string  `json:"operational_notes"`
}

type MilestoneItem struct {
	MilestoneID string    `json:"milestone_id"`
	Title       string    `json:"title"`
	Description string    `json:"description"`
	DueDate     string    `json:"due_date"`
	Status      string    `json:"status"`
	Owner       PersonRef `json:"owner"`
}

type ProjectReportGenerateRequest struct {
	ReportType        string `json:"report_type" binding:"required"`
	Period            string `json:"period" binding:"required"`
	Year              int    `json:"year" binding:"required"`
	IncludeFinancials bool   `json:"include_financials"`
	IncludeOperations bool   `json:"include_operations"`
	IncludeOutlook    bool   `json:"include_outlook"`
}

type ProjectReportResponse struct {
	ReportID    string  `json:"report_id"`
	ProjectID   string  `json:"project_id"`
	ReportType  string  `json:"report_type"`
	Content     JSONMap `json:"content"`
	Status      string  `json:"status"`
	GeneratedAt string  `json:"generated_at"`
}

type ProjectReportItem struct {
	ReportID   string    `json:"report_id"`
	ReportType string    `json:"report_type"`
	Period     string    `json:"period"`
	Year       int       `json:"year"`
	Status     string    `json:"status"`
	Author     PersonRef `json:"author"`
	CreatedAt  string    `json:"created_at"`
}

type BoardInfoResponse struct {
	BoardID      string      `json:"board_id"`
	CompanyName  string      `json:"company_name"`
	BoardSeats   []JSONMap   `json:"board_seats"`
	Observers    []PersonRef `json:"observers"`
	NextMeeting  string      `json:"next_meeting"`
	VotingRights JSONMap     `json:"voting_rights"`
}

type BoardVoteRequest struct {
	ResolutionID      string  `json:"resolution_id" binding:"required"`
	Vote              string  `json:"vote" binding:"required"`
	SharesRepresented float64 `json:"shares_represented" binding:"required"`
	Comments          string  `json:"comments"`
}

type BoardVoteResponse struct {
	VoteID       string `json:"vote_id"`
	ResolutionID string `json:"resolution_id"`
	Vote         string `json:"vote"`
	VotedBy      string `json:"voted_by"`
	VotedAt      string `json:"voted_at"`
}

type RiskAlertQuery struct {
	Severity string `form:"severity"`
	Status   string `form:"status"`
	PaginationQuery
}

type RiskAlertItem struct {
	AlertID           string `json:"alert_id"`
	ProjectID         string `json:"project_id"`
	Title             string `json:"title"`
	Description       string `json:"description"`
	Severity          string `json:"severity"`
	RiskType          string `json:"risk_type"`
	Status            string `json:"status"`
	RecommendedAction string `json:"recommended_action"`
	CreatedAt         string `json:"created_at"`
}

type ResolveRiskAlertRequest struct {
	Status      string `json:"status" binding:"required"`
	Resolution  string `json:"resolution"`
	ActionTaken string `json:"action_taken"`
}
