package statemachine

const (
	DueDiligencePending    = "pending"
	DueDiligenceInProgress = "in_progress"
	DueDiligenceReview     = "review"
	DueDiligenceCompleted  = "completed"
	DueDiligenceFailed     = "failed"
	DueDiligenceCancelled  = "cancelled"
)

var dueDiligenceTransitions = map[string][]string{
	DueDiligencePending:    {DueDiligenceInProgress, DueDiligenceCancelled},
	DueDiligenceInProgress: {DueDiligenceReview, DueDiligenceFailed, DueDiligenceCancelled},
	DueDiligenceReview:     {DueDiligenceCompleted, DueDiligenceFailed},
	DueDiligenceFailed:     {DueDiligenceInProgress},
}

func CanTransitionDueDiligence(from, to string) bool {
	return canTransition(dueDiligenceTransitions, from, to)
}
