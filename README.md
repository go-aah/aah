# essentials - aah framework

[![Build Status](https://travis-ci.org/go-aah/essentials.svg?branch=master)](https://travis-ci.org/go-aah/essentials)  [![Go Report Card](https://goreportcard.com/badge/github.com/go-aah/essentials)](https://goreportcard.com/report/github.com/go-aah/essentials) [![GoDoc](https://godoc.org/github.com/go-aah/essentials?status.svg)](https://godoc.org/github.com/go-aah/essentials)  [![License](https://img.shields.io/badge/license-MIT-blue.svg)](LICENSE)

essentials contain simple & useful utils for go lang. aah framework utilizes essentials (aka `ess`) library across.

## Quick Start

```
go get -u github.com/go-aah/essentials
```

## Sample

You refer package as `ess` as a short form.

```go
fmt.Println("String is empty:", ess.StrIsEmpty("  Welcome  "))

// Output
String is empty: false


fmt.Println("String is empty:", ess.StrIsEmpty("    "))

// Output
String is empty: true
```

## Author
Jeevanandam M. - jeeva@myjeeva.com

## Contributors
Have a look on [Contributors](https://github.com/go-aah/essentials/graphs/contributors) page.

## License
Released under MIT license, refer [LICENSE](LICENSE) file.
