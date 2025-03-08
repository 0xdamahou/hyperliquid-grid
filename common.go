package main

const (
	//Grid的两种状态
	RunningStatus    = "Running"
	TerminatedStatus = "Terminated"
	//分别对应：开单未成交，开单成交，关闭单未成交，关闭单成交
	InitStatus      = "Init"
	CompletedStatus = "Completed"
	CancelledStatus = "Cancelled"
	OkStatus        = "ok"
	SellSide        = "Sell"
	BuySide         = "Buy"
	StartAct        = "Start"
	StopAct         = "Stop"

	MarketOrderFee     = 0.00035
	LimitPriceOrderFee = 0.0001
)

var (
	SmallSlipper = 0.3
)
