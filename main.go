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
	configPath string
	out        string
)

// DefaultConfigPath is the default CoreOS cloud config file path
// to copy into OEM PXE image.
var DefaultConfigPath = "cloud-config.yml"

func init() {
	flag.Usage = usage
	flag.StringVar(&configPath, "c", DefaultConfigPath, "coreos cloud config path")
	flag.StringVar(&out, "o", "", "Write output to file")
}

var help = `
cpic creates an OEM CoreOS PXE image by copying the source PXE
image along with the CoreOS cloud-config.yml into a new PXE image.

The -o flag specifies the output file name. If not specified, the
output file name depends on the arguments and derives from the name
of the source PXE image. If the source PXE image is in the CWD
then the source PXE image will be renamed with the ".old" ext. 

The -c flag specifies the cloud-config file name. If not specified,
the cloud-config file name will be set to "cloud-config.yml". The
cloud-config file must exist.
`

func usage() {
	fmt.Fprintf(os.Stderr, "usage: cpic [-c cloud-config] [-o output] initramfs.cpio.gz\n")
	fmt.Fprintf(os.Stderr, help)
}

func handleError(err error) {
	if err != nil {
		log.Fatal(err)
	}
}

func copyArchive(dst *cpio.Writer, src *cpio.Reader) error {
	for {
		h, err := src.Next()
		if err != nil {
			log.Println(err.Error())
			break
		}
		if h.IsTrailer() {
			break
		}

		if h.Type == cpio.TYPE_DIR {
			if h.Name == "." {
				continue
			}
			err := dst.WriteHeader(h)
			if err != nil {
				return err
			}
			continue
		}
		err = dst.WriteHeader(h)
		if err != nil {
			return err
		}
		_, err = io.Copy(dst, src)
		if err != nil {
			log.Println(err.Error())
			return err
		}
	}
	return nil
}

func createOEM(dst *cpio.Writer, configPath string) error {
	dirs := []string{"usr", "usr/share", "usr/share/oem"}
	for _, d := range dirs {
		h := cpio.Header{
			Name:  d,
			Mode:  0755,
			Mtime: time.Now().Unix(),
			Type:  cpio.TYPE_DIR,
		}
		err := dst.WriteHeader(&h)
		if err != nil {
			return err
		}
	}
	f, err := os.Open(configPath)
	if err != nil {
		return err
	}
	defer f.Close()

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
	err = dst.WriteHeader(&h)
	if err != nil {
		return err
	}
	_, err = io.Copy(dst, f)
	if err != nil {
		return err
	}
	return nil
}

func cpioGzReader()

func main() {
	// Parse the commandline flags. 
	flag.Parse()
	if flag.Arg(0) == "" {
		log.Fatal("cpic: no pxe image provided")
	}
	pxeIn := flag.Arg(0)
	pxeOut := path.Base(pxeIn)
	if out != "" {
		pxeOut = out
	}

	// Setup the gzip reader.
	in, err := os.Open(pxeIn)
	handleError(err)
	gzr, err := gzip.NewReader(in)
	handleError(err)
	src := cpio.NewReader(gzr)

	// Setup the gzip writer.
	temp, err := ioutil.TempFile("", "")
	handleError(err)
	gzw := gzip.NewWriter(temp)
	handleError(err)
	dst := cpio.NewWriter(gzw)

	err = copyArchive(dst, src)
	handleError(err)

	err = createOEM(dst, configPath)
	handleError(err)

	// Cleanup.
	err = gzr.Close()
	handleError(err)
	err = in.Close()
	handleError(err)
	err = dst.Close()
	handleError(err)
	err = gzw.Close()
	handleError(err)
	err = temp.Close()
	handleError(err)

	err = os.Rename(temp.Name(), pxeOut)
	handleError(err)
}