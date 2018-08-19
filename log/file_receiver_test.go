// Copyright (c) Jeevanandam M (https://github.com/jeevatkm)
// go-aah/log source code and usage is governed by a MIT style
// license that can be found in the LICENSE file.

package log

import (
	"io/ioutil"
	"testing"

	"aahframework.org/config"
	"github.com/stretchr/testify/assert"
)

func TestFileLoggerRotation(t *testing.T) {
	cleaupFiles("*.log")
	fileConfigStr1 := `
  log {
    receiver = "file"
    level = "debug"
    pattern = "%utctime:2006-01-02 15:04:05.000 %level:-5 %longfile %line %custom:- %message"
    file = "daily-aah-filename.log"
    rotate {
      policy = "daily"
    }
  }
  `

	testFileLogger(t, fileConfigStr1, 5000)
	cleaupFiles("*.log")

	fileConfigStr2 := `
  log {
    receiver = "file"
    level = "debug"
    pattern = "%utctime:2006-01-02 15:04:05.000 %level:-5 %longfile %line %custom:- %message"
    file = "daily-aah-filename.log"
    rotate {
      policy = "lines"
      lines = 10000
    }
  }
  `
	testFileLogger(t, fileConfigStr2, 5000)
	cleaupFiles("*.log")

	fileConfigStr3 := `
  log {
    receiver = "file"
    level = "debug"
    pattern = "%time:2006-01-02 15:04:05.000 %level:-5 %shortfile %line %custom:- %message"
    file = "daily-aah-filename.log"
    rotate {
      policy = "size"
      size = "500kb"
    }
  }
  `
	testFileLogger(t, fileConfigStr3, 5000)
	cleaupFiles("*.log")

	fileConfigStr4 := `
  log {
    receiver = "file"
    level = "debug"
    file = "daily-aah-filename.log"
    rotate {
      mode = ""
    }
  }
  `

	testFileLogger(t, fileConfigStr4, 50)
	cleaupFiles("*.log")

	// JSON
	fileConfigStrJSON := `
  log {
    receiver = "file"
    level = "debug"
    format = "json"
    file = "daily-aah-filename.log"
  }
  `
	testFileLogger(t, fileConfigStrJSON, 1000)

	cleaupFiles("*.log")
}

func TestFileLoggerFileOpenError(t *testing.T) {
	fileConfigStr := `
  log {
    receiver = "file"
		file = ""
  }
  `
	cfg, _ := config.ParseString(fileConfigStr)
	logger, err := New(cfg)
	assert.Nil(t, logger)
	assert.Equal(t, "open : no such file or directory", err.Error())
}

func TestFileLoggerUnsupportedFormat(t *testing.T) {
	defer cleaupFiles("*.log")
	configStr := `
  log {
    receiver = "file"
    file = "daily-aah-filename.log"
    format = "xml"
  }
  `
	cfg, _ := config.ParseString(configStr)
	logger, err := New(cfg)
	assert.Equal(t, "log: unsupported format 'xml'", err.Error())
	assert.Nil(t, logger)

}

func TestFileLoggerUnknownFormatFlag(t *testing.T) {
	defer cleaupFiles("*.log")
	configStr := `
  log {
    receiver = "file"
    file = "daily-aah-filename.log"
    pattern = "%time:2006-01-02 15:04:05.000 %level:-5 %myfile %line %custom:- %message"
  }
  `
	cfg, _ := config.ParseString(configStr)
	logger, err := New(cfg)
	assert.Nil(t, logger)
	assert.Equal(t, "fmtflag: unknown flag 'myfile'", err.Error())
}

func TestFileLoggerIncorrectSizeValue(t *testing.T) {
	defer cleaupFiles("*.log")
	configStr := `
  log {
    receiver = "file"
    level = "debug"
    pattern = "%time:2006-01-02 15:04:05.000 %level:-5 %shortfile %line %custom:- %message"
    file = "daily-aah-filename.log"
    rotate {
      policy = "size"
      size = "500kbs"
    }
  }
	`
	cfg, _ := config.ParseString(configStr)
	_, err := New(cfg)
	assert.Equal(t, "format: invalid input '500kbs'", err.Error())
}

func testFileLogger(t *testing.T, cfgStr string, loop int) {
	cfg, _ := config.ParseString(cfgStr)
	logger, err := New(cfg)
	assert.Nil(t, err, "unexpected error")

	for i := 0; i < loop; i++ {
		logger.Trace("I shoudn't see this msg, because standard logger level is DEBUG")
		logger.Tracef("I shoudn't see this msg, because standard logger level is DEBUG: %v", 4)

		logger.Debug("I would like to see this message, debug is useful for dev")
		logger.Debugf("I would like to see this message, debug is useful for %v", "dev")

		logger.Info("Yes, I would love to see")
		logger.Infof("Yes, I would love to %v", "see")

		logger.Warn("Yes, yes it's an warning")
		logger.Warnf("Yes, yes it's an %v", "warning")

		logger.Error("Yes, yes, yes - finally an error")
		logger.Errorf("Yes, yes, yes - %v", "finally an error")
	}

	assert.NotNil(t, logger.ToGoLogger())
	logger.SetWriter(ioutil.Discard)
}
