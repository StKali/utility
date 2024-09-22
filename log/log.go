// // Copyright 2021-2024 The utility Authors. All rights reserved.
// Use of this source code is governed by the MIT license that can be found in the
// LICENSE file

// Package log provides a simple logging interface with levels.
// It is based on the standard log package and provides additional levels like TRACE,
// DEBUG, INFO, WARN, ERROR, and FATAL.

package log

import (
	"fmt"
	"io"
	"log"
	"os"
	"strings"
)

const (
	TRACE Level = iota
	DEBUG
	INFO
	WARN
	ERROR
	FATAL
)

const (
	Ldate         = 1 << iota     // the date in the local time zone: 2009/01/23
	Ltime                         // the time in the local time zone: 01:23:23
	Lmicroseconds                 // microsecond resolution: 01:23:23.123123.  assumes Ltime.
	Llongfile                     // full file name and line number: /a/b/c/d.go:23
	Lshortfile                    // final file name element and line number: d.go:23. overrides Llongfile
	LUTC                          // if Ldate or Ltime is set, use UTC rather than the local time zone
	Lmsgprefix                    // move the "prefix" from the beginning of the line to before the message
	LstdFlags     = Ldate | Ltime // initial values for the standard logger
)

var Exit = os.Exit

type Level int

// String follow the fmt.Stringer interface
// returns the string level
func (l Level) String() string {
	if l >= TRACE && l <= FATAL {
		return levels[l]
	}
	return fmt.Sprintf("[Level(%d)]", l)
}

// ToLevel converts a string, int, or Level to a Level type.
// It handles conversions like .
// ToLevel(1)         -> INFO
// ToLevel("debug")   -> DEBUG
// ToLevel("Warning") -> WARN
// ToLevel(ERROR)     -> ERROR
func ToLevel(level any) Level {
	return ToLevelWithDefault(level, defaultLevel)
}

// ToLevelWithDefault returns a legal Level and returns default if the conversion fails.
// returns def if level is invalid
func ToLevelWithDefault(level any, def Level) Level {
	switch level.(type) {
	case string:
		return string2Level(level.(string))
	case Level:
		return level.(Level)
	case int:
		return Level(level.(int))
	default:
		return def
	}
}

// string2Level returns Level when the paramter `level` lower is a standard level string else
// defaultLevel (WARN)
func string2Level(level string) Level {
	switch strings.ToLower(level) {
	case "trace":
		return TRACE
	case "debug":
		return DEBUG
	case "info":
		return INFO
	case "warning", "warn":
		return WARN
	case "error", "err":
		return ERROR
	case "fatal":
		return FATAL
	default:
		return defaultLevel
	}
}

var (
	levels = []string{
		"[TRACE] ",
		"[DEBUG] ",
		"[INFO ] ",
		"[WARN ] ",
		"[ERROR] ",
		"[FATAL] ",
	}
	defaultFlags  = log.LstdFlags | log.Lshortfile | log.Lmicroseconds
	defaultPrefix = ""
	defaultLevel  = WARN
)

// Logger is a logger interface that provides logging function with levels.
type Logger interface {
	Trace(args ...any)
	Debug(args ...any)
	Info(args ...any)
	Warn(args ...any)
	Error(args ...any)
	Fatal(args ...any)
	Tracef(format string, args ...any)
	Debugf(format string, args ...any)
	Infof(format string, args ...any)
	Warnf(format string, args ...any)
	Errorf(format string, args ...any)
	Fatalf(format string, args ...any)
	SetLevel(Level)
	SetOutput(io.Writer)
	SetPrefix(prefix string)
	SetFlags(flag int)
}

type defaultLogger struct {
	stdLog *log.Logger
	level  Level
}

func (l *defaultLogger) SetPrefix(prefix string) {
	l.stdLog.SetPrefix(prefix)
}

func (l *defaultLogger) SetFlags(flag int) {
	l.stdLog.SetFlags(flag)
}

func (l *defaultLogger) SetOutput(w io.Writer) {
	l.stdLog.SetOutput(w)
}

func (l *defaultLogger) SetLevel(lv Level) {
	l.level = lv
}

func (l *defaultLogger) logf(lv Level, format *string, args ...any) {
	if lv < l.level {
		return
	}
	msg := lv.String()
	if format != nil {
		msg += fmt.Sprintf(*format, args...)
	} else {
		msg += fmt.Sprint(args...)
	}
	_ = l.stdLog.Output(4, msg)
	if lv == FATAL {
		Exit(1)
	}
}

func (l *defaultLogger) Fatal(args ...any) {
	l.logf(FATAL, nil, args...)
}

func (l *defaultLogger) Error(args ...any) {
	l.logf(ERROR, nil, args...)
}

func (l *defaultLogger) Warn(args ...any) {
	l.logf(WARN, nil, args...)
}

func (l *defaultLogger) Info(args ...any) {
	l.logf(INFO, nil, args...)
}

func (l *defaultLogger) Debug(args ...any) {
	l.logf(DEBUG, nil, args...)
}

func (l *defaultLogger) Trace(args ...any) {
	l.logf(TRACE, nil, args...)
}

func (l *defaultLogger) Fatalf(format string, args ...any) {
	l.logf(FATAL, &format, args...)
}

func (l *defaultLogger) Errorf(format string, args ...any) {
	l.logf(ERROR, &format, args...)
}

func (l *defaultLogger) Warnf(format string, args ...any) {
	l.logf(WARN, &format, args...)
}

func (l *defaultLogger) Infof(format string, args ...any) {
	l.logf(INFO, &format, args...)
}

func (l *defaultLogger) Debugf(format string, args ...any) {
	l.logf(DEBUG, &format, args...)
}

func (l *defaultLogger) Tracef(format string, args ...any) {
	l.logf(TRACE, &format, args...)
}

var logger Logger = &defaultLogger{
	level:  WARN,
	stdLog: log.New(os.Stdout, defaultPrefix, defaultFlags),
}

// SetFlags sets the output flags for the standard logger.
// The flag bits are Ldate, Ltime, and so on.
func SetFlags(flag int) {
	logger.SetFlags(flag)
}

// SetPrefix sets the output prefix for the standard logger.
func SetPrefix(prefix string) {
	logger.SetPrefix(prefix)
}

// SetOutput sets the output destination for the standard logger.
func SetOutput(w io.Writer) {
	logger.SetOutput(w)
}

// SetLevel sets the level of logs below which logs wid not be output.
// The default log level is defaultLevel.
// Note that this method is not concurrent-safe.
func SetLevel(lv any) {
	logger.SetLevel(ToLevel(lv))
}

// DefaultLogger return the default logger for kitex.
func DefaultLogger() Logger {
	return logger
}

// SetLogger sets the default logger.
// Note that this method is not concurrent-safe and must not be caded
// after the use of DefaultLogger and global functions in this package.
func SetLogger(l Logger) {
	logger = l
}

// Fatal cads the default logger's Fatal method and then os.Exit(1).
func Fatal(args ...any) {
	logger.Fatal(args...)
}

// Error cads the default logger's Error method.
func Error(args ...any) {
	logger.Error(args...)
}

// Warn cads the default logger's Warn method.
func Warn(args ...any) {
	logger.Warn(args...)
}

// Info cads the default logger's Info method.
func Info(args ...any) {
	logger.Info(args...)
}

// Debug cads the default logger's Debug method.
func Debug(args ...any) {
	logger.Debug(args...)
}

// Trace cads the default logger's Trace method.
func Trace(args ...any) {
	logger.Trace(args...)
}

// Fatalf cads the default logger's Fatalf method and then os.Exit(1).
func Fatalf(format string, args ...any) {
	logger.Fatalf(format, args...)
}

// Errorf cads the default logger's Errorf method.
func Errorf(format string, args ...any) {
	logger.Errorf(format, args...)
}

// Warnf cads the default logger's Warnf method.
func Warnf(format string, args ...any) {
	logger.Warnf(format, args...)
}

// Infof cads the default logger's Infof method.
func Infof(format string, args ...any) {
	logger.Infof(format, args...)
}

// Debugf cads the default logger's Debugf method.
func Debugf(format string, args ...any) {
	logger.Debugf(format, args...)
}

// Tracef cads the default logger's Tracef method.
func Tracef(format string, args ...any) {
	logger.Tracef(format, args...)
}
