package image

import (
	"compress/gzip"
	"io"
	"os"
	"time"

	"github.com/surma/gocpio"
)

type Reader struct {
	z *gzip.Reader
	c *cpio.Reader
}

type Writer struct {
	z *gzip.Writer
	c *cpio.Writer
}

func NewReader(r io.Reader) (*Reader, error) {
	z, err := gzip.NewReader(r)
	if err != nil {
		return nil, err
	}
	return &Reader{z, cpio.NewReader(z)}, nil
}

func (r *Reader) Close() error {
	if err := r.z.Close(); err != nil {
		return err
	}
	return nil
}

func NewWriter(w io.Writer) (*Writer, error) {
	z := gzip.NewWriter(w)
	return &Writer{z, cpio.NewWriter(z)}, nil
}

func (w *Writer) Close() error {
	if err := w.c.Close(); err != nil {
		return err
	}
	if err := w.z.Close(); err != nil {
		return err
	}
	return nil
}

func (w *Writer) Write(p []byte) (n int, err error) {
	return w.c.Write(p)
}

func (w *Writer) WriteHeader(hdr *cpio.Header) error {
	return w.c.WriteHeader(hdr)
}

func Copy(dst *Writer, src *Reader) error {
	for {
		h, err := src.c.Next()
		if err != nil {
			return err
		}
		if h.IsTrailer() {
			break
		}
		if h.Type == cpio.TYPE_DIR {
			if h.Name == "." {
				continue
			}
			if err := dst.c.WriteHeader(h); err != nil {
				return err
			}
			continue
		}
		if err := dst.c.WriteHeader(h); err != nil {
			return err
		}
		if _, err = io.Copy(dst.c, src.c); err != nil {
			return err
		}
	}
	return nil
}

func (w *Writer) WriteFile(path, name string) error {
	f, err := os.Open(path)
	if err != nil {
		return err
	}
	defer f.Close()
	fi, err := f.Stat()
	if err != nil {
		return err
	}
	h := cpio.Header{
		Name:  name,
		Mode:  0644,
		Mtime: time.Now().Unix(),
		Size:  fi.Size(),
		Type:  cpio.TYPE_REG,
	}
	if err := w.WriteHeader(&h); err != nil {
		return err
	}
	if _, err = io.Copy(w, f); err != nil {
		return err
	}
	return nil
}

func (w *Writer) WriteDir(name string) error {
	h := cpio.Header{
		Name:  name,
		Mode:  0755,
		Mtime: time.Now().Unix(),
		Type:  cpio.TYPE_DIR,
	}
	if err := w.WriteHeader(&h); err != nil {
		return err
	}
	return nil
}
