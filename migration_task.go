package migrate

import (
	"time"

	"github.com/golang-migrate/migrate/v4/database"
)

const (
	// TaskVersionOffset is a sentinel offset added to a normal SQL
	// migration version. During the migration task for version V, the
	// database version is persisted as (V + TaskVersionOffset). This
	// enables the migrate package to detect and re-run a migration task
	// which errored after the corresponding SQL migration was applied
	// successfully.
	// Note that only the migration task is re-run, the SQL migration is
	// not re-applied.
	//
	// If the persisted version is >= TaskVersionOffset and not `dirty`,
	// then the migration task for (version - TaskVersionOffset) will be
	// re-run on the next migration run.
	// If the persisted version is `dirty`, manual intervention is required,
	// as it's not possible by the migration framework to determine whether
	// the migration task actually was executed successfully or not during
	// the last execution attempt.
	//
	// NOTE:
	// Changing this value is a breaking change for any database that
	// currently has a migration task version recorded. Do not change it
	// unless you also provide a safe transition strategy.
	// Also note that no SQL migration can use a
	// version >= TaskVersionOffset. Such versions are reserved for the
	// migration task phase, and any SQL migrations with such versions will
	// cause an error.
	TaskVersionOffset = 1000000000

	// DefaultSingleMigReadTimeout is the default timeout we use when
	// waiting for a single migration to be read.
	DefaultSingleMigReadTimeout = 30 * time.Second
)

// InTaskVersionRange returns true if the passed version is a migration task
// version.
func InTaskVersionRange(version int) bool {
	return version >= TaskVersionOffset
}

// SQLMigrationVersion returns the corresponding SQL migration version for the
// given version. If the version passed is a migration task version, the
// function will return the version for the corresponding SQL migration.
// If the version passed already is a SQL migration version, the function will
// return the passed version as is.
func SQLMigrationVersion(version int) int {
	if InTaskVersionRange(version) {
		return version - TaskVersionOffset
	}

	return version
}

// MigrationTask is a callback function type that can be used to execute a
// Golang based migration step after a SQL based migration step has been
// executed. The callback function receives the migration and the database
// driver as arguments.
type MigrationTask func(migr *Migration, driver database.Driver) error

// options is a set of optional options that can be set when a Migrate instance
// is created.
type options struct {
	// tasks is a map of MigrationTask functions that can be used to execute
	// a Golang based migration step after a SQL based migration step ha
	// been executed. The key is the migration version and the value is the
	// callback function that should be run _after_ the step was executed
	// (but within the same database transaction).
	tasks map[uint]MigrationTask
}

// defaultOptions returns a new options struct with default values.
func defaultOptions() options {
	return options{
		tasks: make(map[uint]MigrationTask),
	}
}

// Option is a function that can be used to set options on a Migrate instance.
type Option func(*options)

// WithMigrationTasks is an option that can be used to set a map of
// MigrationTask functions that can be used to execute a Golang based migration
// step after a SQL based migration step has been executed. The key is the
// migration version and the value is the task function that should be run
// _after_ the step was executed (but before the version is marked as cleanly
// executed). An error returned from the task will cause the migration to fail
// and will set the database version to the (version + TaskVersionOffset) but
// in a clean state. When the next migration run is executed, the task
// function will be re-run.
func WithMigrationTasks(tasks map[uint]MigrationTask) Option {
	return func(o *options) {
		o.tasks = tasks
	}
}

// WithMigrationTask is an option that can be used to set a MigrationTask
// function that can be used to execute a Golang based migration step after the
// SQL based migration step with the given version number has been executed. The
// task is the function that should be run _after_ the step was executed
// (but before the version is marked as cleanly executed). An error returned
// from the task will cause the migration to fail and will set the database
// version to the (version + TaskVersionOffset) but in a clean state. When the
// next migration run is executed, the task function will be re-run.
func WithMigrationTask(version uint, task MigrationTask) Option {
	return func(o *options) {
		o.tasks[version] = task
	}
}
