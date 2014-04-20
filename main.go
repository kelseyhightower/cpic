// Copyright 2014 Kelsey Hightower. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package main

import (
	"flag"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"path"

	"github.com/kelseyhightower/cpic/image"
)

var (
	config string
	output string
)

const version = "0.0.1"

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
	flag.PrintDefaults()
}

func init() {
	flag.Usage = usage
	flag.StringVar(&config, "c", DefaultConfigPath, "coreos cloud config")
	flag.StringVar(&output, "o", "", "write output to file")
}

func copyConfig(w *image.Writer, path string) error {
	for _, d := range []string{"usr", "usr/share", "usr/share/oem"} {
		if err := w.WriteDir(d); err != nil {
			return err
		}
	}
	if err := w.WriteFile(path, "usr/share/oem/cloud-config.yml"); err != nil {
		return err
	}
	return nil
}

// Customize the CoreOS PXE image by creating the necessary OEM directories
// and copying the cloud-config file in place.
// See the "Adding a Custom OEM" section in the Booting CoreOS via PXE
// documentation - http://goo.gl/QrWvqN.
func customizeImage(in, config string) (temp *os.File, err error) {
	i, err := os.Open(in)
	if err != nil {
		return
	}
	defer i.Close()
	r, err := image.NewReader(i)
	if err != nil {
		return
	}
	defer r.Close()
	temp, err = ioutil.TempFile("", "")
	if err != nil {
		return
	}
	defer temp.Close()
	w, err := image.NewWriter(temp)
	if err != nil {
		return
	}
	if err = copyConfig(w, config); err != nil {
		return
	}
	defer w.Close()
	if err = image.Copy(w, r); err != nil {
		return
	}
	return
}

func Version() string {
	return fmt.Sprintf("cpic version %s", version)
}

func main() {
	flag.Parse()
	if len(os.Args) > 1 {
		switch os.Args[1] {
		case "version":
			fmt.Println(Version())
			os.Exit(0)
		case "help":
			usage()
			fmt.Fprintf(os.Stderr, help)
			os.Exit(0)
		}
	}
	if flag.Arg(0) == "" {
		log.Fatal("cpic: no pxe image provided")
	}
	in := flag.Arg(0)
	out := path.Base(in)
	if output != "" {
		out = output
	}
	temp, err := customizeImage(in, config)
	if err != nil {
		log.Fatal(err)
	}
	if err := os.Rename(temp.Name(), out); err != nil {
		log.Fatal(err)
	}
}
