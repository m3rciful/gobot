package bootstrap

import (
	"fmt"

	"github.com/jmoiron/sqlx"

	coreconfig "gobot/core/config"
	coredatabase "gobot/core/database"
	"gobot/core/logger"
)

// Options control the generic bootstrap pipeline shared between bots.
type Options struct {
	Config   *coreconfig.Config
	Database coredatabase.Config

	LoggerInit func(*coreconfig.Config) error
	Connect    func(coredatabase.Config) (*sqlx.DB, error)
	Migrate    func(coredatabase.Config) error
}

// Result exposes infrastructure initialized by the bootstrap pipeline.
type Result struct {
	DB *sqlx.DB
}

// Run initializes the logger, connects to the database, and applies migrations.
func Run(opts Options) (*Result, error) {
	if opts.Config == nil {
		return nil, fmt.Errorf("bootstrap: nil config provided")
	}

	loggerInit := opts.LoggerInit
	if loggerInit == nil {
		loggerInit = logger.InitLogger
	}
	if err := loggerInit(opts.Config); err != nil {
		return nil, fmt.Errorf("bootstrap: logger init failed: %w", err)
	}

	connect := opts.Connect
	if connect == nil {
		connect = coredatabase.Connect
	}
	db, err := connect(opts.Database)
	if err != nil {
		return nil, fmt.Errorf("bootstrap: database initialization failed: %w", err)
	}

	migrate := opts.Migrate
	if migrate == nil {
		migrate = coredatabase.RunMigrations
	}
	if err := migrate(opts.Database); err != nil {
		_ = db.Close()
		return nil, fmt.Errorf("bootstrap: migrations failed: %w", err)
	}

	return &Result{DB: db}, nil
}
