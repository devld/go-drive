package logging

import (
	"fmt"
	"io"
	"log"
	"os"
	"strings"
	"sync/atomic"
)

const (
	logLevelFieldWidth       = len("[ERROR]")
	logComponentContentWidth = len("db-mgrt")
	logContinuationIndent    = "  "
)

// Level controls which log messages are written.
type Level uint8

const (
	DebugLevel Level = iota
	InfoLevel
	WarnLevel
	ErrorLevel
)

func (l Level) String() string {
	switch l {
	case DebugLevel:
		return "DEBUG"
	case InfoLevel:
		return "INFO"
	case WarnLevel:
		return "WARN"
	case ErrorLevel:
		return "ERROR"
	default:
		return "UNKNOWN"
	}
}

// ParseLevel parses a configured log level. An empty value uses the default
// Info level.
func ParseLevel(value string) (Level, error) {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case "", "info":
		return InfoLevel, nil
	case "debug":
		return DebugLevel, nil
	case "warn", "warning":
		return WarnLevel, nil
	case "error":
		return ErrorLevel, nil
	default:
		return InfoLevel, fmt.Errorf("unsupported log level %q", value)
	}
}

// Sanitize escapes line breaks that could break the line-oriented log format
// when logging user-controlled values.
func Sanitize(value string) string {
	return strings.NewReplacer("\r", `\r`, "\n", `\n`).Replace(value)
}

// Logger is a lightweight component view over one process-wide standard
// library logger. Component views share the same output lock.
type Logger struct {
	out       *log.Logger
	component string
	level     *atomic.Int32
}

func New(w io.Writer) *Logger {
	if w == nil {
		w = io.Discard
	}
	return &Logger{
		out:   log.New(w, "", log.LstdFlags|log.Lmicroseconds),
		level: newLevel(InfoLevel),
	}
}

var defaultLogger = New(os.Stdout)

func newLevel(level Level) *atomic.Int32 {
	value := &atomic.Int32{}
	value.Store(int32(level))
	return value
}

// Default returns the process-wide application logger.
func Default() *Logger {
	return defaultLogger
}

// SetLevel changes the minimum level of the process-wide application logger.
func SetLevel(level Level) {
	defaultLogger.SetLevel(level)
}

// Enabled reports whether a message at level would be written by the default
// application logger.
func Enabled(level Level) bool {
	return defaultLogger.Enabled(level)
}

// For returns a component-specific view of the process-wide application
// logger.
func For(component string) *Logger {
	return defaultLogger.For(component)
}

// For returns a component-specific view that shares the underlying logger.
func (l *Logger) For(component string) *Logger {
	if l == nil {
		l = Default()
	}
	return &Logger{out: l.out, component: component, level: l.level}
}

// SetLevel changes the minimum level for this logger and all of its component
// views.
func (l *Logger) SetLevel(level Level) {
	if l == nil || l.level == nil {
		return
	}
	l.level.Store(int32(normalizeLevel(level)))
}

// Enabled reports whether a message at level would be written.
func (l *Logger) Enabled(level Level) bool {
	if l == nil || l.out == nil {
		return false
	}
	if l.level == nil {
		return level >= InfoLevel
	}
	return normalizeLevel(level) >= normalizeLevel(Level(l.level.Load()))
}

func normalizeLevel(level Level) Level {
	if level > ErrorLevel {
		return ErrorLevel
	}
	return level
}

// Debugf writes a debug message when debug logging is enabled.
func (l *Logger) Debugf(format string, args ...any) {
	l.log(DebugLevel, format, args...)
}

// Infof writes an informational message.
func (l *Logger) Infof(format string, args ...any) {
	l.log(InfoLevel, format, args...)
}

// Warnf writes a warning message.
func (l *Logger) Warnf(format string, args ...any) {
	l.log(WarnLevel, format, args...)
}

// Errorf writes an error message.
func (l *Logger) Errorf(format string, args ...any) {
	l.log(ErrorLevel, format, args...)
}

// Printf is kept as an Info-level compatibility entry point.
// Newlines in the message, for example an error stack, are preserved.
func (l *Logger) Printf(format string, args ...any) {
	l.Infof(format, args...)
}

func formatComponent(component string) string {
	padding := logComponentContentWidth - len(component)
	if padding <= 0 {
		return "[" + component + "]"
	}
	left := padding / 2
	return "[" + strings.Repeat(" ", left) + component +
		strings.Repeat(" ", padding-left) + "]"
}

func (l *Logger) log(level Level, format string, args ...any) {
	if !l.Enabled(level) {
		return
	}

	if l == nil || l.out == nil {
		return
	}

	message := indentContinuationLines(fmt.Sprintf(format, args...))
	if l.component != "" {
		message = fmt.Sprintf("%*s %s %s",
			logLevelFieldWidth, "["+level.String()+"]",
			formatComponent(l.component), message)
	} else {
		message = fmt.Sprintf("%*s %s", logLevelFieldWidth, "["+level.String()+"]", message)
	}
	l.out.Output(3, message)
}

func indentContinuationLines(message string) string {
	if !strings.Contains(message, "\n") {
		return message
	}

	lines := strings.Split(message, "\n")
	for i := 1; i < len(lines); i++ {
		if i == len(lines)-1 && lines[i] == "" {
			continue
		}
		lines[i] = logContinuationIndent + lines[i]
	}
	return strings.Join(lines, "\n")
}

// Writer adapts component logging to APIs that require io.Writer, such as
// Gin's debug output hooks.
func (l *Logger) Writer() io.Writer {
	if l == nil {
		l = Default()
	}
	return loggerWriter{logger: l, level: InfoLevel}
}

// ErrorWriter adapts component logging to an io.Writer that emits errors.
func (l *Logger) ErrorWriter() io.Writer {
	if l == nil {
		l = Default()
	}
	return loggerWriter{logger: l, level: ErrorLevel}
}

// StdLogger adapts component logging to APIs that require *log.Logger, such
// as http.Server.ErrorLog.
func (l *Logger) StdLogger() *log.Logger {
	if l == nil {
		l = Default()
	}
	return log.New(l.ErrorWriter(), "", 0)
}

type loggerWriter struct {
	logger *Logger
	level  Level
}

func (w loggerWriter) Write(p []byte) (int, error) {
	if len(p) > 0 {
		w.logger.log(w.level, "%s", string(p))
	}
	return len(p), nil
}
