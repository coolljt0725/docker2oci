package main

import (
	"archive/tar"
	"bufio"
	"bytes"
	"compress/bzip2"
	"compress/gzip"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"
)

func getReader(comp string, buf io.Reader) (io.Reader, error) {
	switch comp {
	case "gzip":
		return gzip.NewReader(buf)
	case "bzip2":
		return bzip2.NewReader(buf), nil
	case "xz":
		return nil, fmt.Errorf("xz format tar are not supported")
	default:
		return buf, nil
	}
}

// DetectCompression detects the compression algorithm of the source.
func DetectCompression(r *bufio.Reader) (string, error) {
	source, err := r.Peek(10)
	if err != nil {
		return "", err
	}

	for compression, m := range map[string][]byte{
		"bzip2": {0x42, 0x5A, 0x68},
		"gzip":  {0x1F, 0x8B, 0x08},
	} {
		if len(source) < len(m) {
			continue
		}
		if bytes.Equal(m, source[:len(m)]) {
			return compression, nil
		}
	}
	return "plain", nil
}

func unpack(dest string, r io.Reader) error {
	buf := bufio.NewReader(r)

	comp, err := DetectCompression(buf)
	if err != nil {
		return err
	}
	reader, err := getReader(comp, buf)
	if err != nil {
		return err
	}

	var dirs []*tar.Header
	tr := tar.NewReader(reader)

loop:
	for {
		hdr, err := tr.Next()
		switch err {
		case io.EOF:
			break loop
		case nil:
			// success, continue below
		default:
			return fmt.Errorf("error advancing tar stream: %v", err)
		}

		hdr.Name = filepath.Clean(hdr.Name)
		if !strings.HasSuffix(hdr.Name, string(os.PathSeparator)) {
			// Not the root directory, ensure that the parent directory exists
			parent := filepath.Dir(hdr.Name)
			parentPath := filepath.Join(dest, parent)
			if _, err2 := os.Lstat(parentPath); err2 != nil && os.IsNotExist(err2) {
				if err3 := os.MkdirAll(parentPath, 0755); err3 != nil {
					return err3
				}
			}
		}

		path := filepath.Join(dest, hdr.Name)
		rel, err := filepath.Rel(dest, path)
		if err != nil {
			return err
		}
		info := hdr.FileInfo()
		if strings.HasPrefix(rel, ".."+string(os.PathSeparator)) {
			return fmt.Errorf("%q is outside of %q", hdr.Name, dest)
		}

		switch hdr.Typeflag {
		case tar.TypeDir:
			if fi, err := os.Lstat(path); !(err == nil && fi.IsDir()) {
				if err2 := os.MkdirAll(path, 0755); err2 != nil {
					return fmt.Errorf("error creating directory: %v", err2)
				}
			}

		case tar.TypeReg, tar.TypeRegA:
			f, err := os.OpenFile(path, os.O_CREATE|os.O_WRONLY, info.Mode())
			if err != nil {
				return fmt.Errorf("unable to open file: %v", err)
			}

			if _, err := io.Copy(f, tr); err != nil {
				f.Close()
				return fmt.Errorf("unable to copy: %v", err)
			}
			f.Close()

		case tar.TypeLink:
			target := filepath.Join(dest, hdr.Linkname)

			if !strings.HasPrefix(target, dest) {
				return fmt.Errorf("invalid hardlink %q -> %q", target, hdr.Linkname)
			}

			if err := os.Link(target, path); err != nil {
				return err
			}

		case tar.TypeSymlink:
			target := filepath.Join(filepath.Dir(path), hdr.Linkname)

			if !strings.HasPrefix(target, dest) {
				return fmt.Errorf("invalid symlink %q -> %q", path, hdr.Linkname)
			}

			if err := os.Symlink(hdr.Linkname, path); err != nil {
				return err
			}
		case tar.TypeXGlobalHeader:
			return nil
		}
		// Directory mtimes must be handled at the end to avoid further
		// file creation in them to modify the directory mtime
		// We also have to set the mode at the end in the event that they're read-only
		// in the tarball.
		if hdr.Typeflag == tar.TypeDir {
			dirs = append(dirs, hdr)
		}
	}
	for _, hdr := range dirs {
		path := filepath.Join(dest, hdr.Name)

		finfo := hdr.FileInfo()
		if err := os.Chmod(path, finfo.Mode()); err != nil {
			return fmt.Errorf("error changing mode: %w", err)
		}
		// I believe the old version was using time.Now().UTC() to overcome an
		// invalid error from chtimes.....but here we lose hdr.AccessTime like this...
		if err := os.Chtimes(path, time.Now().UTC(), finfo.ModTime()); err != nil {
			return fmt.Errorf("error changing time: %w", err)
		}
	}
	return nil
}
