package mdathome

import (
	"time"

	colorable "github.com/mattn/go-colorable"
	"github.com/sirupsen/logrus"
	"github.com/snowzach/rotatefilehook"
	"github.com/spf13/viper"
)

var log = logrus.New()

func initLogger(logLevelString string, maxLogSizeInMb int, maxLogBackups int, maxLogAgeInDays int) {
	logLevel, _ := logrus.ParseLevel(logLevelString)

	rotateFileHook, err := rotatefilehook.NewRotateFileHook(rotatefilehook.RotateFileConfig{
		Filename:   viper.GetString("log.directory") + "/mdathome.log",
		MaxSize:    maxLogSizeInMb,
		MaxBackups: 3,
		MaxAge:     28,
		Level:      logrus.TraceLevel,
		Formatter: &logrus.JSONFormatter{
			TimestampFormat: time.RFC822,
		},
	})

	if err != nil {
		log.Fatalf("Failed to initialize file rotate hook: %v", err)
	}

	log.SetLevel(logLevel)
	log.SetOutput(colorable.NewColorableStdout())
	log.SetFormatter(&logrus.TextFormatter{
		ForceColors:     true,
		FullTimestamp:   true,
		TimestampFormat: time.RFC3339,
	})
	log.AddHook(rotateFileHook)
}
