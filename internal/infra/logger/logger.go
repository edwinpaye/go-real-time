package logger

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"sync"
	"time"
)

type Level string

const (
	LevelDebug Level = "DEBUG"
	LevelInfo  Level = "INFO"
	LevelWarn  Level = "WARN"
	LevelError Level = "ERROR"
)

// Fields represents key-value structured context.
type Fields map[string]interface{}

// Logger provides thread-safe structured JSON logging.
type Logger struct {
	mu     sync.Mutex
	out    io.Writer
	env    string
	fields Fields
}

var defaultLogger *Logger

func init() {
	defaultLogger = New(os.Stdout, "production")
}

// New instantiates a new Logger.
func New(out io.Writer, env string) *Logger {
	return &Logger{
		out:    out,
		env:    env,
		fields: make(Fields),
	}
}

// SetDefault sets the global default logger instance.
func SetDefault(l *Logger) {
	defaultLogger = l
}

// WithFields creates a child logger with appended contextual metadata.
func (l *Logger) WithFields(f Fields) *Logger {
	l.mu.Lock()
	defer l.mu.Unlock()

	newFields := make(Fields, len(l.fields)+len(f))
	for k, v := range l.fields {
		newFields[k] = v
	}
	for k, v := range f {
		newFields[k] = v
	}

	return &Logger{
		out:    l.out,
		env:    l.env,
		fields: newFields,
	}
}

func (l *Logger) log(level Level, msg string, extra Fields) {
	l.mu.Lock()
	defer l.mu.Unlock()

	entry := make(Fields, len(l.fields)+len(extra)+4)
	for k, v := range l.fields {
		entry[k] = v
	}
	for k, v := range extra {
		entry[k] = v
	}

	entry["timestamp"] = time.Now().UTC().Format(time.RFC3339Nano)
	entry["level"] = level
	entry["message"] = msg
	entry["env"] = l.env

	data, err := json.Marshal(entry)
	if err != nil {
		fmt.Fprintf(l.out, `{"timestamp":"%s","level":"ERROR","message":"failed to marshal log entry: %v"}`+"\n", time.Now().UTC().Format(time.RFC3339Nano), err)
		return
	}
	l.out.Write(append(data, '\n'))
}

func (l *Logger) Debug(msg string, fields ...Fields) {
	f := mergeFields(fields...)
	l.log(LevelDebug, msg, f)
}

func (l *Logger) Info(msg string, fields ...Fields) {
	f := mergeFields(fields...)
	l.log(LevelInfo, msg, f)
}

func (l *Logger) Warn(msg string, fields ...Fields) {
	f := mergeFields(fields...)
	l.log(LevelWarn, msg, f)
}

func (l *Logger) Error(msg string, fields ...Fields) {
	f := mergeFields(fields...)
	l.log(LevelError, msg, f)
}

func Debug(msg string, fields ...Fields) { defaultLogger.Debug(msg, fields...) }
func Info(msg string, fields ...Fields)  { defaultLogger.Info(msg, fields...) }
func Warn(msg string, fields ...Fields)  { defaultLogger.Warn(msg, fields...) }
func Error(msg string, fields ...Fields) { defaultLogger.Error(msg, fields...) }
func WithFields(f Fields) *Logger        { return defaultLogger.WithFields(f) }

func mergeFields(fields ...Fields) Fields {
	merged := make(Fields)
	for _, f := range fields {
		for k, v := range f {
			merged[k] = v
		}
	}
	return merged
}
