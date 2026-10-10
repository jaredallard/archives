package archives_test

import (
	"bytes"
	"io/fs"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"go.rgst.io/jaredallard/archives/v2"
	"gotest.tools/v3/assert"
)

// createAndExtract archives src with ext, then extracts it into a new
// directory, which is returned along with the archive.
func createAndExtract(t *testing.T, src, ext string) (dest string, archive []byte) {
	t.Helper()

	buf := new(bytes.Buffer)
	assert.NilError(t, archives.Create(buf, src, archives.CreateOptions{Extension: ext}))
	archive = buf.Bytes()

	dest = t.TempDir()
	assert.NilError(t, archives.Extract(bytes.NewReader(archive), dest, archives.ExtractOptions{Extension: ext}))
	return dest, archive
}

// assertMode asserts that the permission bits of path are want.
func assertMode(t *testing.T, path string, want fs.FileMode) {
	t.Helper()

	fi, err := os.Lstat(path)
	assert.NilError(t, err)
	assert.Equal(t, fi.Mode().Perm(), want, "mode of %s", path)
}

func TestCreateRoundTrip(t *testing.T) {
	src := t.TempDir()
	assert.NilError(t, os.MkdirAll(filepath.Join(src, "a", "b"), 0o755))
	assert.NilError(t, os.Mkdir(filepath.Join(src, "empty"), 0o700))
	assert.NilError(t, os.WriteFile(filepath.Join(src, "a", "b", "exec.sh"), []byte("#!/bin/sh"), 0o755))
	assert.NilError(t, os.WriteFile(filepath.Join(src, "private.txt"), []byte("secret"), 0o600))
	// Ensure the modes aren't affected by the umask.
	assert.NilError(t, os.Chmod(filepath.Join(src, "a", "b", "exec.sh"), 0o755))
	assert.NilError(t, os.Chmod(filepath.Join(src, "empty"), 0o700))

	for _, ext := range writableExtensions {
		t.Run(ext, func(t *testing.T) {
			dest, _ := createAndExtract(t, src, ext)

			assertFileContent(t, filepath.Join(dest, "a", "b", "exec.sh"), "#!/bin/sh")
			assertFileContent(t, filepath.Join(dest, "private.txt"), "secret")

			fi, err := os.Stat(filepath.Join(dest, "empty"))
			assert.NilError(t, err)
			assert.Assert(t, fi.IsDir())

			if runtime.GOOS != "windows" {
				assertMode(t, filepath.Join(dest, "a", "b", "exec.sh"), 0o755)
				assertMode(t, filepath.Join(dest, "private.txt"), 0o600)
				assertMode(t, filepath.Join(dest, "empty"), 0o700)
			}
		})
	}
}

func TestCreateSymlinks(t *testing.T) {
	skipIfNoSymlinks(t)

	src := t.TempDir()
	assert.NilError(t, os.Mkdir(filepath.Join(src, "data"), 0o755))
	assert.NilError(t, os.WriteFile(filepath.Join(src, "data", "file.txt"), []byte("hello"), 0o644))

	links := map[string]string{
		"rel":     "data/file.txt",
		"abs":     "/etc/hosts",
		"up":      "../outside",
		"dangle":  "missing",
		"dirlink": "data",
	}
	for name, target := range links {
		assert.NilError(t, os.Symlink(target, filepath.Join(src, name)))
	}

	for _, ext := range []string{".tar.gz", ".zip"} {
		t.Run(ext, func(t *testing.T) {
			dest, b := createAndExtract(t, src, ext)

			entries := readAll(t, bytes.NewReader(b), ext)
			for name, target := range links {
				e, ok := entries[name]
				assert.Assert(t, ok, "missing entry %s", name)
				assert.Equal(t, e.h.Type, archives.HeaderSymlink, "type of %s", name)
				assert.Equal(t, e.h.Linkname, target, "target of %s", name)
			}
			for name := range entries {
				assert.Assert(t, !strings.HasPrefix(name, "dirlink/"), "followed symlinked directory: %s", name)
			}

			for name, target := range links {
				got, err := os.Readlink(filepath.Join(dest, name))
				assert.NilError(t, err)
				assert.Equal(t, got, target, "target of %s", name)
			}
			assertFileContent(t, filepath.Join(dest, "rel"), "hello")
		})
	}
}

func TestCreateMissingSource(t *testing.T) {
	err := archives.Create(new(bytes.Buffer), filepath.Join(t.TempDir(), "nope"), archives.CreateOptions{Extension: ".tar"})
	assert.ErrorContains(t, err, "failed to open source")
}
