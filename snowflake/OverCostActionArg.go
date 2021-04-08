package snowflake

// OverCostActionArg .
type OverCostActionArg struct {
	ActionType             uint32
	TimeTick               uint64
	WorkerId               uint16
	OverCostCountInOneTerm uint32
	GenCountInOneTerm      uint32
	TermIndex              uint32
}

// OverCostActionArg .
func (orca OverCostActionArg) OverCostActionArg(workerId uint16, timeTick uint64, actionType uint32, overCostCountInOneTerm uint32, genCountWhenOverCost uint32, index uint32) {
	orca.ActionType = actionType
	orca.TimeTick = timeTick
	orca.WorkerId = workerId
	orca.OverCostCountInOneTerm = overCostCountInOneTerm
	orca.GenCountInOneTerm = genCountWhenOverCost
	orca.TermIndex = index
}
