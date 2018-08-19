// Copyright (c) Jeevanandam M (https://github.com/jeevatkm)
// aahframework.org/aruntime source code and usage is governed by a MIT style
// license that can be found in the LICENSE file.

package aruntime

import (
	"bytes"
	"testing"

	"aahframework.org/config"
	"github.com/stretchr/testify/assert"
)

func TestStacktrace(t *testing.T) {
	strace := Stacktrace{
		Raw:          getStacktrace(),
		Recover:      "this is test case",
		StripSrcBase: true,
	}

	buf := &bytes.Buffer{}
	strace.Print(buf)
	t.Log(buf.String())

	assert.Equal(t, 5, len(strace.GoRoutines))
	assert.Equal(t, "goroutine 5 [running]:", strace.GoRoutines[0].Header)
	assert.Equal(t, "goroutine 1 [running]:", strace.GoRoutines[1].Header)
	assert.Equal(t, "goroutine 1 [IO wait]:", strace.GoRoutines[2].Header)
	assert.Equal(t, "goroutine 17 [syscall, locked to thread]:", strace.GoRoutines[3].Header)

	assert.Equal(t, 10, len(strace.GoRoutines[0].Packages))
	assert.Equal(t, 10, len(strace.GoRoutines[0].Functions))

	assert.Equal(t, 8, len(strace.GoRoutines[1].Packages))
	assert.Equal(t, 8, len(strace.GoRoutines[1].Functions))

	assert.Equal(t, 11, len(strace.GoRoutines[2].Packages))
	assert.Equal(t, 11, len(strace.GoRoutines[2].Functions))

	assert.Equal(t, 1, len(strace.GoRoutines[3].Packages))
	assert.Equal(t, 1, len(strace.GoRoutines[3].Functions))
}

func TestSingleStacktrace(t *testing.T) {
	strace := Stacktrace{
		Raw:     getSingleStacktrace(),
		Recover: "this is single test case",
	}

	buf := &bytes.Buffer{}
	strace.Print(buf)
	t.Log(buf.String())

	assert.Equal(t, 1, len(strace.GoRoutines))
	assert.Equal(t, "goroutine 18 [running]:", strace.GoRoutines[0].Header)
}

func TestNewStacktrace(t *testing.T) {
	cfg, _ := config.ParseString(``)
	strace := NewStacktrace("testing", cfg)

	assert.NotNil(t, strace)
}

func TestFallbackBufSizeStacktrace(t *testing.T) {
	cfg, _ := config.ParseString(`
    runtime {
			debug {
				stack_buffer_size = "4"
				all_goroutines = true
			}
    }
  `)
	strace := NewStacktrace("testing", cfg)

	assert.NotNil(t, strace)
}

func TestNewFullStacktrace(t *testing.T) {
	cfg, _ := config.ParseString(`
    runtime {
			debug {
				all_goroutines = true
			}
    }
  `)

	strace := NewStacktrace("testing", cfg)

	assert.NotNil(t, strace)
}

func TestPrintNilWriter(t *testing.T) {
	strace := Stacktrace{}
	strace.Print(nil)
}

func getStacktrace() string {
	return `goroutine 5 [running]:
runtime/debug.Stack(0xc420041bc0, 0x2, 0x2)
	/usr/local/go/src/runtime/debug/stack.go:24 +0x79
aahframework.org/aah.Init.func1()
	/Users/jeeva/pscm/go-home/src/aahframework.org/aah/aah.go:155 +0x15c
panic(0x33f780, 0xc420011cc0)
	/usr/local/go/src/runtime/panic.go:458 +0x243
aahframework.org/config.(*Config).String(0xc420103160, 0x39f26f, 0x11, 0xc4200fe5b8, 0x3, 0xc420016a01)
	/Users/jeeva/pscm/go-home/src/aahframework.org/config/config.go:98 +0x90
aahframework.org/config.(*Config).StringDefault(0xc420103160, 0x39f26f, 0x11, 0x39a58b, 0x3, 0xc4200fe5b8, 0x3)
	/Users/jeeva/pscm/go-home/src/aahframework.org/config/config.go:107 +0x3f
aahframework.org/aah.initAppVariables(0x0, 0x0)
	/Users/jeeva/pscm/go-home/src/aahframework.org/aah/aah.go:325 +0x143
aahframework.org/aah.Init(0x3a1f18, 0x18)
	/Users/jeeva/pscm/go-home/src/aahframework.org/aah/aah.go:164 +0xc1
aahframework.org/aah.TestStart(0xc420090180)
	/Users/jeeva/pscm/go-home/src/aahframework.org/aah/aah_test.go:48 +0x36
testing.tRunner(0xc420090180, 0x3c9d68)
	/usr/local/go/src/testing/testing.go:610 +0x81
created by testing.(*T).Run
	/usr/local/go/src/testing/testing.go:646 +0x2ec

goroutine 1 [running]:
runtime/debug.Stack(0xc420051b90, 0x2, 0x2)
	/usr/local/go/src/runtime/debug/stack.go:24 +0x79
aahframework.org/aah.Init.func1()
	/Users/jeeva/pscm/go-home/src/aahframework.org/aah/aah.go:155 +0x15c
panic(0x31cf00, 0xc4200117c0)
	/usr/local/go/src/runtime/panic.go:458 +0x243
aahframework.org/config.(*Config).String(0xc4200f9100, 0x377f69, 0x11, 0xc4200ee338, 0x3, 0xc420016a01)
	/Users/jeeva/pscm/go-home/src/aahframework.org/config/config.go:98 +0x90
aahframework.org/config.(*Config).StringDefault(0xc4200f9100, 0x377f69, 0x11, 0x373727, 0x3, 0xc4200ee338, 0x3)
	/Users/jeeva/pscm/go-home/src/aahframework.org/config/config.go:107 +0x3f
aahframework.org/aah.initAppVariables(0x0, 0x0)
	/Users/jeeva/pscm/go-home/src/aahframework.org/aah/aah.go:325 +0x143
aahframework.org/aah.Init(0x37adff, 0x19)
	/Users/jeeva/pscm/go-home/src/aahframework.org/aah/aah.go:164 +0xc1
main.main()
	/Users/jeeva/pscm/go-home/src/aahtest.com/user/niceapp/main.go:18 +0x36

goroutine 1 [IO wait]:
net.runtime_pollWait(0x1849060, 0x72, 0x0)
	/usr/local/go/src/runtime/netpoll.go:160 +0x59
net.(*pollDesc).wait(0xc42014ab50, 0x72, 0xc420051b50, 0xc42000c0b8)
	/usr/local/go/src/net/fd_poll_runtime.go:73 +0x38
net.(*pollDesc).waitRead(0xc42014ab50, 0x4e5f40, 0xc42000c0b8)
	/usr/local/go/src/net/fd_poll_runtime.go:78 +0x34
net.(*netFD).accept(0xc42014aaf0, 0x0, 0x4e4900, 0xc42017f300)
	/usr/local/go/src/net/fd_unix.go:419 +0x238
net.(*TCPListener).accept(0xc4200301d8, 0x29e8d60800, 0x0, 0x0)
	/usr/local/go/src/net/tcpsock_posix.go:132 +0x2e
net.(*TCPListener).AcceptTCP(0xc4200301d8, 0xc420051c78, 0xc420051c80, 0xc420051c70)
	/usr/local/go/src/net/tcpsock.go:209 +0x49
net/http.tcpKeepAliveListener.Accept(0xc4200301d8, 0x3a4ed8, 0xc420080e00, 0x4e97c0, 0xc420174bd0)
	/usr/local/go/src/net/http/server.go:2608 +0x2f
net/http.(*Server).Serve(0xc420080d00, 0x4e9300, 0xc4200301d8, 0x0, 0x0)
	/usr/local/go/src/net/http/server.go:2273 +0x1ce
net/http.(*Server).ListenAndServe(0xc420080d00, 0x20, 0xc42017de10)
	/usr/local/go/src/net/http/server.go:2219 +0xb4
aahframework.org/aah.Start()
	/Users/jeeva/pscm/go-home/src/aahframework.org/aah/aah.go:235 +0xb93
main.main()
	/Users/jeeva/pscm/go-home/src/aahtest.com/user/niceapp/main.go:150 +0x3b

goroutine 17 [syscall, locked to thread]:
runtime.goexit()
	/usr/local/go/src/runtime/asm_amd64.s:2086 +0x1

goroutine 18 [running]:
testing.tRunner.func1(0xc0420ea0f0)
        c:/Go/src/testing/testing.go:742 +0x2a4
panic(0x5dd180, 0x6538f0)
        c:/Go/src/runtime/panic.go:505 +0x237
aahframework.org/aruntime%2ev0.(*Stacktrace).Parse(0xc04203df28)
        C:/Users/jeeva/go/src/aahframework.org/aruntime.v0/stacktrace.go:146 +0xb37
aahframework.org/aruntime%2ev0.(*Stacktrace).Print(0xc04204bf28, 0x654960, 0xc0420cefc0)
        C:/Users/jeeva/go/src/aahframework.org/aruntime.v0/stacktrace.go:156 +0x446
aahframework.org/aruntime%2ev0.TestMeStacktrace(0xc0420ea0f0)
        C:/Users/jeeva/go/src/aahframework.org/aruntime.v0/stacktrace_test.go:25 +0xc9
testing.tRunner(0xc0420ea0f0, 0x63aa40)
        c:/Go/src/testing/testing.go:777 +0xd7
created by testing.(*T).Run
        c:/Go/src/testing/testing.go:824 +0x2e7
	`
}

func getSingleStacktrace() string {
	return `goroutine 18 [running]:
testing.tRunner.func1(0xc0420ea0f0)
        c:/Go/src/testing/testing.go:742 +0x2a4
panic(0x5dd180, 0x6538f0)
        c:/Go/src/runtime/panic.go:505 +0x237
aahframework.org/aruntime%2ev0.(*Stacktrace).Parse(0xc04203df28)
        C:/Users/jeeva/go/src/aahframework.org/aruntime.v0/stacktrace.go:146 +0xb37
aahframework.org/aruntime%2ev0.(*Stacktrace).Print(0xc04204bf28, 0x654960, 0xc0420cefc0)
        C:/Users/jeeva/go/src/aahframework.org/aruntime.v0/stacktrace.go:156 +0x446
aahframework.org/aruntime%2ev0.TestMeStacktrace(0xc0420ea0f0)
        C:/Users/jeeva/go/src/aahframework.org/aruntime.v0/stacktrace_test.go:25 +0xc9
testing.tRunner(0xc0420ea0f0, 0x63aa40)
        c:/Go/src/testing/testing.go:777 +0xd7
created by testing.(*T).Run
        c:/Go/src/testing/testing.go:824 +0x2e7`
}
