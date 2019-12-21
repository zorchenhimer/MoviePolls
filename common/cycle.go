package common

import (
	"fmt"
	"time"
)

type Cycle struct {
	Id int

	Start     time.Time
	End       time.Time
	EndingSet bool // has an end time been set? (ignore End value if false)
}

func (c Cycle) String() string {
	return fmt.Sprintf("Cycle{Id:%d}", c.Id)
}
