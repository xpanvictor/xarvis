package runtime

import "github.com/looplab/fsm"

// Every user has a runtime in the system
// Round robin algorithm to handle concurrency
// and fair timing for each user (asides trigger wake)
// Runtime is an FSM
// States:
//
//	sleep -> wake -> (thinking || responding || acting)
type UserRuntime struct {
	UserID       string
	StateMachine *fsm.FSM
}
