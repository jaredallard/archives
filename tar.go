// Copyright (C) 2024 archives contributors
//
// This program is free software: you can redistribute it and/or modify
// it under the terms of the GNU Lesser General Public License as
// published by the Free Software Foundation, either version 3 of the
// License, or (at your option) any later version.
//
// This program is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
// GNU Lesser General Public License for more details.
//
// You should have received a copy of the GNU Lesser General Public
// License along with this program. If not, see
// <https://www.gnu.org/licenses/>.
//
// SPDX-License-Identifier: LGPL-3.0

package archives

import (
	stdtar "archive/tar"
	"fmt"
	"io"
)

// _ ensures that tar implements the [Archiver] interface.
var _ Archiver = (&tar{})

// tar implements the [Archiver] interface for tar archives and their
// compressed variants.
type tar struct{}

// Extensions returns the supported extensions for the tar extractor.
func (t *tar) Extensions() []string {
	return []string{"tar", "tgz", "tar.gz", "txz", "tar.xz", "tbz2", "tar.bz2"}
}

func (t *tar) Open(r io.Reader, ext string) (Archive, error) {
	// Determine if we're dealing with a compressed tar archive and if so,
	// create the appropriate reader.
	var container io.ReadCloser
	switch ext {
	case "tar":
		container = io.NopCloser(r)
	case "tgz", "tar.gz":
		var err error
		container, err = newGzipReader(r)
		if err != nil {
			return nil, fmt.Errorf("failed to create gzip reader: %w", err)
		}
	case "tbz2", "tar.bz2":
		container = newBzip2Reader(r)
	case "txz", "tar.xz":
		var err error
		container, err = newXZReader(r)
		if err != nil {
			return nil, fmt.Errorf("failed to create xz reader: %w", err)
		}
	default:
		// This only happens if we're missing a case in the switch statement.
		return nil, fmt.Errorf("unsupported tar extension: %s", ext)
	}

	tr := stdtar.NewReader(container)
	return &tarArchive{tr, container}, nil
}

type tarArchive struct {
	*stdtar.Reader
	closer io.Closer
}

func (t *tarArchive) Close() error {
	return t.closer.Close()
}

func (t *tarArchive) Next() (*Header, error) {
	h, err := t.Reader.Next()
	if err != nil {
		return nil, err
	}

	hType := HeaderFile
	if h.FileInfo().IsDir() {
		hType = HeaderDir
	}

	return &Header{
		Name:       h.Name,
		Type:       hType,
		Mode:       h.FileInfo().Mode(),
		Size:       h.Size,
		AccessTime: h.AccessTime,
		ModTime:    h.ModTime,
		UID:        h.Uid,
		GID:        h.Gid,
	}, nil
}
