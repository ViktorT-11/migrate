package migrate

import (
	"bytes"
	"database/sql"
	"errors"
	"io"
	"log"
	"os"
	"strings"
	"testing"

	"github.com/golang-migrate/migrate/v4/database"
	dStub "github.com/golang-migrate/migrate/v4/database/stub"
	"github.com/golang-migrate/migrate/v4/source"
	sStub "github.com/golang-migrate/migrate/v4/source/stub"
)

// sourceStubMigrations hold the following migrations:
// u = up migration, d = down migration, n = version
//
//	|  1  |  -  |  3  |  4  |  5  |  -  |  7  |
//	| u d |  -  | u   | u d |   d |  -  | u d |
var sourceStubMigrations *source.Migrations

const (
	srcDrvNameStub = "stub"
	dbDrvNameStub  = "stub"
)

func init() {
	sourceStubMigrations = source.NewMigrations()
	sourceStubMigrations.Append(&source.Migration{Version: 1, Direction: source.Up, Identifier: "CREATE 1"})
	sourceStubMigrations.Append(&source.Migration{Version: 1, Direction: source.Down, Identifier: "DROP 1"})
	sourceStubMigrations.Append(&source.Migration{Version: 3, Direction: source.Up, Identifier: "CREATE 3"})
	sourceStubMigrations.Append(&source.Migration{Version: 4, Direction: source.Up, Identifier: "CREATE 4"})
	sourceStubMigrations.Append(&source.Migration{Version: 4, Direction: source.Down, Identifier: "DROP 4"})
	sourceStubMigrations.Append(&source.Migration{Version: 5, Direction: source.Down, Identifier: "DROP 5"})
	sourceStubMigrations.Append(&source.Migration{Version: 7, Direction: source.Up, Identifier: "CREATE 7"})
	sourceStubMigrations.Append(&source.Migration{Version: 7, Direction: source.Down, Identifier: "DROP 7"})
}

type DummyInstance struct{ Name string }

func TestNew(t *testing.T) {
	m, err := New("stub://", "stub://")
	if err != nil {
		t.Fatal(err)
	}

	if m.sourceName != srcDrvNameStub {
		t.Errorf("expected stub, got %v", m.sourceName)
	}
	if m.sourceDrv == nil {
		t.Error("expected sourceDrv not to be nil")
	}

	if m.databaseName != dbDrvNameStub {
		t.Errorf("expected stub, got %v", m.databaseName)
	}
	if m.databaseDrv == nil {
		t.Error("expected databaseDrv not to be nil")
	}
}

func ExampleNew() {
	// Read migrations from /home/mattes/migrations and connect to a local postgres database.
	m, err := New("file:///home/mattes/migrations", "postgres://mattes:secret@localhost:5432/database?sslmode=disable")
	if err != nil {
		log.Fatal(err)
	}

	// Migrate all the way up ...
	if err := m.Up(); err != nil && err != ErrNoChange {
		log.Fatal(err)
	}
}

func TestNewWithDatabaseInstance(t *testing.T) {
	dummyDb := &DummyInstance{"database"}
	dbInst, err := dStub.WithInstance(dummyDb, &dStub.Config{})
	if err != nil {
		t.Fatal(err)
	}

	m, err := NewWithDatabaseInstance("stub://", dbDrvNameStub, dbInst)
	if err != nil {
		t.Fatal(err)
	}

	if m.sourceName != srcDrvNameStub {
		t.Errorf("expected stub, got %v", m.sourceName)
	}
	if m.sourceDrv == nil {
		t.Error("expected sourceDrv not to be nil")
	}

	if m.databaseName != dbDrvNameStub {
		t.Errorf("expected stub, got %v", m.databaseName)
	}
	if m.databaseDrv == nil {
		t.Error("expected databaseDrv not to be nil")
	}
}

func ExampleNewWithDatabaseInstance() {
	// Create and use an existing database instance.
	db, err := sql.Open("postgres", "postgres://mattes:secret@localhost:5432/database?sslmode=disable")
	if err != nil {
		log.Fatal(err)
	}
	defer func() {
		if err := db.Close(); err != nil {
			log.Fatal(err)
		}
	}()

	// Create driver instance from db.
	// Check each driver if it supports the WithInstance function.
	// `import "github.com/golang-migrate/migrate/v4/database/postgres"`
	instance, err := dStub.WithInstance(db, &dStub.Config{})
	if err != nil {
		log.Fatal(err)
	}

	// Read migrations from /home/mattes/migrations and connect to a local postgres database.
	m, err := NewWithDatabaseInstance("file:///home/mattes/migrations", "postgres", instance)
	if err != nil {
		log.Fatal(err)
	}

	// Migrate all the way up ...
	if err := m.Up(); err != nil {
		log.Fatal(err)
	}
}

func TestNewWithSourceInstance(t *testing.T) {
	dummySource := &DummyInstance{"source"}
	sInst, err := sStub.WithInstance(dummySource, &sStub.Config{})
	if err != nil {
		t.Fatal(err)
	}

	m, err := NewWithSourceInstance(srcDrvNameStub, sInst, "stub://")
	if err != nil {
		t.Fatal(err)
	}

	if m.sourceName != srcDrvNameStub {
		t.Errorf("expected stub, got %v", m.sourceName)
	}
	if m.sourceDrv == nil {
		t.Error("expected sourceDrv not to be nil")
	}

	if m.databaseName != dbDrvNameStub {
		t.Errorf("expected stub, got %v", m.databaseName)
	}
	if m.databaseDrv == nil {
		t.Error("expected databaseDrv not to be nil")
	}
}

func ExampleNewWithSourceInstance() {
	di := &DummyInstance{"think any client required for a source here"}

	// Create driver instance from DummyInstance di.
	// Check each driver if it support the WithInstance function.
	// `import "github.com/golang-migrate/migrate/v4/source/stub"`
	instance, err := sStub.WithInstance(di, &sStub.Config{})
	if err != nil {
		log.Fatal(err)
	}

	// Read migrations from Stub and connect to a local postgres database.
	m, err := NewWithSourceInstance(srcDrvNameStub, instance, "postgres://mattes:secret@localhost:5432/database?sslmode=disable")
	if err != nil {
		log.Fatal(err)
	}

	// Migrate all the way up ...
	if err := m.Up(); err != nil {
		log.Fatal(err)
	}
}

func TestNewWithInstance(t *testing.T) {
	dummyDb := &DummyInstance{"database"}
	dbInst, err := dStub.WithInstance(dummyDb, &dStub.Config{})
	if err != nil {
		t.Fatal(err)
	}

	dummySource := &DummyInstance{"source"}
	sInst, err := sStub.WithInstance(dummySource, &sStub.Config{})
	if err != nil {
		t.Fatal(err)
	}

	m, err := NewWithInstance(srcDrvNameStub, sInst, dbDrvNameStub, dbInst)
	if err != nil {
		t.Fatal(err)
	}

	if m.sourceName != srcDrvNameStub {
		t.Errorf("expected stub, got %v", m.sourceName)
	}
	if m.sourceDrv == nil {
		t.Error("expected sourceDrv not to be nil")
	}

	if m.databaseName != dbDrvNameStub {
		t.Errorf("expected stub, got %v", m.databaseName)
	}
	if m.databaseDrv == nil {
		t.Error("expected databaseDrv not to be nil")
	}
}

func ExampleNewWithInstance() {
	// See NewWithDatabaseInstance and NewWithSourceInstance for an example.
}

func TestClose(t *testing.T) {
	m, _ := New("stub://", "stub://")
	sourceErr, databaseErr := m.Close()
	if sourceErr != nil {
		t.Error(sourceErr)
	}
	if databaseErr != nil {
		t.Error(databaseErr)
	}
}

func TestMigrate(t *testing.T) {
	m, _ := New("stub://", "stub://")
	m.sourceDrv.(*sStub.Stub).Migrations = sourceStubMigrations
	dbDrv := m.databaseDrv.(*dStub.Stub)

	tt := []struct {
		version       uint
		expectErr     error
		expectVersion uint
		expectSeq     migrationSequence
	}{
		// migrate all the way Up in single steps
		{
			version:   0,
			expectErr: os.ErrNotExist,
		},
		{
			version:       1,
			expectVersion: 1,
			expectSeq: migrationSequence{
				mr("CREATE 1"),
			},
		},
		{
			version:   2,
			expectErr: os.ErrNotExist,
			expectSeq: migrationSequence{
				mr("CREATE 1"),
			},
		},
		{
			version:       3,
			expectVersion: 3,
			expectSeq: migrationSequence{
				mr("CREATE 1"),
				mr("CREATE 3"),
			},
		},
		{
			version:       4,
			expectVersion: 4,
			expectSeq: migrationSequence{
				mr("CREATE 1"),
				mr("CREATE 3"),
				mr("CREATE 4"),
			},
		},
		{
			version:       5,
			expectVersion: 5,
			expectSeq: migrationSequence{ // 5 has no up migration
				mr("CREATE 1"),
				mr("CREATE 3"),
				mr("CREATE 4"),
			},
		},
		{
			version:   6,
			expectErr: os.ErrNotExist,
			expectSeq: migrationSequence{
				mr("CREATE 1"),
				mr("CREATE 3"),
				mr("CREATE 4"),
			},
		},
		{
			version:       7,
			expectVersion: 7,
			expectSeq: migrationSequence{
				mr("CREATE 1"),
				mr("CREATE 3"),
				mr("CREATE 4"),
				mr("CREATE 7"),
			},
		},
		{
			version:   8,
			expectErr: os.ErrNotExist,
			expectSeq: migrationSequence{
				mr("CREATE 1"),
				mr("CREATE 3"),
				mr("CREATE 4"),
				mr("CREATE 7"),
			},
		},

		// migrate all the way Down in single steps
		{
			version:   6,
			expectErr: os.ErrNotExist,
			expectSeq: migrationSequence{
				mr("CREATE 1"),
				mr("CREATE 3"),
				mr("CREATE 4"),
				mr("CREATE 7"),
			},
		},
		{
			version:       5,
			expectVersion: 5,
			expectSeq: migrationSequence{
				mr("CREATE 1"),
				mr("CREATE 3"),
				mr("CREATE 4"),
				mr("CREATE 7"),
				mr("DROP 7"),
			},
		},
		{
			version:       4,
			expectVersion: 4,
			expectSeq: migrationSequence{
				mr("CREATE 1"),
				mr("CREATE 3"),
				mr("CREATE 4"),
				mr("CREATE 7"),
				mr("DROP 7"),
				mr("DROP 5"),
			},
		},
		{
			version:       3,
			expectVersion: 3,
			expectSeq: migrationSequence{
				mr("CREATE 1"),
				mr("CREATE 3"),
				mr("CREATE 4"),
				mr("CREATE 7"),
				mr("DROP 7"),
				mr("DROP 5"),
				mr("DROP 4"),
			},
		},
		{
			version:   2,
			expectErr: os.ErrNotExist,
			expectSeq: migrationSequence{
				mr("CREATE 1"),
				mr("CREATE 3"),
				mr("CREATE 4"),
				mr("CREATE 7"),
				mr("DROP 7"),
				mr("DROP 5"),
				mr("DROP 4"),
			},
		},
		{
			version:       1,
			expectVersion: 1,
			expectSeq: migrationSequence{ // 3 has no down migration
				mr("CREATE 1"),
				mr("CREATE 3"),
				mr("CREATE 4"),
				mr("CREATE 7"),
				mr("DROP 7"),
				mr("DROP 5"),
				mr("DROP 4"),
			},
		},
		{
			version:   0,
			expectErr: os.ErrNotExist,
			expectSeq: migrationSequence{
				mr("CREATE 1"),
				mr("CREATE 3"),
				mr("CREATE 4"),
				mr("CREATE 7"),
				mr("DROP 7"),
				mr("DROP 5"),
				mr("DROP 4"),
			},
		},

		// migrate all the way Up in one step
		{
			version:       7,
			expectVersion: 7,
			expectSeq: migrationSequence{
				mr("CREATE 1"),
				mr("CREATE 3"),
				mr("CREATE 4"),
				mr("CREATE 7"),
				mr("DROP 7"),
				mr("DROP 5"),
				mr("DROP 4"),
				mr("CREATE 3"),
				mr("CREATE 4"),
				mr("CREATE 7"),
			},
		},

		// migrate all the way Down in one step
		{
			version:       1,
			expectVersion: 1,
			expectSeq: migrationSequence{
				mr("CREATE 1"),
				mr("CREATE 3"),
				mr("CREATE 4"),
				mr("CREATE 7"),
				mr("DROP 7"),
				mr("DROP 5"),
				mr("DROP 4"),
				mr("CREATE 3"),
				mr("CREATE 4"),
				mr("CREATE 7"),
				mr("DROP 7"),
				mr("DROP 5"),
				mr("DROP 4"),
			},
		},

		// can't migrate the same version twice
		{
			version:   1,
			expectErr: ErrNoChange,
			expectSeq: migrationSequence{
				mr("CREATE 1"),
				mr("CREATE 3"),
				mr("CREATE 4"),
				mr("CREATE 7"),
				mr("DROP 7"),
				mr("DROP 5"),
				mr("DROP 4"),
				mr("CREATE 3"),
				mr("CREATE 4"),
				mr("CREATE 7"),
				mr("DROP 7"),
				mr("DROP 5"),
				mr("DROP 4"),
			},
		},
	}

	for i, v := range tt {
		err := m.Migrate(v.version)
		if (v.expectErr == os.ErrNotExist && !errors.Is(err, os.ErrNotExist)) ||
			(v.expectErr != os.ErrNotExist && err != v.expectErr) {
			t.Errorf("expected err %v, got %v, in %v", v.expectErr, err, i)

		} else if err == nil {
			version, _, err := m.Version()
			if err != nil {
				t.Error(err)
			}
			if version != v.expectVersion {
				t.Errorf("expected version %v, got %v, in %v", v.expectVersion, version, i)
			}
		}
		equalDbSeq(t, i, v.expectSeq, dbDrv)
	}
}

func TestMigrateDirty(t *testing.T) {
	m, _ := New("stub://", "stub://")
	dbDrv := m.databaseDrv.(*dStub.Stub)
	if err := dbDrv.SetVersion(0, true); err != nil {
		t.Fatal(err)
	}

	err := m.Migrate(1)
	if _, ok := err.(ErrDirty); !ok {
		t.Fatalf("expected ErrDirty, got %v", err)
	}
}

func TestSteps(t *testing.T) {
	m, _ := New("stub://", "stub://")
	m.sourceDrv.(*sStub.Stub).Migrations = sourceStubMigrations
	dbDrv := m.databaseDrv.(*dStub.Stub)

	tt := []struct {
		steps         int
		expectErr     error
		expectVersion int
		expectSeq     migrationSequence
	}{
		// step must be != 0
		{
			steps:     0,
			expectErr: ErrNoChange,
		},

		// can't go Down if ErrNilVersion
		{
			steps:     -1,
			expectErr: os.ErrNotExist,
		},

		// migrate all the way Up
		{
			steps:         1,
			expectVersion: 1,
			expectSeq: migrationSequence{
				mr("CREATE 1")},
		},
		{
			steps:         1,
			expectVersion: 3,
			expectSeq: migrationSequence{
				mr("CREATE 1"),
				mr("CREATE 3"),
			},
		},
		{
			steps:         1,
			expectVersion: 4,
			expectSeq: migrationSequence{
				mr("CREATE 1"),
				mr("CREATE 3"),
				mr("CREATE 4"),
			},
		},
		{
			steps:         1,
			expectVersion: 5,
			expectSeq: migrationSequence{
				mr("CREATE 1"),
				mr("CREATE 3"),
				mr("CREATE 4"),
			},
		},
		{
			steps:         1,
			expectVersion: 7,
			expectSeq: migrationSequence{
				mr("CREATE 1"),
				mr("CREATE 3"),
				mr("CREATE 4"),
				mr("CREATE 7"),
			},
		},
		{
			steps:     1,
			expectErr: os.ErrNotExist,
			expectSeq: migrationSequence{
				mr("CREATE 1"),
				mr("CREATE 3"),
				mr("CREATE 4"),
				mr("CREATE 7"),
			},
		},

		// migrate all the way Down
		{
			steps:         -1,
			expectVersion: 5,
			expectSeq: migrationSequence{
				mr("CREATE 1"),
				mr("CREATE 3"),
				mr("CREATE 4"),
				mr("CREATE 7"),
				mr("DROP 7"),
			},
		},
		{
			steps:         -1,
			expectVersion: 4,
			expectSeq: migrationSequence{
				mr("CREATE 1"),
				mr("CREATE 3"),
				mr("CREATE 4"),
				mr("CREATE 7"),
				mr("DROP 7"),
				mr("DROP 5"),
			},
		},
		{
			steps:         -1,
			expectVersion: 3,
			expectSeq: migrationSequence{
				mr("CREATE 1"),
				mr("CREATE 3"),
				mr("CREATE 4"),
				mr("CREATE 7"),
				mr("DROP 7"),
				mr("DROP 5"),
				mr("DROP 4"),
			},
		},
		{
			steps:         -1,
			expectVersion: 1,
			expectSeq: migrationSequence{
				mr("CREATE 1"),
				mr("CREATE 3"),
				mr("CREATE 4"),
				mr("CREATE 7"),
				mr("DROP 7"),
				mr("DROP 5"),
				mr("DROP 4"),
			},
		},
		{
			steps:         -1,
			expectVersion: -1,
			expectSeq: migrationSequence{
				mr("CREATE 1"),
				mr("CREATE 3"),
				mr("CREATE 4"),
				mr("CREATE 7"),
				mr("DROP 7"),
				mr("DROP 5"),
				mr("DROP 4"),
				mr("DROP 1"),
			},
		},

		// migrate Up in bigger step
		{
			steps:         4,
			expectVersion: 5,
			expectSeq: migrationSequence{
				mr("CREATE 1"),
				mr("CREATE 3"),
				mr("CREATE 4"),
				mr("CREATE 7"),
				mr("DROP 7"),
				mr("DROP 5"),
				mr("DROP 4"),
				mr("DROP 1"),
				mr("CREATE 1"),
				mr("CREATE 3"),
				mr("CREATE 4"),
			},
		},

		// apply one migration, then reaches out of boundary
		{
			steps:         2,
			expectErr:     ErrShortLimit{1},
			expectVersion: 7,
			expectSeq: migrationSequence{
				mr("CREATE 1"),
				mr("CREATE 3"),
				mr("CREATE 4"),
				mr("CREATE 7"),
				mr("DROP 7"),
				mr("DROP 5"),
				mr("DROP 4"),
				mr("DROP 1"),
				mr("CREATE 1"),
				mr("CREATE 3"),
				mr("CREATE 4"),
				mr("CREATE 7"),
			},
		},

		// migrate Down in bigger step
		{
			steps:         -4,
			expectVersion: 1,
			expectSeq: migrationSequence{
				mr("CREATE 1"),
				mr("CREATE 3"),
				mr("CREATE 4"),
				mr("CREATE 7"),
				mr("DROP 7"),
				mr("DROP 5"),
				mr("DROP 4"),
				mr("DROP 1"),
				mr("CREATE 1"),
				mr("CREATE 3"),
				mr("CREATE 4"),
				mr("CREATE 7"),
				mr("DROP 7"),
				mr("DROP 5"),
				mr("DROP 4"),
			},
		},

		// apply one migration, then reaches out of boundary
		{
			steps:         -2,
			expectErr:     ErrShortLimit{1},
			expectVersion: -1,
			expectSeq: migrationSequence{
				mr("CREATE 1"),
				mr("CREATE 3"),
				mr("CREATE 4"),
				mr("CREATE 7"),
				mr("DROP 7"),
				mr("DROP 5"),
				mr("DROP 4"),
				mr("DROP 1"),
				mr("CREATE 1"),
				mr("CREATE 3"),
				mr("CREATE 4"),
				mr("CREATE 7"),
				mr("DROP 7"),
				mr("DROP 5"),
				mr("DROP 4"),
				mr("DROP 1"),
			},
		},
	}

	for i, v := range tt {
		err := m.Steps(v.steps)
		if (v.expectErr == os.ErrNotExist && !errors.Is(err, os.ErrNotExist)) ||
			(v.expectErr != os.ErrNotExist && err != v.expectErr) {
			t.Errorf("expected err %v, got %v, in %v", v.expectErr, err, i)

		} else if err == nil {
			version, _, err := m.Version()
			if err != ErrNilVersion && err != nil {
				t.Error(err)
			}
			if v.expectVersion == -1 && err != ErrNilVersion {
				t.Errorf("expected ErrNilVersion, got %v, in %v", version, i)

			} else if v.expectVersion >= 0 && version != uint(v.expectVersion) {
				t.Errorf("expected version %v, got %v, in %v", v.expectVersion, version, i)
			}
		}
		equalDbSeq(t, i, v.expectSeq, dbDrv)
	}
}

func TestStepsDirty(t *testing.T) {
	m, _ := New("stub://", "stub://")
	dbDrv := m.databaseDrv.(*dStub.Stub)
	if err := dbDrv.SetVersion(0, true); err != nil {
		t.Fatal(err)
	}

	err := m.Steps(1)
	if _, ok := err.(ErrDirty); !ok {
		t.Fatalf("expected ErrDirty, got %v", err)
	}
}

func TestUpAndDown(t *testing.T) {
	m, _ := New("stub://", "stub://")
	m.sourceDrv.(*sStub.Stub).Migrations = sourceStubMigrations
	dbDrv := m.databaseDrv.(*dStub.Stub)

	// go Up first
	if err := m.Up(); err != nil {
		t.Fatal(err)
	}
	expectedSequence := migrationSequence{
		mr("CREATE 1"),
		mr("CREATE 3"),
		mr("CREATE 4"),
		mr("CREATE 7"),
	}
	equalDbSeq(t, 0, expectedSequence, dbDrv)

	// go Down
	if err := m.Down(); err != nil {
		t.Fatal(err)
	}
	expectedSequence = migrationSequence{
		mr("CREATE 1"),
		mr("CREATE 3"),
		mr("CREATE 4"),
		mr("CREATE 7"),
		mr("DROP 7"),
		mr("DROP 5"),
		mr("DROP 4"),
		mr("DROP 1"),
	}
	equalDbSeq(t, 1, expectedSequence, dbDrv)

	// go 1 Up and then all the way Up
	if err := m.Steps(1); err != nil {
		t.Fatal(err)
	}
	expectedSequence = migrationSequence{
		mr("CREATE 1"),
		mr("CREATE 3"),
		mr("CREATE 4"),
		mr("CREATE 7"),
		mr("DROP 7"),
		mr("DROP 5"),
		mr("DROP 4"),
		mr("DROP 1"),
		mr("CREATE 1"),
	}
	equalDbSeq(t, 2, expectedSequence, dbDrv)

	if err := m.Up(); err != nil {
		t.Fatal(err)
	}
	expectedSequence = migrationSequence{
		mr("CREATE 1"),
		mr("CREATE 3"),
		mr("CREATE 4"),
		mr("CREATE 7"),
		mr("DROP 7"),
		mr("DROP 5"),
		mr("DROP 4"),
		mr("DROP 1"),
		mr("CREATE 1"),
		mr("CREATE 3"),
		mr("CREATE 4"),
		mr("CREATE 7"),
	}
	equalDbSeq(t, 3, expectedSequence, dbDrv)

	// go 1 Down and then all the way Down
	if err := m.Steps(-1); err != nil {
		t.Fatal(err)
	}
	expectedSequence = migrationSequence{
		mr("CREATE 1"),
		mr("CREATE 3"),
		mr("CREATE 4"),
		mr("CREATE 7"),
		mr("DROP 7"),
		mr("DROP 5"),
		mr("DROP 4"),
		mr("DROP 1"),
		mr("CREATE 1"),
		mr("CREATE 3"),
		mr("CREATE 4"),
		mr("CREATE 7"),
		mr("DROP 7"),
	}
	equalDbSeq(t, 1, expectedSequence, dbDrv)

	if err := m.Down(); err != nil {
		t.Fatal(err)
	}
	expectedSequence = migrationSequence{
		mr("CREATE 1"),
		mr("CREATE 3"),
		mr("CREATE 4"),
		mr("CREATE 7"),
		mr("DROP 7"),
		mr("DROP 5"),
		mr("DROP 4"),
		mr("DROP 1"),
		mr("CREATE 1"),
		mr("CREATE 3"),
		mr("CREATE 4"),
		mr("CREATE 7"),
		mr("DROP 7"),
		mr("DROP 5"),
		mr("DROP 4"),
		mr("DROP 1"),
	}
	equalDbSeq(t, 1, expectedSequence, dbDrv)
}

func TestMigrationTask(t *testing.T) {
	m, _ := New("stub://", "stub://", WithMigrationTasks(
		map[uint]MigrationTask{
			1: func(m *Migration, driver database.Driver) error {
				return driver.Run(
					strings.NewReader("CALLBACK 1"),
				)
			},
			7: func(m *Migration, driver database.Driver) error {
				return driver.Run(
					strings.NewReader("CALLBACK 7"),
				)
			},
		},
	))
	m.sourceDrv.(*sStub.Stub).Migrations = sourceStubMigrations
	dbDrv := m.databaseDrv.(*dStub.Stub)

	// go Up first
	if err := m.Up(); err != nil {
		t.Fatal(err)
	}
	expectedSequence := migrationSequence{
		mr("CREATE 1"),
		mr("CALLBACK 1"),
		mr("CREATE 3"),
		mr("CREATE 4"),
		mr("CREATE 7"),
		mr("CALLBACK 7"),
	}
	equalDbSeq(t, 0, expectedSequence, dbDrv)

	if !bytes.Equal(dbDrv.LastRunMigration, []byte("CALLBACK 7")) {
		t.Fatalf("expected database last migration to be callback 7, "+
			"got %s", dbDrv.LastRunMigration)
	}

	// go Down
	if err := m.Down(); err != nil {
		t.Fatal(err)
	}
	expectedSequence = migrationSequence{
		mr("CREATE 1"),
		mr("CALLBACK 1"),
		mr("CREATE 3"),
		mr("CREATE 4"),
		mr("CREATE 7"),
		mr("CALLBACK 7"),
		mr("DROP 7"),
		mr("CALLBACK 7"),
		mr("DROP 5"),
		mr("DROP 4"),
		mr("DROP 1"),
		mr("CALLBACK 1"),
	}
	equalDbSeq(t, 1, expectedSequence, dbDrv)

	if !bytes.Equal(dbDrv.LastRunMigration, []byte("CALLBACK 1")) {
		t.Fatalf("expected database last migration to be callback 1, "+
			"got %s", dbDrv.LastRunMigration)
	}

	// go 1 Up and then all the way Up
	if err := m.Steps(1); err != nil {
		t.Fatal(err)
	}
	expectedSequence = migrationSequence{
		mr("CREATE 1"),
		mr("CALLBACK 1"),
		mr("CREATE 3"),
		mr("CREATE 4"),
		mr("CREATE 7"),
		mr("CALLBACK 7"),
		mr("DROP 7"),
		mr("CALLBACK 7"),
		mr("DROP 5"),
		mr("DROP 4"),
		mr("DROP 1"),
		mr("CALLBACK 1"),
		mr("CREATE 1"),
		mr("CALLBACK 1"),
	}
	equalDbSeq(t, 2, expectedSequence, dbDrv)

	if err := m.Up(); err != nil {
		t.Fatal(err)
	}
	expectedSequence = migrationSequence{
		mr("CREATE 1"),
		mr("CALLBACK 1"),
		mr("CREATE 3"),
		mr("CREATE 4"),
		mr("CREATE 7"),
		mr("CALLBACK 7"),
		mr("DROP 7"),
		mr("CALLBACK 7"),
		mr("DROP 5"),
		mr("DROP 4"),
		mr("DROP 1"),
		mr("CALLBACK 1"),
		mr("CREATE 1"),
		mr("CALLBACK 1"),
		mr("CREATE 3"),
		mr("CREATE 4"),
		mr("CREATE 7"),
		mr("CALLBACK 7"),
	}
	equalDbSeq(t, 3, expectedSequence, dbDrv)
}

func TestMigrationTaskError(t *testing.T) {
	// This test simulates a migration task that fails, and ensures that:
	// 1) The migration process stops and returns the task error.
	// 2) The migration version is set to the migration task version
	//    (version + TaskVersionOffset) and is not dirty.
	// 3) Re-running Up will re-attempt the migration task.
	// 4) Down will execute also re-execute a failed migration task
	//    at a given version, before attempting the downgrade.
	// 5) Once the migration task succeeds, the migration can finalize
	//    and reach the latest version cleanly.
	var (
		cbError    = errors.New("migration task failure")
		shouldFail = true
	)

	m, _ := New("stub://", "stub://",
		// Create a task that will error when shouldFail is true.
		WithMigrationTask(7, func(migr *Migration,
			driver database.Driver) error {

			// record that the task was executed
			if shouldFail {
				err := driver.Run(strings.NewReader(
					"CALLBACK 7 FAILURE",
				))
				if err != nil {
					return err
				}

				return cbError
			}

			err := driver.Run(strings.NewReader(
				"CALLBACK 7 SUCCESS",
			))
			if err != nil {
				return err
			}

			return nil
		}),
	)
	m.sourceDrv.(*sStub.Stub).Migrations = sourceStubMigrations
	dbDrv := m.databaseDrv.(*dStub.Stub)

	// Helper to check the migration version and dirty state.
	checkVersion := func(expVer int) {
		v, dirty, err := dbDrv.Version()
		if err != nil {
			t.Fatal(err)
		}

		if v != expVer {
			t.Fatalf("expected version %d, got v=%d", expVer, v)
		}
		if dirty {
			t.Fatalf("expected clean version, but was dirty")
		}
	}

	// 1) Run Up — the migration task for 7 should fail.
	err := m.Up()
	if !errors.Is(err, cbError) {
		t.Fatal("expected cbError from failing migration task")
	}

	// The sequence should show the migrations up to 7, and then the
	// failing callback.
	expectedSequence := migrationSequence{
		mr("CREATE 1"),
		mr("CREATE 3"),
		mr("CREATE 4"),
		mr("CREATE 7"),
		mr("CALLBACK 7 FAILURE"),
	}
	equalDbSeq(t, 0, expectedSequence, dbDrv)

	if !bytes.Equal(dbDrv.LastRunMigration, []byte("CALLBACK 7 FAILURE")) {
		t.Fatalf("expected database last migration to be callback 7, "+
			"got %s", dbDrv.LastRunMigration)
	}

	// Version should be the migration task version and not dirty.
	expectedTaskVer := int(7) + TaskVersionOffset
	checkVersion(expectedTaskVer)

	// 2) Re-run Up — since we are at the migration task version, it should
	// try the task again and fail again.
	err = m.Up()
	if !errors.Is(err, cbError) {
		t.Fatal("expected cbError from failing migration task")
	}

	expectedSequence = migrationSequence{
		mr("CREATE 1"),
		mr("CREATE 3"),
		mr("CREATE 4"),
		mr("CREATE 7"),
		mr("CALLBACK 7 FAILURE"),
		mr("CALLBACK 7 FAILURE"),
	}
	equalDbSeq(t, 0, expectedSequence, dbDrv)

	if !bytes.Equal(dbDrv.LastRunMigration, []byte("CALLBACK 7 FAILURE")) {
		t.Fatalf("expected database last migration to be callback 7, "+
			"got %s", dbDrv.LastRunMigration)
	}

	checkVersion(expectedTaskVer)

	// 3) Attempt down — but since we are still at the migration task
	// version, it should try the callback again and fail again and never
	// downgrade.
	err = m.Down()
	if !errors.Is(err, cbError) {
		t.Fatal("expected cbError from failing migration task")
	}

	expectedSequence = migrationSequence{
		mr("CREATE 1"),
		mr("CREATE 3"),
		mr("CREATE 4"),
		mr("CREATE 7"),
		mr("CALLBACK 7 FAILURE"),
		mr("CALLBACK 7 FAILURE"),
		mr("CALLBACK 7 FAILURE"),
	}
	equalDbSeq(t, 0, expectedSequence, dbDrv)

	if !bytes.Equal(dbDrv.LastRunMigration, []byte("CALLBACK 7 FAILURE")) {
		t.Fatalf("expected database last migration to be callback 7, "+
			"got %s", dbDrv.LastRunMigration)
	}

	checkVersion(expectedTaskVer)

	// 3) Make the callback succeed by setting shouldFail to false and run
	// again. It should now successfully re-run the callback and then
	// finalize the migration cleanly to the latest version (7).
	shouldFail = false

	err = m.Up()
	if !errors.Is(err, ErrNoChange) {
		t.Fatalf("unexpected error after migration task succeed: %v",
			err)
	}

	expectedSequence = migrationSequence{
		mr("CREATE 1"),
		mr("CREATE 3"),
		mr("CREATE 4"),
		mr("CREATE 7"),
		mr("CALLBACK 7 FAILURE"),
		mr("CALLBACK 7 FAILURE"),
		mr("CALLBACK 7 FAILURE"),
		mr("CALLBACK 7 SUCCESS"),
	}
	equalDbSeq(t, 0, expectedSequence, dbDrv)

	// And the last run migration should be from the callback.
	if !bytes.Equal(dbDrv.LastRunMigration, []byte("CALLBACK 7 SUCCESS")) {
		t.Fatalf("expected last run migration to be from migration "+
			"task, got %q", dbDrv.LastRunMigration)
	}

	// The version should now be the latest non migration task version 7
	// and not dirty.
	checkVersion(7)

	// 4) Finally, try Up again — it should be a no-op since we are at the
	// latest version, and shouldn't run the task again.
	err = m.Up()
	if !errors.Is(err, ErrNoChange) {
		t.Fatalf("unexpected error after migration task succeed: %v",
			err)
	}

	// Ensure the callback wasn't re-run.
	expectedSequence = migrationSequence{
		mr("CREATE 1"),
		mr("CREATE 3"),
		mr("CREATE 4"),
		mr("CREATE 7"),
		mr("CALLBACK 7 FAILURE"),
		mr("CALLBACK 7 FAILURE"),
		mr("CALLBACK 7 FAILURE"),
		mr("CALLBACK 7 SUCCESS"),
	}
	equalDbSeq(t, 0, expectedSequence, dbDrv)

	// And the last run migration should be from the previous callback.
	if !bytes.Equal(dbDrv.LastRunMigration, []byte("CALLBACK 7 SUCCESS")) {
		t.Fatalf("expected last run migration to be from migration "+
			"task, got %q", dbDrv.LastRunMigration)
	}

	checkVersion(7)
}

func TestUpDirty(t *testing.T) {
	m, _ := New("stub://", "stub://")
	dbDrv := m.databaseDrv.(*dStub.Stub)
	if err := dbDrv.SetVersion(0, true); err != nil {
		t.Fatal(err)
	}

	err := m.Up()
	if _, ok := err.(ErrDirty); !ok {
		t.Fatalf("expected ErrDirty, got %v", err)
	}
}

func TestDownDirty(t *testing.T) {
	m, _ := New("stub://", "stub://")
	dbDrv := m.databaseDrv.(*dStub.Stub)
	if err := dbDrv.SetVersion(0, true); err != nil {
		t.Fatal(err)
	}

	err := m.Down()
	if _, ok := err.(ErrDirty); !ok {
		t.Fatalf("expected ErrDirty, got %v", err)
	}
}

func TestDrop(t *testing.T) {
	m, _ := New("stub://", "stub://")
	m.sourceDrv.(*sStub.Stub).Migrations = sourceStubMigrations
	dbDrv := m.databaseDrv.(*dStub.Stub)

	if err := m.Drop(); err != nil {
		t.Fatal(err)
	}

	if dbDrv.MigrationSequence[len(dbDrv.MigrationSequence)-1] != dStub.DROP {
		t.Fatalf("expected database to DROP, got sequence %v", dbDrv.MigrationSequence)
	}
}

func TestVersion(t *testing.T) {
	m, _ := New("stub://", "stub://")
	dbDrv := m.databaseDrv.(*dStub.Stub)

	_, _, err := m.Version()
	if err != ErrNilVersion {
		t.Fatalf("expected ErrNilVersion, got %v", err)
	}

	if err := dbDrv.Run(bytes.NewBufferString("1_up")); err != nil {
		t.Fatal(err)
	}

	if err := dbDrv.SetVersion(1, false); err != nil {
		t.Fatal(err)
	}

	v, _, err := m.Version()
	if err != nil {
		t.Fatal(err)
	}

	if v != 1 {
		t.Fatalf("expected version 1, got %v", v)
	}
}

func TestRun(t *testing.T) {
	m, _ := New("stub://", "stub://")

	mx, err := NewMigration(nil, "", 1, 2)
	if err != nil {
		t.Fatal(err)
	}

	if err := m.Run(mx); err != nil {
		t.Fatal(err)
	}

	v, _, err := m.Version()
	if err != nil {
		t.Fatal(err)
	}

	if v != 2 {
		t.Errorf("expected version 2, got %v", v)
	}
}

func TestRunDirty(t *testing.T) {
	m, _ := New("stub://", "stub://")
	dbDrv := m.databaseDrv.(*dStub.Stub)
	if err := dbDrv.SetVersion(0, true); err != nil {
		t.Fatal(err)
	}

	migr, err := NewMigration(nil, "", 1, 2)
	if err != nil {
		t.Fatal(err)
	}

	err = m.Run(migr)
	if _, ok := err.(ErrDirty); !ok {
		t.Fatalf("expected ErrDirty, got %v", err)
	}
}

func TestForce(t *testing.T) {
	m, _ := New("stub://", "stub://")
	m.sourceDrv.(*sStub.Stub).Migrations = sourceStubMigrations

	if err := m.Force(7); err != nil {
		t.Fatal(err)
	}

	v, dirty, err := m.Version()
	if err != nil {
		t.Fatal(err)
	}
	if dirty {
		t.Errorf("expected dirty to be false")
	}
	if v != 7 {
		t.Errorf("expected version to be 7")
	}
}

func TestForceDirty(t *testing.T) {
	m, _ := New("stub://", "stub://")
	dbDrv := m.databaseDrv.(*dStub.Stub)
	if err := dbDrv.SetVersion(0, true); err != nil {
		t.Fatal(err)
	}

	if err := m.Force(1); err != nil {
		t.Fatal(err)
	}
}

func TestRead(t *testing.T) {
	m, _ := New("stub://", "stub://")
	m.sourceDrv.(*sStub.Stub).Migrations = sourceStubMigrations

	tt := []struct {
		from             int
		to               int
		expectErr        error
		expectMigrations migrationSequence
	}{
		{from: -1, to: -1, expectErr: ErrNoChange},
		{from: -1, to: 0, expectErr: os.ErrNotExist},
		{from: -1, to: 1, expectErr: nil, expectMigrations: newMigSeq(M(1))},
		{from: -1, to: 2, expectErr: os.ErrNotExist},
		{from: -1, to: 3, expectErr: nil, expectMigrations: newMigSeq(M(1), M(3))},
		{from: -1, to: 4, expectErr: nil, expectMigrations: newMigSeq(M(1), M(3), M(4))},
		{from: -1, to: 5, expectErr: nil, expectMigrations: newMigSeq(M(1), M(3), M(4), M(5))},
		{from: -1, to: 6, expectErr: os.ErrNotExist},
		{from: -1, to: 7, expectErr: nil, expectMigrations: newMigSeq(M(1), M(3), M(4), M(5), M(7))},
		{from: -1, to: 8, expectErr: os.ErrNotExist},

		{from: 0, to: -1, expectErr: os.ErrNotExist},
		{from: 0, to: 0, expectErr: os.ErrNotExist},
		{from: 0, to: 1, expectErr: os.ErrNotExist},
		{from: 0, to: 2, expectErr: os.ErrNotExist},
		{from: 0, to: 3, expectErr: os.ErrNotExist},
		{from: 0, to: 4, expectErr: os.ErrNotExist},
		{from: 0, to: 5, expectErr: os.ErrNotExist},
		{from: 0, to: 6, expectErr: os.ErrNotExist},
		{from: 0, to: 7, expectErr: os.ErrNotExist},
		{from: 0, to: 8, expectErr: os.ErrNotExist},

		{from: 1, to: -1, expectErr: nil, expectMigrations: newMigSeq(M(1, -1))},
		{from: 1, to: 0, expectErr: os.ErrNotExist},
		{from: 1, to: 1, expectErr: ErrNoChange},
		{from: 1, to: 2, expectErr: os.ErrNotExist},
		{from: 1, to: 3, expectErr: nil, expectMigrations: newMigSeq(M(3))},
		{from: 1, to: 4, expectErr: nil, expectMigrations: newMigSeq(M(3), M(4))},
		{from: 1, to: 5, expectErr: nil, expectMigrations: newMigSeq(M(3), M(4), M(5))},
		{from: 1, to: 6, expectErr: os.ErrNotExist},
		{from: 1, to: 7, expectErr: nil, expectMigrations: newMigSeq(M(3), M(4), M(5), M(7))},
		{from: 1, to: 8, expectErr: os.ErrNotExist},

		{from: 2, to: -1, expectErr: os.ErrNotExist},
		{from: 2, to: 0, expectErr: os.ErrNotExist},
		{from: 2, to: 1, expectErr: os.ErrNotExist},
		{from: 2, to: 2, expectErr: os.ErrNotExist},
		{from: 2, to: 3, expectErr: os.ErrNotExist},
		{from: 2, to: 4, expectErr: os.ErrNotExist},
		{from: 2, to: 5, expectErr: os.ErrNotExist},
		{from: 2, to: 6, expectErr: os.ErrNotExist},
		{from: 2, to: 7, expectErr: os.ErrNotExist},
		{from: 2, to: 8, expectErr: os.ErrNotExist},

		{from: 3, to: -1, expectErr: nil, expectMigrations: newMigSeq(M(3, 1), M(1, -1))},
		{from: 3, to: 0, expectErr: os.ErrNotExist},
		{from: 3, to: 1, expectErr: nil, expectMigrations: newMigSeq(M(3, 1))},
		{from: 3, to: 2, expectErr: os.ErrNotExist},
		{from: 3, to: 3, expectErr: ErrNoChange},
		{from: 3, to: 4, expectErr: nil, expectMigrations: newMigSeq(M(4))},
		{from: 3, to: 5, expectErr: nil, expectMigrations: newMigSeq(M(4), M(5))},
		{from: 3, to: 6, expectErr: os.ErrNotExist},
		{from: 3, to: 7, expectErr: nil, expectMigrations: newMigSeq(M(4), M(5), M(7))},
		{from: 3, to: 8, expectErr: os.ErrNotExist},

		{from: 4, to: -1, expectErr: nil, expectMigrations: newMigSeq(M(4, 3), M(3, 1), M(1, -1))},
		{from: 4, to: 0, expectErr: os.ErrNotExist},
		{from: 4, to: 1, expectErr: nil, expectMigrations: newMigSeq(M(4, 3), M(3, 1))},
		{from: 4, to: 2, expectErr: os.ErrNotExist},
		{from: 4, to: 3, expectErr: nil, expectMigrations: newMigSeq(M(4, 3))},
		{from: 4, to: 4, expectErr: ErrNoChange},
		{from: 4, to: 5, expectErr: nil, expectMigrations: newMigSeq(M(5))},
		{from: 4, to: 6, expectErr: os.ErrNotExist},
		{from: 4, to: 7, expectErr: nil, expectMigrations: newMigSeq(M(5), M(7))},
		{from: 4, to: 8, expectErr: os.ErrNotExist},

		{from: 5, to: -1, expectErr: nil, expectMigrations: newMigSeq(M(5, 4), M(4, 3), M(3, 1), M(1, -1))},
		{from: 5, to: 0, expectErr: os.ErrNotExist},
		{from: 5, to: 1, expectErr: nil, expectMigrations: newMigSeq(M(5, 4), M(4, 3), M(3, 1))},
		{from: 5, to: 2, expectErr: os.ErrNotExist},
		{from: 5, to: 3, expectErr: nil, expectMigrations: newMigSeq(M(5, 4), M(4, 3))},
		{from: 5, to: 4, expectErr: nil, expectMigrations: newMigSeq(M(5, 4))},
		{from: 5, to: 5, expectErr: ErrNoChange},
		{from: 5, to: 6, expectErr: os.ErrNotExist},
		{from: 5, to: 7, expectErr: nil, expectMigrations: newMigSeq(M(7))},
		{from: 5, to: 8, expectErr: os.ErrNotExist},

		{from: 6, to: -1, expectErr: os.ErrNotExist},
		{from: 6, to: 0, expectErr: os.ErrNotExist},
		{from: 6, to: 1, expectErr: os.ErrNotExist},
		{from: 6, to: 2, expectErr: os.ErrNotExist},
		{from: 6, to: 3, expectErr: os.ErrNotExist},
		{from: 6, to: 4, expectErr: os.ErrNotExist},
		{from: 6, to: 5, expectErr: os.ErrNotExist},
		{from: 6, to: 6, expectErr: os.ErrNotExist},
		{from: 6, to: 7, expectErr: os.ErrNotExist},
		{from: 6, to: 8, expectErr: os.ErrNotExist},

		{from: 7, to: -1, expectErr: nil, expectMigrations: newMigSeq(M(7, 5), M(5, 4), M(4, 3), M(3, 1), M(1, -1))},
		{from: 7, to: 0, expectErr: os.ErrNotExist},
		{from: 7, to: 1, expectErr: nil, expectMigrations: newMigSeq(M(7, 5), M(5, 4), M(4, 3), M(3, 1))},
		{from: 7, to: 2, expectErr: os.ErrNotExist},
		{from: 7, to: 3, expectErr: nil, expectMigrations: newMigSeq(M(7, 5), M(5, 4), M(4, 3))},
		{from: 7, to: 4, expectErr: nil, expectMigrations: newMigSeq(M(7, 5), M(5, 4))},
		{from: 7, to: 5, expectErr: nil, expectMigrations: newMigSeq(M(7, 5))},
		{from: 7, to: 6, expectErr: os.ErrNotExist},
		{from: 7, to: 7, expectErr: ErrNoChange},
		{from: 7, to: 8, expectErr: os.ErrNotExist},

		{from: 8, to: -1, expectErr: os.ErrNotExist},
		{from: 8, to: 0, expectErr: os.ErrNotExist},
		{from: 8, to: 1, expectErr: os.ErrNotExist},
		{from: 8, to: 2, expectErr: os.ErrNotExist},
		{from: 8, to: 3, expectErr: os.ErrNotExist},
		{from: 8, to: 4, expectErr: os.ErrNotExist},
		{from: 8, to: 5, expectErr: os.ErrNotExist},
		{from: 8, to: 6, expectErr: os.ErrNotExist},
		{from: 8, to: 7, expectErr: os.ErrNotExist},
		{from: 8, to: 8, expectErr: os.ErrNotExist},
	}

	for i, v := range tt {
		ret := make(chan interface{})
		go m.read(v.from, v.to, ret)
		migrations, err := migrationsFromChannel(ret)

		if (v.expectErr == os.ErrNotExist && !errors.Is(err, os.ErrNotExist)) ||
			(v.expectErr != os.ErrNotExist && v.expectErr != err) {
			t.Errorf("expected %v, got %v, in %v", v.expectErr, err, i)
			t.Logf("%v, in %v", migrations, i)
		}
		if len(v.expectMigrations) > 0 {
			equalMigSeq(t, i, v.expectMigrations, migrations)
		}
	}
}

func TestReadUp(t *testing.T) {
	m, _ := New("stub://", "stub://")
	m.sourceDrv.(*sStub.Stub).Migrations = sourceStubMigrations

	tt := []struct {
		from             int
		limit            int // -1 means no limit
		expectErr        error
		expectMigrations migrationSequence
	}{
		{from: -1, limit: -1, expectErr: nil, expectMigrations: newMigSeq(M(1), M(3), M(4), M(5), M(7))},
		{from: -1, limit: 0, expectErr: ErrNoChange},
		{from: -1, limit: 1, expectErr: nil, expectMigrations: newMigSeq(M(1))},
		{from: -1, limit: 2, expectErr: nil, expectMigrations: newMigSeq(M(1), M(3))},

		{from: 0, limit: -1, expectErr: os.ErrNotExist},
		{from: 0, limit: 0, expectErr: os.ErrNotExist},
		{from: 0, limit: 1, expectErr: os.ErrNotExist},
		{from: 0, limit: 2, expectErr: os.ErrNotExist},

		{from: 1, limit: -1, expectErr: nil, expectMigrations: newMigSeq(M(3), M(4), M(5), M(7))},
		{from: 1, limit: 0, expectErr: ErrNoChange},
		{from: 1, limit: 1, expectErr: nil, expectMigrations: newMigSeq(M(3))},
		{from: 1, limit: 2, expectErr: nil, expectMigrations: newMigSeq(M(3), M(4))},

		{from: 2, limit: -1, expectErr: os.ErrNotExist},
		{from: 2, limit: 0, expectErr: os.ErrNotExist},
		{from: 2, limit: 1, expectErr: os.ErrNotExist},
		{from: 2, limit: 2, expectErr: os.ErrNotExist},

		{from: 3, limit: -1, expectErr: nil, expectMigrations: newMigSeq(M(4), M(5), M(7))},
		{from: 3, limit: 0, expectErr: ErrNoChange},
		{from: 3, limit: 1, expectErr: nil, expectMigrations: newMigSeq(M(4))},
		{from: 3, limit: 2, expectErr: nil, expectMigrations: newMigSeq(M(4), M(5))},

		{from: 4, limit: -1, expectErr: nil, expectMigrations: newMigSeq(M(5), M(7))},
		{from: 4, limit: 0, expectErr: ErrNoChange},
		{from: 4, limit: 1, expectErr: nil, expectMigrations: newMigSeq(M(5))},
		{from: 4, limit: 2, expectErr: nil, expectMigrations: newMigSeq(M(5), M(7))},

		{from: 5, limit: -1, expectErr: nil, expectMigrations: newMigSeq(M(7))},
		{from: 5, limit: 0, expectErr: ErrNoChange},
		{from: 5, limit: 1, expectErr: nil, expectMigrations: newMigSeq(M(7))},
		{from: 5, limit: 2, expectErr: ErrShortLimit{1}, expectMigrations: newMigSeq(M(7))},

		{from: 6, limit: -1, expectErr: os.ErrNotExist},
		{from: 6, limit: 0, expectErr: os.ErrNotExist},
		{from: 6, limit: 1, expectErr: os.ErrNotExist},
		{from: 6, limit: 2, expectErr: os.ErrNotExist},

		{from: 7, limit: -1, expectErr: ErrNoChange},
		{from: 7, limit: 0, expectErr: ErrNoChange},
		{from: 7, limit: 1, expectErr: os.ErrNotExist},
		{from: 7, limit: 2, expectErr: os.ErrNotExist},

		{from: 8, limit: -1, expectErr: os.ErrNotExist},
		{from: 8, limit: 0, expectErr: os.ErrNotExist},
		{from: 8, limit: 1, expectErr: os.ErrNotExist},
		{from: 8, limit: 2, expectErr: os.ErrNotExist},
	}

	for i, v := range tt {
		ret := make(chan interface{})
		go m.readUp(v.from, v.limit, ret)
		migrations, err := migrationsFromChannel(ret)

		if (v.expectErr == os.ErrNotExist && !errors.Is(err, os.ErrNotExist)) ||
			(v.expectErr != os.ErrNotExist && v.expectErr != err) {
			t.Errorf("expected %v, got %v, in %v", v.expectErr, err, i)
			t.Logf("%v, in %v", migrations, i)
		}
		if len(v.expectMigrations) > 0 {
			equalMigSeq(t, i, v.expectMigrations, migrations)
		}
	}
}

func TestReadDown(t *testing.T) {
	m, _ := New("stub://", "stub://")
	m.sourceDrv.(*sStub.Stub).Migrations = sourceStubMigrations

	tt := []struct {
		from             int
		limit            int // -1 means no limit
		expectErr        error
		expectMigrations migrationSequence
	}{
		{from: -1, limit: -1, expectErr: ErrNoChange},
		{from: -1, limit: 0, expectErr: ErrNoChange},
		{from: -1, limit: 1, expectErr: os.ErrNotExist},
		{from: -1, limit: 2, expectErr: os.ErrNotExist},

		{from: 0, limit: -1, expectErr: os.ErrNotExist},
		{from: 0, limit: 0, expectErr: os.ErrNotExist},
		{from: 0, limit: 1, expectErr: os.ErrNotExist},
		{from: 0, limit: 2, expectErr: os.ErrNotExist},

		{from: 1, limit: -1, expectErr: nil, expectMigrations: newMigSeq(M(1, -1))},
		{from: 1, limit: 0, expectErr: ErrNoChange},
		{from: 1, limit: 1, expectErr: nil, expectMigrations: newMigSeq(M(1, -1))},
		{from: 1, limit: 2, expectErr: ErrShortLimit{1}, expectMigrations: newMigSeq(M(1, -1))},

		{from: 2, limit: -1, expectErr: os.ErrNotExist},
		{from: 2, limit: 0, expectErr: os.ErrNotExist},
		{from: 2, limit: 1, expectErr: os.ErrNotExist},
		{from: 2, limit: 2, expectErr: os.ErrNotExist},

		{from: 3, limit: -1, expectErr: nil, expectMigrations: newMigSeq(M(3, 1), M(1, -1))},
		{from: 3, limit: 0, expectErr: ErrNoChange},
		{from: 3, limit: 1, expectErr: nil, expectMigrations: newMigSeq(M(3, 1))},
		{from: 3, limit: 2, expectErr: nil, expectMigrations: newMigSeq(M(3, 1), M(1, -1))},

		{from: 4, limit: -1, expectErr: nil, expectMigrations: newMigSeq(M(4, 3), M(3, 1), M(1, -1))},
		{from: 4, limit: 0, expectErr: ErrNoChange},
		{from: 4, limit: 1, expectErr: nil, expectMigrations: newMigSeq(M(4, 3))},
		{from: 4, limit: 2, expectErr: nil, expectMigrations: newMigSeq(M(4, 3), M(3, 1))},

		{from: 5, limit: -1, expectErr: nil, expectMigrations: newMigSeq(M(5, 4), M(4, 3), M(3, 1), M(1, -1))},
		{from: 5, limit: 0, expectErr: ErrNoChange},
		{from: 5, limit: 1, expectErr: nil, expectMigrations: newMigSeq(M(5, 4))},
		{from: 5, limit: 2, expectErr: nil, expectMigrations: newMigSeq(M(5, 4), M(4, 3))},

		{from: 6, limit: -1, expectErr: os.ErrNotExist},
		{from: 6, limit: 0, expectErr: os.ErrNotExist},
		{from: 6, limit: 1, expectErr: os.ErrNotExist},
		{from: 6, limit: 2, expectErr: os.ErrNotExist},

		{from: 7, limit: -1, expectErr: nil, expectMigrations: newMigSeq(M(7, 5), M(5, 4), M(4, 3), M(3, 1), M(1, -1))},
		{from: 7, limit: 0, expectErr: ErrNoChange},
		{from: 7, limit: 1, expectErr: nil, expectMigrations: newMigSeq(M(7, 5))},
		{from: 7, limit: 2, expectErr: nil, expectMigrations: newMigSeq(M(7, 5), M(5, 4))},

		{from: 8, limit: -1, expectErr: os.ErrNotExist},
		{from: 8, limit: 0, expectErr: os.ErrNotExist},
		{from: 8, limit: 1, expectErr: os.ErrNotExist},
		{from: 8, limit: 2, expectErr: os.ErrNotExist},
	}

	for i, v := range tt {
		ret := make(chan interface{})
		go m.readDown(v.from, v.limit, ret)
		migrations, err := migrationsFromChannel(ret)

		if (v.expectErr == os.ErrNotExist && !errors.Is(err, os.ErrNotExist)) ||
			(v.expectErr != os.ErrNotExist && v.expectErr != err) {
			t.Errorf("expected %v, got %v, in %v", v.expectErr, err, i)
			t.Logf("%v, in %v", migrations, i)
		}
		if len(v.expectMigrations) > 0 {
			equalMigSeq(t, i, v.expectMigrations, migrations)
		}
	}
}

func TestLock(t *testing.T) {
	m, _ := New("stub://", "stub://")
	if err := m.lock(); err != nil {
		t.Fatal(err)
	}

	if err := m.lock(); err == nil {
		t.Fatal("should be locked already")
	}
}

func migrationsFromChannel(ret chan interface{}) ([]*Migration, error) {
	slice := make([]*Migration, 0)
	for r := range ret {
		switch t := r.(type) {
		case error:
			return slice, t

		case *Migration:
			slice = append(slice, t)
		}
	}
	return slice, nil
}

type migrationSequence []*Migration

func newMigSeq(migr ...*Migration) migrationSequence {
	return migr
}

func (m *migrationSequence) add(migr ...*Migration) migrationSequence { // nolint:unused
	*m = append(*m, migr...)
	return *m
}

func (m *migrationSequence) bodySequence() []string {
	r := make([]string, 0)
	for _, v := range *m {
		if v.Body != nil {
			body, err := io.ReadAll(v.Body)
			if err != nil {
				panic(err) // that should never happen
			}

			// reset body reader
			// TODO: is there a better/nicer way?
			v.Body = io.NopCloser(bytes.NewReader(body))

			r = append(r, string(body[:]))
		} else {
			r = append(r, "<empty>")
		}
	}
	return r
}

// M is a convenience func to create a new *Migration
func M(version uint, targetVersion ...int) *Migration {
	if len(targetVersion) > 1 {
		panic("only one targetVersion allowed")
	}
	ts := int(version)
	if len(targetVersion) == 1 {
		ts = targetVersion[0]
	}

	m, _ := New("stub://", "stub://")
	m.sourceDrv.(*sStub.Stub).Migrations = sourceStubMigrations
	migr, err := m.newMigration(version, ts)
	if err != nil {
		panic(err)
	}
	return migr
}

// mr is a convenience func to create a new *Migration from the raw database query
func mr(value string) *Migration {
	return &Migration{
		Body: io.NopCloser(strings.NewReader(value)),
	}
}

func equalMigSeq(t *testing.T, i int, expected, got migrationSequence) {
	if len(expected) != len(got) {
		t.Errorf("expected migrations %v, got %v, in %v", expected, got, i)

	} else {
		for ii := 0; ii < len(expected); ii++ {
			if expected[ii].Version != got[ii].Version {
				t.Errorf("expected version %v, got %v, in %v", expected[ii].Version, got[ii].Version, i)
			}

			if expected[ii].TargetVersion != got[ii].TargetVersion {
				t.Errorf("expected targetVersion %v, got %v, in %v", expected[ii].TargetVersion, got[ii].TargetVersion, i)
			}
		}
	}
}

func equalDbSeq(t *testing.T, i int, expected migrationSequence, got *dStub.Stub) {
	bs := expected.bodySequence()
	if !got.EqualSequence(bs) {
		t.Fatalf("\nexpected sequence %v,\ngot               %v, in %v", bs, got.MigrationSequence, i)
	}
}
