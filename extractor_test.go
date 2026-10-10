package archives_test

import (
	stdtar "archive/tar"
	stdzip "archive/zip"
	"bytes"
	"io/fs"
	"os"
	"path/filepath"
	"runtime"
	"testing"
	"time"

	"go.rgst.io/jaredallard/archives/v2"
	"go.rgst.io/jaredallard/archives/v2/internal/tartest"
	"gotest.tools/v3/assert"
	is "gotest.tools/v3/assert/cmp"
)

// tarEntry is an alias to keep test tables short.
type tarEntry = tartest.Entry

// extractTar extracts a tar archive of the provided entries into dest.
func extractTar(t *testing.T, dest string, opts archives.ExtractOptions, entries ...tarEntry) error {
	t.Helper()

	r, err := tartest.Create(tartest.WithEntries(entries...))
	assert.NilError(t, err)

	opts.Extension = ".tar"
	return archives.Extract(r, dest, opts)
}

// assertFileContent asserts that the file at path contains want.
func assertFileContent(t *testing.T, path, want string) {
	t.Helper()

	b, err := os.ReadFile(path)
	assert.NilError(t, err)
	assert.Equal(t, string(b), want)
}

// assertNotExist asserts that nothing exists at path.
func assertNotExist(t *testing.T, path string) {
	t.Helper()

	_, err := os.Lstat(path)
	assert.Assert(t, is.ErrorIs(err, fs.ErrNotExist))
}

// skipIfNoSymlinks skips the test if symlinks cannot be created.
func skipIfNoSymlinks(t *testing.T) {
	t.Helper()

	dir := t.TempDir()
	if err := os.Symlink("target", filepath.Join(dir, "link")); err != nil {
		t.Skipf("symlinks not supported: %v", err)
	}
}

func TestExtractSymlinks(t *testing.T) {
	skipIfNoSymlinks(t)

	dest := t.TempDir()
	assert.NilError(t, extractTar(t, dest, archives.ExtractOptions{},
		tarEntry{Header: stdtar.Header{Name: "data/file.txt"}, Body: "hello"},
		tarEntry{Header: stdtar.Header{Name: "rel", Typeflag: stdtar.TypeSymlink, Linkname: "data/file.txt"}},
		tarEntry{Header: stdtar.Header{Name: "abs", Typeflag: stdtar.TypeSymlink, Linkname: "/etc/hosts"}},
		tarEntry{Header: stdtar.Header{Name: "up", Typeflag: stdtar.TypeSymlink, Linkname: "../outside"}},
	))

	for name, target := range map[string]string{
		"rel": "data/file.txt",
		"abs": "/etc/hosts",
		"up":  "../outside",
	} {
		fi, err := os.Lstat(filepath.Join(dest, name))
		assert.NilError(t, err)
		assert.Assert(t, fi.Mode()&fs.ModeSymlink != 0, "%s is not a symlink: %v", name, fi.Mode())

		got, err := os.Readlink(filepath.Join(dest, name))
		assert.NilError(t, err)
		assert.Equal(t, got, target)
	}

	assertFileContent(t, filepath.Join(dest, "rel"), "hello")
}

func TestExtractZipSymlink(t *testing.T) {
	skipIfNoSymlinks(t)

	buf := new(bytes.Buffer)
	zw := stdzip.NewWriter(buf)
	fh := &stdzip.FileHeader{Name: "link"}
	fh.SetMode(fs.ModeSymlink | 0o777)
	w, err := zw.CreateHeader(fh)
	assert.NilError(t, err)
	_, err = w.Write([]byte("target.txt"))
	assert.NilError(t, err)
	assert.NilError(t, zw.Close())

	dest := t.TempDir()
	assert.NilError(t, archives.Extract(buf, dest, archives.ExtractOptions{Extension: ".zip"}))

	got, err := os.Readlink(filepath.Join(dest, "link"))
	assert.NilError(t, err)
	assert.Equal(t, got, "target.txt")
}

func TestExtractHardlink(t *testing.T) {
	dest := t.TempDir()
	assert.NilError(t, extractTar(t, dest, archives.ExtractOptions{},
		tarEntry{Header: stdtar.Header{Name: "a/file.txt"}, Body: "hello"},
		tarEntry{Header: stdtar.Header{Name: "b/link.txt", Typeflag: stdtar.TypeLink, Linkname: "a/file.txt"}},
	))

	a, err := os.Stat(filepath.Join(dest, "a", "file.txt"))
	assert.NilError(t, err)
	b, err := os.Stat(filepath.Join(dest, "b", "link.txt"))
	assert.NilError(t, err)
	assert.Assert(t, os.SameFile(a, b), "hard link does not share an inode with its target")
}

func TestExtractRejectsEscapes(t *testing.T) {
	testCases := []struct {
		name    string
		entries []tarEntry
	}{
		{
			name:    "sibling directory sharing a prefix",
			entries: []tarEntry{{Header: stdtar.Header{Name: "../game-evil/x"}, Body: "pwned"}},
		},
		{
			name:    "parent directory",
			entries: []tarEntry{{Header: stdtar.Header{Name: "a/../../x"}, Body: "pwned"}},
		},
		{
			name:    "absolute path",
			entries: []tarEntry{{Header: stdtar.Header{Name: "/x"}, Body: "pwned"}},
		},
		{
			name: "hard link to outside",
			entries: []tarEntry{
				{Header: stdtar.Header{Name: "x", Typeflag: stdtar.TypeLink, Linkname: "../game-evil/x"}},
			},
		},
	}
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			parent := t.TempDir()
			dest := filepath.Join(parent, "game")

			err := extractTar(t, dest, archives.ExtractOptions{}, tc.entries...)
			assert.ErrorContains(t, err, "tainted")

			assertNotExist(t, filepath.Join(parent, "game-evil"))
			assertNotExist(t, filepath.Join(parent, "x"))
		})
	}
}

func TestExtractDoesNotWriteThroughSymlinks(t *testing.T) {
	skipIfNoSymlinks(t)

	outside := t.TempDir()
	dest := t.TempDir()

	err := extractTar(t, dest, archives.ExtractOptions{},
		tarEntry{Header: stdtar.Header{Name: "evil", Typeflag: stdtar.TypeSymlink, Linkname: outside}},
		tarEntry{Header: stdtar.Header{Name: "evil/pwned"}, Body: "pwned"},
	)
	assert.Assert(t, err != nil, "expected error writing through symlink")

	assertNotExist(t, filepath.Join(outside, "pwned"))
}

func TestExtractReplacesExistingSymlink(t *testing.T) {
	skipIfNoSymlinks(t)

	outside := filepath.Join(t.TempDir(), "victim.txt")
	assert.NilError(t, os.WriteFile(outside, []byte("original"), 0o600))

	dest := t.TempDir()
	assert.NilError(t, os.Symlink(outside, filepath.Join(dest, "file.txt")))

	assert.NilError(t, extractTar(t, dest, archives.ExtractOptions{},
		tarEntry{Header: stdtar.Header{Name: "file.txt"}, Body: "restored"},
	))

	assertFileContent(t, outside, "original")

	fi, err := os.Lstat(filepath.Join(dest, "file.txt"))
	assert.NilError(t, err)
	assert.Assert(t, fi.Mode().IsRegular())
}

func TestExtractDirectoryMetadataAppliedLast(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("directory permissions are not supported on Windows")
	}

	mtime := time.Date(2020, 1, 2, 3, 4, 5, 0, time.UTC)
	dest := t.TempDir()
	t.Cleanup(func() {
		// Allow t.TempDir to clean up the read-only directory.
		_ = os.Chmod(filepath.Join(dest, "ro"), 0o755) //nolint:errcheck // Why: Best effort.
	})

	entries := []tarEntry{
		{Header: stdtar.Header{Name: "ro/", Typeflag: stdtar.TypeDir, Mode: 0o555, ModTime: mtime}},
		{Header: stdtar.Header{Name: "ro/file.txt", ModTime: mtime}, Body: "hello"},
		{Header: stdtar.Header{Name: "ro/sub/nested.txt", ModTime: mtime}, Body: "nested"},
	}
	assert.NilError(t, extractTar(t, dest, archives.ExtractOptions{}, entries...))

	fi, err := os.Stat(filepath.Join(dest, "ro"))
	assert.NilError(t, err)
	assert.Equal(t, fi.Mode().Perm(), fs.FileMode(0o555))
	assert.Assert(t, fi.ModTime().Equal(mtime), "directory mtime %v != %v", fi.ModTime(), mtime)

	assertFileContent(t, filepath.Join(dest, "ro", "sub", "nested.txt"), "nested")

	// Extracting again on top of the read-only directory should work.
	assert.NilError(t, extractTar(t, dest, archives.ExtractOptions{}, entries...))
}

func TestExtractPreservesSetuid(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("setuid is not supported on Windows")
	}

	dest := t.TempDir()
	opts := archives.ExtractOptions{PreserveOwnership: os.Geteuid() == 0}
	assert.NilError(t, extractTar(t, dest, opts,
		tarEntry{Header: stdtar.Header{
			Name: "bin", Mode: 0o4755, Uid: os.Getuid(), Gid: os.Getgid(),
		}, Body: "#!/bin/sh"},
	))

	fi, err := os.Stat(filepath.Join(dest, "bin"))
	assert.NilError(t, err)
	assert.Equal(t, fi.Mode()&(fs.ModeSetuid|fs.ModePerm), fs.ModeSetuid|0o755)
}

func TestExtractSync(t *testing.T) {
	dest := t.TempDir()
	assert.NilError(t, extractTar(t, dest, archives.ExtractOptions{Sync: true},
		tarEntry{Header: stdtar.Header{Name: "dir/", Typeflag: stdtar.TypeDir, Mode: 0o755}},
		tarEntry{Header: stdtar.Header{Name: "dir/file.txt"}, Body: "hello"},
		tarEntry{Header: stdtar.Header{Name: "implicit/file.txt"}, Body: "world"},
	))

	assertFileContent(t, filepath.Join(dest, "implicit", "file.txt"), "world")
}

func TestExtractRejectsUnsupportedTypes(t *testing.T) {
	dest := t.TempDir()
	err := extractTar(t, dest, archives.ExtractOptions{},
		tarEntry{Header: stdtar.Header{Name: "fifo", Typeflag: stdtar.TypeFifo}},
	)
	assert.ErrorContains(t, err, "unsupported file type")
	assertNotExist(t, filepath.Join(dest, "fifo"))

	// Rejected entries must not modify existing files.
	assert.NilError(t, os.WriteFile(filepath.Join(dest, "fifo"), []byte("existing"), 0o600))
	err = extractTar(t, dest, archives.ExtractOptions{},
		tarEntry{Header: stdtar.Header{Name: "fifo", Typeflag: stdtar.TypeFifo}},
	)
	assert.ErrorContains(t, err, "unsupported file type")
	assertFileContent(t, filepath.Join(dest, "fifo"), "existing")
}

func TestExtractRootDirectoryEntry(t *testing.T) {
	// Archives created with `tar -C dir .` contain a "./" entry.
	dest := t.TempDir()
	assert.NilError(t, extractTar(t, dest, archives.ExtractOptions{},
		tarEntry{Header: stdtar.Header{Name: "./", Typeflag: stdtar.TypeDir, Mode: 0o755}},
		tarEntry{Header: stdtar.Header{Name: "./file.txt"}, Body: "hello"},
	))

	assertFileContent(t, filepath.Join(dest, "file.txt"), "hello")
}
