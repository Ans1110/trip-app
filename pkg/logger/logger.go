package logger

import (
	"os"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

const (
	Red    = "\033[31m"
	Yellow = "\033[33m"
	Blue   = "\033[34m"
	Reset  = "\033[0m"
)

func New(env string) *zap.Logger {
	encoderConfig := zap.NewProductionEncoderConfig()
	encoderConfig.EncodeTime = zapcore.ISO8601TimeEncoder

	if env == "development" {
		encoderConfig.EncodeLevel = colorLevelEncoder
	}

	encoder := getEncoder(env, encoderConfig)

	infoWriter := newLumberjackLogger("logs/info.log")
	errorWriter := newLumberjackLogger("logs/error.log")

	core := zapcore.NewTee(
		zapcore.NewCore(encoder, zapcore.AddSync(os.Stdout), zap.DebugLevel),
		zapcore.NewCore(encoder, infoWriter, zap.InfoLevel),
		zapcore.NewCore(encoder, errorWriter, zap.ErrorLevel),
	)

	logger := zap.New(core, zap.AddCaller(), zap.AddStacktrace(zap.ErrorLevel))
	zap.ReplaceGlobals(logger)
	return logger

}

func getEncoder(env string, cfg zapcore.EncoderConfig) zapcore.Encoder {
	if env == "development" {
		return zapcore.NewConsoleEncoder(cfg)
	}
	return zapcore.NewJSONEncoder(cfg)
}

func colorLevelEncoder(level zapcore.Level, enc zapcore.PrimitiveArrayEncoder) {
	switch level {
	case zapcore.InfoLevel:
		enc.AppendString(Blue + "INFO" + Reset)
	case zapcore.WarnLevel:
		enc.AppendString(Yellow + "WARN" + Reset)
	case zapcore.ErrorLevel, zapcore.DPanicLevel, zapcore.PanicLevel, zapcore.FatalLevel:
		enc.AppendString(Red + "ERROR" + Reset)
	default:
		enc.AppendString(level.CapitalString())
	}
}
