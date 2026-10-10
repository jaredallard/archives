// Copyright (C) 2026 archives contributors
//
// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at https://mozilla.org/MPL/2.0/.
//
// SPDX-License-Identifier: MPL-2.0

package archives

import (
	"errors"
	"fmt"
	"io"
	"io/fs"
	"os"
)

// create writes every entry in src to aw. Entries are written in
// lexical order and symlinks are never followed.
func create(aw ArchiveWriter, src string) error {
	root, err := os.OpenRoot(src)
	if err != nil {
		return fmt.Errorf("failed to open source: %w", err)
	}
	defer root.Close()

	fsys := root.FS()
	return fs.WalkDir(fsys, ".", func(name string, _ fs.DirEntry, err error) error {
		if err != nil {
			return fmt.Errorf("failed to walk %s: %w", name, err)
		}
		if name == "." {
			return nil
		}

		return createEntry(aw, fsys, name)
	})
}

// createEntry writes the entry at name in fsys to aw.
func createEntry(aw ArchiveWriter, fsys fs.FS, name string) error {
	fi, err := fs.Lstat(fsys, name)
	if err != nil {
		return fmt.Errorf("failed to stat %s: %w", name, err)
	}

	h := &Header{
		Name:    name,
		Mode:    fi.Mode() & (fs.ModePerm | fs.ModeSetuid | fs.ModeSetgid | fs.ModeSticky),
		ModTime: fi.ModTime(),
	}

	//nolint:exhaustive // Why: All other types are unsupported.
	switch fi.Mode().Type() {
	case fs.ModeDir:
		h.Type = HeaderDir
		return aw.WriteHeader(h)
	case fs.ModeSymlink:
		target, err := fs.ReadLink(fsys, name)
		if err != nil {
			return fmt.Errorf("failed to read symlink %s: %w", name, err)
		}
		h.Type = HeaderSymlink
		h.Linkname = target
		// Symlink permissions vary by platform and are never applied,
		// so store the conventional mode.
		h.Mode = fs.ModePerm
		return aw.WriteHeader(h)
	case 0:
		h.Type = HeaderFile
		h.Size = fi.Size()
		return createFile(aw, fsys, h)
	default:
		return fmt.Errorf("unsupported file type (%s: %v)", name, fi.Mode().Type())
	}
}

// createFile writes the header h and the contents of the regular file
// it describes to aw. Exactly h.Size bytes are written, so files that
// change while being read can't corrupt the archive.
func createFile(aw ArchiveWriter, fsys fs.FS, h *Header) error {
	f, err := fsys.Open(h.Name)
	if err != nil {
		return fmt.Errorf("failed to open %s: %w", h.Name, err)
	}
	defer f.Close()

	if err := aw.WriteHeader(h); err != nil {
		return err
	}

	if _, err := io.CopyN(aw, f, h.Size); err != nil {
		if errors.Is(err, io.EOF) {
			return fmt.Errorf("%s shrank while being archived", h.Name)
		}
		return fmt.Errorf("failed to write contents of %s: %w", h.Name, err)
	}

	return nil
}
