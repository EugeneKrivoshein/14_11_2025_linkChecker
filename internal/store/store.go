package store

import "github.com/EugeneKrivoshein/14_11_2025_linkChecker/models"

type Store interface {
	CreateSet([]string) (int64, *models.LinkSet, error)
	GetSet(int64) (*models.LinkSet, error)
	UpdateLinkResult(int64, string, models.LinkResult) error
	ListUnfinished() ([]*models.LinkSet, error)
	ListSets([]int64) ([]*models.LinkSet, error)
}
