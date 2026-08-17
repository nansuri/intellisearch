package services

import (
	"intellisearch/internal/models/entities"
	"intellisearch/internal/repositories"
)

type SiteService struct{ repo *repositories.SiteRepository }

func NewSiteService(repo *repositories.SiteRepository) *SiteService { return &SiteService{repo: repo} }
func (s *SiteService) Public() (entities.SiteSettings, error)       { return s.repo.Get() }
func (s *SiteService) Update(name string, tagline *string, copyright *string) (entities.SiteSettings, error) {
	value, err := s.repo.Get()
	if err != nil {
		return value, err
	}
	value.SiteName = name
	value.Tagline = tagline
	value.Copyright = copyright
	return value, s.repo.Update(value)
}
