package statemachine

const (
	DistributionDraft      = "draft"
	DistributionCalculated = "calculated"
	DistributionApproved   = "approved"
	DistributionPaying     = "paying"
	DistributionPaid       = "paid"
	DistributionCancelled  = "cancelled"
)

var distributionTransitions = map[string][]string{
	DistributionDraft:      {DistributionCalculated, DistributionCancelled},
	DistributionCalculated: {DistributionApproved, DistributionCancelled},
	DistributionApproved:   {DistributionPaying},
	DistributionPaying:     {DistributionPaid},
}

func CanTransitionDistribution(from, to string) bool {
	return canTransition(distributionTransitions, from, to)
}
