// Copyright (c) Jeevanandam M. (https://github.com/jeevatkm)
// go-aah/essentials source code and usage is governed by a MIT style
// license that can be found in the LICENSE file.

package ess

import (
	"crypto/md5"
	"crypto/rand"
	"encoding/binary"
	"encoding/hex"
	"errors"
	"io"
	"os"
	"sync/atomic"
	"time"
)

//‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾
// GUID generation
// Code inspired from mgo/bson ObjectId
//___________________________________

var (
	// guidCounter is atomically incremented when generating a new GUID
	// using UniqueID() function. It's used as a counter part of an id.
	guidCounter = readRandomUint32()

	// machineID stores machine id generated once and used in subsequent calls
	// to UniqueId function.
	machineID = readMachineID()

	// processID is current Process Id
	processID = os.Getpid()
)

//‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾
// Global methods
//___________________________________

// NewGUID method returns a new Globally Unique identifier (GUID).
//
// The 12-byte `UniqueId` consists of-
//   - 4-byte value representing the seconds since the Unix epoch,
//   - 3-byte machine identifier,
//   - 2-byte process id, and
//   - 3-byte counter, starting with a random value.
//
// NewGUID generation using Mongo Object ID algorithm to generate globally
// unique ids - https://docs.mongodb.com/manual/reference/method/ObjectId/
func NewGUID() string {
	var b [12]byte
	// Timestamp, 4 bytes, big endian
	binary.BigEndian.PutUint32(b[:], uint32(time.Now().Unix()))

	// Machine, first 3 bytes of md5(hostname)
	b[4], b[5], b[6] = machineID[0], machineID[1], machineID[2]

	// Pid, 2 bytes, specs don't specify endianness, but we use big endian.
	b[7], b[8] = byte(processID>>8), byte(processID)

	// Increment, 3 bytes, big endian
	i := atomic.AddUint32(&guidCounter, 1)
	b[9], b[10], b[11] = byte(i>>16), byte(i>>8), byte(i)

	return hex.EncodeToString(b[:])
}

//‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾
// Unexported methods
//___________________________________

// readRandomUint32 returns a random guidCounter.
func readRandomUint32() uint32 {
	var b [4]byte
	if _, err := io.ReadFull(rand.Reader, b[:]); err == nil {
		return (uint32(b[0]) << 0) | (uint32(b[1]) << 8) | (uint32(b[2]) << 16) | (uint32(b[3]) << 24)
	}

	panic(errors.New("ess - guid: unable to generate random object id"))
}

// readMachineID generates and returns a machine id.
// If this function fails to get the hostname it will cause a runtime error.
func readMachineID() []byte {
	var sum [3]byte
	id := sum[:]

	if hostname, err := os.Hostname(); err == nil {
		hw := md5.New()
		_, _ = hw.Write([]byte(hostname))
		copy(id, hw.Sum(nil))
		return id
	}

	if _, err := io.ReadFull(rand.Reader, id); err == nil {
		return id
	}

	// return nil, errors.New("guid: unable to get hostname and random bytes")
	panic(errors.New("ess - guid: unable to get hostname and random bytes"))
}
