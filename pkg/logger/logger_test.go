package logger

import (
	"bytes"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

func TestNew_DefaultConfiguration(t *testing.T) {
	logger := New()
	assert.NotNil(t, logger)
	assert.NotNil(t, logger.SugaredLogger)
}

func TestLogger_Info(t *testing.T) {
	// Create a logger with memory output for testing
	var logBuffer bytes.Buffer
	core := zapcore.NewCore(
		zapcore.NewJSONEncoder(zap.NewProductionEncoderConfig()),
		zapcore.AddSync(&logBuffer),
		zapcore.InfoLevel,
	)
	zapLogger := zap.New(core)
	logger := &Logger{SugaredLogger: zapLogger.Sugar()}

	// Test info logging
	logger.Info("test message with key: ", "value")

	output := logBuffer.String()
	assert.Contains(t, output, "test message")
}

func TestLogger_Error(t *testing.T) {
	// Create a logger with memory output for testing
	var logBuffer bytes.Buffer
	core := zapcore.NewCore(
		zapcore.NewJSONEncoder(zap.NewProductionEncoderConfig()),
		zapcore.AddSync(&logBuffer),
		zapcore.ErrorLevel,
	)
	zapLogger := zap.New(core)
	logger := &Logger{SugaredLogger: zapLogger.Sugar()}

	// Test error logging
	logger.Error("error message: ", "test error")

	output := logBuffer.String()
	assert.Contains(t, output, "error message")
}

func TestLogger_Debug(t *testing.T) {
	// Create a logger with memory output for testing at debug level
	var logBuffer bytes.Buffer
	core := zapcore.NewCore(
		zapcore.NewJSONEncoder(zap.NewProductionEncoderConfig()),
		zapcore.AddSync(&logBuffer),
		zapcore.DebugLevel,
	)
	zapLogger := zap.New(core)
	logger := &Logger{SugaredLogger: zapLogger.Sugar()}

	// Test debug logging
	logger.Debug("debug message: ", "test debug")

	output := logBuffer.String()
	assert.Contains(t, output, "debug message")
}

func TestLogger_Warn(t *testing.T) {
	// Create a logger with memory output for testing
	var logBuffer bytes.Buffer
	core := zapcore.NewCore(
		zapcore.NewJSONEncoder(zap.NewProductionEncoderConfig()),
		zapcore.AddSync(&logBuffer),
		zapcore.WarnLevel,
	)
	zapLogger := zap.New(core)
	logger := &Logger{SugaredLogger: zapLogger.Sugar()}

	// Test warn logging
	logger.Warn("warn message: ", "test warning")

	output := logBuffer.String()
	assert.Contains(t, output, "warn message")
}

func TestLogger_WithFields(t *testing.T) {
	// Create a logger with memory output for testing
	var logBuffer bytes.Buffer
	core := zapcore.NewCore(
		zapcore.NewJSONEncoder(zap.NewProductionEncoderConfig()),
		zapcore.AddSync(&logBuffer),
		zapcore.InfoLevel,
	)
	zapLogger := zap.New(core)
	logger := &Logger{SugaredLogger: zapLogger.Sugar()}

	// Test logging with multiple fields using With
	contextLogger := logger.With("field1", "value1", "field2", 42, "field3", true)
	contextLogger.Info("test with fields")

	output := logBuffer.String()
	assert.Contains(t, output, "test with fields")
	assert.Contains(t, output, "field1")
	assert.Contains(t, output, "value1")
	assert.Contains(t, output, "field2")
	assert.Contains(t, output, "42")
	assert.Contains(t, output, "field3")
	assert.Contains(t, output, "true")
}

func TestLogger_LogLevels(t *testing.T) {
	tests := []struct {
		name      string
		level     zapcore.Level
		logFunc   func(*Logger, ...interface{})
		message   string
		shouldLog bool
	}{
		{
			name:      "Debug level with debug message",
			level:     zapcore.DebugLevel,
			logFunc:   (*Logger).Debug,
			message:   "debug message",
			shouldLog: true,
		},
		{
			name:      "Info level with debug message",
			level:     zapcore.InfoLevel,
			logFunc:   (*Logger).Debug,
			message:   "debug message",
			shouldLog: false,
		},
		{
			name:      "Info level with info message",
			level:     zapcore.InfoLevel,
			logFunc:   (*Logger).Info,
			message:   "info message",
			shouldLog: true,
		},
		{
			name:      "Warn level with info message",
			level:     zapcore.WarnLevel,
			logFunc:   (*Logger).Info,
			message:   "info message",
			shouldLog: false,
		},
		{
			name:      "Error level with error message",
			level:     zapcore.ErrorLevel,
			logFunc:   (*Logger).Error,
			message:   "error message",
			shouldLog: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var logBuffer bytes.Buffer
			core := zapcore.NewCore(
				zapcore.NewJSONEncoder(zap.NewProductionEncoderConfig()),
				zapcore.AddSync(&logBuffer),
				tt.level,
			)
			zapLogger := zap.New(core)
			logger := &Logger{SugaredLogger: zapLogger.Sugar()}

			tt.logFunc(logger, tt.message)

			output := logBuffer.String()
			if tt.shouldLog {
				assert.Contains(t, output, tt.message)
			} else {
				assert.NotContains(t, output, tt.message)
			}
		})
	}
}

func TestLogger_WithRequestID(t *testing.T) {
	// Create a logger with memory output for testing
	var logBuffer bytes.Buffer
	core := zapcore.NewCore(
		zapcore.NewJSONEncoder(zap.NewProductionEncoderConfig()),
		zapcore.AddSync(&logBuffer),
		zapcore.InfoLevel,
	)
	zapLogger := zap.New(core)
	logger := &Logger{SugaredLogger: zapLogger.Sugar()}

	// Test WithRequestID method
	requestLogger := logger.WithRequestID("req-12345")
	requestLogger.Info("test with request ID")

	output := logBuffer.String()
	assert.Contains(t, output, "test with request ID")
	assert.Contains(t, output, "request_id")
	assert.Contains(t, output, "req-12345")
}

func TestLogger_ContextualLogging(t *testing.T) {
	// Create a logger with memory output for testing
	var logBuffer bytes.Buffer
	core := zapcore.NewCore(
		zapcore.NewJSONEncoder(zap.NewProductionEncoderConfig()),
		zapcore.AddSync(&logBuffer),
		zapcore.InfoLevel,
	)
	zapLogger := zap.New(core)
	logger := &Logger{SugaredLogger: zapLogger.Sugar()}

	// Test contextual logging with common patterns
	tests := []struct {
		name   string
		fields []interface{}
	}{
		{
			name: "User context",
			fields: []interface{}{
				"user_id", "12345",
				"operation", "create_task",
			},
		},
		{
			name: "Request context",
			fields: []interface{}{
				"request_id", "req-abc123",
				"method", "POST",
				"path", "/api/v1/tasks",
				"status_code", 201,
			},
		},
		{
			name: "Error context",
			fields: []interface{}{
				"error_type", "ValidationError",
				"component", "task_service",
				"function", "CreateTask",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			logBuffer.Reset()
			contextLogger := logger.With(tt.fields...)
			contextLogger.Info("Test ", tt.name)

			output := logBuffer.String()
			assert.Contains(t, output, "Test")
			assert.Contains(t, output, tt.name)

			// Verify fields are present - check every other item as key
			for i := 0; i < len(tt.fields); i += 2 {
				key := tt.fields[i].(string)
				assert.Contains(t, output, key)
			}
		})
	}
}

func TestLogger_JSONFormat(t *testing.T) {
	// Create a logger with memory output for testing
	var logBuffer bytes.Buffer
	core := zapcore.NewCore(
		zapcore.NewJSONEncoder(zap.NewProductionEncoderConfig()),
		zapcore.AddSync(&logBuffer),
		zapcore.InfoLevel,
	)
	zapLogger := zap.New(core)
	logger := &Logger{SugaredLogger: zapLogger.Sugar()}

	logger.Info("json format test")

	output := logBuffer.String()

	// Verify JSON format
	assert.Contains(t, output, "{")
	assert.Contains(t, output, "}")
	assert.Contains(t, output, "\"level\":")
	assert.Contains(t, output, "\"msg\":")
}

func TestLogger_Performance(t *testing.T) {
	// Create a no-op logger for performance testing
	logger := &Logger{SugaredLogger: zap.NewNop().Sugar()}

	// Test that logging doesn't panic under load
	assert.NotPanics(t, func() {
		for i := 0; i < 1000; i++ {
			logger.Info("performance test ", i)
		}
	})
}

func TestLogger_ThreadSafety(t *testing.T) {
	// Create a logger with memory output for testing
	var logBuffer bytes.Buffer
	core := zapcore.NewCore(
		zapcore.NewJSONEncoder(zap.NewProductionEncoderConfig()),
		zapcore.AddSync(&logBuffer),
		zapcore.InfoLevel,
	)
	zapLogger := zap.New(core)
	logger := &Logger{SugaredLogger: zapLogger.Sugar()}

	// Test concurrent logging doesn't panic
	done := make(chan bool, 10)

	for i := 0; i < 10; i++ {
		go func(id int) {
			logger.Info("concurrent test ", id)
			done <- true
		}(i)
	}

	// Wait for all goroutines to complete
	for i := 0; i < 10; i++ {
		<-done
	}

	// Should not panic and should have logged messages
	assert.NotEmpty(t, logBuffer.String())
}

func TestLogger_VariousLogLevels(t *testing.T) {
	// Create a logger with memory output for testing
	var logBuffer bytes.Buffer
	core := zapcore.NewCore(
		zapcore.NewJSONEncoder(zap.NewProductionEncoderConfig()),
		zapcore.AddSync(&logBuffer),
		zapcore.DebugLevel, // Allow all levels
	)
	zapLogger := zap.New(core)
	logger := &Logger{SugaredLogger: zapLogger.Sugar()}

	// Test all log levels
	logger.Debug("debug message")
	logger.Info("info message")
	logger.Warn("warn message")
	logger.Error("error message")

	output := logBuffer.String()

	// Count occurrences
	debugCount := strings.Count(output, "debug message")
	infoCount := strings.Count(output, "info message")
	warnCount := strings.Count(output, "warn message")
	errorCount := strings.Count(output, "error message")

	assert.Equal(t, 1, debugCount)
	assert.Equal(t, 1, infoCount)
	assert.Equal(t, 1, warnCount)
	assert.Equal(t, 1, errorCount)
}

func TestLogger_FormattedLogging(t *testing.T) {
	// Create a logger with memory output for testing
	var logBuffer bytes.Buffer
	core := zapcore.NewCore(
		zapcore.NewJSONEncoder(zap.NewProductionEncoderConfig()),
		zapcore.AddSync(&logBuffer),
		zapcore.InfoLevel,
	)
	zapLogger := zap.New(core)
	logger := &Logger{SugaredLogger: zapLogger.Sugar()}

	// Test formatted logging (sugared logger style)
	logger.Infof("formatted message with %s and %d", "string", 42)

	output := logBuffer.String()
	assert.Contains(t, output, "formatted message with string and 42")
}

func TestLogger_ChainedContext(t *testing.T) {
	// Create a logger with memory output for testing
	var logBuffer bytes.Buffer
	core := zapcore.NewCore(
		zapcore.NewJSONEncoder(zap.NewProductionEncoderConfig()),
		zapcore.AddSync(&logBuffer),
		zapcore.InfoLevel,
	)
	zapLogger := zap.New(core)
	logger := &Logger{SugaredLogger: zapLogger.Sugar()}

	// Test chained context building
	userLogger := &Logger{SugaredLogger: logger.With("user_id", "123")}
	requestLogger := userLogger.WithRequestID("req-456")
	requestLogger.Info("chained context test")

	output := logBuffer.String()
	assert.Contains(t, output, "chained context test")
	assert.Contains(t, output, "user_id")
	assert.Contains(t, output, "123")
	assert.Contains(t, output, "request_id")
	assert.Contains(t, output, "req-456")
}
