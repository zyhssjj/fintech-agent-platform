package statemachine

const (
	ApprovalDraft     = "draft"
	ApprovalPending   = "pending"
	ApprovalApproved  = "approved"
	ApprovalRejected  = "rejected"
	ApprovalCancelled = "cancelled"
)

var approvalTransitions = map[string][]string{
	ApprovalDraft:   {ApprovalPending, ApprovalCancelled},
	ApprovalPending: {ApprovalApproved, ApprovalRejected, ApprovalCancelled},
}

func CanTransitionApproval(from, to string) bool {
	return canTransition(approvalTransitions, from, to)
}
