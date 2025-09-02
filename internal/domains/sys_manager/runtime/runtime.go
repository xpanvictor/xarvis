package runtime

import (
	"sync"

	"github.com/looplab/fsm"
	"github.com/xpanvictor/xarvis/internal/domains/conversation/brain"
)

// Every user has a runtime in the system
// Round robin algorithm to handle concurrency
// and fair timing for each user (asides trigger wake)
// Runtime is an FSM
// States:
//
//	sleep -> wake -> (thinking || responding || acting)
type UserRuntime struct {
	// identity
	UserID string
	// state
	StateMachine *fsm.FSM
	// dependencies
	BrainModule *brain.Brain
	// persistence layer
	// processing pipeline
	// IO system // per device mapping
	// Locking mech -> also per user act lock
	sync.Mutex
}

func (ur *UserRuntime) GenerateFSM() *fsm.FSM {
	ur.Lock()
	defer ur.Unlock()

	currentState := ASLEEP // fetch from persistence
	return fsm.NewFSM(
		string(currentState),
		fsm.Events{
			{Name: string(WAKE), Src: []string{string(AWAKE)}, Dst: string(AWAKE)},
		},
		fsm.Callbacks{},
	)
}
