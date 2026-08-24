package service

import (
	"mycocycle-growbag-transfer-gate/internal/catalog"
)

// CatalogSummary is the read model of the reference catalog: strain lineages,
// substrate revisions, sterilizer runs, inoculation lines and reviewers.
type CatalogSummary struct {
	Strains          []catalog.Strain          `json:"strains"`
	Substrates       []catalog.Substrate       `json:"substrates"`
	SterilizerRuns   []catalog.SterilizerRun   `json:"sterilizer_runs"`
	InoculationLines []catalog.InoculationLine `json:"inoculation_lines"`
	Reviewers        []catalog.Reviewer        `json:"reviewers"`
}

// CatalogSummary returns the full reference catalog used for lock validation.
func (s *Service) CatalogSummary() CatalogSummary {
	return CatalogSummary{
		Strains:          s.catalog.Strains(),
		Substrates:       s.catalog.Substrates(),
		SterilizerRuns:   s.catalog.SterilizerRuns(),
		InoculationLines: s.catalog.InoculationLines(),
		Reviewers:        s.catalog.Reviewers(),
	}
}
