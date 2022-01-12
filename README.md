# LAN-Share

[![Build Status](https://travis-ci.com/jinliming2/LAN-Share.svg?branch=main)](https://travis-ci.com/jinliming2/LAN-Share)
[![Go Report Card](https://goreportcard.com/badge/github.com/jinliming2/LAN-Share)](https://goreportcard.com/report/github.com/jinliming2/LAN-Share)

Share plain text, images and files in Local area network.

## Usage

```bash
$ lan-share -h
Usage of lan-share:
  -addr string
        Listen on address (default "[::]")
  -history int
        Chat history count, mind the memory usage (default 999)
  -limit int
        The byte size limit per message, default to 16Mib, large file please send via 'file' option (default 16777216)
  -port int
        Listen on port (default 8080)
  -version
        Show version and exit
```

After the server starts, open the address in your modern browser.

Supports:
* `Edge` >=79
* `Firefox` >=75
* `Chrome` >=76
* `Safari` ***Some features may be broken***
* `Opera` >=63

## Build

```bash
$ git clone https://github.com/jinliming2/LAN-Share.git
$ cd LAN-Share
$ make linux_amd64 # replace to your os and arch, or just run `make` to build all targets
```
