# archives

[![Latest Version](https://img.shields.io/github/v/release/jaredallard/archives?style=for-the-badge)](https://github.com/jaredallard/archives/releases)
[![License](https://img.shields.io/github/license/jaredallard/archives?style=for-the-badge)](https://github.com/jaredallard/archives/blob/main/LICENSE)
[![Github Workflow Status](https://img.shields.io/github/actions/workflow/status/jaredallard/archives/tests.yaml?style=for-the-badge)](https://github.com/jaredallard/archives/actions/workflows/tests.yaml)
[![Codecov](https://img.shields.io/codecov/c/github/jaredallard/archives?style=for-the-badge)](https://app.codecov.io/gh/jaredallard/archives)

Go library for extracting archives (tar, zip, etc.)

## Supported Archive Types

* `tar`
  * `tar.xz`
  * `tar.bz2`
  * `tar.gz`

## Usage

See our [Go docs](https://pkg.go.dev/github.com/jaredallard/archives).

### cgo

cgo is used for extracting `xz` archives by default. If you wish to not
use cgo, simply set `CGO_ENABLED` to `0`. This library will
automatically use a pure-Go implementation instead.

## License

LGPL-3.0
