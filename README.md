# Security - aah framework
[![Build Status](https://travis-ci.org/go-aah/security.svg?branch=master)](https://travis-ci.org/go-aah/security) [![codecov](https://codecov.io/gh/go-aah/security/branch/master/graph/badge.svg)](https://codecov.io/gh/go-aah/security/branch/master) [![Go Report Card](https://goreportcard.com/badge/aahframework.org/security.v0)](https://goreportcard.com/report/aahframework.org/security.v0) [![Version](https://img.shields.io/badge/version-0.3.1-blue.svg)](https://github.com/go-aah/security/releases/latest) [![GoDoc](https://godoc.org/aahframework.org/security.v0?status.svg)](https://godoc.org/aahframework.org/security.v0)  [![License](https://img.shields.io/github/license/go-aah/security.svg)](LICENSE)

***v0.3.1 [released](https://github.com/go-aah/security/releases/latest) and tagged on Apr 15, 2017***

Security library houses all the application security implementation (Session, Basic Auth, Token Auth, CORS, CSRF, Security Headers, etc.) by aah framework.

### Session - HTTP State Management

Features:
  * Extensible session store interface (just key-value pair)
  * Signed session data
  * Encrypted session data

*`security` developed for aah framework. However, it's an independent library, can be used separately with any `Go` language project. Feel free to use it.*

# Installation
#### Stable Version - Production Ready
```sh
# install the library
go get -u aahframework.org/security.v0
```

See official page [TODO]
