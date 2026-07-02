// Package logger 日志模块
package logger

import (
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"os"
)

var Sugar *zap.SugaredLogger
var Logger *zap.Logger

// InitLogger 初始化日志
func InitLogger(level string) {
	var lvl zapcore.Level
	switch level {
	case "debug":
		lvl = zapcore.DebugLevel
	case "info":
		lvl = zapcore.InfoLevel
	case "warn":
		lvl = zapcore.WarnLevel
	case "error":
		lvl = zapcore.ErrorLevel
	default:
		lvl = zapcore.InfoLevel
	}

	encoderConfig := zapcore.EncoderConfig{
		TimeKey:       "time",
		LevelKey:      "level",
		NameKey:       "logger",
		CallerKey:     "caller",
		MessageKey:    "msg",
		StacktraceKey: "stacktrace",
		LineEnding:    zapcore.DefaultLineEnding,
		EncodeLevel:   zapcore.CapitalLevelEncoder,
		EncodeTime: zapcore.TimeEncoderOfLayout("2006-01-02 15:04:05"),
		EncodeCaller: zapcore.ShortCallerEncoder,
	}

	core := zapcore.NewCore(
		zapcore.NewConsoleEncoder(encoderConfig),
		zapcore.AddSync(os.Stdout),
		lvl,
	)

	Logger = zap.New(core, zap.AddCaller(), zap.AddCallerSkip(1))
	Sugar = Logger.Sugar()
}

// Debug 记录 debug 日志
func Debug(msg string, args ...interface{}) {
	if len(args) > 0 {
		Sugar.Debugf(msg, args...)
	} else {
		Sugar.Debug(msg)
	}
}

// Info 记录 info 日志
func Info(msg string, args ...interface{}) {
	if len(args) > 0 {
		Sugar.Infof(msg, args...)
	} else {
		Sugar.Info(msg)
	}
}

// Error 记录 error 日志
func Error(msg string, args ...interface{}) {
	if len(args) > 0 {
		Sugar.Errorf(msg, args...)
	} else {
		Sugar.Error(msg)
	}
}
