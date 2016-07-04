# log - aah framework
[![Build Status](https://travis-ci.org/go-aah/log.svg?branch=master)](https://travis-ci.org/go-aah/log)  [![Go Report Card](https://goreportcard.com/badge/github.com/go-aah/log)](https://goreportcard.com/report/github.com/go-aah/log) [![GoDoc](https://godoc.org/github.com/go-aah/log?status.svg)](https://godoc.org/github.com/go-aah/log)  [![License](https://img.shields.io/badge/license-MIT-blue.svg)](LICENSE)

Package `log` is used across **aah framework**. It's independent library, can be used separately with any `Go` language project. Feel free to use it.

`log` supports following receivers:
* Console
  * Log level is colored (for windows it's plain)
* File
  * Log Rotation
    * Daily
    * Size
    * No. of lines
  * No. of Backups (upcoming)
* Remote - TCP & UDP (upcoming)

## Quick Start

```
go get -u github.com/go-aah/log
```

## Sample
```go
log.Info("Welcome ", "to ", "aah ", "logger")
log.Infof("%v, %v, & %v", "simple", "flexible", "powerful logger")

// Output:
2016-07-03 19:22:11.504 INFO  - Welcome to aah logger
2016-07-03 19:22:11.504 INFO  - simple, flexible, & powerful logger
```

## Author
Jeevanandam M. - jeeva@myjeeva.com

## Contributors
Have a look on [Contributors](https://github.com/go-aah/log/graphs/contributors) page.

## License
Released under MIT license, refer [LICENSE](LICENSE) file.
