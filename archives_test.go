package archives_test

import (
	"io"
	"os"
	"path/filepath"
	"testing"

	"github.com/jaredallard/archives"
	"github.com/jaredallard/archives/internal/tartest"
	"gotest.tools/v3/assert"
)

// TestCanPickFromTar serves as a canary test for the entire package's
// functionality. It creates a tar archive with a single file, opens it,
// and picks the file from the archive.
func TestCanPickFromTar(t *testing.T) {
	tarArchive, err := tartest.Create()
	assert.NilError(t, err)

	a, err := archives.Open(tarArchive, archives.OpenOptions{
		Extension: ".tar",
	})
	assert.NilError(t, err)

	r, err := archives.Pick(a, archives.PickFilterByName("file.txt"))
	assert.NilError(t, err)

	got, err := io.ReadAll(r)
	assert.NilError(t, err)

	assert.Equal(t, string(got), "hello world")
}

// TestCanExtractFromTar serves as a canary test for the entire
// package's functionality. It creates a tar archive with a single file,
// extracts the archive, and reads the extracted file.
func TestCanExtractFromTar(t *testing.T) {
	targetDir := t.TempDir()
	tarArchive, err := tartest.Create()
	assert.NilError(t, err)

	// extract the archive
	assert.NilError(t, archives.Extract(tarArchive, targetDir, archives.ExtractOptions{
		Extension: ".tar",
	}))

	// read the extracted file
	got, err := os.ReadFile(filepath.Join(targetDir, "file.txt"))
	assert.NilError(t, err)

	assert.Equal(t, string(got), "hello world")
}
