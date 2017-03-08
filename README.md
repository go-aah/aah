# router - aah framework
[![Build Status](https://travis-ci.org/go-aah/router.svg?branch=master)](https://travis-ci.org/go-aah/router) [![codecov](https://codecov.io/gh/go-aah/router/branch/master/graph/badge.svg)](https://codecov.io/gh/go-aah/router/branch/master) [![Go Report Card](https://goreportcard.com/badge/aahframework.org/router.v0)](https://goreportcard.com/report/aahframework.org/router.v0) [![Version](https://img.shields.io/badge/version-0.1-blue.svg)](https://github.com/go-aah/router/releases/latest) [![GoDoc](https://godoc.org/aahframework.org/router.v0?status.svg)](https://godoc.org/aahframework.org/router.v0)  [![License](https://img.shields.io/github/license/go-aah/router.svg)](LICENSE)

***v0.1 [released](https://github.com/go-aah/router/releases/latest) and tagged on Mar 07, 2017***

HTTP Router it supports domain and sub-domains routing. It built around [httprouter](https://github.com/julienschmidt/httprouter) library.

The router is optimized for high performance and a small memory footprint. It scales well even with very long paths and a large number of routes. A compressing dynamic trie (radix tree) structure is used for efficient matching.

*`router` developed for aah framework. However, it's an independent library, can be used separately with any `Go` language project. Feel free to use it.*

# Installation
#### Stable Version - Production Ready
```sh
# install the library
go get -u aahframework.org/router.v0
```

#### Development Version - Edge
```sh
# install the development version
go get -u aahframework.org/router.v0-unstable
```

See official page [TODO]
