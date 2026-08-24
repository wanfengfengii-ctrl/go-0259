package catalog

// InoculationLine is an inoculation line capability record (接种线能力). A line
// is bound to the sterilizer runs it serves and is only allowed to inoculate a
// fixed set of strain lineages.
type InoculationLine struct {
	ID             string   `json:"id"`
	AllowedStrains []string `json:"allowed_strains"`
}

// AllowsStrain reports whether the line may inoculate the given strain
// revision.
func (l InoculationLine) AllowsStrain(revision string) bool {
	for _, s := range l.AllowedStrains {
		if s == revision {
			return true
		}
	}
	return false
}

// InoculationLine returns the line capability with the given id.
func (r *Repo) InoculationLine(id string) (InoculationLine, bool) {
	l, ok := r.inoculationLines[id]
	return l, ok
}

// AddInoculationLine registers a line capability.
func (r *Repo) AddInoculationLine(l InoculationLine) { r.inoculationLines[l.ID] = l }

// InoculationLines returns all registered line capabilities.
func (r *Repo) InoculationLines() []InoculationLine {
	out := make([]InoculationLine, 0, len(r.inoculationLines))
	for _, l := range r.inoculationLines {
		out = append(out, l)
	}
	return out
}
