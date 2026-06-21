package main

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"
)

// PrettyLogWriter renders structured, leveled log lines. The console copy is
// colored by level; the file copy stays plain. WARN/ERROR lines can be forwarded
// to an external sink (the log group) without ever mixing into the debug stream.
type PrettyLogWriter struct {
	mu       sync.Mutex
	console  io.Writer
	file     io.Writer
	minLevel int
	color    bool
	forward  func(level int, label, line string)
}

const (
	logDebug = iota
	logInfo
	logWarn
	logError
)

var levelColors = map[int]string{
	logDebug: "90", // bright black / gray
	logInfo:  "36", // cyan
	logWarn:  "33", // yellow
	logError: "31", // red
}

func NewPrettyLogWriter(console, file io.Writer, level string, color bool) *PrettyLogWriter {
	return &PrettyLogWriter{console: console, file: file, minLevel: parseLogLevel(level), color: color}
}

// SetForward registers a sink for WARN/ERROR lines. It must never log on its own
// (directly or indirectly) or it will feed itself in a loop.
func (w *PrettyLogWriter) SetForward(fn func(level int, label, line string)) {
	w.mu.Lock()
	w.forward = fn
	w.mu.Unlock()
}

func (w *PrettyLogWriter) Write(p []byte) (int, error) {
	line := strings.TrimRight(string(p), "\n")
	if strings.TrimSpace(line) == "" {
		return len(p), nil
	}
	level, label, mark := classifyLog(line)
	if level < w.minLevel {
		return len(p), nil
	}
	timestamp := time.Now().Format("2006-01-02 15:04:05.000")
	plain := fmt.Sprintf("%s │ %-5s │ %s %s\n", timestamp, label, mark, line)
	w.mu.Lock()
	if w.console != nil {
		if w.color {
			_, _ = io.WriteString(w.console, colorize(level, plain))
		} else {
			_, _ = io.WriteString(w.console, plain)
		}
	}
	if w.file != nil {
		_, _ = io.WriteString(w.file, plain)
	}
	forward := w.forward
	w.mu.Unlock()
	if forward != nil && level >= logWarn {
		forward(level, label, line)
	}
	return len(p), nil
}

func colorize(level int, text string) string {
	code, ok := levelColors[level]
	if !ok {
		return text
	}
	return "\x1b[" + code + "m" + strings.TrimRight(text, "\n") + "\x1b[0m\n"
}

// classifyLog maps a raw log line to a level, a short label and an emoji marker.
// The order matters: debug is checked before the generic info keywords so that a
// "debug" line never bleeds into the info/event stream.
func classifyLog(line string) (int, string, string) {
	lower := strings.ToLower(line)
	switch {
	case strings.HasPrefix(lower, "debug") || strings.Contains(lower, "[debug]"):
		return logDebug, "DEBUG", "🔎"
	case strings.Contains(lower, "panic") || strings.Contains(lower, "fatal") || strings.Contains(lower, "failed") || strings.Contains(lower, "error"):
		return logError, "ERROR", "🚨"
	case strings.Contains(lower, "warning") || strings.Contains(lower, "недоступ") || strings.Contains(lower, "не удалось"):
		return logWarn, "WARN", "⚠️"
	case strings.HasPrefix(lower, "event") || strings.HasPrefix(lower, "audit"):
		return logInfo, "EVENT", "📣"
	case strings.Contains(lower, "started") || strings.Contains(lower, "completed") || strings.Contains(lower, "created") || strings.Contains(lower, "approved"):
		return logInfo, "INFO", "✅"
	case strings.Contains(lower, "upload") || strings.Contains(lower, "webhook") || strings.Contains(lower, "payment"):
		return logInfo, "INFO", "📡"
	default:
		return logInfo, "INFO", "ℹ️"
	}
}

func parseLogLevel(level string) int {
	switch strings.ToLower(strings.TrimSpace(level)) {
	case "debug":
		return logDebug
	case "warn", "warning":
		return logWarn
	case "error":
		return logError
	default:
		return logInfo
	}
}

// colorEnabled decides whether ANSI colors are used on the console. It honors the
// NO_COLOR convention and an explicit LOG_COLOR override, defaulting to on.
func colorEnabled() bool {
	switch strings.ToLower(strings.TrimSpace(os.Getenv("LOG_COLOR"))) {
	case "always", "on", "1", "true", "yes":
		return true
	case "never", "off", "0", "false", "no":
		return false
	}
	if _, disabled := os.LookupEnv("NO_COLOR"); disabled {
		return false
	}
	return true
}

func logFilePath(dir string) string {
	if strings.TrimSpace(dir) == "" {
		dir = "logs"
	}
	return filepath.Join(dir, "bot-go.log")
}
