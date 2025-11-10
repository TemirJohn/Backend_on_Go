package utils

import (
	"os"

	"github.com/sirupsen/logrus"
	"gopkg.in/natefinch/lumberjack.v2"
)

var Log *logrus.Logger

// InitLogger initializes the structured logger
func InitLogger() {
	Log = logrus.New()

	// Set log level from environment
	logLevel := os.Getenv("LOG_LEVEL")
	switch logLevel {
	case "debug":
		Log.SetLevel(logrus.DebugLevel)
	case "info":
		Log.SetLevel(logrus.InfoLevel)
	case "warn":
		Log.SetLevel(logrus.WarnLevel)
	case "error":
		Log.SetLevel(logrus.ErrorLevel)
	default:
		Log.SetLevel(logrus.InfoLevel)
	}

	// Set formatter (JSON for production, Text for development)
	env := os.Getenv("GIN_MODE")
	if env == "release" {
		Log.SetFormatter(&logrus.JSONFormatter{
			TimestampFormat: "2006-01-02 15:04:05",
			PrettyPrint:     false,
		})
	} else {
		Log.SetFormatter(&logrus.TextFormatter{
			FullTimestamp:   true,
			TimestampFormat: "2006-01-02 15:04:05",
			ForceColors:     true,
		})
	}

	// Configure log rotation with lumberjack
	logFile := &lumberjack.Logger{
		Filename:   "logs/app.log",
		MaxSize:    10,   // MB
		MaxBackups: 5,    // Keep 5 old log files
		MaxAge:     30,   // Days
		Compress:   true, // Compress old logs
	}

	// Write to both file and stdout
	Log.SetOutput(logFile)

	// Also log to stdout in development
	if env != "release" {
		Log.SetOutput(os.Stdout)
	}

	Log.Info("Logger initialized successfully")
}

// LogWithFields logs with structured fields
func LogInfo(message string, fields map[string]interface{}) {
	Log.WithFields(logrus.Fields(fields)).Info(message)
}

func LogError(message string, fields map[string]interface{}) {
	Log.WithFields(logrus.Fields(fields)).Error(message)
}

func LogWarn(message string, fields map[string]interface{}) {
	Log.WithFields(logrus.Fields(fields)).Warn(message)
}

func LogDebug(message string, fields map[string]interface{}) {
	Log.WithFields(logrus.Fields(fields)).Debug(message)
}