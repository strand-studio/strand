package logger

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"time"

	"github.com/sirupsen/logrus"
)

var (
	Log     *logrus.Logger
	logFile *os.File
)

type compactFormatter struct {
	logrus.TextFormatter
}

func (f *compactFormatter) Format(entry *logrus.Entry) ([]byte, error) {
	level := strings.ToUpper(entry.Level.String())[0:4]
	timestamp := entry.Time.Format("01-02 15:04:05")

	caller := ""
	for i := 3; i < 10; i++ {
		_, file, line, ok := runtime.Caller(i)
		if !ok {
			break
		}
		if !strings.Contains(file, "logrus") && !strings.HasSuffix(file, "logger/logger.go") {
			dir := filepath.Dir(file)
			pkg := filepath.Base(dir)
			caller = fmt.Sprintf("%s/%s:%d", pkg, filepath.Base(file), line)
			break
		}
	}

	msg := fmt.Sprintf("%s [%s] %s %s\n", timestamp, level, caller, entry.Message)
	return []byte(msg), nil
}

func init() {
	Log = logrus.New()
	Log.SetLevel(logrus.InfoLevel)
	Log.SetFormatter(&compactFormatter{})
	Log.SetOutput(os.Stdout)
}

// ============================================================================
// Initialization functions
// ============================================================================

// Init initializes the global logger
func Init(cfg *Config) error {
	Log = logrus.New()

	if cfg == nil {
		cfg = &Config{Level: "info"}
	}

	cfg.SetDefault()

	level, err := logrus.ParseLevel(cfg.Level)
	if err != nil {
		level = logrus.InfoLevel
	}
	Log.SetLevel(level)

	Log.SetFormatter(&compactFormatter{})

	logDir := "data"
	if err := os.MkdirAll(logDir, 0755); err == nil {
		logFileName := filepath.Join(logDir, fmt.Sprintf("strand_%s.log", time.Now().Format("2006-01-02")))
		f, err := os.OpenFile(logFileName, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
		if err == nil {
			logFile = f
			// Write to both stdout and file
			Log.SetOutput(io.MultiWriter(os.Stdout, f))
		} else {
			Log.SetOutput(os.Stdout)
		}
	} else {
		Log.SetOutput(os.Stdout)
	}

	Log.SetReportCaller(true)

	return nil
}

func InitWithSimpleConfig(level string) error {
	return Init(&Config{Level: level})
}

// Shutdown gracefully shuts down the logger
func Shutdown() {
	if logFile != nil {
		logFile.Close()
		logFile = nil
	}
}

// ============================================================================
// Logging functions
// ============================================================================

func WithFields(fields logrus.Fields) *logrus.Entry {
	return Log.WithFields(fields)
}

func WithField(key string, value any) *logrus.Entry {
	return Log.WithField(key, value)
}

func Debug(args ...any) {
	Log.Debug(args...)
}

func Info(args ...any) {
	Log.Info(args...)
}

func Warn(args ...any) {
	Log.Warn(args...)
}

func Debugf(format string, args ...any) {
	Log.Debugf(format, args...)
}

func Infof(format string, args ...any) {
	Log.Infof(format, args...)
}

func Warnf(format string, args ...any) {
	Log.Warnf(format, args...)
}

func Error(args ...any) {
	Log.Error(args...)
}

func Errorf(format string, args ...any) {
	Log.Errorf(format, args...)
}

func Fatal(args ...any) {
	Log.Fatal(args...)
}

func Fatalf(format string, args ...any) {
	Log.Fatalf(format, args...)
}

func Panic(args ...any) {
	Log.Panic(args...)
}

func Panicf(format string, args ...any) {
	Log.Panicf(format, args...)
}

// ============================================================================
// MCP Logger adapter
// ============================================================================

// MCPLogger adapter that allows MCP package to use the global logger
type MCPLogger struct{}

func NewMCPLogger() *MCPLogger {
	return &MCPLogger{}
}

func (l *MCPLogger) Debugf(format string, args ...any) {
	Log.Debugf(format, args...)
}

func (l *MCPLogger) Infof(format string, args ...any) {
	Log.Infof(format, args...)
}

func (l *MCPLogger) Warnf(format string, args ...any) {
	Log.Warnf(format, args...)
}

func (l *MCPLogger) Errorf(format string, args ...any) {
	Log.Errorf(format, args...)
}
