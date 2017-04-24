# Security - aah framework
[![Build Status](https://travis-ci.org/go-aah/security.svg?branch=master)](https://travis-ci.org/go-aah/security) [![codecov](https://codecov.io/gh/go-aah/security/branch/master/graph/badge.svg)](https://codecov.io/gh/go-aah/security/branch/master) [![Go Report Card](https://goreportcard.com/badge/aahframework.org/security.v0)](https://goreportcard.com/report/aahframework.org/security.v0) [![Version](https://img.shields.io/badge/version-0.4-blue.svg)](https://github.com/go-aah/security/releases/latest) [![GoDoc](https://godoc.org/aahframework.org/security.v0?status.svg)](https://godoc.org/aahframework.org/security.v0)  [![License](https://img.shields.io/github/license/go-aah/security.svg)](LICENSE)

***v0.4 [released](https://github.com/go-aah/security/releases/latest) and tagged on Apr 23, 2017***

Security library houses all the application security implementation (Session, Basic Auth, Token Auth, CORS, CSRF, Security Headers, etc.) by aah framework.

### Session - HTTP State Management

Features:
  * Extensible session store interface (just key-value pair)
  * HMAC Signed session data
  * AES Encrypted session data

*`security` developed for aah framework. However, it's an independent library, can be used separately with any `Go` language project. Feel free to use it.*

# Installation
#### Stable Version - Production Ready
```bash
# install the library
go get -u aahframework.org/security.v0
```

Visit official website https://aahframework.org to learn more.
