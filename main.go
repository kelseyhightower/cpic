// Copyright 2014 Kelsey Hightower. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package main

import (
	"flag"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"os"
	"path"
	"time"

	"github.com/kelseyhightower/cpic/image"
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

func init() {
	flag.Usage = usage
	flag.StringVar(&config, "c", DefaultConfigPath, "coreos cloud config")
	flag.StringVar(&out, "o", "", "write output to file")
}

func copyConfig(iw *image.Writer, path string) error {
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
		if err := iw.WriteHeader(&h); err != nil {
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
	if err := iw.WriteHeader(&h); err != nil {
		return err
	}
	if _, err = io.Copy(iw, f); err != nil {
		return err
	}
	return nil
}

// Customize the CoreOS PXE image by creating the necessary OEM directories
// and copying the cloud-config file in place.
// See the "Adding a Custom OEM" section in the Booting CoreOS via PXE
// documentation - http://goo.gl/QrWvqN.
func customizeImage(in, out, config string) error {
	i, err := os.Open(in)
	if err != nil {
		return err
	}
	defer i.Close()
	ir, err := image.NewReader(i)
	if err != nil {
		return err
	}
	temp, err := ioutil.TempFile("", "")
	if err != nil {
		return err
	}
	iw, err := image.NewWriter(temp)
	if err != nil {
		return err
	}
	if err := image.Copy(iw, ir); err != nil {
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
