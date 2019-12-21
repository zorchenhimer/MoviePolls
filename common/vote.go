package common

import ()

type Vote struct {
	User  *User
	Movie *Movie
	// Decay based on cycles active.
	CycleAdded *Cycle
}
