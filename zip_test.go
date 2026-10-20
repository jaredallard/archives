package archives_test

import (
	stdzip "archive/zip"
	"bytes"
	"errors"
	"io"
	"os"
	"path/filepath"
	"testing"

	"go.rgst.io/jaredallard/archives/v2"
	"gotest.tools/v3/assert"
)

// newZip returns a zip archive containing file.txt with the contents
// "hello world".
func newZip(t *testing.T) []byte {
	t.Helper()

	buf := new(bytes.Buffer)
	zw := stdzip.NewWriter(buf)
	w, err := zw.Create("file.txt")
	assert.NilError(t, err)
	_, err = w.Write([]byte("hello world"))
	assert.NilError(t, err)
	assert.NilError(t, zw.Close())
	return buf.Bytes()
}

// assertZipContents opens r as a zip archive and asserts that file.txt
// contains "hello world".
func assertZipContents(t *testing.T, r io.Reader) {
	t.Helper()

	a, err := archives.Open(r, archives.OpenOptions{
		Extension: ".zip",
	})
	assert.NilError(t, err)
	defer a.Close()

	pr, err := archives.Pick(a, archives.PickFilterByName("file.txt"))
	assert.NilError(t, err)

	b, err := io.ReadAll(pr)
	assert.NilError(t, err)

	assert.Equal(t, string(b), "hello world")
}

// noReadReader is a random access reader that fails if it is read as a
// stream.
type noReadReader struct {
	*bytes.Reader
}

// Read always fails.
func (noReadReader) Read([]byte) (int, error) {
	return 0, errors.New("unexpected call to Read")
}

// noSeekReader is a reader that implements [io.Seeker], but can't seek
// (e.g., a pipe).
type noSeekReader struct {
	*bytes.Reader
}

// Seek always fails.
func (noSeekReader) Seek(int64, int) (int64, error) {
	return 0, errors.New("illegal seek")
}

func TestZip(t *testing.T) {
	assertZipContents(t, bytes.NewBuffer(newZip(t)))
}

func TestZipFile(t *testing.T) {
	path := filepath.Join(t.TempDir(), "archive.zip")
	assert.NilError(t, os.WriteFile(path, newZip(t), 0o600))

	f, err := os.Open(path)
	assert.NilError(t, err)
	defer f.Close()

	assertZipContents(t, f)
}

func TestZipReadsInPlace(t *testing.T) {
	assertZipContents(t, noReadReader{bytes.NewReader(newZip(t))})
}

func TestZipReadsFromCurrentPosition(t *testing.T) {
	prefix := []byte("not part of the archive")
	r := bytes.NewReader(append(prefix, newZip(t)...))
	_, err := r.Seek(int64(len(prefix)), io.SeekStart)
	assert.NilError(t, err)

	assertZipContents(t, noReadReader{r})
}

func TestZipStreamFallback(t *testing.T) {
	r := struct{ io.Reader }{bytes.NewReader(newZip(t))}
	assertZipContents(t, r)
}

func TestZipSeekFailureFallback(t *testing.T) {
	assertZipContents(t, noSeekReader{bytes.NewReader(newZip(t))})
}
