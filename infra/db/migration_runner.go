package db

import (
	"context"
	"database/sql"
	"embed"
	"errors"
	"fmt"
	"strconv"

	"github.com/golang-migrate/migrate/v4"
	"github.com/golang-migrate/migrate/v4/database/postgres"
	"github.com/golang-migrate/migrate/v4/source/iofs"
	_ "github.com/jackc/pgx/v5/stdlib"
)

const migrationsDirectory = "migrations"

// MigrationFiles is the immutable migration source embedded in the NRServer binary.
//
//go:embed migrations/*.sql
var MigrationFiles embed.FS

type MigrationStatus struct {
	Applied  uint
	Expected uint
	Dirty    bool
}

func (s MigrationStatus) Current() bool {
	return !s.Dirty && s.Applied == s.Expected
}

type SchemaOutdatedError struct {
	Applied  uint
	Expected uint
}

func (e *SchemaOutdatedError) Error() string {
	return fmt.Sprintf("schema desatualizado (versão %d, esperado %d) — rode `nrserver migrate up`", e.Applied, e.Expected)
}

type SchemaDirtyError struct {
	Version uint
}

func (e *SchemaDirtyError) Error() string {
	return fmt.Sprintf("schema de banco dirty na versão %d — corrija a migration antes de iniciar o NRServer", e.Version)
}

func LatestMigrationVersion() (uint, error) {
	entries, err := MigrationFiles.ReadDir(migrationsDirectory)
	if err != nil {
		return 0, fmt.Errorf("read embedded migrations: %w", err)
	}

	var latest uint
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		matches := migrationFilePattern.FindStringSubmatch(entry.Name())
		if len(matches) == 0 || matches[2] != "up" {
			continue
		}
		version, err := strconv.ParseUint(matches[1], 10, 32)
		if err != nil {
			return 0, fmt.Errorf("parse embedded migration version from %q: %w", entry.Name(), err)
		}
		if uint(version) > latest {
			latest = uint(version)
		}
	}
	if latest == 0 {
		return 0, fmt.Errorf("no embedded migrations found")
	}
	return latest, nil
}

func MigrationStatusFor(ctx context.Context, dsn string) (MigrationStatus, error) {
	expected, err := LatestMigrationVersion()
	if err != nil {
		return MigrationStatus{}, err
	}

	m, err := open(ctx, dsn)
	if err != nil {
		return MigrationStatus{}, err
	}
	defer closeMigrate(m)

	applied, dirty, err := m.Version()
	if errors.Is(err, migrate.ErrNilVersion) {
		return MigrationStatus{Expected: expected}, nil
	}
	if err != nil {
		return MigrationStatus{}, fmt.Errorf("read schema version: %w", err)
	}
	return MigrationStatus{Applied: applied, Expected: expected, Dirty: dirty}, nil
}

func CheckMigrationsCurrent(ctx context.Context, dsn string) error {
	status, err := MigrationStatusFor(ctx, dsn)
	if err != nil {
		return err
	}
	if status.Dirty {
		return &SchemaDirtyError{Version: status.Applied}
	}
	if !status.Current() {
		return &SchemaOutdatedError{Applied: status.Applied, Expected: status.Expected}
	}
	return nil
}

func MigrateUp(ctx context.Context, dsn string) error {
	return run(ctx, dsn, func(m *migrate.Migrate) error {
		return m.Up()
	})
}

func MigrateDown(ctx context.Context, dsn string, all bool) error {
	return run(ctx, dsn, func(m *migrate.Migrate) error {
		if all {
			return m.Down()
		}
		return m.Steps(-1)
	})
}

// MigrateForce reconciles the migration ledger with a schema that was
// independently verified. It never applies or reverts schema statements.
func MigrateForce(ctx context.Context, dsn string, version int) error {
	return run(ctx, dsn, func(m *migrate.Migrate) error {
		return m.Force(version)
	})
}

func run(ctx context.Context, dsn string, operation func(*migrate.Migrate) error) error {
	m, err := open(ctx, dsn)
	if err != nil {
		return err
	}
	defer closeMigrate(m)

	err = operation(m)
	if errors.Is(err, migrate.ErrNoChange) {
		return nil
	}
	if err != nil {
		return fmt.Errorf("run database migrations: %w", err)
	}
	return nil
}

func open(ctx context.Context, dsn string) (*migrate.Migrate, error) {
	db, err := sql.Open("pgx", dsn)
	if err != nil {
		return nil, fmt.Errorf("open migration database connection: %w", err)
	}
	if err := db.PingContext(ctx); err != nil {
		_ = db.Close()
		return nil, fmt.Errorf("ping migration database connection: %w", err)
	}

	source, err := iofs.New(MigrationFiles, migrationsDirectory)
	if err != nil {
		_ = db.Close()
		return nil, fmt.Errorf("open embedded migrations: %w", err)
	}

	driver, err := postgres.WithInstance(db, &postgres.Config{})
	if err != nil {
		_ = source.Close()
		_ = db.Close()
		return nil, fmt.Errorf("create postgres migration driver: %w", err)
	}

	m, err := migrate.NewWithInstance("iofs", source, "postgres", driver)
	if err != nil {
		_ = source.Close()
		_ = driver.Close()
		return nil, fmt.Errorf("create migration runner: %w", err)
	}
	return m, nil
}

func closeMigrate(m *migrate.Migrate) {
	if m == nil {
		return
	}
	_, _ = m.Close()
}
