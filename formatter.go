// Copyright (c) Jeevanandam M (https://github.com/jeevatkm)
// go-aah/log source code and usage is governed by a MIT style
// license that can be found in the LICENSE file.

package log

import (
	"encoding/json"
	"fmt"
	"path/filepath"
	"strings"
)

const (
	textFmt = "text"
	jsonFmt = "json"
)

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

//‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾
// textFormatter
//___________________________________

// textFormatter formats the `Entry` object details as per log `pattern`
// 	For e.g.:
// 		2016-07-02 22:26:01.530 INFO formatter_test.go L29 - Yes, I would love to see
func textFormatter(flags *[]FlagPart, entry *Entry) []byte {
	buf := getBuffer()
	defer putBuffer(buf)

	for _, part := range *flags {
		switch part.Flag {
		case FmtFlagLevel:
			buf.WriteString(fmt.Sprintf(part.Format, levelToLevelName[entry.Level]))
		case FmtFlagTime:
			buf.WriteString(entry.Time.Format(part.Format))
		case FmtFlagUTCTime:
			buf.WriteString(entry.Time.UTC().Format(part.Format))
		case FmtFlagLongfile, FmtFlagShortfile:
			if part.Flag == FmtFlagShortfile {
				entry.File = filepath.Base(entry.File)
			}
			buf.WriteString(fmt.Sprintf(part.Format, entry.File))
		case FmtFlagLine:
			buf.WriteString(fmt.Sprintf(part.Format, entry.Line))
		case FmtFlagMessage:
			buf.WriteString(entry.Message)
		case FmtFlagCustom:
			buf.WriteString(part.Format)
		}

		buf.WriteByte(' ')
	}

	buf.WriteByte('\n')
	return buf.Bytes()
}

//‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾
// jsonFormatter
//___________________________________

func jsonFormatter(entry *Entry) ([]byte, error) {
	return json.Marshal(entry)
}

//‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾
// Unexported methods
//___________________________________

func applyFormatter(formatter string, flags *[]FlagPart, entry *Entry) []byte {
	if formatter == textFmt {
		return textFormatter(flags, entry)
	}

	lm, _ := jsonFormatter(entry)
	return lm
}

// parseFlag it parses the log message formart into flag parts
//  For e.g.:
//    %time:2006-01-02 15:04:05.000 %level %custom:- %msg
func parseFlag(format string) (*[]FlagPart, error) {
	var flagParts []FlagPart
	format = strings.TrimSpace(format)
	formatFlags := strings.Split(format, flagSeparator)[1:]
	for _, f := range formatFlags {
		parts := strings.SplitN(strings.TrimSpace(f), flagValueSeparator, 2)
		flag := fmtFlagByName(parts[0])
		if flag == FmtFlagUnknown {
			return nil, fmt.Errorf("unrecognized log format flag: %v", strings.TrimSpace(f))
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
			if flag == FmtFlagLine {
				part.Format = "L" + defaultFormat
			}
		}

		flagParts = append(flagParts, part)
	}

	return &flagParts, nil
}
