package archives_test

import (
	stdzip "archive/zip"
	"bytes"
	"io"
	"testing"

	"github.com/jaredallard/archives"
	"gotest.tools/v3/assert"
)

func TestZip(t *testing.T) {
	buf := new(bytes.Buffer)
	zw := stdzip.NewWriter(buf)
	w, err := zw.Create("file.txt")
	assert.NilError(t, err)
	_, err = w.Write([]byte("hello world"))
	assert.NilError(t, err)
	assert.NilError(t, zw.Close())

	a, err := archives.Open(buf, archives.OpenOptions{
		Extension: ".zip",
	})
	assert.NilError(t, err)

	r, err := archives.Pick(a, archives.PickFilterByName("file.txt"))
	assert.NilError(t, err)

	b, err := io.ReadAll(r)
	assert.NilError(t, err)

	assert.Equal(t, string(b), "hello world")
}
