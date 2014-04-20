package image

import (
	"compress/gzip"
	"io"

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

func (i *Reader) Close() error {
	if err := i.z.Close(); err != nil {
		return err
	}
	return nil
}

func NewWriter(w io.Writer) (*Writer, error) {
	z := gzip.NewWriter(w)
	return &Writer{z, cpio.NewWriter(z)}, nil
}

func (i *Writer) Close() error {
	if err := i.c.Close(); err != nil {
		return err
	}
	if err := i.z.Close(); err != nil {
		return err
	}
	return nil
}

func (i *Writer) Write(p []byte) (n int, err error) {
	return i.c.Write(p)
}

func (i *Writer) WriteHeader(hdr *cpio.Header) error {
	return i.c.WriteHeader(hdr)
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
