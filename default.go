// Copyright (c) 2016 Jeevanandam M (https://github.com/jeevatkm)
// resty source code and usage is governed by a MIT style
// license that can be found in the LICENSE file.

package log

var stdLogger Logger

// Error logs message as `LevelError`
func Error(v ...interface{}) {
	stdLogger.Error(v...)
}

// Errorf logs message as `LevelError`
func Errorf(format string, v ...interface{}) {
	stdLogger.Errorf(format, v...)
}

// Warn logs message as `LevelWarn`
func Warn(v ...interface{}) {
	stdLogger.Warn(v...)
}

// Warnf logs message as `LevelWarn`
func Warnf(format string, v ...interface{}) {
	stdLogger.Warnf(format, v...)
}

// Info logs message as `LevelInfo`
func Info(v ...interface{}) {
	stdLogger.Info(v...)
}

// Infof logs message as `LevelInfo`
func Infof(format string, v ...interface{}) {
	stdLogger.Infof(format, v...)
}

// Debug logs message as `LevelDebug`
func Debug(v ...interface{}) {
	stdLogger.Debug(v...)
}

// Debugf logs message as `LevelDebug`
func Debugf(format string, v ...interface{}) {
	stdLogger.Debugf(format, v...)
}

// Trace logs message as `LevelTrace`
func Trace(v ...interface{}) {
	stdLogger.Trace(v...)
}

// Tracef logs message as `LevelTrace`
func Tracef(format string, v ...interface{}) {
	stdLogger.Tracef(format, v...)
}

// Stats returns current logger statistics like number of lines written,
// number of bytes written, etc.
func Stats() *ReceiverStats {
	return stdLogger.Stats()
}

func init() {
	stdLogger, _ = New(`receiver = "CONSOLE"; level = "DEBUG";`)
}
