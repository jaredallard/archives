package archives

import (
	"io"
	"os/exec"
	"testing"

	"github.com/jaredallard/archives/internal/tartest"
	"gotest.tools/v3/assert"
)

func TestTarContainers(t *testing.T) {
	type containerWithExtension struct {
		value tartest.Container
		ext   string
	}

	containers := []containerWithExtension{
		{tartest.ContainerGz, "gz"},
		{tartest.ContainerXz, "xz"},
		{tartest.ContainerBz2, "bz2"},
	}
	for _, container := range containers {
		t.Run(container.ext, func(t *testing.T) {
			if container.value == tartest.ContainerBz2 {
				if _, err := exec.LookPath("bzip2"); err != nil {
					t.Skip("bzip2 not found on host")
				}
			}

			tarArchive, err := tartest.Create(tartest.WithContainer(container.value))
			assert.NilError(t, err)

			a, err := Open(tarArchive, OpenOptions{
				Extension: ".tar." + container.ext,
			})
			assert.NilError(t, err)

			r, err := Pick(a, PickFilterByName("file.txt"))
			assert.NilError(t, err)

			b, err := io.ReadAll(r)
			assert.NilError(t, err)

			assert.Equal(t, string(b), "hello world")
		})
	}
}
