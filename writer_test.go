package archives_test

import (
	"bytes"
	"errors"
	"io"
	"io/fs"
	"math/rand/v2"
	"strings"
	"testing"
	"time"

	"go.rgst.io/jaredallard/archives/v2"
	"gotest.tools/v3/assert"
	is "gotest.tools/v3/assert/cmp"
)

// writableExtensions are all extensions that archives can be written
// as.
var writableExtensions = []string{".tar", ".tgz", ".tar.gz", ".txz", ".tar.xz", ".tar.zst", ".zip"}

// readEntry is a header and the contents read from an archive.
type readEntry struct {
	h    *archives.Header
	body string
}

// readAll reads every entry from the archive in r, keyed by name.
func readAll(t *testing.T, r io.Reader, ext string) map[string]readEntry {
	t.Helper()

	a, err := archives.Open(r, archives.OpenOptions{Extension: ext})
	assert.NilError(t, err)
	defer a.Close()

	entries := map[string]readEntry{}
	for {
		h, err := a.Next()
		if errors.Is(err, io.EOF) {
			break
		}
		assert.NilError(t, err)

		e := readEntry{h: h}
		if h.Type == archives.HeaderFile {
			b, err := io.ReadAll(a)
			assert.NilError(t, err)
			e.body = string(b)
		}
		entries[strings.TrimSuffix(h.Name, "/")] = e
	}
	return entries
}

func TestNewWriterRoundTrip(t *testing.T) {
	modTime := time.Date(2024, 1, 2, 3, 4, 5, 0, time.UTC)

	for _, ext := range writableExtensions {
		t.Run(ext, func(t *testing.T) {
			buf := new(bytes.Buffer)
			w, err := archives.NewWriter(buf, archives.WriterOptions{Extension: ext})
			assert.NilError(t, err)

			writeFile := func(name, body string, mode fs.FileMode) {
				assert.NilError(t, w.WriteHeader(&archives.Header{
					Name: name, Type: archives.HeaderFile, Size: int64(len(body)),
					Mode: mode, ModTime: modTime,
				}))
				_, err := io.WriteString(w, body)
				assert.NilError(t, err)
			}

			assert.NilError(t, w.WriteHeader(&archives.Header{
				Name: "dir", Type: archives.HeaderDir, Mode: 0o750, ModTime: modTime,
			}))
			writeFile("dir/file.txt", "hello world", 0o644)
			writeFile("suid", "x", 0o755|fs.ModeSetuid)
			assert.NilError(t, w.WriteHeader(&archives.Header{
				Name: "link", Type: archives.HeaderSymlink, Linkname: "dir/file.txt",
				Mode: 0o777, ModTime: modTime,
			}))
			assert.NilError(t, w.Close())

			entries := readAll(t, buf, ext)
			assert.Equal(t, len(entries), 4)

			dir := entries["dir"]
			assert.Equal(t, dir.h.Type, archives.HeaderDir)
			assert.Equal(t, dir.h.Mode.Perm(), fs.FileMode(0o750))

			file := entries["dir/file.txt"]
			assert.Equal(t, file.h.Type, archives.HeaderFile)
			assert.Equal(t, file.h.Mode, fs.FileMode(0o644))
			assert.Equal(t, file.body, "hello world")
			assert.Assert(t, file.h.ModTime.Equal(modTime), "got mtime %v", file.h.ModTime)

			suid := entries["suid"]
			assert.Equal(t, suid.h.Mode, 0o755|fs.ModeSetuid)

			link := entries["link"]
			assert.Equal(t, link.h.Type, archives.HeaderSymlink)
			assert.Equal(t, link.h.Linkname, "dir/file.txt")
		})
	}
}

func TestNewWriterHardlink(t *testing.T) {
	buf := new(bytes.Buffer)
	w, err := archives.NewWriter(buf, archives.WriterOptions{Extension: ".tar"})
	assert.NilError(t, err)
	assert.NilError(t, w.WriteHeader(&archives.Header{Name: "a", Type: archives.HeaderFile}))
	assert.NilError(t, w.WriteHeader(&archives.Header{Name: "b", Type: archives.HeaderHardlink, Linkname: "a"}))
	assert.NilError(t, w.Close())

	entries := readAll(t, buf, ".tar")
	assert.Equal(t, entries["b"].h.Type, archives.HeaderHardlink)
	assert.Equal(t, entries["b"].h.Linkname, "a")
}

func TestNewWriterErrors(t *testing.T) {
	t.Run("bzip2 is unsupported", func(t *testing.T) {
		for _, ext := range []string{".tar.bz2", ".tbz2"} {
			_, err := archives.NewWriter(new(bytes.Buffer), archives.WriterOptions{Extension: ext})
			assert.ErrorContains(t, err, "bzip2")
		}
	})

	t.Run("unknown extension", func(t *testing.T) {
		_, err := archives.NewWriter(new(bytes.Buffer), archives.WriterOptions{Extension: ".rar"})
		assert.ErrorContains(t, err, "unsupported archive extension")
	})

	t.Run("zip rejects hard links", func(t *testing.T) {
		w, err := archives.NewWriter(new(bytes.Buffer), archives.WriterOptions{Extension: ".zip"})
		assert.NilError(t, err)
		err = w.WriteHeader(&archives.Header{Name: "b", Type: archives.HeaderHardlink, Linkname: "a"})
		assert.ErrorContains(t, err, "hard links")
	})

	t.Run("zip rejects writes to non-files", func(t *testing.T) {
		w, err := archives.NewWriter(new(bytes.Buffer), archives.WriterOptions{Extension: ".zip"})
		assert.NilError(t, err)
		assert.NilError(t, w.WriteHeader(&archives.Header{Name: "d", Type: archives.HeaderDir}))
		_, err = w.Write([]byte("x"))
		assert.ErrorContains(t, err, "not a file")
	})

	t.Run("tar rejects unsupported types", func(t *testing.T) {
		w, err := archives.NewWriter(new(bytes.Buffer), archives.WriterOptions{Extension: ".tar"})
		assert.NilError(t, err)
		err = w.WriteHeader(&archives.Header{Name: "x", Type: archives.HeaderUnsupported})
		assert.ErrorContains(t, err, "unsupported header type")
	})

	t.Run("tar rejects size mismatches", func(t *testing.T) {
		w, err := archives.NewWriter(new(bytes.Buffer), archives.WriterOptions{Extension: ".tar"})
		assert.NilError(t, err)
		assert.NilError(t, w.WriteHeader(&archives.Header{Name: "a", Type: archives.HeaderFile, Size: 2}))
		_, err = w.Write([]byte("abc"))
		assert.Assert(t, err != nil)

		w, err = archives.NewWriter(new(bytes.Buffer), archives.WriterOptions{Extension: ".tar"})
		assert.NilError(t, err)
		assert.NilError(t, w.WriteHeader(&archives.Header{Name: "a", Type: archives.HeaderFile, Size: 2}))
		_, err = w.Write([]byte("a"))
		assert.NilError(t, err)
		assert.Assert(t, is.ErrorContains(w.Close(), ""))
	})
}

// compressionLevels are all supported compression levels.
var compressionLevels = []archives.CompressionLevel{
	archives.CompressionDefault,
	archives.CompressionFastest,
	archives.CompressionBetter,
	archives.CompressionBest,
}

// compressibleData returns n bytes of deterministic, compressible
// text.
func compressibleData(n int) []byte {
	words := []string{"lorem", "ipsum", "dolor", "sit", "amet", "consectetur", "adipiscing", "elit"}
	rng := rand.New(rand.NewPCG(1, 2)) //nolint:gosec // Why: Deterministic test data.

	var b bytes.Buffer
	for b.Len() < n {
		b.WriteString(words[rng.IntN(len(words))])
		b.WriteByte(' ')
	}
	return b.Bytes()[:n]
}

// writeWithLevel writes body as a single file to an archive of ext
// compressed with level.
func writeWithLevel(t *testing.T, ext string, level archives.CompressionLevel, body []byte) []byte {
	t.Helper()

	buf := new(bytes.Buffer)
	w, err := archives.NewWriter(buf, archives.WriterOptions{Extension: ext, CompressionLevel: level})
	assert.NilError(t, err)
	assert.NilError(t, w.WriteHeader(&archives.Header{
		Name: "file.txt", Type: archives.HeaderFile, Size: int64(len(body)), Mode: 0o644,
	}))
	_, err = w.Write(body)
	assert.NilError(t, err)
	assert.NilError(t, w.Close())
	return buf.Bytes()
}

func TestNewWriterCompressionLevels(t *testing.T) {
	body := compressibleData(1 << 20)

	for _, ext := range writableExtensions {
		t.Run(ext, func(t *testing.T) {
			sizes := map[archives.CompressionLevel]int{}
			for _, level := range compressionLevels {
				b := writeWithLevel(t, ext, level, body)
				sizes[level] = len(b)

				entries := readAll(t, bytes.NewReader(b), ext)
				assert.Equal(t, entries["file.txt"].body, string(body), "level %d", level)
			}

			// The pure-Go xz encoder only varies its dictionary size, which
			// doesn't affect small inputs, so only check other formats.
			switch ext {
			case ".tar":
				assert.Equal(t, sizes[archives.CompressionBest], sizes[archives.CompressionFastest])
			case ".tgz", ".tar.gz", ".tar.zst", ".zip":
				assert.Assert(t, sizes[archives.CompressionBest] < sizes[archives.CompressionFastest],
					"best (%d) should be smaller than fastest (%d)",
					sizes[archives.CompressionBest], sizes[archives.CompressionFastest])
			}
		})
	}
}

func TestInvalidCompressionLevel(t *testing.T) {
	_, err := archives.NewWriter(new(bytes.Buffer), archives.WriterOptions{
		Extension: ".tar.gz", CompressionLevel: archives.CompressionLevel(99),
	})
	assert.ErrorContains(t, err, "unknown compression level")

	err = archives.Create(new(bytes.Buffer), t.TempDir(), archives.CreateOptions{
		Extension: ".tar.gz", CompressionLevel: archives.CompressionLevel(99),
	})
	assert.ErrorContains(t, err, "unknown compression level")
}
