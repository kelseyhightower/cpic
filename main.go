// Copyright 2014 Kelsey Hightower. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package main

import (
	"compress/gzip"
	"flag"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"os"
	"path"
	"time"

	"github.com/surma/gocpio"
)

var (
	config string
	out    string
)

// DefaultConfigPath is the default CoreOS cloud config file path to copy
// into OEM PXE image.
var DefaultConfigPath = "cloud-config.yml"

var help = `
cpic creates an OEM CoreOS PXE image by copying the source PXE
image along with the CoreOS cloud-config.yml into a new PXE image.

The -o flag specifies the output file name. If not specified, the
output file name depends on the arguments and derives from the name
of the source PXE image. If the source PXE image is in the current
working directory it will be overwritten.

The -c flag specifies the cloud-config file name. If not specified,
the cloud-config file name will be set to "cloud-config.yml". The
cloud-config file must exist.
`

func usage() {
	fmt.Fprintf(os.Stderr, "usage: cpic [-c cloud-config] [-o output] coreos_production_pxe_image.cpio.gz\n")
	fmt.Fprintf(os.Stderr, help)
}

type ImageReader struct {
	z *gzip.Reader
	c *cpio.Reader
}

func NewImageReader(r io.Reader) (*ImageReader, error) {
	z, err := gzip.NewReader(r)
	if err != nil {
		return nil, err
	}
	return &ImageReader{z, cpio.NewReader(z)}, nil
}

func (i *ImageReader) Close() error {
	if err := i.z.Close(); err != nil {
		return err
	}
	return nil
}

type ImageWriter struct {
	z *gzip.Writer
	c *cpio.Writer
}

func NewImageWriter(w io.Writer) (*ImageWriter, error) {
	z := gzip.NewWriter(w)
	return &ImageWriter{z, cpio.NewWriter(z)}, nil
}

func (i *ImageWriter) Close() error {
	if err := i.c.Close(); err != nil {
		return err
	}
	if err := i.z.Close(); err != nil {
		return err
	}
	return nil
}

func init() {
	flag.Usage = usage
	flag.StringVar(&config, "c", DefaultConfigPath, "coreos cloud config")
	flag.StringVar(&out, "o", "", "write output to file")
}

func copyImage(dst *ImageWriter, src *ImageReader) error {
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

func copyConfig(iw *ImageWriter, path string) error {
	f, err := os.Open(path)
	if err != nil {
		return err
	}
	defer f.Close()
	for _, d := range []string{"usr", "usr/share", "usr/share/oem"} {
		h := cpio.Header{
			Name:  d,
			Mode:  0755,
			Mtime: time.Now().Unix(),
			Type:  cpio.TYPE_DIR,
		}
		if err := iw.c.WriteHeader(&h); err != nil {
			return err
		}
	}
	fi, err := f.Stat()
	if err != nil {
		return err
	}
	h := cpio.Header{
		Name:  "usr/share/oem/cloud-config.yml",
		Mode:  0644,
		Mtime: time.Now().Unix(),
		Size:  fi.Size(),
		Type:  cpio.TYPE_REG,
	}
	if err := iw.c.WriteHeader(&h); err != nil {
		return err
	}
	if _, err = io.Copy(iw.c, f); err != nil {
		return err
	}
	return nil
}


// Customize the CoreOS PXE image by creating the necessary OEM directories
// and copying the cloud-config file in place.
// See the "Adding a Custom OEM" section in the Booting CoreOS via PXE 
// documentation - http://goo.gl/QrWvqN. 
func customizeImage(in, out, config string) error {
	image, err := os.Open(in)
	if err != nil {
		return err
	}
	defer image.Close()
	ir, err := NewImageReader(image)
	if err != nil {
		return err
	}
	temp, err := ioutil.TempFile("", "")
	if err != nil {
		return err
	}
	iw, err := NewImageWriter(temp)
	if err != nil {
		return err
	}
	if err := copyImage(iw, ir); err != nil {
		return err
	}
	if err := copyConfig(iw, config); err != nil {
		return err
	}
	if err := ir.Close(); err != nil {
		return err
	}
	if err := iw.Close(); err != nil {
		return err
	}
	if err := os.Rename(temp.Name(), out); err != nil {
		return err
	}
	return nil
}

func main() {
	// Parse the commandline flags.
	flag.Parse()
	if flag.Arg(0) == "" {
		log.Fatal("cpic: no pxe image provided")
	}
	in := flag.Arg(0)
	out := path.Base(in)
	if out != "" {
		out = out
	}
	if err := customizeImage(in, out, config); err != nil {
		log.Fatal(err)
	}
}
