package migrate

import "github.com/golang-migrate/migrate/v4/database"

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
// and the step to be marked as dirty.
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
// from the task will cause the migration to fail and the step to be marked as
// dirty.
func WithMigrationTask(version uint, task MigrationTask) Option {
	return func(o *options) {
		o.tasks[version] = task
	}
}
