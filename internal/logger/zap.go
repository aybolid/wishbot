package logger

import (
	"fmt"
	"os"
	"time"

	"github.com/aybolid/wishbot/internal/env"
	"go.uber.org/zap"
)

const (
	LOGS_DIR         = "logs"
	FILE_DATE_FORMAT = "2006-01-02_15-04-05"
)

var Sugared *zap.SugaredLogger

// Init initializes the sugared logger.
// Panics if an error occurs during initialization.
func Init() {
	if Sugared != nil {
		return
	}

	var (
		err    error
		logger *zap.Logger
	)

	if env.Vars.Mode == env.DEV_MODE {
		logger, err = zap.NewDevelopment()
	} else {
		logger, err = newProdLogger()
	}

	if err != nil {
		panic(err)
	}

	Sugared = logger.Sugar()
}

// Shutdown flushes any buffered log entries.
// It should be called at program termination.
func Shutdown() {
	if Sugared != nil {
		_ = Sugared.Sync()
	}
}

func newProdLogger() (*zap.Logger, error) {
	cfg := zap.NewProductionConfig()

	if _, err := os.Stat(LOGS_DIR); os.IsNotExist(err) {
		if err := os.Mkdir(LOGS_DIR, 0755); err != nil {
			return nil, fmt.Errorf("failed to create logs directory: %w", err)
		}
	}

	logFile := fmt.Sprintf("%s/%s.log", LOGS_DIR, time.Now().Format(FILE_DATE_FORMAT))
	cfg.OutputPaths = []string{logFile}

	return cfg.Build()
}
