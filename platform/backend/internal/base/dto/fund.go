package dto

// 本文件定义募资管理接口 FUND-LP、FUND-MGMT、FUND-COMPLIANCE 的传输结构。
type LPQuery struct {
	LPType       string `form:"lp_type"`
	RiskAppetite string `form:"risk_appetite"`
	Keyword      string `form:"keyword"`
	PaginationQuery
}

type LPItem struct {
	LPID             string      `json:"lp_id"`
	LPName           string      `json:"lp_name"`
	LPType           string      `json:"lp_type"`
	TotalCommitment  float64     `json:"total_commitment"`
	PaidInRate       float64     `json:"paid_in_rate"`
	PreferenceSector []string    `json:"preference_sector"`
	RiskAppetite     string      `json:"risk_appetite"`
	Contacts         []PersonRef `json:"contacts"`
}

type CreateLPRequest struct {
	LPName           string      `json:"lp_name" binding:"required"`
	LPType           string      `json:"lp_type" binding:"required"`
	TotalCommitment  float64     `json:"total_commitment"`
	PreferenceSector []string    `json:"preference_sector"`
	RiskAppetite     string      `json:"risk_appetite"`
	Contacts         []PersonRef `json:"contacts"`
}

type LPMatchRequest struct {
	FundID     string   `json:"fund_id" binding:"required"`
	Industries []string `json:"industries"`
	Amount     float64  `json:"amount"`
	RiskLevel  string   `json:"risk_level"`
}

type LPProfileResponse struct {
	LPItem
	HistoryInvestments []JSONMap `json:"history_investments"`
	ComplianceStatus   string    `json:"compliance_status"`
	MatchScore         float64   `json:"match_score"`
}

type FundQuery struct {
	Status string `form:"status"`
	Type   string `form:"type"`
	PaginationQuery
}

type FundItem struct {
	FundID       string    `json:"fund_id"`
	FundName     string    `json:"fund_name"`
	FundType     string    `json:"fund_type"`
	TargetSize   float64   `json:"target_size"`
	RaisedAmount float64   `json:"raised_amount"`
	Status       string    `json:"status"`
	Manager      PersonRef `json:"manager"`
	CreatedAt    string    `json:"created_at"`
}

type CreateFundRequest struct {
	FundName   string   `json:"fund_name" binding:"required"`
	FundType   string   `json:"fund_type" binding:"required"`
	TargetSize float64  `json:"target_size" binding:"required"`
	Strategy   string   `json:"strategy"`
	Industries []string `json:"industries"`
	ManagerID  string   `json:"manager_id"`
}

type FundSubscribeRequest struct {
	LPID       string    `json:"lp_id" binding:"required"`
	Amount     float64   `json:"amount" binding:"required"`
	SignedDate string    `json:"signed_date"`
	Documents  []FileRef `json:"documents"`
}

type FundDistributionRequest struct {
	DistributionDate string  `json:"distribution_date" binding:"required"`
	TotalAmount      float64 `json:"total_amount" binding:"required"`
	AllocationRules  JSONMap `json:"allocation_rules"`
}

type ComplianceCheckRequest struct {
	SubjectID   string    `json:"subject_id" binding:"required"`
	SubjectType string    `json:"subject_type" binding:"required"`
	CheckTypes  []string  `json:"check_types" binding:"required"`
	Documents   []FileRef `json:"documents"`
}

type ComplianceCheckResponse struct {
	CheckID     string    `json:"check_id"`
	Status      string    `json:"status"`
	RiskLevel   string    `json:"risk_level"`
	Issues      []JSONMap `json:"issues"`
	Conclusion  string    `json:"conclusion"`
	GeneratedAt string    `json:"generated_at"`
}
