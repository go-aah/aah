// Copyright (c) Jeevanandam M. (https://github.com/jeevatkm)
// go-aah/essentials source code and usage is governed by a MIT style
// license that can be found in the LICENSE file.

package ess

import (
	"encoding/binary"
	"encoding/hex"
	"testing"
	"time"

	"aahframework.org/test.v0/assert"
)

func TestGUIDNew(t *testing.T) {
	// Generate 10 ids
	ids := make([]string, 10)
	for i := 0; i < 10; i++ {
		ids[i] = NewGUID()
	}
	for i := 1; i < 10; i++ {
		prevID := ids[i-1]
		id := ids[i]
		// Test for uniqueness among all other 9 generated ids
		for j, tid := range ids {
			if j != i {
				assert.NotEqualf(t, id, tid, "Generated ID is not unique")
			}
		}

		// Check that timestamp was incremented and is within 30 seconds of the previous one
		secs := getTime(id).Sub(getTime(prevID)).Seconds()
		assert.Equalf(t, (secs >= 0 && secs <= 30), true, "wrong timestamp in generated Id")

		// Check that machine ids are the same
		assert.Equal(t, getMachine(id), getMachine(prevID))

		// Check that pids are the same
		assert.Equal(t, getPid(id), getPid(prevID))

		// Test for proper increment
		delta := int(getCounter(id) - getCounter(prevID))
		assert.Equalf(t, delta, 1, "wrong increment in generated Id")
	}
}

type guidParts struct {
	id        string
	timestamp int64
	machine   []byte
	pid       uint16
	counter   int32
}

var uniqueIds = []guidParts{
	{
		"4d88e15b60f486e428412dc9",
		1300816219,
		[]byte{0x60, 0xf4, 0x86},
		0xe428,
		4271561,
	},
	{
		"000000000000000000000000",
		0,
		[]byte{0x00, 0x00, 0x00},
		0x0000,
		0,
	},
	{
		"00000000aabbccddee000001",
		0,
		[]byte{0xaa, 0xbb, 0xcc},
		0xddee,
		1,
	},
}

func TestGUIDPartsExtraction(t *testing.T) {
	for i, v := range uniqueIds {
		assert.Equalf(t, getTime(v.id), time.Unix(v.timestamp, 0), "#%d timestamp", i)
		assert.Equalf(t, getMachine(v.id), v.machine, "#%d machine", i)
		assert.Equalf(t, getPid(v.id), v.pid, "#%d pid", i)
		assert.Equalf(t, getCounter(v.id), v.counter, "#%d counter", i)
	}
}

func BenchmarkNewGUID(b *testing.B) {
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			_ = NewGUID()
		}
	})
}

func getMachine(id string) []byte {
	return byteSlice(id, 4, 7)
}

func getTime(id string) time.Time {
	secs := int64(binary.BigEndian.Uint32(byteSlice(id, 0, 4)))
	return time.Unix(secs, 0)
}

func getPid(id string) uint16 {
	return binary.BigEndian.Uint16(byteSlice(id, 7, 9))
}

func getCounter(id string) int32 {
	b := byteSlice(id, 9, 12)
	// Counter is stored as big-endian 3-byte value
	return int32(uint32(b[0])<<16 | uint32(b[1])<<8 | uint32(b[2]))
}

func byteSlice(id string, s, e int) []byte {
	if len(id) == 24 {
		b, _ := hex.DecodeString(id)
		return b[s:e]
	}
	return nil
}
