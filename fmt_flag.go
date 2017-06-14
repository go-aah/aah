// Copyright (c) Jeevanandam M. (https://github.com/jeevatkm)
// go-aah/essentials source code and usage is governed by a MIT style
// license that can be found in the LICENSE file.

package ess

import (
	"fmt"
	"strings"
)

var (
	// FmtFlagSeparator is used parse flags pattern.
	FmtFlagSeparator = "%"

	// FmtFlagValueSeparator is used to parse into flag and value.
	FmtFlagValueSeparator = ":"

	defaultFormat = "%v"
)

type (
	// FmtFlagPart is indiviual flag details
	//  For e.g.:
	//    part := FmtFlagPart{
	//      Flag:   FmtFlagTime,
	//      Name:   "time",
	//      Format: "2006-01-02 15:04:05.000",
	//    }
	FmtFlagPart struct {
		Flag   FmtFlag
		Name   string
		Format string
	}

	// FmtFlag type definition
	FmtFlag uint8
)

// ParseFmtFlag it parses the given pattern, format flags into format flag parts.
//  For e.g.:
//    %time:2006-01-02 15:04:05.000 %level:-5 %message
//    %clientip %reqid %reqtime %restime %resstatus %ressize %reqmethod %requrl %reqhdr:Referer %reshdr:Server
func ParseFmtFlag(pattern string, fmtFlags map[string]FmtFlag) ([]FmtFlagPart, error) {
	var flagParts []FmtFlagPart
	pattern = strings.TrimSpace(pattern)
	formatFlags := strings.Split(pattern, FmtFlagSeparator)[1:]
	for _, f := range formatFlags {
		f = strings.TrimSpace(f)
		parts := strings.SplitN(f, FmtFlagValueSeparator, 2)
		flag, found := fmtFlags[parts[0]]
		if !found {
			return nil, fmt.Errorf("fmtflag: unknown flag '%s'", f)
		}

		part := FmtFlagPart{Flag: flag, Name: parts[0]}
		switch len(parts) {
		case 2:
			// handle `time` related flag, `custom` flag
			// and `hdr` flag particularly
			if strings.Contains(parts[0], "time") || parts[0] == "custom" ||
				strings.HasSuffix(parts[0], "hdr") {
				part.Format = parts[1]
			} else {
				part.Format = "%" + parts[1] + "v"
			}
		default:
			part.Format = defaultFormat
		}

		flagParts = append(flagParts, part)
	}

	return flagParts, nil
}
