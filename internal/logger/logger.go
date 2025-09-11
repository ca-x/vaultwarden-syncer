package logger

import (
	"os"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"gopkg.in/natefinch/lumberjack.v2"
)

var Logger *zap.Logger

func InitLogger(level string, logFile string) {
	var zapLevel zapcore.Level
	switch level {
	case "debug":
		zapLevel = zapcore.DebugLevel
	case "info":
		zapLevel = zapcore.InfoLevel
	case "warn":
		zapLevel = zapcore.WarnLevel
	case "error":
		zapLevel = zapcore.ErrorLevel
	default:
		zapLevel = zapcore.InfoLevel
	}

	encoder := zapcore.NewJSONEncoder(zapcore.EncoderConfig{
		TimeKey:        "time",
		LevelKey:       "level",
		NameKey:        "logger",
		CallerKey:      "caller",
		MessageKey:     "message",
		StacktraceKey:  "stacktrace",
		LineEnding:     zapcore.DefaultLineEnding,
		EncodeLevel:    zapcore.LowercaseLevelEncoder,
		EncodeTime:     zapcore.ISO8601TimeEncoder,
		EncodeDuration: zapcore.SecondsDurationEncoder,
		EncodeCaller:   zapcore.ShortCallerEncoder,
	})

	var cores []zapcore.Core

	consoleCore := zapcore.NewCore(
		zapcore.NewConsoleEncoder(zapcore.EncoderConfig{
			TimeKey:        "time",
			LevelKey:       "level",
			NameKey:        "logger",
			CallerKey:      "caller",
			MessageKey:     "message",
			StacktraceKey:  "stacktrace",
			LineEnding:     zapcore.DefaultLineEnding,
			EncodeLevel:    zapcore.CapitalColorLevelEncoder,
			EncodeTime:     zapcore.ISO8601TimeEncoder,
			EncodeDuration: zapcore.SecondsDurationEncoder,
			EncodeCaller:   zapcore.ShortCallerEncoder,
		}),
		zapcore.AddSync(os.Stdout),
		zapLevel,
	)
	cores = append(cores, consoleCore)

	if logFile != "" {
		fileWriter := &lumberjack.Logger{
			Filename:   logFile,
			MaxSize:    100,
			MaxBackups: 5,
			MaxAge:     30,
			Compress:   true,
		}

		fileCore := zapcore.NewCore(
			encoder,
			zapcore.AddSync(fileWriter),
			zapLevel,
		)
		cores = append(cores, fileCore)
	}

	core := zapcore.NewTee(cores...)
	Logger = zap.New(core, zap.AddCaller(), zap.AddStacktrace(zapcore.ErrorLevel))
}

func GetLogger() *zap.Logger {
	if Logger == nil {
		InitLogger("info", "")
	}
	return Logger
}

func Sync() {
	if Logger != nil {
		Logger.Sync()
	}
}