package MoviePolls

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"time"
)

type JsonConnector struct {
	filename     string
	CurrentCycle int

	Cycles []*Cycle
	//Cycles []struct {
	//    Id int
	//    Start time.Time
	//    End time.Time
	//}

	Movies []struct {
		Id           int
		Name         string
		Links        []string
		Description  string
		CycleAddedId int
		Removed      bool
		Approved     bool
	}

	Users []struct {
		Id                 int
		Name               string
		Email              string
		NotifyCycleEnd     bool
		NotifyVoteSelected bool
		Privilege          PrivilegeLevel
	}

	Votes []struct {
		Id    int
		Cycle int
		Movie int
	}
}

func LoadJson(filename string) (*JsonConnector, error) {
	raw, err := ioutil.ReadFile(filename)
	if err != nil {
		return nil, err
	}

	var data *JsonConnector
	err = json.Unmarshal(raw, data)
	if err != nil {
		return nil, fmt.Errorf("Unable to read JSON data: %v", err)
	}

	return data, nil
}

func (j *JsonConnector) Save(filename string) error {
	raw, err := json.Marshal(j)
	if err != nil {
		return fmt.Errorf("Unable to marshal JSON data: %v", err)
	}

	err = ioutil.WriteFile(filename, raw, 0777)
	if err != nil {
		return fmt.Errorf("Unable to write JSON data: %v", err)
	}

	return nil
}

/*
   On determining the current cycle.

   Should the current cycle have an end date?
   If so, this would be the automatic end date for the cycle.
   If not, only the current cycle would have an end date, which would define
   the current cycle as the cycle without an end date.

   Otherwise, just store the current cycle's ID somewhere (current
   functionality).
*/
func (j *JsonConnector) GetCurrentCycle() *Cycle {
	for _, c := range j.Cycles {
		if j.CurrentCycle == c.Id {
			return c
		}
	}
	return nil
}

func (j *JsonConnector) AddCycle(end *time.Time) error {
	c := Cycle{
		Id:    j.nextCycleId(),
		Start: time.Now(),
	}

	if end != nil && end.After(c.Start) {
		c.End = *end
	}

	return nil
}

func (j *JsonConnector) nextCycleId() int {
	highest := 0
	for _, c := range j.Cycles {
		if c.Id > highest {
			highest = c.Id
		}
	}
	return highest + 1
}
