#!/bin/bash

# Original version was from:
# https://github.com/taskcluster/slugid-go/blob/master/.travis_rename_releases.sh

set -euxo pipefail

binary='docker-image-cleaner'

# The deploy is called per arch and os combination - so we only release one file here.
# We just need to work out which file we built, rename it to something unique, and
# set an environment variable for its location that we can use in .travis.yml for
# publishing back to github.

# all cross-compiled binaries are in subdirectories: ${GOPATH}/bin/${GIMME_OS}_${GIMME_ARCH}/
# linux 64 bit, not cross-compiled, breaks this rule and is in ${GOPATH}/bin
# therefore move it to match the convention of the others, to simplify subsequent steps
# note: we don't know what we built, so only move it if we happen to be linux amd64 travis job
if [ -f "${GOPATH}/bin/${binary}" ]; then
  mkdir "${GOPATH}/bin/linux_amd64"
  mv "${GOPATH}/bin/${binary}" "${GOPATH}/bin/linux_amd64/${binary}"
fi

# Ah, Window's file extensions.
if [ "${GIMME_OS}" == "windows" ]; then
  FILE_EXT=".exe"
else
  FILE_EXT=""
fi

# Let's rename the release file because it has a 1:1 mapping with what it is called on
# github releases, and therefore the name for each platform needs to be unique so that
# they don't overwrite each other. Set a variable that can be used in .travis.yml
export RELEASE_FILE="${TRAVIS_BUILD_DIR}/${binary}-${GIMME_OS}-${GIMME_ARCH}${FILE_EXT}"
mv "${GOPATH}/bin/${GIMME_OS}_${GIMME_ARCH}/${binary}${FILE_EXT}" "${RELEASE_FILE}"
