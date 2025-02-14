package logger

import (
	"github.com/aybolid/wishbot/internal/env"
	"go.uber.org/zap"
)

const PROD_OUTPUT_PATH = "logs.json"

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

	if env.Vars.Debug {
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
	cfg.OutputPaths = []string{
		PROD_OUTPUT_PATH,
	}
	return cfg.Build()
}
