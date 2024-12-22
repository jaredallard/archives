# archives

[![Latest Version](https://img.shields.io/github/v/release/jaredallard/archives?style=for-the-badge)](https://github.com/jaredallard/archives/releases)
[![License](https://img.shields.io/github/license/jaredallard/archives?style=for-the-badge)](https://github.com/jaredallard/archives/blob/main/LICENSE)
[![Github Workflow Status](https://img.shields.io/github/actions/workflow/status/jaredallard/archives/tests.yaml?style=for-the-badge)](https://github.com/jaredallard/archives/actions/workflows/tests.yaml)
[![Codecov](https://img.shields.io/codecov/c/github/jaredallard/archives?style=for-the-badge)](https://app.codecov.io/gh/jaredallard/archives)

Go library for extracting archives (tar, zip, etc.)

## Supported Archive Types

- `tar`
  - `tar.xz` - xz
  - `tar.bz2` - bzip2
  - `tar.gz` - gzip
  - `tar.zst` - zstd
- `zip`

## Usage

For complete documentation and examples, see our [pkg.go.dev]
documentation.

### Extracting Archives

Extracting archives is simple with this package. Simply provide an
[io.Reader] and the extension (which you can use [archives.Ext] to get!)
and you're good to go.

```go
resp, err := http.Get("https://getsamplefiles.com/download/zip/sample-1.zip")
if err != nil {}
defer resp.Body.Close()

err := archives.Extract(resp.Body, "dir-to-extract-into", &archives.ExtractOptions{
  Extension: archives.Ext("sample-1.zip"),
})
if err != nil {}

// Do something with the files in dir-to-extract-into
```

### Picking a File out of an Archive

Sometimes you want to only grab a single file out of an archive.
[archives.Pick] is helpful here.

```go
resp, err := http.Get("https://getsamplefiles.com/download/zip/sample-1.zip")
if err != nil {}
defer resp.Body.Close()

a, err := archives.Open(resp.Body, &archives.OpenOptions{
  Extension: archives.Ext("sample-1.zip"),
})
if err != nil {}

// Pick a single file out of the zip archive
r, err := archives.Pick(a, archives.PickFilterByName("sample-1/sample-1.webp"))
if err != nil {}

// Do something with the returned [io.Reader] (r).
```

### Working with Archives

You can also work with an archive directly, much like [tar.Reader].

```go
resp, err := http.Get("https://getsamplefiles.com/download/tar/sample-1.tar")
if err != nil {}
defer resp.Body.Close()

a, err := archives.Open(resp.Body, &archives.OpenOptions{
  Extension: archives.Ext("sample-1.tar"),
})
if err != nil {}

h, err := a.Next()
if err != nil {}

// Read the current file using `a` ([Archive]) which is an io.Reader,
// or only handle the `h` ([Header]). Your choice!

// Close out the archiver parser(s).
a.Close()
```

### CGO

CGO is used for extracting `xz` archives by default. If you wish to not
use CGO, simply set `CGO_ENABLED` to `0`. This library will
automatically use a pure-Go implementation instead.

## License

LGPL-3.0

[archives.Ext]: https://pkg.go.dev/github.com/jaredallard/archives#Ext
[archives.Pick]: https://pkg.go.dev/github.com/jaredallard/archives#Pick
[io.Reader]: https://pkg.go.dev/io#Reader
[pkg.go.dev]: https://pkg.go.dev/github.com/jaredallard/archives
[tar.Reader]: https://pkg.go.dev/archive/tar#Reader
