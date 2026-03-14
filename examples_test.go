package archives_test

import (
	"fmt"
	"io"
	"os"
	"path/filepath"

	"go.rgst.io/jaredallard/archives/v2"
	"go.rgst.io/jaredallard/archives/v2/internal/tartest"
)

func ExamplePick() {
	tarArchive, err := tartest.Create()
	if err != nil {
		panic(err)
	}

	// Open the archive.
	a, err := archives.Open(tarArchive, archives.OpenOptions{
		Extension: ".tar",
	})
	if err != nil {
		panic(err)
	}

	// Pick a single file from the archive.
	r, err := archives.Pick(a, archives.PickFilterByName("file.txt"))
	if err != nil {
		panic(err)
	}

	// Do something with the reader.
	b, err := io.ReadAll(r)
	if err != nil {
		panic(err)
	}
	fmt.Println(string(b))

	// Output:
	// hello world
}

func ExampleExtract() {
	tarArchive, err := tartest.Create()
	if err != nil {
		panic(err)
	}

	// Create a temporary directory to extract the archive to.
	tmpDir, err := os.MkdirTemp("", "archives-test")
	if err != nil {
		panic(err)
	}
	defer os.RemoveAll(tmpDir) // Remove the temporary directory when done.

	// Open the archive.
	if err := archives.Extract(tarArchive, tmpDir, archives.ExtractOptions{
		Extension: ".tar",
	}); err != nil {
		panic(err)
	}

	// Read the extracted file.
	got, err := os.ReadFile(filepath.Join(tmpDir, "file.txt"))
	if err != nil {
		panic(err)
	}

	fmt.Println(string(got))

	// Output:
	// hello world
}
