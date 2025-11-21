package database

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/m3rciful/gobot/core/logger"

	"github.com/golang-migrate/migrate/v4"
	_ "github.com/golang-migrate/migrate/v4/database/postgres"
	_ "github.com/golang-migrate/migrate/v4/source/file"
	"log/slog"
)

// RunMigrations applies all up migrations from the migrations directory.
func RunMigrations(cfg Config) error {
	dsn := fmt.Sprintf(
		"postgres://%s:%s@%s:%s/%s?sslmode=%s",
		cfg.User, cfg.Password, cfg.Host, cfg.Port, cfg.Name, cfg.SSLMode,
	)
	if err := WaitForPostgres(dsn, 30*time.Second); err != nil {
		logger.MIG.Error("db not ready",
			slog.String("event", "db.migrate"),
			slog.String("err", err.Error()),
		)
		return fmt.Errorf("database not ready: %w", err)
	}

	cwd, err := os.Getwd()
	if err != nil {
		logger.MIG.Error("cwd lookup failed",
			slog.String("event", "db.migrate"),
			slog.String("err", err.Error()),
		)
		return fmt.Errorf("get working directory: %w", err)
	}
	migrationsPath := filepath.Join(cwd, "migrations")
	sourceURL := "file://" + migrationsPath

	files := listMigrationFiles(migrationsPath)
	preview, truncated := logger.SummarizeStrings(files, 6)
	args := []any{
		slog.String("event", "resolve"),
		slog.String("path", migrationsPath),
		slog.Int("files_total", len(files)),
	}
	if preview != "" {
		args = append(args, slog.String("files_preview", preview))
	}
	if truncated {
		args = append(args, slog.Bool("files_truncated", true))
	}
	logger.MIG.Debug("migrations resolved", args...)

	m, err := migrate.New(sourceURL, dsn)
	if err != nil {
		logger.MIG.Error("init failed",
			slog.String("event", "db.migrate"),
			slog.String("err", err.Error()),
		)
		return fmt.Errorf("failed to initialize migrations: %w", err)
	}

	fromVer, _, _ := m.Version()

	start := time.Now()
	upErr := m.Up()
	took := time.Since(start)

	switch upErr {
	case nil:
	case migrate.ErrNoChange:
		logger.MIG.Info("migrations summary",
			slog.String("event", "summary"),
			slog.Uint64("from_ver", uint64(fromVer)),
			slog.Uint64("to_ver", uint64(fromVer)),
			slog.Int("files", 0),
			slog.Duration("duration", logger.RoundMS(took)),
		)
		return nil
	default:
		logger.MIG.Error("migration failed",
			slog.String("event", "apply"),
			slog.String("err", upErr.Error()),
			slog.Duration("duration", logger.RoundMS(took)),
		)
		return fmt.Errorf("migration execution failed: %w", upErr)
	}

	toVer, _, _ := m.Version()
	applied := countApplied(files, uint64(fromVer), uint64(toVer))

	if applied > 0 {
		appliedNames := selectApplied(files, uint64(fromVer), uint64(toVer))
		previewApplied, truncatedApplied := logger.SummarizeStrings(appliedNames, 6)
		args := []any{
			slog.String("event", "apply"),
			slog.Int("files_total", len(appliedNames)),
		}
		if previewApplied != "" {
			args = append(args, slog.String("files_preview", previewApplied))
		}
		if truncatedApplied {
			args = append(args, slog.Bool("files_truncated", true))
		}
		logger.MIG.Debug("applied files", args...)
	}

	logger.MIG.Info("migrations summary",
		slog.String("event", "summary"),
		slog.Uint64("from_ver", uint64(fromVer)),
		slog.Uint64("to_ver", uint64(toVer)),
		slog.Int("files", applied),
		slog.Duration("duration", logger.RoundMS(took)),
	)

	return nil
}

func listMigrationFiles(dir string) []string {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil
	}
	var names []string
	for _, e := range entries {
		if e.IsDir() {
			continue
		}
		name := e.Name()
		if strings.HasSuffix(name, ".up.sql") {
			names = append(names, name)
		}
	}
	sort.Strings(names)
	return names
}

func parseVersion(name string) uint64 {
	parts := strings.SplitN(name, "_", 2)
	if len(parts) == 0 {
		return 0
	}
	v, _ := strconv.ParseUint(parts[0], 10, 64)
	return v
}

func countApplied(files []string, from, to uint64) int {
	if to <= from {
		return 0
	}
	c := 0
	for _, f := range files {
		v := parseVersion(f)
		if v > from && v <= to {
			c++
		}
	}
	return c
}

func selectApplied(files []string, from, to uint64) []string {
	var out []string
	for _, f := range files {
		v := parseVersion(f)
		if v > from && v <= to {
			out = append(out, f)
		}
	}
	return out
}
