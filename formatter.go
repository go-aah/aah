// Copyright (c) 2016 Jeevanandam M (https://github.com/jeevatkm)
// resty source code and usage is governed by a MIT style
// license that can be found in the LICENSE file.

package log

import (
	"errors"
	"fmt"
	"os"
	"strconv"
	"strings"
)

// FmtFlag type definition
type FmtFlag uint8

// Format flags used to define log message format for each log entry
const (
	FmtFlagLevel FmtFlag = iota
	FmtFlagTime
	FmtFlagUTCTime
	FmtFlagLongfile
	FmtFlagShortfile
	FmtFlagLine
	FmtFlagMessage
	FmtFlagCustom
	FmtFlagUnknown
)

var (
	// FmtFlags is the list of log format flags supported by aah/log library
	// Usage of flag order is upto format composition.
	//    level     - outputs INFO, DEBUG, ERROR, so on
	//    time      - outputs local time as per format supplied
	//    utctime   - outputs UTC time as per format supplied
	//    longfile  - outputs full file name: /a/b/c/d.go
	//    shortfile - outputs final file name element: d.go
	//    line      - outputs file line number: L23
	//    message   - outputs given message along supplied arguments if they present
	//    custom    - outputs string as-is into log entry
	FmtFlags = map[string]FmtFlag{
		"level":     FmtFlagLevel,
		"time":      FmtFlagTime,
		"utctime":   FmtFlagUTCTime,
		"longfile":  FmtFlagLongfile,
		"shortfile": FmtFlagShortfile,
		"line":      FmtFlagLine,
		"message":   FmtFlagMessage,
		"custom":    FmtFlagCustom,
	}

	// DefaultPattern is default log entry pattern in aah/log
	// For e.g:
	//    2006-01-02 15:04:05.000 INFO  - This is my message
	DefaultPattern = "%time:2006-01-02 15:04:05.000 %level:-5 %custom:- %message"

	// BackupTimeFormat is used for timestamp with filename on rotation
	BackupTimeFormat = "2006-01-02-15-04-05.000"

	// ErrFormatStringEmpty returned when log format parameter is empty
	ErrFormatStringEmpty = errors.New("log format string is empty")

	flagSeparator      = "%"
	flagValueSeparator = ":"
	defaultFormat      = "%v"
	filePermission     = os.FileMode(0755)
)

// FormatterFunc is the handler function to implement log entry
// formatting log entry based on log format flags.
type FormatterFunc func(flags *[]FlagPart, entry *Entry, isColor bool) (*[]byte, error)

// FlagPart is indiviual flag details
//  For e.g.:
//    part := FlagPart{
//      Flag:   fmtFlagTime,
//      Name:   "time",
//      Format: "2006-01-02 15:04:05.000",
//    }
type FlagPart struct {
	Flag   FmtFlag
	Name   string
	Format string
}

// parseFlag it parses the log message formart into flag parts
//  For e.g.:
//    %time:2006-01-02 15:04:05.000 %level %custom:- %msg
func parseFlag(format string) (*[]FlagPart, error) {
	if strIsEmpty(format) {
		return nil, ErrFormatStringEmpty
	}

	var flagParts []FlagPart
	format = strings.TrimSpace(format)
	formatFlags := strings.Split(format, flagSeparator)[1:]
	for _, f := range formatFlags {
		parts := strings.SplitN(strings.TrimSpace(f), flagValueSeparator, 2)
		flag := getFmtFlagByName(parts[0])
		if flag == FmtFlagUnknown {
			return nil, fmt.Errorf("unrecognized log format flag: %v", f)
		}

		part := FlagPart{Flag: flag, Name: parts[0]}
		switch len(parts) {
		case 2:
			if flag == FmtFlagTime || flag == FmtFlagUTCTime ||
				flag == FmtFlagCustom {
				part.Format = parts[1]
			} else {
				part.Format = "%" + parts[1] + "v"
			}
		default:
			part.Format = defaultFormat
		}

		flagParts = append(flagParts, part)
	}

	return &flagParts, nil
}

// DefaultFormatter formats the `Entry` object details as per log `pattern`
// 	For e.g.:
// 		2016-07-02 22:26:01.530 INFO formatter_test.go L29 - Yes, I would love to see
func DefaultFormatter(flags *[]FlagPart, entry *Entry, isColor bool) (*[]byte, error) {
	var buf []byte

	if isColor {
		buf = append(buf, levelToColor[entry.Level]...)
	}

	for _, part := range *flags {
		switch part.Flag {
		case FmtFlagLevel:
			buf = append(buf, fmt.Sprintf(part.Format, entry.Level)...)
		case FmtFlagTime:
			buf = append(buf, entry.Time.Format(part.Format)...)
		case FmtFlagUTCTime:
			buf = append(buf, entry.Time.UTC().Format(part.Format)...)
		case FmtFlagLongfile, FmtFlagShortfile:
			if part.Flag == FmtFlagShortfile {
				if slash := strings.LastIndex(entry.File, "/"); slash >= 0 {
					entry.File = entry.File[slash+1:]
				}
			}
			buf = append(buf, fmt.Sprintf(part.Format, entry.File)...)
		case FmtFlagLine:
			buf = append(buf, fmt.Sprintf(part.Format, "L"+strconv.Itoa(entry.Line))...)
		case FmtFlagMessage:
			if strIsEmpty(entry.Format) {
				buf = append(buf, fmt.Sprint(entry.Values...)...)
			} else {
				buf = append(buf, fmt.Sprintf(entry.Format, entry.Values...)...)
			}
		case FmtFlagCustom:
			buf = append(buf, part.Format...)
		}

		buf = append(buf, ' ')
	}

	if isColor {
		buf = append(buf, resetColor...)
	}

	buf = append(buf, '\n')

	return &buf, nil
}

// unexported methods

func getFmtFlagByName(name string) FmtFlag {
	if flag, ok := FmtFlags[name]; ok {
		return flag
	}

	return FmtFlagUnknown
}

func isFileFlagExists(flags *[]FlagPart) bool {
	return (isFmtFlagExists(flags, FmtFlagShortfile) ||
		isFmtFlagExists(flags, FmtFlagLongfile))
}

func isFmtFlagExists(flags *[]FlagPart, flag FmtFlag) bool {
	for _, f := range *flags {
		if f.Flag == flag {
			return true
		}
	}

	return false
}
