package statemachine

const (
	FundDraft            = "draft"
	FundRaising          = "raising"
	FundEstablished      = "established"
	FundInvestmentPeriod = "investment_period"
	FundExitPeriod       = "exit_period"
	FundLiquidating      = "liquidating"
	FundClosed           = "closed"
	FundTerminated       = "terminated"
)

var fundTransitions = map[string][]string{
	FundDraft:            {FundRaising, FundTerminated},
	FundRaising:          {FundEstablished, FundTerminated},
	FundEstablished:      {FundInvestmentPeriod, FundTerminated},
	FundInvestmentPeriod: {FundExitPeriod},
	FundExitPeriod:       {FundLiquidating},
	FundLiquidating:      {FundClosed},
}

func CanTransitionFund(from, to string) bool {
	return canTransition(fundTransitions, from, to)
}
