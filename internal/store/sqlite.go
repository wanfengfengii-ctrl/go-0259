package store

import (
	"context"
	"database/sql"
	"fmt"
	"sync"

	_ "modernc.org/sqlite"

	"mycocycle-growbag-transfer-gate/internal/catalog"
)

// SQLite is the SQLite WAL-backed implementation of Store.
type SQLite struct {
	db *sql.DB
	mu sync.Mutex
}

// Open opens (or creates) the SQLite database at path with WAL journaling and
// returns a migrated, seeded store.
func Open(path string) (*SQLite, error) {
	dsn := fmt.Sprintf("file:%s?_pragma=journal_mode(WAL)&_pragma=busy_timeout(5000)&_pragma=foreign_keys(0)", path)
	db, err := sql.Open("sqlite", dsn)
	if err != nil {
		return nil, err
	}
	// A single connection keeps the in-memory ordering and write serialization
	// deterministic for the single-node service.
	db.SetMaxOpenConns(1)
	if err := db.Ping(); err != nil {
		db.Close()
		return nil, err
	}
	if err := migrate(db); err != nil {
		db.Close()
		return nil, err
	}
	s := &SQLite{db: db}
	if err := s.seedCatalogIfEmpty(); err != nil {
		db.Close()
		return nil, err
	}
	return s, nil
}

// OpenMemory opens an in-memory store (used by tests and smoke runs).
func OpenMemory() (*SQLite, error) {
	return Open(":memory:")
}

// Close releases the database handle.
func (s *SQLite) Close() error {
	return s.db.Close()
}

// withTx runs fn inside a single immediate transaction. It serializes writers
// via the store mutex so that check-then-insert adjudication is atomic.
func (s *SQLite) withTx(ctx context.Context, fn func(*sql.Tx) error) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	if err := fn(tx); err != nil {
		_ = tx.Rollback()
		return err
	}
	return tx.Commit()
}

// seedCatalogIfEmpty populates the catalog tables from the deterministic seed
// dataset when they are empty.
func (s *SQLite) seedCatalogIfEmpty() error {
	var n int
	if err := s.db.QueryRow(`SELECT COUNT(*) FROM catalog_strains`).Scan(&n); err != nil {
		return err
	}
	if n > 0 {
		return nil
	}
	r := catalog.NewRepo()
	catalog.Seed(r)
	return s.withTx(context.Background(), func(tx *sql.Tx) error {
		for _, st := range r.Strains() {
			if _, err := tx.Exec(`
				INSERT INTO catalog_strains
				(id, revision, allowed_substrates, default_schedule,
				 maturity_min_value, maturity_min_scale, maturity_max_value, maturity_max_scale)
				VALUES (?, ?, ?, ?, ?, ?, ?, ?)`,
				st.ID, st.Revision, marshalStringSlice(st.AllowedSubstrates),
				marshalDayAges(st.DefaultSchedule.DayAges),
				st.MaturityMin.Value, st.MaturityMin.Scale,
				st.MaturityMax.Value, st.MaturityMax.Scale); err != nil {
				return err
			}
		}
		for _, sub := range r.Substrates() {
			if _, err := tx.Exec(`
				INSERT INTO catalog_substrates
				(id, revision, summary, moisture_target_value, moisture_target_scale,
				 ph_target_value, ph_target_scale, valid_from, voided)
				VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)`,
				sub.ID, sub.Revision, sub.Summary,
				sub.MoistureTarget.Value, sub.MoistureTarget.Scale,
				sub.PHTarget.Value, sub.PHTarget.Scale,
				sub.ValidFrom, boolInt(sub.Voided)); err != nil {
				return err
			}
		}
		for _, run := range r.SterilizerRuns() {
			if _, err := tx.Exec(`
				INSERT INTO catalog_sterilizer_runs
				(id, summary, completed_at, inoculation_line, freshness_window)
				VALUES (?, ?, ?, ?, ?)`,
				run.ID, run.Summary, run.CompletedAt, run.InoculationLine,
				run.FreshnessWindow); err != nil {
				return err
			}
		}
		for _, line := range r.InoculationLines() {
			if _, err := tx.Exec(`
				INSERT INTO catalog_inoculation_lines (id, allowed_strains)
				VALUES (?, ?)`,
				line.ID, marshalStringSlice(line.AllowedStrains)); err != nil {
				return err
			}
		}
		for _, rv := range r.Reviewers() {
			if _, err := tx.Exec(`
				INSERT INTO catalog_reviewers (person, qualification, valid)
				VALUES (?, ?, ?)`,
				rv.Person, rv.Qualification, boolInt(rv.Valid)); err != nil {
				return err
			}
		}
		return nil
	})
}
