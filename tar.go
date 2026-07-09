// Copyright (C) 2026 archives contributors
//
// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at https://mozilla.org/MPL/2.0/.
//
// SPDX-License-Identifier: MPL-2.0

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
	return []string{"tar", "tgz", "tar.gz", "txz", "tar.xz", "tbz2", "tar.bz2", "tar.zst"}
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
	case "tar.zst":
		var err error
		container, err = newZstdReader(r)
		if err != nil {
			return nil, fmt.Errorf("failed to create zstd reader: %w", err)
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
