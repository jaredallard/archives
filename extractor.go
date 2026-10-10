// Copyright (C) 2026 archives contributors
//
// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at https://mozilla.org/MPL/2.0/.
//
// SPDX-License-Identifier: MPL-2.0

package archives

import (
	"cmp"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"slices"
	"strings"
)

// localName converts an archive entry name into a cleaned path relative
// to the extraction root. Names that are absolute or that escape the
// root (e.g., "../x") are rejected.
//
// This is a lexical check only. Escapes through symlinks are prevented
// by performing all filesystem operations through an [os.Root].
func localName(name string) (string, error) {
	cleaned := filepath.Clean(filepath.FromSlash(name))
	if !filepath.IsLocal(cleaned) {
		return "", fmt.Errorf("content filepath is tainted: %s", name)
	}

	return cleaned, nil
}

// extractor holds the state for a single extraction.
type extractor struct {
	a    Archive
	root *os.Root

	// perms and owner mirror [ExtractOptions]. syncFiles and syncDirs are
	// true if entries should be synced individually, which depends on
	// the platform (see syncEachFile and syncEachDir).
	perms, owner, syncFiles, syncDirs bool

	// dirs contains every directory that was created or written to
	// during extraction, keyed by its local name. The value is the
	// header for the directory, or nil if the archive did not contain
	// an entry for it. Directory metadata is applied after all entries
	// have been extracted so that restrictive modes don't prevent
	// writing their contents and mtimes aren't clobbered by children.
	dirs map[string]*Header
}

// extract contains low level logic for extracting archives.
func extract(a Archive, dest string, opts *ExtractOptions) error {
	//nolint:gosec // Why: acceptable, we're an archive extractor.
	if err := os.MkdirAll(dest, 0o755); err != nil {
		return fmt.Errorf("failed to create destination: %w", err)
	}

	root, err := os.OpenRoot(dest)
	if err != nil {
		return fmt.Errorf("failed to open destination: %w", err)
	}
	defer root.Close()

	x := &extractor{
		a:         a,
		root:      root,
		perms:     *opts.PreservePermissions,
		owner:     opts.PreserveOwnership,
		syncFiles: opts.Sync && syncEachFile,
		syncDirs:  opts.Sync && syncEachDir,
		dirs:      map[string]*Header{".": nil},
	}
	for {
		h, err := a.Next()
		if err != nil {
			if errors.Is(err, io.EOF) {
				break
			}

			return fmt.Errorf("failed to read archive header: %w", err)
		}

		if err := x.extractEntry(h); err != nil {
			return err
		}
	}

	if err := x.finalizeDirs(); err != nil {
		return err
	}

	if opts.Sync {
		if err := syncFinal(root); err != nil {
			return fmt.Errorf("failed to sync destination: %w", err)
		}
	}

	return nil
}

// extractEntry extracts a single entry from the archive. Entries are
// fully validated before the filesystem is modified.
func (x *extractor) extractEntry(h *Header) error {
	name, err := localName(h.Name)
	if err != nil {
		return err
	}

	var create func(name string, h *Header) error
	switch h.Type {
	case HeaderDir:
		return x.extractDir(name, h)
	case HeaderFile:
		create = x.extractFile
	case HeaderSymlink:
		if h.Linkname == "" {
			return fmt.Errorf("symlink has no target: %s", h.Name)
		}
		create = x.extractSymlink
	case HeaderHardlink:
		// The link shares the metadata of its target, so none is applied.
		target, err := localName(h.Linkname)
		if err != nil {
			return fmt.Errorf("invalid hard link target for %s: %w", h.Name, err)
		}
		create = func(name string, _ *Header) error {
			if err := x.root.Link(target, name); err != nil {
				return fmt.Errorf("failed to create hard link: %w", err)
			}
			return nil
		}
	case HeaderUnsupported:
		return fmt.Errorf("unsupported file type in archive (%s: %v)", h.Name, h.Type)
	default:
		return fmt.Errorf("unknown header type in archive (%s: %v)", h.Name, h.Type)
	}

	if name == "." {
		return fmt.Errorf("refusing to replace extraction root with non-directory: %s", h.Name)
	}

	if err := x.ensureParent(name); err != nil {
		return err
	}

	// Replace anything already at name instead of writing through it.
	// Creation never follows an existing link, it fails with ErrExist.
	err = create(name, h)
	if errors.Is(err, fs.ErrExist) {
		if err := x.root.Remove(name); err != nil {
			return fmt.Errorf("failed to replace existing file: %w", err)
		}
		delete(x.dirs, name)
		err = create(name, h)
	}

	return err
}

// ensureParent creates the parent directories of name, unless they were
// already created during this extraction.
func (x *extractor) ensureParent(name string) error {
	dir := filepath.Dir(name)
	if _, ok := x.dirs[dir]; ok {
		return nil
	}

	//nolint:gosec // Why: acceptable, we're an archive extractor.
	if err := x.root.MkdirAll(dir, 0o755); err != nil {
		return fmt.Errorf("failed to create directory: %w", err)
	}

	// Record dir and its ancestors as written to.
	for ; dir != "."; dir = filepath.Dir(dir) {
		if _, ok := x.dirs[dir]; ok {
			break
		}
		x.dirs[dir] = nil
	}

	return nil
}

// extractDir creates a directory. Its metadata is applied later by
// [extractor.finalizeDirs].
func (x *extractor) extractDir(name string, h *Header) error {
	if err := x.ensureParent(name); err != nil {
		return err
	}

	//nolint:gosec // Why: acceptable, we're an archive extractor.
	err := x.root.Mkdir(name, 0o755)
	if errors.Is(err, fs.ErrExist) {
		err = x.reuseDir(name)
	}
	if err != nil {
		return fmt.Errorf("failed to create directory: %w", err)
	}

	x.dirs[name] = h
	return nil
}

// reuseDir prepares an existing path for use as a directory. Existing
// non-directories are replaced. Existing directories are made writable
// (e.g., read-only ones from a previous extraction), as their final
// mode is applied by [extractor.finalizeDirs].
func (x *extractor) reuseDir(name string) error {
	fi, err := x.root.Lstat(name)
	if err != nil {
		return err
	}

	if !fi.IsDir() {
		if err := x.root.Remove(name); err != nil {
			return err
		}
		//nolint:gosec // Why: acceptable, we're an archive extractor.
		return x.root.Mkdir(name, 0o755)
	}

	if perm := fi.Mode().Perm(); x.perms && perm&0o700 != 0o700 {
		return x.root.Chmod(name, perm|0o700)
	}

	return nil
}

// extractFile writes a regular file from the archive.
func (x *extractor) extractFile(name string, h *Header) error {
	// When preserving permissions, start private and widen to the final
	// mode once the contents have been written.
	perm := os.FileMode(0o666)
	if x.perms {
		perm = 0o600
	}

	f, err := x.root.OpenFile(name, os.O_WRONLY|os.O_CREATE|os.O_EXCL, perm)
	if err != nil {
		return fmt.Errorf("failed to create file: %w", err)
	}
	defer f.Close()

	if _, err := io.Copy(f, x.a); err != nil {
		return fmt.Errorf("failed to copy file contents: %w", err)
	}

	// Set before syncing so that the metadata is also durable.
	if err := x.setMetadata(f, name, h); err != nil {
		return err
	}

	if x.syncFiles {
		if err := syncEntry(f); err != nil {
			return fmt.Errorf("failed to sync file: %w", err)
		}
	}

	if err := f.Close(); err != nil {
		return fmt.Errorf("failed to close file: %w", err)
	}

	return nil
}

// setMetadata applies the ownership, mode and times from h to f, which
// was opened from name.
func (x *extractor) setMetadata(f *os.File, name string, h *Header) error {
	// Ownership must be set before the mode, as changing ownership may
	// clear setuid/setgid bits.
	if x.owner {
		if err := f.Chown(h.UID, h.GID); err != nil {
			return fmt.Errorf("failed to set ownership of %s: %w", name, err)
		}
	}

	if x.perms {
		if err := f.Chmod(h.Mode); err != nil {
			return fmt.Errorf("failed to set permissions of %s: %w", name, err)
		}
	}

	if err := x.root.Chtimes(name, h.AccessTime, h.ModTime); err != nil {
		return fmt.Errorf("failed to set times of %s: %w", name, err)
	}

	return nil
}

// extractSymlink creates a symlink. The target is recreated exactly as
// recorded in the archive, even if it points outside of the extraction
// root. Extraction never follows links outside of the root.
//
// The mode and times of symlinks are not restored, as doing so would
// modify the link's target instead.
func (x *extractor) extractSymlink(name string, h *Header) error {
	if err := x.root.Symlink(h.Linkname, name); err != nil {
		return fmt.Errorf("failed to create symlink: %w", err)
	}

	if x.owner {
		if err := x.root.Lchown(name, h.UID, h.GID); err != nil {
			return fmt.Errorf("failed to set symlink ownership: %w", err)
		}
	}

	return nil
}

// finalizeDirs applies directory metadata, deepest directories first,
// so that a directory remains writable until all of its children have
// been finalized. If [ExtractOptions.Sync] is set, each directory is
// also synced so that its entries are durable.
func (x *extractor) finalizeDirs() error {
	names := make([]string, 0, len(x.dirs))
	for name, h := range x.dirs {
		if h != nil || x.syncDirs {
			names = append(names, name)
		}
	}
	slices.SortFunc(names, func(a, b string) int {
		return cmp.Or(cmp.Compare(dirDepth(b), dirDepth(a)), strings.Compare(a, b))
	})

	for _, name := range names {
		if err := x.finalizeDir(name, x.dirs[name]); err != nil {
			return err
		}
	}

	return nil
}

// finalizeDir applies the metadata from h (if not nil) to the directory
// at name and syncs it if requested.
func (x *extractor) finalizeDir(name string, h *Header) error {
	d, err := x.root.Open(name)
	if err != nil {
		return fmt.Errorf("failed to open directory: %w", err)
	}
	defer d.Close()

	if h != nil {
		if err := x.setMetadata(d, name, h); err != nil {
			return err
		}
	}

	if x.syncDirs {
		if err := syncEntry(d); err != nil {
			return fmt.Errorf("failed to sync directory: %w", err)
		}
	}

	return nil
}

// dirDepth returns the depth of a local directory name, where "." is
// shallowest.
func dirDepth(name string) int {
	if name == "." {
		return -1
	}
	return strings.Count(name, string(filepath.Separator))
}
