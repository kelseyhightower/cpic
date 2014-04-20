# cpic

The cpic commandline utility automates the CoreOS PXE image [customization process](https://coreos.com/docs/running-coreos/bare-metal/booting-with-pxe/#adding-a-custom-oem).

## Installation

Binary downloads on the way.

```
go install github.com/kelseyhightower/cpic
```

## Usage

```
cpic -h
usage: cpic [-c cloud-config] [-o output] coreos_production_pxe_image.cpio.gz
```

## Examples

### Customize a CoreOS PXE Image in place.

The follow command assumes the follow files exist in the current directory

- CoreOS cloud config named cloud-config.yml
- CoresOS PXE image

```
cpic coreos_production_pxe_image.cpio.gz
```

### Specify output file and cloud config file

```
cpio -o /tmp/coreos_production_pxe_image.cpio.gz -c /tmp/config.yml coreos_production_pxe_image.cpio.gz
```
