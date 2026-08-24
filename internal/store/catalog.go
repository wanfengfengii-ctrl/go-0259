package store

import (
	"context"

	"mycocycle-growbag-transfer-gate/internal/catalog"
)

// LoadCatalog reconstructs the catalog repository from the database.
func (s *SQLite) LoadCatalog() (*catalog.Repo, error) {
	r := catalog.NewRepo()
	rows, err := s.db.QueryContext(context.Background(), `
		SELECT id, revision, allowed_substrates, default_schedule,
		       maturity_min_value, maturity_min_scale, maturity_max_value, maturity_max_scale
		FROM catalog_strains`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	for rows.Next() {
		var st catalog.Strain
		var allowed, sched string
		if err := rows.Scan(&st.ID, &st.Revision, &allowed, &sched,
			&st.MaturityMin.Value, &st.MaturityMin.Scale,
			&st.MaturityMax.Value, &st.MaturityMax.Scale); err != nil {
			return nil, err
		}
		st.AllowedSubstrates = unmarshalStringSlice(allowed)
		st.DefaultSchedule = catalog.ScheduleTemplate{DayAges: unmarshalDayAges(sched)}
		r.AddStrain(st)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}

	srows, err := s.db.QueryContext(context.Background(), `
		SELECT id, revision, summary, moisture_target_value, moisture_target_scale,
		       ph_target_value, ph_target_scale, valid_from, voided
		FROM catalog_substrates`)
	if err != nil {
		return nil, err
	}
	defer srows.Close()
	for srows.Next() {
		var sub catalog.Substrate
		var voided int
		if err := srows.Scan(&sub.ID, &sub.Revision, &sub.Summary,
			&sub.MoistureTarget.Value, &sub.MoistureTarget.Scale,
			&sub.PHTarget.Value, &sub.PHTarget.Scale,
			&sub.ValidFrom, &voided); err != nil {
			return nil, err
		}
		sub.Voided = intBool(voided)
		r.AddSubstrate(sub)
	}
	if err := srows.Err(); err != nil {
		return nil, err
	}

	rrows, err := s.db.QueryContext(context.Background(), `
		SELECT id, summary, completed_at, inoculation_line, freshness_window
		FROM catalog_sterilizer_runs`)
	if err != nil {
		return nil, err
	}
	defer rrows.Close()
	for rrows.Next() {
		var run catalog.SterilizerRun
		if err := rrows.Scan(&run.ID, &run.Summary, &run.CompletedAt,
			&run.InoculationLine, &run.FreshnessWindow); err != nil {
			return nil, err
		}
		r.AddSterilizerRun(run)
	}
	if err := rrows.Err(); err != nil {
		return nil, err
	}

	lrows, err := s.db.QueryContext(context.Background(), `
		SELECT id, allowed_strains FROM catalog_inoculation_lines`)
	if err != nil {
		return nil, err
	}
	defer lrows.Close()
	for lrows.Next() {
		var line catalog.InoculationLine
		var allowed string
		if err := lrows.Scan(&line.ID, &allowed); err != nil {
			return nil, err
		}
		line.AllowedStrains = unmarshalStringSlice(allowed)
		r.AddInoculationLine(line)
	}
	if err := lrows.Err(); err != nil {
		return nil, err
	}

	vrows, err := s.db.QueryContext(context.Background(), `
		SELECT person, qualification, valid FROM catalog_reviewers`)
	if err != nil {
		return nil, err
	}
	defer vrows.Close()
	for vrows.Next() {
		var rv catalog.Reviewer
		var valid int
		if err := vrows.Scan(&rv.Person, &rv.Qualification, &valid); err != nil {
			return nil, err
		}
		rv.Valid = intBool(valid)
		r.AddReviewer(rv)
	}
	if err := vrows.Err(); err != nil {
		return nil, err
	}
	return r, nil
}
