# archives

Go library for extracting and creating archives (tar, zip, etc.)

## Supported Archive Types

- `tar`
  - `tar.xz` - xz
  - `tar.bz2` - bzip2 (extraction only)
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

Symlinks and hard links are restored, and entries are never written
outside of the destination directory. Set `Sync: true` in
[archives.ExtractOptions] to fsync every extracted file and directory
before `Extract` returns (e.g., when restoring a disk).

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

### Creating Archives

[archives.Create] packs the contents of a directory into an archive.
Symlinks are stored as symlinks (never followed) with their targets
exactly as they are on disk.

```go
f, err := os.Create("dir.tar.gz")
if err != nil {}
defer f.Close()

err := archives.Create(f, "dir-to-archive", archives.CreateOptions{
  Extension: archives.Ext("dir.tar.gz"),
})
if err != nil {}
```

Permissions are always stored. Set `PreserveOwnership: true` to also
store each entry's user and group ID (tar only, on unix). Set
`CompressionLevel` to one of `CompressionFastest`, `CompressionBetter`
or `CompressionBest` to trade speed for size. This is also available
in [archives.WriterOptions].

```go
err := archives.Create(f, "dir-to-archive", archives.CreateOptions{
  Extension:         ".tar.zst",
  CompressionLevel:  archives.CompressionBest,
  PreserveOwnership: true,
})
```

For full control over the entries, use [archives.NewWriter], which
works much like [tar.Writer].

```go
w, err := archives.NewWriter(f, archives.WriterOptions{
  Extension: ".zip",
})
if err != nil {}

body := []byte("hello world")
err = w.WriteHeader(&archives.Header{
  Name: "file.txt",
  Type: archives.HeaderFile,
  Size: int64(len(body)),
  Mode: 0o644,
})
if err != nil {}

_, err = w.Write(body)
if err != nil {}

// Finish the archive. This does not close f.
err = w.Close()
```

Creating bzip2 compressed archives is not supported.

### CGO

CGO is used for extracting and creating `xz` archives by default. If you wish to not
use CGO, simply set `CGO_ENABLED` to `0`. This library will
automatically use a pure-Go implementation instead.

## License

MPL-2.0

[archives.Create]: https://pkg.go.dev/go.rgst.io/jaredallard/archives/v2#Create
[archives.NewWriter]: https://pkg.go.dev/go.rgst.io/jaredallard/archives/v2#NewWriter
[archives.Ext]: https://pkg.go.dev/go.rgst.io/jaredallard/archives/v2#Ext
[archives.ExtractOptions]: https://pkg.go.dev/go.rgst.io/jaredallard/archives/v2#ExtractOptions
[archives.WriterOptions]: https://pkg.go.dev/go.rgst.io/jaredallard/archives/v2#WriterOptions
[archives.Pick]: https://pkg.go.dev/go.rgst.io/jaredallard/archives/v2#Pick
[io.Reader]: https://pkg.go.dev/io#Reader
[pkg.go.dev]: https://pkg.go.dev/go.rgst.io/jaredallard/archives/v2
[tar.Reader]: https://pkg.go.dev/archive/tar#Reader
[tar.Writer]: https://pkg.go.dev/archive/tar#Writer
