package domain

// Cmp compares f and o after aligning their scales, returning -1, 0 or 1.
// Threshold and range comparisons across the domain are expressed in terms of
// Cmp so that scale alignment and overflow are checked in one place.
func (f Fixed) Cmp(o Fixed) (int, error) {
	scale := f.Scale
	if o.Scale > scale {
		scale = o.Scale
	}
	ff, err := f.Rescale(scale)
	if err != nil {
		return 0, err
	}
	oo, err := o.Rescale(scale)
	if err != nil {
		return 0, err
	}
	switch {
	case ff.Value < oo.Value:
		return -1, nil
	case ff.Value > oo.Value:
		return 1, nil
	default:
		return 0, nil
	}
}
