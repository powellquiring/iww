#!/bin/bash
set -ex
pwd
ls -l
echo $GITHUB_SHA > Release.txt
exit 0

function build {
  FILENAME=extra-$1-$2
  echo "Building ${FILENAME}"
  GOOS=$1 GOARCH=$2 go build -o $FILENAME
}

# https://www.digitalocean.com/community/tutorials/how-to-build-go-executables-for-multiple-platforms-on-ubuntu-16-04
cd cmd/plugin
go build -v ./...
build linux amd64
build darwin amd64
build windows amd64