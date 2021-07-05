package logic

import "github.com/zorchenhimer/MoviePolls/models"

func (b *backend) AddLink(link *models.Link) (int, error) {
	return b.data.AddLink(link)
}
