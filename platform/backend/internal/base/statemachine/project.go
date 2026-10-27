package statemachine

const (
	ProjectDraft           = "draft"
	ProjectScreening       = "screening"
	ProjectDueDiligence    = "due_diligence"
	ProjectDecisionPending = "decision_pending"
	ProjectApproved        = "approved"
	ProjectRejected        = "rejected"
	ProjectInvested        = "invested"
	ProjectMonitoring      = "monitoring"
	ProjectExitPreparing   = "exit_preparing"
	ProjectExited          = "exited"
)

var projectTransitions = map[string][]string{
	ProjectDraft:           {ProjectScreening},
	ProjectScreening:       {ProjectDueDiligence, ProjectRejected},
	ProjectDueDiligence:    {ProjectDecisionPending, ProjectRejected},
	ProjectDecisionPending: {ProjectApproved, ProjectRejected},
	ProjectApproved:        {ProjectInvested},
	ProjectInvested:        {ProjectMonitoring},
	ProjectMonitoring:      {ProjectExitPreparing},
	ProjectExitPreparing:   {ProjectExited},
}

func CanTransitionProject(from, to string) bool {
	return canTransition(projectTransitions, from, to)
}
