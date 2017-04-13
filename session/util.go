// Copyright (c) Jeevanandam M. (https://github.com/jeevatkm)
// go-aah/security source code and usage is governed by a MIT style
// license that can be found in the LICENSE file.

package session

import (
	"fmt"
	"strings"
	"time"
)

// currentTimestamp method return current UTC time in unix format.
func currentTimestamp() int64 {
	return time.Now().UTC().Unix()
}

// toBytes method encodes into byte slice.
func toBytes(v interface{}) ([]byte, error) {
	switch v.(type) {
	case []byte:
		return v.([]byte), nil
	default:
		return encodeGob(v)
	}
}

// toSeconds method converts string value into seconds.
func toSeconds(value string) (int64, error) {
	if strings.HasSuffix(value, "m") || strings.HasSuffix(value, "h") {
		d, err := time.ParseDuration(value)
		if err != nil {
			return 0, err
		}
		return int64(d.Seconds()), nil
	}
	return 0, fmt.Errorf("unsupported time unit '%s' on 'session.ttl'", value)
}
