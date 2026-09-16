package db

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestEmbeddedMigrationsHaveMatchingUpAndDownFiles(t *testing.T) {
	entries, err := MigrationFiles.ReadDir(migrationsDirectory)
	require.NoError(t, err)

	upVersions := map[string]bool{}
	downVersions := map[string]bool{}
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		matches := migrationFilePattern.FindStringSubmatch(entry.Name())
		if len(matches) == 0 {
			continue
		}
		contents, err := MigrationFiles.ReadFile(filepath.Join(migrationsDirectory, entry.Name()))
		require.NoError(t, err)
		require.NotEmpty(t, contents, entry.Name())
		if matches[2] == "up" {
			upVersions[matches[1]] = true
			continue
		}
		downVersions[matches[1]] = true
	}

	require.Equal(t, upVersions, downVersions)
	version, err := LatestMigrationVersion()
	require.NoError(t, err)
	require.EqualValues(t, 3, version)
}

func TestCreateMigrationCreatesAnUpAndDownPair(t *testing.T) {
	dir := t.TempDir()
	require.NoError(t, os.WriteFile(filepath.Join(dir, "000002_existing.up.sql"), []byte("SELECT 1;\n"), 0o644))
	require.NoError(t, os.WriteFile(filepath.Join(dir, "000002_existing.down.sql"), []byte("SELECT 1;\n"), 0o644))

	upPath, downPath, err := CreateMigration(dir, "Add Topics")
	require.NoError(t, err)
	require.Equal(t, filepath.Join(dir, "000003_add_topics.up.sql"), upPath)
	require.Equal(t, filepath.Join(dir, "000003_add_topics.down.sql"), downPath)

	up, err := os.ReadFile(upPath)
	require.NoError(t, err)
	require.Contains(t, string(up), "forward migration")
	down, err := os.ReadFile(downPath)
	require.NoError(t, err)
	require.Contains(t, string(down), "reverse migration")
}

func TestSchemaOutdatedErrorProvidesMigrationCommand(t *testing.T) {
	err := &SchemaOutdatedError{Applied: 1, Expected: 2}
	require.Equal(t, "schema desatualizado (versão 1, esperado 2) — rode `nrserver migrate up`", err.Error())
}
