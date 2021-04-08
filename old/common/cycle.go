package common

import (
	"fmt"
	"time"
)

type Cycle struct {
	Id int

	PlannedEnd *time.Time
	Ended      *time.Time

	// List of movies watched this cycle.  If cycle has not ended, this will be
	// nil.
	Watched []*Movie
}

func (c Cycle) PlannedEndString() string {
	if c.PlannedEnd == nil {
		return ""
	}
	return c.PlannedEnd.Format("Mon Jan 2")
}

func (c Cycle) EndedString() string {
	if c.Ended == nil {
		return ""
	}
	return c.Ended.Format("Mon Jan 2, 2006")
}

func (c Cycle) String() string {
	return fmt.Sprintf("Cycle{Id:%d PlannedEnd:%s Ended: %s}", c.Id, c.PlannedEndString(), c.EndedString())
}
