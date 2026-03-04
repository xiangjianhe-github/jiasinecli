// Package logger 提供统一的日志管理
package logger

import (
	"os"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

var log *zap.SugaredLogger

// Init 初始化日志
func Init(verbose bool) error {
	level := zapcore.InfoLevel
	if verbose {
		level = zapcore.DebugLevel
	}

	encoderConfig := zapcore.EncoderConfig{
		TimeKey:        "time",
		LevelKey:       "level",
		NameKey:        "logger",
		CallerKey:      "caller",
		FunctionKey:    zapcore.OmitKey,
		MessageKey:     "msg",
		StacktraceKey:  "stacktrace",
		LineEnding:     zapcore.DefaultLineEnding,
		EncodeLevel:    zapcore.CapitalColorLevelEncoder,
		EncodeTime:     zapcore.TimeEncoderOfLayout("2006-01-02 15:04:05"),
		EncodeDuration: zapcore.StringDurationEncoder,
		EncodeCaller:   zapcore.ShortCallerEncoder,
	}

	core := zapcore.NewCore(
		zapcore.NewConsoleEncoder(encoderConfig),
		zapcore.AddSync(os.Stdout),
		level,
	)

	logger := zap.New(core, zap.AddCaller(), zap.AddCallerSkip(1))
	log = logger.Sugar()

	return nil
}

// getLogger 内部获取 logger，确保已初始化
func getLogger() *zap.SugaredLogger {
	if log == nil {
		Init(false)
	}
	return log
}

// Debug 输出调试日志
func Debug(msg string, keysAndValues ...interface{}) {
	getLogger().Debugw(msg, keysAndValues...)
}

// Info 输出信息日志
func Info(msg string, keysAndValues ...interface{}) {
	getLogger().Infow(msg, keysAndValues...)
}

// Warn 输出警告日志
func Warn(msg string, keysAndValues ...interface{}) {
	getLogger().Warnw(msg, keysAndValues...)
}

// Error 输出错误日志
func Error(msg string, keysAndValues ...interface{}) {
	getLogger().Errorw(msg, keysAndValues...)
}

// Fatal 输出致命错误日志并退出
func Fatal(msg string, keysAndValues ...interface{}) {
	getLogger().Fatalw(msg, keysAndValues...)
}

// Sync 刷新日志缓冲
func Sync() {
	if log != nil {
		log.Sync()
	}
}
