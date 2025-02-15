package logger

import (
	"os"
	"time"

	"github.com/aybolid/wishbot/internal/env"
	"go.uber.org/zap"
)

const LOGS_DIR = "logs"

var Sugared *zap.SugaredLogger

// Initializes the sugared logger.
//
// Panics if an error occurs.
func Init() {
	if Sugared != nil {
		return
	}

	var err error
	var logger *zap.Logger

	if env.Vars.Mode == "dev" {
		logger, err = zap.NewDevelopment()
	} else {
		logger, err = newProdLogger()
	}

	if err != nil {
		panic(err)
	}
	defer logger.Sync()

	Sugared = logger.Sugar()
}

func newProdLogger() (*zap.Logger, error) {
	cfg := zap.NewProductionConfig()

	if _, err := os.Stat(LOGS_DIR); os.IsNotExist(err) {
		os.Mkdir(LOGS_DIR, 0755)
	}

	cfg.OutputPaths = []string{
		LOGS_DIR + "/" + time.Now().Format("2006-01-02") + ".log",
	}
	return cfg.Build()
}
