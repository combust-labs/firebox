#!/usr/bin/env bash

set -o errexit
set -o nounset
set -o pipefail
set -o xtrace

# prerequisite: make docker-build
BASE_IMAGE=local/firebox

ROOT_DIR=$( cd "$(dirname "$0")" ; pwd -P )
mkdir -p "${ROOT_DIR}/../bin"
BIN_DIR=$( cd "${ROOT_DIR}/../bin" ; pwd -P )


ROOTFS_DIR=${BIN_DIR}/firebox-rootfs
rm -rf ${ROOTFS_DIR}
mkdir -p ${ROOTFS_DIR}

# extract a container's filesystem
container_id=$(docker create $BASE_IMAGE)
function cleanup() {
  docker rm "$container_id"
}
trap cleanup EXIT
docker export "$container_id" | tar -C "$ROOTFS_DIR" -xvf -

## TODO....