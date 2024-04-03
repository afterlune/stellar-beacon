package zlog

import (
	"fmt"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"os"
	"time"
)

var (
	logger    *zap.Logger
	ErrorInfo string
)

func init() {
	// 创建终端编码器
	developmentEncoderConfig := zap.NewDevelopmentEncoderConfig()
	developmentEncoderConfig.EncodeCaller = zapcore.ShortCallerEncoder
	//developmentEncoderConfig.EncodeCaller = nil
	developmentEncoderConfig.EncodeTime = zapcore.ISO8601TimeEncoder
	consoleEncoder := zapcore.NewConsoleEncoder(developmentEncoderConfig)

	// 创建文件编码器
	fileEncoderConfig := zap.NewProductionEncoderConfig()
	fileEncoderConfig.EncodeTime = zapcore.ISO8601TimeEncoder
	fileEncoderConfig.EncodeDuration = zapcore.StringDurationEncoder
	fileEncoderConfig.EncodeCaller = zapcore.ShortCallerEncoder
	fileEncoder := zapcore.NewConsoleEncoder(fileEncoderConfig)

	// 创建一个核心，只接受错误级别的日志，使用文件编码器，写入文件
	file, err := os.OpenFile(fmt.Sprintf("error_%s.log", time.Now().Format(time.DateOnly)), os.O_CREATE|os.O_APPEND|os.O_RDWR, 0777)
	if err != nil {
		panic(err)
	}

	fileCore := zapcore.NewCore(
		fileEncoder,
		zapcore.AddSync(file),
		zap.ErrorLevel,
	)

	// 创建一个核心，使用终端编码器，接受所有级别的日志
	consoleCore := zapcore.NewCore(
		consoleEncoder,
		zapcore.AddSync(os.Stdout),
		zapcore.DebugLevel,
	)

	// 同时输出到终端和文件
	logger = zap.New(zapcore.NewTee(consoleCore, fileCore),
		zap.AddCaller(),
		zap.AddCallerSkip(1),
		zap.AddStacktrace(zapcore.ErrorLevel),
	)
}

// Info 输出Info级别日志
func Info(msg string, fields ...zap.Field) {
	logger.Info(msg, fields...)
}

// Error 输出Error级别日志
func Error(msg string, fields ...zap.Field) {
	for _, field := range fields {
		ErrorInfo += field.String
	}
	logger.Error(msg, fields...)
}

// Warn 输出Warn级别日志
func Warn(msg string, fields ...zap.Field) {
	logger.Warn(msg, fields...)
}

// Debug 输出Debug级别日志
func Debug(msg string, fields ...zap.Field) {
	logger.Debug(msg, fields...)
}

// Fatal 输出Fatal级别日志
func Fatal(msg string, fields ...zap.Field) {
	logger.Fatal(msg, fields...)
}

func Unwrap(err error, fields ...zap.Field) {
	if err != nil {
		fields = append(fields, zap.Error(err))
		logger.Error("", fields...)
	}
}

func UnwrapWithMessage(msg string, err error, fields ...zap.Field) {
	if err != nil {
		fields = append(fields, zap.Error(err))
		logger.Error("", fields...)
	}
}
