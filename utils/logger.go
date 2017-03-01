package utils

import (
	"os"

	"github.com/Sirupsen/logrus"
)

var (
	// Logger log self message.
	// TODO: change log level from command line.
	Logger = &logrus.Logger{
		Out: os.Stdout,
		Formatter: &logrus.TextFormatter{
			TimestampFormat: timeFormat,
		},
		Hooks: make(logrus.LevelHooks),
		Level: logrus.InfoLevel,
	}
)