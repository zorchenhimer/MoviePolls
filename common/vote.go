package common

import (
	"fmt"
)

type Vote struct {
	User  *User
	Movie *Movie
	// Decay based on cycles active.
	CycleAdded *Cycle
}

func (v Vote) String() string {
	uid := 0
	mid := 0
	cid := 0

	if v.User != nil {
		uid = v.User.Id
	}
	if v.Movie != nil {
		mid = v.Movie.Id
	}
	if v.CycleAdded != nil {
		cid = v.CycleAdded.Id
	}

	return fmt.Sprintf("{Vote User:%d Movie:%d Cycle:%d}", uid, mid, cid)
}
