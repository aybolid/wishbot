package logger

import "go.uber.org/zap"

var SUGAR *zap.SugaredLogger

// Initializes the sugared logger.
//
// Panics if an error occurs.
func Init() {
	if SUGAR != nil {
		return
	}

	logger, err := zap.NewDevelopment()
	if err != nil {
		panic(err)
	}
	defer logger.Sync()

	SUGAR = logger.Sugar()
}
