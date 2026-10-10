//go:build unix

package archives_test

import (
	"bytes"
	"path/filepath"
	"syscall"
	"testing"

	"go.rgst.io/jaredallard/archives/v2"
	"gotest.tools/v3/assert"
)

func TestCreateRejectsUnsupportedTypes(t *testing.T) {
	src := t.TempDir()
	assert.NilError(t, syscall.Mkfifo(filepath.Join(src, "fifo"), 0o644))

	err := archives.Create(new(bytes.Buffer), src, archives.CreateOptions{Extension: ".tar"})
	assert.ErrorContains(t, err, "unsupported file type")
}
