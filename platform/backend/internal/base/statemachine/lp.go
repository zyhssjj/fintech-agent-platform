package statemachine

const (
	LPLead       = "lead"
	LPContacted  = "contacted"
	LPQualified  = "qualified"
	LPCommitted  = "committed"
	LPSubscribed = "subscribed"
	LPActive     = "active"
	LPDefaulted  = "defaulted"
	LPExited     = "exited"
)

var lpTransitions = map[string][]string{
	LPLead:       {LPContacted},
	LPContacted:  {LPQualified},
	LPQualified:  {LPCommitted},
	LPCommitted:  {LPSubscribed, LPDefaulted},
	LPSubscribed: {LPActive, LPDefaulted},
	LPActive:     {LPExited, LPDefaulted},
	LPDefaulted:  {LPExited},
}

func CanTransitionLP(from, to string) bool {
	return canTransition(lpTransitions, from, to)
}
