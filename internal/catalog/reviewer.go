package catalog

import "mycocycle-growbag-transfer-gate/internal/domain"

// Reviewer is a person's qualification record (复核人员资质). A reviewer may
// participate in sampling confirmation or independent review only when their
// qualification is currently valid.
type Reviewer struct {
	Person        domain.PersonID        `json:"person"`
	Qualification domain.QualificationID `json:"qualification"`
	Valid         bool                   `json:"valid"`
}

// IsQualified reports whether the reviewer currently holds a valid
// qualification.
func (r Reviewer) IsQualified() bool {
	return r.Valid && r.Qualification != ""
}
