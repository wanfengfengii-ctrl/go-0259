package store

import "database/sql"

// schema is the full SQLite schema. Every table corresponds to a documented
// entity in the project data model, plus the idempotency ledger, sampling
// confirmations and audit events required by the domain rules and interfaces.
const schema = `
CREATE TABLE IF NOT EXISTS catalog_strains (
	id               TEXT PRIMARY KEY,
	revision         TEXT NOT NULL UNIQUE,
	allowed_substrates TEXT NOT NULL,
	default_schedule TEXT NOT NULL,
	maturity_min_value INTEGER NOT NULL,
	maturity_min_scale INTEGER NOT NULL,
	maturity_max_value INTEGER NOT NULL,
	maturity_max_scale INTEGER NOT NULL
);

CREATE TABLE IF NOT EXISTS catalog_substrates (
	id             TEXT PRIMARY KEY,
	revision       TEXT NOT NULL UNIQUE,
	summary        TEXT NOT NULL,
	moisture_target_value INTEGER NOT NULL,
	moisture_target_scale INTEGER NOT NULL,
	ph_target_value INTEGER NOT NULL,
	ph_target_scale INTEGER NOT NULL,
	valid_from     INTEGER NOT NULL,
	voided         INTEGER NOT NULL
);

CREATE TABLE IF NOT EXISTS catalog_sterilizer_runs (
	id                TEXT PRIMARY KEY,
	summary           TEXT NOT NULL UNIQUE,
	completed_at      INTEGER NOT NULL,
	inoculation_line  TEXT NOT NULL,
	freshness_window  INTEGER NOT NULL
);

CREATE TABLE IF NOT EXISTS catalog_inoculation_lines (
	id              TEXT PRIMARY KEY,
	allowed_strains TEXT NOT NULL
);

CREATE TABLE IF NOT EXISTS catalog_reviewers (
	person        TEXT PRIMARY KEY,
	qualification TEXT NOT NULL,
	valid         INTEGER NOT NULL
);

CREATE TABLE IF NOT EXISTS inspection_tasks (
	id            TEXT PRIMARY KEY,
	bag_batch     TEXT NOT NULL,
	generation    INTEGER NOT NULL,
	state         TEXT NOT NULL,
	snapshot      TEXT NOT NULL,
	created_at    INTEGER NOT NULL,
	final_version INTEGER NOT NULL
);

CREATE TABLE IF NOT EXISTS locked_bag_positions (
	task_id   TEXT NOT NULL,
	batch     TEXT NOT NULL,
	position  TEXT NOT NULL,
	sealed    INTEGER NOT NULL,
	sealed_by TEXT NOT NULL,
	sealed_at INTEGER NOT NULL,
	PRIMARY KEY (task_id, position)
);

CREATE TABLE IF NOT EXISTS occupancy_leases (
	id             INTEGER PRIMARY KEY AUTOINCREMENT,
	resource_type  TEXT NOT NULL,
	resource_id    TEXT NOT NULL,
	position       TEXT NOT NULL,
	window_start   INTEGER NOT NULL,
	window_end     INTEGER NOT NULL,
	task_id        TEXT NOT NULL,
	generation     INTEGER NOT NULL,
	state          TEXT NOT NULL,
	release_reason TEXT NOT NULL
);

CREATE UNIQUE INDEX IF NOT EXISTS idx_lease_active_exclusive
	ON occupancy_leases(resource_type, resource_id)
	WHERE state = 'active' AND resource_type != 'probe_window';

CREATE TABLE IF NOT EXISTS maturity_cells (
	task_id              TEXT NOT NULL,
	generation           INTEGER NOT NULL,
	day_age              INTEGER NOT NULL,
	position             TEXT NOT NULL,
	mycelium_value       INTEGER NOT NULL,
	mycelium_scale       INTEGER NOT NULL,
	contamination_count  INTEGER NOT NULL,
	bag_damage           INTEGER NOT NULL,
	missing              INTEGER NOT NULL,
	summary              TEXT NOT NULL,
	observer             TEXT NOT NULL,
	PRIMARY KEY (task_id, generation, day_age, position)
);

CREATE TABLE IF NOT EXISTS physchem_readings (
	id            INTEGER PRIMARY KEY AUTOINCREMENT,
	task_id       TEXT NOT NULL,
	generation    INTEGER NOT NULL,
	position      TEXT NOT NULL,
	day_age       INTEGER NOT NULL,
	metric        TEXT NOT NULL,
	value         INTEGER NOT NULL,
	scale         INTEGER NOT NULL,
	source_device TEXT NOT NULL,
	status        TEXT NOT NULL,
	derived       TEXT NOT NULL
);

CREATE TABLE IF NOT EXISTS device_attempts (
	id          INTEGER PRIMARY KEY AUTOINCREMENT,
	device_type TEXT NOT NULL,
	device_id   TEXT NOT NULL,
	object      TEXT NOT NULL,
	task_id     TEXT NOT NULL,
	generation  INTEGER NOT NULL,
	at          INTEGER NOT NULL,
	script_seq  INTEGER NOT NULL,
	result      TEXT NOT NULL,
	retry_count INTEGER NOT NULL,
	error_code  TEXT NOT NULL
);

CREATE TABLE IF NOT EXISTS contamination_evidence (
	id                  INTEGER PRIMARY KEY AUTOINCREMENT,
	task_id             TEXT NOT NULL,
	recheck_generation  INTEGER NOT NULL,
	version             INTEGER NOT NULL,
	position            TEXT NOT NULL,
	day_age             INTEGER NOT NULL,
	well                TEXT NOT NULL,
	source              TEXT NOT NULL,
	positive            INTEGER NOT NULL,
	summary             TEXT NOT NULL
);

CREATE TABLE IF NOT EXISTS reviews (
	task_id       TEXT NOT NULL,
	generation    INTEGER NOT NULL,
	person        TEXT NOT NULL,
	qualification TEXT NOT NULL,
	conclusion    TEXT NOT NULL,
	digest        TEXT NOT NULL,
	operation     TEXT NOT NULL,
	at            INTEGER NOT NULL,
	PRIMARY KEY (task_id, generation, person)
);

CREATE TABLE IF NOT EXISTS final_decisions (
	task_id           TEXT PRIMARY KEY,
	final_type        TEXT NOT NULL,
	credential        TEXT NOT NULL,
	winning_operation TEXT NOT NULL,
	written_at        INTEGER NOT NULL,
	summary           TEXT NOT NULL
);

CREATE TABLE IF NOT EXISTS operations (
	task_id     TEXT NOT NULL,
	generation  INTEGER NOT NULL,
	operation   TEXT NOT NULL,
	digest      TEXT NOT NULL,
	result_json TEXT NOT NULL,
	at          INTEGER NOT NULL,
	PRIMARY KEY (task_id, generation, operation)
);

CREATE TABLE IF NOT EXISTS sampling_confirmations (
	task_id          TEXT NOT NULL,
	generation       INTEGER NOT NULL,
	person           TEXT NOT NULL,
	batch            TEXT NOT NULL,
	positions_digest TEXT NOT NULL,
	operation        TEXT NOT NULL,
	at               INTEGER NOT NULL,
	PRIMARY KEY (task_id, generation, person)
);

CREATE TABLE IF NOT EXISTS audit_events (
	id         INTEGER PRIMARY KEY AUTOINCREMENT,
	task_id    TEXT NOT NULL,
	generation INTEGER NOT NULL,
	at         INTEGER NOT NULL,
	operation  TEXT NOT NULL,
	action     TEXT NOT NULL,
	detail     TEXT NOT NULL
);
`

func migrate(db *sql.DB) error {
	_, err := db.Exec(schema)
	return err
}
