#!/bin/bash
set -ex
pwd
ls -l
echo $GITHUB_SHA > Release.txt
(
  cd cmd/plugin
  make
)
(
  cd cmd/iww
  make
)