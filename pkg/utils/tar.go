package utils

import (
	"archive/tar"
	"io"
	"os"
	"path"
	"path/filepath"
)

func Untar(reader io.Reader, to string) error {
	tr := tar.NewReader(reader)

	decFile := func(hdr *tar.Header) error {
		file := path.Join(to, hdr.Name)
		err := os.MkdirAll(filepath.Dir(file), 0755)
		if err != nil {
			return err
		}
		fw, err := os.OpenFile(file, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, hdr.FileInfo().Mode())
		if err != nil {
			return err
		}
		defer fw.Close()

		_, err = io.Copy(fw, tr)
		return err
	}

	for {
		hdr, err := tr.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			return err
		}
		switch hdr.Typeflag {
		case tar.TypeDir:
			if err := os.MkdirAll(path.Join(to, hdr.Name), hdr.FileInfo().Mode()); err != nil {
				return err
			}
		case tar.TypeSymlink:
			if err = os.Symlink(hdr.Linkname, filepath.Join(to, hdr.Name)); err != nil {
				return err
			}
		default:
			if err := decFile(hdr); err != nil {
				return err
			}
		}
	}
	return nil
}
