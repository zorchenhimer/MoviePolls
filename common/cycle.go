package common

import (
	"fmt"
	"time"
)

type Cycle struct {
	Id int

	Start time.Time
	End   *time.Time
	//EndingSet bool // has an end time been set? (ignore End value if false)
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
