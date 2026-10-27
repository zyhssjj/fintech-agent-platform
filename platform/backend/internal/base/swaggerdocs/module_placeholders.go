package swaggerdocs

import "wfagent/internal/base/response"

// listAgents godoc
// @Summary List agents
// @Tags Agent
// @Produce json
// @Security BearerAuth
// @Success 200 {object} response.Body
// @Router /v1/agents [get]
func listAgents() {}

// getAgent godoc
// @Summary Get agent detail
// @Tags Agent
// @Produce json
// @Security BearerAuth
// @Param agent_id path string true "Agent ID"
// @Success 200 {object} response.Body
// @Router /v1/agents/{agent_id} [get]
func getAgent() {}

// invokeAgent godoc
// @Summary Invoke agent
// @Tags Agent
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param agent_id path string true "Agent ID"
// @Param body body map[string]any true "Invoke payload"
// @Success 200 {object} response.Body
// @Router /v1/agents/{agent_id}/invoke [post]
func invokeAgent() {}

// listCustomerTodos godoc
// @Summary List customer todos
// @Tags Agent
// @Produce json
// @Security BearerAuth
// @Success 200 {object} response.Body
// @Router /v1/agents/customer/todos [get]
func listCustomerTodos() {}

// createCustomerTodo godoc
// @Summary Create customer todo
// @Tags Agent
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param body body map[string]any true "Todo payload"
// @Success 200 {object} response.Body
// @Router /v1/agents/customer/todos [post]
func createCustomerTodo() {}

// updateCustomerTodoStatus godoc
// @Summary Update customer todo status
// @Tags Agent
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param todo_id path string true "Todo ID"
// @Param body body map[string]any true "Status payload"
// @Success 200 {object} response.Body
// @Router /v1/agents/customer/todos/{todo_id}/status [patch]
func updateCustomerTodoStatus() {}

// createNotification godoc
// @Summary Create customer notification
// @Tags Agent
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param body body map[string]any true "Notification payload"
// @Success 200 {object} response.Body
// @Router /v1/agents/customer/notifications [post]
func createNotification() {}

// uploadKnowledgeDocument godoc
// @Summary Upload knowledge document
// @Tags Agent
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param body body map[string]any true "Document payload"
// @Success 200 {object} response.Body
// @Router /v1/agents/knowledge/documents [post]
func uploadKnowledgeDocument() {}

// searchKnowledge godoc
// @Summary Search knowledge base
// @Tags Agent
// @Produce json
// @Security BearerAuth
// @Success 200 {object} response.Body
// @Router /v1/agents/knowledge/search [get]
func searchKnowledge() {}

// getCompetitorIntelligence godoc
// @Summary Get competitor intelligence
// @Tags Agent
// @Produce json
// @Security BearerAuth
// @Success 200 {object} response.Body
// @Router /v1/agents/knowledge/competitor-intelligence [get]
func getCompetitorIntelligence() {}

// listEmails godoc
// @Summary List emails
// @Tags Agent
// @Produce json
// @Security BearerAuth
// @Success 200 {object} response.Body
// @Router /v1/agents/communication/emails [get]
func listEmails() {}

// createEmail godoc
// @Summary Create email
// @Tags Agent
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param body body map[string]any true "Email payload"
// @Success 200 {object} response.Body
// @Router /v1/agents/communication/emails [post]
func createEmail() {}

// listEmailTemplates godoc
// @Summary List email templates
// @Tags Agent
// @Produce json
// @Security BearerAuth
// @Success 200 {object} response.Body
// @Router /v1/agents/communication/templates [get]
func listEmailTemplates() {}

// createMeetingMinutes godoc
// @Summary Create meeting minutes
// @Tags Agent
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param body body map[string]any true "Meeting minutes payload"
// @Success 200 {object} response.Body
// @Router /v1/agents/cooperation/meeting-minutes [post]
func createMeetingMinutes() {}

// listFileVersions godoc
// @Summary List file versions
// @Tags Agent
// @Produce json
// @Security BearerAuth
// @Param file_id path string true "File ID"
// @Success 200 {object} response.Body
// @Router /v1/agents/cooperation/files/{file_id}/versions [get]
func listFileVersions() {}

// listLPs godoc
// @Summary List LPs
// @Tags Fundraising
// @Produce json
// @Security BearerAuth
// @Success 200 {object} response.Body
// @Router /v1/fundraising/lps [get]
func listLPs() {}

// createLP godoc
// @Summary Create LP
// @Tags Fundraising
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param body body map[string]any true "LP payload"
// @Success 200 {object} response.Body
// @Router /v1/fundraising/lps [post]
func createLP() {}

// matchLPs godoc
// @Summary Match LPs
// @Tags Fundraising
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param body body map[string]any true "Match payload"
// @Success 200 {object} response.Body
// @Router /v1/fundraising/lps/match [post]
func matchLPs() {}

// getLPProfile godoc
// @Summary Get LP profile
// @Tags Fundraising
// @Produce json
// @Security BearerAuth
// @Param lp_id path string true "LP ID"
// @Success 200 {object} response.Body
// @Router /v1/fundraising/lps/{lp_id}/profile [get]
func getLPProfile() {}

// listFunds godoc
// @Summary List funds
// @Tags Fundraising
// @Produce json
// @Security BearerAuth
// @Success 200 {object} response.Body
// @Router /v1/fundraising/funds [get]
func listFunds() {}

// createFund godoc
// @Summary Create fund
// @Tags Fundraising
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param body body map[string]any true "Fund payload"
// @Success 200 {object} response.Body
// @Router /v1/fundraising/funds [post]
func createFund() {}

// subscribeFund godoc
// @Summary Subscribe fund
// @Tags Fundraising
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param fund_id path string true "Fund ID"
// @Param body body map[string]any true "Subscription payload"
// @Success 200 {object} response.Body
// @Router /v1/fundraising/funds/{fund_id}/subscribe [post]
func subscribeFund() {}

// createDistribution godoc
// @Summary Create fund distribution
// @Tags Fundraising
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param fund_id path string true "Fund ID"
// @Param body body map[string]any true "Distribution payload"
// @Success 200 {object} response.Body
// @Router /v1/fundraising/funds/{fund_id}/distributions [post]
func createDistribution() {}

// createFundComplianceReport godoc
// @Summary Create fund compliance report
// @Tags Fundraising
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param fund_id path string true "Fund ID"
// @Param body body map[string]any true "Report payload"
// @Success 200 {object} response.Body
// @Router /v1/fundraising/funds/{fund_id}/compliance-report [post]
func createFundComplianceReport() {}

// listFundComplianceHistory godoc
// @Summary List fund compliance history
// @Tags Fundraising
// @Produce json
// @Security BearerAuth
// @Param fund_id path string true "Fund ID"
// @Success 200 {object} response.Body
// @Router /v1/fundraising/funds/{fund_id}/compliance-history [get]
func listFundComplianceHistory() {}

// checkFundCompliance godoc
// @Summary Check fundraising compliance
// @Tags Fundraising
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param body body map[string]any true "Compliance payload"
// @Success 200 {object} response.Body
// @Router /v1/fundraising/compliance/check [post]
func checkFundCompliance() {}

// listInvestmentProjects godoc
// @Summary List investment projects
// @Tags Investment
// @Produce json
// @Security BearerAuth
// @Success 200 {object} response.Body
// @Router /v1/investment/projects [get]
func listInvestmentProjects() {}

// createInvestmentProject godoc
// @Summary Create investment project
// @Tags Investment
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param body body map[string]any true "Project payload"
// @Success 200 {object} response.Body
// @Router /v1/investment/projects [post]
func createInvestmentProject() {}

// filterInvestmentProjects godoc
// @Summary Filter investment projects
// @Tags Investment
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param body body map[string]any true "Filter payload"
// @Success 200 {object} response.Body
// @Router /v1/investment/projects/filter [post]
func filterInvestmentProjects() {}

// createDueDiligence godoc
// @Summary Create due diligence task
// @Tags Investment
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param project_id path string true "Project ID"
// @Param body body map[string]any true "Due diligence payload"
// @Success 200 {object} response.Body
// @Router /v1/investment/projects/{project_id}/due-diligence [post]
func createDueDiligence() {}

// createInvestmentTerms godoc
// @Summary Create investment terms
// @Tags Investment
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param project_id path string true "Project ID"
// @Param body body map[string]any true "Terms payload"
// @Success 200 {object} response.Body
// @Router /v1/investment/projects/{project_id}/terms [post]
func createInvestmentTerms() {}

// updateDueDiligenceProgress godoc
// @Summary Update due diligence progress
// @Tags Investment
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param dd_id path string true "Due diligence ID"
// @Param body body map[string]any true "Progress payload"
// @Success 200 {object} response.Body
// @Router /v1/investment/due-diligence/{dd_id}/progress [patch]
func updateDueDiligenceProgress() {}

// completeDueDiligence godoc
// @Summary Complete due diligence
// @Tags Investment
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param dd_id path string true "Due diligence ID"
// @Param body body map[string]any true "Completion payload"
// @Success 200 {object} response.Body
// @Router /v1/investment/due-diligence/{dd_id}/complete [post]
func completeDueDiligence() {}

// createInvestmentDecision godoc
// @Summary Create investment decision
// @Tags Investment
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param body body map[string]any true "Decision payload"
// @Success 200 {object} response.Body
// @Router /v1/investment/decisions [post]
func createInvestmentDecision() {}

// voteInvestmentDecision godoc
// @Summary Vote investment decision
// @Tags Investment
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param decision_id path string true "Decision ID"
// @Param body body map[string]any true "Vote payload"
// @Success 200 {object} response.Body
// @Router /v1/investment/decisions/{decision_id}/vote [post]
func voteInvestmentDecision() {}

// calculateDCF godoc
// @Summary Calculate DCF valuation
// @Tags Investment
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param body body map[string]any true "DCF payload"
// @Success 200 {object} response.Body
// @Router /v1/investment/valuation/dcf [post]
func calculateDCF() {}

// createInvestmentExecution godoc
// @Summary Create investment execution
// @Tags Investment
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param body body map[string]any true "Execution payload"
// @Success 200 {object} response.Body
// @Router /v1/investment/executions [post]
func createInvestmentExecution() {}

// listPostInvestmentProjects godoc
// @Summary List post-investment projects
// @Tags Project
// @Produce json
// @Security BearerAuth
// @Success 200 {object} response.Body
// @Router /v1/projects/post-investment [get]
func listPostInvestmentProjects() {}

// createProjectOperation godoc
// @Summary Create project operation
// @Tags Project
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param project_id path string true "Project ID"
// @Param body body map[string]any true "Operation payload"
// @Success 200 {object} response.Body
// @Router /v1/projects/{project_id}/operations [post]
func createProjectOperation() {}

// listProjectMilestones godoc
// @Summary List project milestones
// @Tags Project
// @Produce json
// @Security BearerAuth
// @Param project_id path string true "Project ID"
// @Success 200 {object} response.Body
// @Router /v1/projects/{project_id}/milestones [get]
func listProjectMilestones() {}

// createProjectReport godoc
// @Summary Create project report
// @Tags Project
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param project_id path string true "Project ID"
// @Param body body map[string]any true "Report payload"
// @Success 200 {object} response.Body
// @Router /v1/projects/{project_id}/reports [post]
func createProjectReport() {}

// listProjectReports godoc
// @Summary List project reports
// @Tags Project
// @Produce json
// @Security BearerAuth
// @Param project_id path string true "Project ID"
// @Success 200 {object} response.Body
// @Router /v1/projects/{project_id}/reports [get]
func listProjectReports() {}

// getProjectBoard godoc
// @Summary Get project board
// @Tags Project
// @Produce json
// @Security BearerAuth
// @Param project_id path string true "Project ID"
// @Success 200 {object} response.Body
// @Router /v1/projects/{project_id}/board [get]
func getProjectBoard() {}

// createProjectBoardVote godoc
// @Summary Create project board vote
// @Tags Project
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param project_id path string true "Project ID"
// @Param body body map[string]any true "Vote payload"
// @Success 200 {object} response.Body
// @Router /v1/projects/{project_id}/board/votes [post]
func createProjectBoardVote() {}

// listProjectRiskAlerts godoc
// @Summary List project risk alerts
// @Tags Project
// @Produce json
// @Security BearerAuth
// @Param project_id path string true "Project ID"
// @Success 200 {object} response.Body
// @Router /v1/projects/{project_id}/risk-alerts [get]
func listProjectRiskAlerts() {}

// updateProjectRiskAlert godoc
// @Summary Update project risk alert
// @Tags Project
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param project_id path string true "Project ID"
// @Param alert_id path string true "Alert ID"
// @Param body body map[string]any true "Risk alert payload"
// @Success 200 {object} response.Body
// @Router /v1/projects/{project_id}/risk-alerts/{alert_id} [patch]
func updateProjectRiskAlert() {}

// createMeeting godoc
// @Summary Create meeting
// @Tags Meeting
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param body body map[string]any true "Meeting payload"
// @Success 200 {object} response.Body
// @Router /v1/meetings [post]
func createMeeting() {}

// uploadMeetingFile godoc
// @Summary Upload meeting file
// @Tags Meeting
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param meeting_id path string true "Meeting ID"
// @Param body body map[string]any true "Upload payload"
// @Success 200 {object} response.Body
// @Router /v1/meetings/{meeting_id}/upload [post]
func uploadMeetingFile() {}

// transcribeMeeting godoc
// @Summary Transcribe meeting
// @Tags Meeting
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param meeting_id path string true "Meeting ID"
// @Param body body map[string]any true "Transcription payload"
// @Success 200 {object} response.Body
// @Router /v1/meetings/{meeting_id}/transcribe [post]
func transcribeMeeting() {}

// getMeeting godoc
// @Summary Get meeting
// @Tags Meeting
// @Produce json
// @Security BearerAuth
// @Param meeting_id path string true "Meeting ID"
// @Success 200 {object} response.Body
// @Router /v1/meetings/{meeting_id} [get]
func getMeeting() {}

// deleteMeeting godoc
// @Summary Delete meeting
// @Tags Meeting
// @Produce json
// @Security BearerAuth
// @Param meeting_id path string true "Meeting ID"
// @Success 204
// @Router /v1/meetings/{meeting_id} [delete]
func deleteMeeting() {}

// listMeetingActionItems godoc
// @Summary List meeting action items
// @Tags Meeting
// @Produce json
// @Security BearerAuth
// @Param meeting_id path string true "Meeting ID"
// @Success 200 {object} response.Body
// @Router /v1/meetings/{meeting_id}/action-items [get]
func listMeetingActionItems() {}

// listExitOpportunities godoc
// @Summary List exit opportunities
// @Tags Exit
// @Produce json
// @Security BearerAuth
// @Success 200 {object} response.Body
// @Router /v1/exit/opportunities [get]
func listExitOpportunities() {}

// createExitAnalysis godoc
// @Summary Create exit analysis
// @Tags Exit
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param project_id path string true "Project ID"
// @Param body body map[string]any true "Analysis payload"
// @Success 200 {object} response.Body
// @Router /v1/exit/projects/{project_id}/analysis [post]
func createExitAnalysis() {}

// createExitExecution godoc
// @Summary Create exit execution
// @Tags Exit
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param body body map[string]any true "Execution payload"
// @Success 200 {object} response.Body
// @Router /v1/exit/executions [post]
func createExitExecution() {}

// updateExitExecutionProgress godoc
// @Summary Update exit execution progress
// @Tags Exit
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param execution_id path string true "Execution ID"
// @Param body body map[string]any true "Progress payload"
// @Success 200 {object} response.Body
// @Router /v1/exit/executions/{execution_id}/progress [patch]
func updateExitExecutionProgress() {}

// createExitDistribution godoc
// @Summary Create exit distribution
// @Tags Exit
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param execution_id path string true "Execution ID"
// @Param body body map[string]any true "Distribution payload"
// @Success 200 {object} response.Body
// @Router /v1/exit/executions/{execution_id}/distribution [post]
func createExitDistribution() {}

// listLPDisclosures godoc
// @Summary List LP disclosures
// @Tags Exit
// @Produce json
// @Security BearerAuth
// @Success 200 {object} response.Body
// @Router /v1/exit/lp-disclosures [get]
func listLPDisclosures() {}

// generateLPDisclosure godoc
// @Summary Generate LP disclosure
// @Tags Exit
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param body body map[string]any true "Disclosure payload"
// @Success 200 {object} response.Body
// @Router /v1/exit/lp-disclosures/generate [post]
func generateLPDisclosure() {}

// listFundMeetings godoc
// @Summary List fund meetings
// @Tags Exit
// @Produce json
// @Security BearerAuth
// @Success 200 {object} response.Body
// @Router /v1/exit/fund-meetings [get]
func listFundMeetings() {}

// createFundMeeting godoc
// @Summary Create fund meeting
// @Tags Exit
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param body body map[string]any true "Meeting payload"
// @Success 200 {object} response.Body
// @Router /v1/exit/fund-meetings [post]
func createFundMeeting() {}

// voteFundMeeting godoc
// @Summary Vote fund meeting
// @Tags Exit
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param meeting_id path string true "Meeting ID"
// @Param body body map[string]any true "Vote payload"
// @Success 200 {object} response.Body
// @Router /v1/exit/fund-meetings/{meeting_id}/vote [post]
func voteFundMeeting() {}

var _ = response.Body{}
