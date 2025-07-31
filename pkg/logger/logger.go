package logger

import (
    "go.uber.org/zap"
    "go.uber.org/zap/zapcore"
)

type Logger struct {
    *zap.SugaredLogger
}

func New() *Logger {
    config := zap.NewProductionConfig()
    config.OutputPaths = []string{"stdout"}
    config.ErrorOutputPaths = []string{"stderr"}
    config.EncoderConfig.TimeKey = "timestamp"
    config.EncoderConfig.EncodeTime = zapcore.ISO8601TimeEncoder

    logger, err := config.Build()
    if err != nil {
        panic(err)
    }

    return &Logger{
        SugaredLogger: logger.Sugar(),
    }
}

func (l *Logger) WithRequestID(requestID string) *Logger {
    return &Logger{
        SugaredLogger: l.SugaredLogger.With("request_id", requestID),
    }
}