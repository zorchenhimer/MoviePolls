package logic

import (
	"github.com/zorchenhimer/MoviePolls/models"
)

func (b *backend) GetPastCycles(start, count int) ([]*models.Cycle, error) {
	return b.data.GetPastCycles(start, count)
}

func (b *backend) GetPreviousCycle() *models.Cycle {
	past, err := b.GetPastCycles(0, 1)
	if err != nil {
		b.l.Error("Error getting PastCycle: %v", err)
	}

	if len(past) > 0 {
		return past[0]
	}
	return nil
}

func (b *backend) GetCurrentCycle() (*models.Cycle, error) {
	return b.data.GetCurrentCycle()
}
