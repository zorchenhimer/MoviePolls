package common

import (
	"fmt"
	"time"
)

type Cycle struct {
	Id int

	Start time.Time
	End   *time.Time

	// List of movies watched this cycle.  If cycle has not ended, this will be
	// nil.
	Watched []*Movie
}

func (c Cycle) StartString() string {
	return fmt.Sprintf("%s %d, %d", c.Start.Month().String(), c.Start.Day(), c.Start.Year())
}

func (c Cycle) EndString() string {
	if c.End == nil {
		return "no end date"
	}
	return fmt.Sprintf("%s %d, %d", c.End.Month().String(), c.End.Day(), c.End.Year())
}

func (c Cycle) String() string {
	return fmt.Sprintf("Cycle{Id:%d}", c.Id)
}
