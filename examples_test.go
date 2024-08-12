package archives_test

import (
	"fmt"
	"io"

	"github.com/jaredallard/archives"
	"github.com/jaredallard/archives/internal/tartest"
)

func Example() {
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
