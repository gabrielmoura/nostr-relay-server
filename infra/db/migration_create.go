package db

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
)

var migrationFilePattern = regexp.MustCompile(`^(\d+)_.+\.(up|down)\.sql$`)

func CreateMigration(dir, name string) (string, string, error) {
	slug, err := migrationSlug(name)
	if err != nil {
		return "", "", err
	}

	version, err := nextVersion(dir)
	if err != nil {
		return "", "", err
	}

	prefix := fmt.Sprintf("%06d_%s", version, slug)
	upPath := filepath.Join(dir, prefix+".up.sql")
	downPath := filepath.Join(dir, prefix+".down.sql")
	if err := os.WriteFile(upPath, []byte("-- Write the forward migration here.\n"), 0o644); err != nil {
		return "", "", fmt.Errorf("create up migration: %w", err)
	}
	if err := os.WriteFile(downPath, []byte("-- Write the reverse migration here.\n"), 0o644); err != nil {
		_ = os.Remove(upPath)
		return "", "", fmt.Errorf("create down migration: %w", err)
	}
	return upPath, downPath, nil
}

func migrationSlug(name string) (string, error) {
	slug := strings.ToLower(strings.TrimSpace(name))
	slug = strings.Join(strings.Fields(slug), "_")
	if slug == "" {
		return "", fmt.Errorf("migration name cannot be empty")
	}
	for _, char := range slug {
		if char >= 'a' && char <= 'z' || char >= '0' && char <= '9' || char == '_' || char == '-' {
			continue
		}
		return "", fmt.Errorf("invalid migration name %q: use letters, numbers, underscores, or hyphens", name)
	}
	return slug, nil
}

func nextVersion(dir string) (uint, error) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		if errorsIsNotExist(err) {
			return 0, fmt.Errorf("migration directory %q does not exist", dir)
		}
		return 0, fmt.Errorf("read migration directory: %w", err)
	}

	var max uint
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		matches := migrationFilePattern.FindStringSubmatch(entry.Name())
		if len(matches) == 0 {
			continue
		}
		version, err := strconv.ParseUint(matches[1], 10, 32)
		if err != nil {
			return 0, fmt.Errorf("parse migration version from %q: %w", entry.Name(), err)
		}
		if uint(version) > max {
			max = uint(version)
		}
	}
	return max + 1, nil
}

func errorsIsNotExist(err error) bool {
	return os.IsNotExist(err)
}
