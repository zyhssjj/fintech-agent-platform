package dto

// 本文件定义投资管理接口 INVEST-* 的传输结构。
type InvestmentProjectQuery struct {
	Stage    string `form:"stage"`
	Industry string `form:"industry"`
	Status   string `form:"status"`
	Keyword  string `form:"keyword"`
	PaginationQuery
}

type InvestmentProjectItem struct {
	ProjectID   string    `json:"project_id"`
	ProjectName string    `json:"project_name"`
	Stage       string    `json:"stage"`
	Industry    string    `json:"industry"`
	Amount      float64   `json:"amount"`
	ExpectedIRR float64   `json:"expected_irr"`
	RiskLevel   string    `json:"risk_level"`
	LeadPartner PersonRef `json:"lead_partner"`
	Description string    `json:"description"`
	CreatedAt   string    `json:"created_at"`
}

type CreateInvestmentProjectRequest struct {
	ProjectName string    `json:"project_name" binding:"required"`
	Industry    string    `json:"industry" binding:"required"`
	Amount      float64   `json:"amount"`
	Stage       string    `json:"stage"`
	Description string    `json:"description"`
	LeadUserID  string    `json:"lead_user_id"`
	Documents   []FileRef `json:"documents"`
}

type ProjectFilterRequest struct {
	Industries []string `json:"industries"`
	Stage      string   `json:"stage"`
	MinAmount  float64  `json:"min_amount"`
	MaxAmount  float64  `json:"max_amount"`
	RiskLevel  string   `json:"risk_level"`
	Keyword    string   `json:"keyword"`
}

type DueDiligenceCreateRequest struct {
	Scope     []string  `json:"scope" binding:"required"`
	Assignees []string  `json:"assignees"`
	DueDate   string    `json:"due_date"`
	Documents []FileRef `json:"documents"`
}

type DueDiligenceProgressRequest struct {
	CurrentStep string    `json:"current_step" binding:"required"`
	Progress    float64   `json:"progress" binding:"required"`
	Status      string    `json:"status"`
	Notes       string    `json:"notes"`
	Documents   []FileRef `json:"documents"`
}

type DueDiligenceCompleteRequest struct {
	Conclusion string    `json:"conclusion" binding:"required"`
	Score      float64   `json:"score"`
	Risks      []JSONMap `json:"risks"`
	Report     FileRef   `json:"report"`
}

type InvestmentDecisionRequest struct {
	ProjectID   string    `json:"project_id" binding:"required"`
	MeetingDate string    `json:"meeting_date"`
	Agenda      []string  `json:"agenda"`
	Materials   []FileRef `json:"materials"`
	Voters      []string  `json:"voters"`
}

type DecisionVoteRequest struct {
	Vote              string  `json:"vote" binding:"required"`
	SharesRepresented float64 `json:"shares_represented"`
	Comments          string  `json:"comments"`
}

type DCFValuationRequest struct {
	ProjectID          string  `json:"project_id"`
	Revenue            float64 `json:"revenue" binding:"required"`
	GrowthOptimistic   float64 `json:"growth_optimistic"`
	GrowthBase         float64 `json:"growth_base"`
	GrowthConservative float64 `json:"growth_conservative"`
	MarginOptimistic   float64 `json:"margin_optimistic"`
	MarginBase         float64 `json:"margin_base"`
	MarginConservative float64 `json:"margin_conservative"`
	WACC               float64 `json:"wacc" binding:"required"`
	PerpetualGrowth    float64 `json:"perpetual_growth"`
}

type DCFValuationResponse struct {
	ValuationID  string  `json:"valuation_id"`
	ProjectID    string  `json:"project_id"`
	Optimistic   float64 `json:"optimistic"`
	Base         float64 `json:"base"`
	Conservative float64 `json:"conservative"`
	Assumptions  JSONMap `json:"assumptions"`
	GeneratedAt  string  `json:"generated_at"`
}

type InvestmentExecutionRequest struct {
	ProjectID   string    `json:"project_id" binding:"required"`
	Amount      float64   `json:"amount" binding:"required"`
	PaymentDate string    `json:"payment_date"`
	BankAccount JSONMap   `json:"bank_account"`
	Documents   []FileRef `json:"documents"`
}

type InvestmentTermsRequest struct {
	ProjectID         string   `json:"project_id"`
	InvestmentAmount  float64  `json:"investment_amount" binding:"required"`
	PreMoneyValuation float64  `json:"pre_money_valuation"`
	ShareRatio        float64  `json:"share_ratio"`
	SpecialTerms      []string `json:"special_terms"`
}
