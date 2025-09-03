package runtime

type RuntimePhase string

const (
	ASLEEP     RuntimePhase = "asleep"
	AWAKE      RuntimePhase = "awake"      // idle state
	THINKING   RuntimePhase = "thinking"   // processing something
	ACTING     RuntimePhase = "acting"     // executing tasks
	COOLDOWN   RuntimePhase = "cooldown"   // resting + backoff
	RESPONDING RuntimePhase = "responding" // user trigger state
	// sub sleep but waiting to alert user or get approval
	// might hold state of messages to summarize for user
	// or approvals to deliver
	HANGING RuntimePhase = "hanging"
)

type RuntimeEvents string

const (
	WAKE    RuntimeEvents = "wake"
	TRIGGER RuntimeEvents = "trigger"
)
