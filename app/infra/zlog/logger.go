package zlog

import (
	"fmt"
	"os"
	"time"

	"github.com/natefinch/lumberjack"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

// Config 日志配置结构体
type Config struct {
	// 日志级别
	Level string `json:"level" yaml:"level"`
	// 日志文件路径
	Filename string `json:"filename" yaml:"filename"`
	// 日志文件最大大小（MB）
	MaxSize int `json:"max_size" yaml:"max_size"`
	// 最大备份数量
	MaxBackups int `json:"max_backups" yaml:"max_backups"`
	// 最大保留天数
	MaxAge int `json:"max_age" yaml:"max_age"`
	// 是否压缩
	Compress bool `json:"compress" yaml:"compress"`
	// 是否输出到控制台
	Console bool `json:"console" yaml:"console"`
}

var (
	logger           *zap.Logger
	lumberjackLogger *lumberjack.Logger
)

// DefaultConfig 默认配置
var DefaultConfig = Config{
	Level:      "debug",
	Filename:   fmt.Sprintf("error_%s.log", time.Now().Format(time.DateOnly)),
	MaxSize:    50, // 50MB
	MaxBackups: 5,  // 最多保留5个备份
	MaxAge:     7,  // 最多保留7天
	Compress:   true,
	Console:    true,
}

// Init 初始化日志系统
func Init(config Config) {
	// 解析日志级别
	level := zapcore.DebugLevel
	if err := level.UnmarshalText([]byte(config.Level)); err != nil {
		fmt.Printf("Invalid log level, using debug: %v\n", err)
		level = zapcore.DebugLevel
	}

	// 创建终端编码器
	developmentEncoderConfig := zap.NewDevelopmentEncoderConfig()
	developmentEncoderConfig.EncodeCaller = zapcore.ShortCallerEncoder
	developmentEncoderConfig.EncodeTime = zapcore.ISO8601TimeEncoder
	consoleEncoder := zapcore.NewConsoleEncoder(developmentEncoderConfig)

	// 创建文件编码器
	fileEncoderConfig := zap.NewProductionEncoderConfig()
	fileEncoderConfig.EncodeTime = zapcore.ISO8601TimeEncoder
	fileEncoderConfig.EncodeDuration = zapcore.StringDurationEncoder
	fileEncoderConfig.EncodeCaller = zapcore.ShortCallerEncoder
	fileEncoder := zapcore.NewConsoleEncoder(fileEncoderConfig)

	// 创建核心
	var cores []zapcore.Core

	// 文件核心
	lumberjackLogger = &lumberjack.Logger{
		Filename:   config.Filename,
		MaxSize:    config.MaxSize,
		MaxBackups: config.MaxBackups,
		MaxAge:     config.MaxAge,
		Compress:   config.Compress,
	}

	fileCore := zapcore.NewCore(
		fileEncoder,
		zapcore.AddSync(lumberjackLogger),
		zap.ErrorLevel,
	)
	cores = append(cores, fileCore)

	// 控制台核心
	if config.Console {
		consoleCore := zapcore.NewCore(
			consoleEncoder,
			zapcore.AddSync(os.Stdout),
			level,
		)
		cores = append(cores, consoleCore)
	}

	// 同时输出到终端和文件
	logger = zap.New(zapcore.NewTee(cores...),
		zap.AddCaller(),
		zap.AddCallerSkip(1),
		zap.AddStacktrace(zapcore.ErrorLevel),
	)
}

func init() {
	// 使用默认配置初始化
	Init(DefaultConfig)
}

// Info 输出Info级别日志
func Info(msg string, fields ...zap.Field) {
	logger.Info(msg, fields...)
}

// Error 输出Error级别日志
func Error(msg string, fields ...zap.Field) {
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

// Close 关闭日志文件
func Close() {
	if lumberjackLogger != nil {
		lumberjackLogger.Close()
	}
}

// Unwrap 处理错误并输出日志
func Unwrap(err error, fields ...zap.Field) {
	if err != nil {
		fields = append(fields, zap.Error(err))
		logger.Error("", fields...)
	}
}

// UnwrapWithMessage 处理错误并输出带消息的日志
func UnwrapWithMessage(msg string, err error, fields ...zap.Field) {
	if err != nil {
		fields = append(fields, zap.Error(err))
		logger.Error(msg, fields...)
	}
}
