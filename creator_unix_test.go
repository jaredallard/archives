//go:build unix

package archives_test

import (
	"bytes"
	"os"
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

// fileOwner returns the user and group ID of path, without following
// symlinks.
func fileOwner(t *testing.T, path string) (uid, gid int) {
	t.Helper()

	fi, err := os.Lstat(path)
	assert.NilError(t, err)
	st, ok := fi.Sys().(*syscall.Stat_t)
	assert.Assert(t, ok)
	return int(st.Uid), int(st.Gid)
}

func TestCreatePreserveOwnership(t *testing.T) {
	src := t.TempDir()
	assert.NilError(t, os.Mkdir(filepath.Join(src, "dir"), 0o755))
	assert.NilError(t, os.WriteFile(filepath.Join(src, "dir", "file.txt"), []byte("hello"), 0o644))
	assert.NilError(t, os.Symlink("dir/file.txt", filepath.Join(src, "link")))

	names := []string{"dir", "dir/file.txt", "link"}

	t.Run("enabled", func(t *testing.T) {
		buf := new(bytes.Buffer)
		assert.NilError(t, archives.Create(buf, src, archives.CreateOptions{
			Extension: ".tar.gz", PreserveOwnership: true,
		}))

		entries := readAll(t, buf, ".tar.gz")
		for _, name := range names {
			uid, gid := fileOwner(t, filepath.Join(src, name))
			assert.Equal(t, entries[name].h.UID, uid, "uid of %s", name)
			assert.Equal(t, entries[name].h.GID, gid, "gid of %s", name)
		}
	})

	t.Run("disabled by default", func(t *testing.T) {
		buf := new(bytes.Buffer)
		assert.NilError(t, archives.Create(buf, src, archives.CreateOptions{Extension: ".tar.gz"}))

		entries := readAll(t, buf, ".tar.gz")
		for _, name := range names {
			assert.Equal(t, entries[name].h.UID, 0, "uid of %s", name)
			assert.Equal(t, entries[name].h.GID, 0, "gid of %s", name)
		}
	})

	t.Run("ignored by zip", func(t *testing.T) {
		buf := new(bytes.Buffer)
		assert.NilError(t, archives.Create(buf, src, archives.CreateOptions{
			Extension: ".zip", PreserveOwnership: true,
		}))

		entries := readAll(t, buf, ".zip")
		assert.Equal(t, entries["dir/file.txt"].body, "hello")
	})
}
