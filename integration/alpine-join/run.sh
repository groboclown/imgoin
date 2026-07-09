#!/bin/sh

if [ -z "${DOCKER_CMD}" ] ; then
  if which docker >/dev/null 2>&1 ; then
    DOCKER_CMD=docker
  elif which podman >/dev/null 2>&1 ; then
    DOCKER_CMD=podman
  else
    2> echo "ERROR could not find docker or podman; set DOCKER_CMD to force a container builder."
    exit 1
  fi
fi
if [ -z "${SKOPEO_CMD}" ] ; then
  if which skopeo >/dev/null 2>&1 ; then
    SKOPEO_CMD=skopeo
  else
    2> echo "ERROR could not find skopeo; set SKOPEO_CMD to force a JSON file parser."
    exit 1
  fi
fi

cd $(dirname "$0") || exit 1

tmpdir=$(mktemp -d) || exit 1

"${DOCKER_CMD}" build --platform linux/amd64 -t local-alpine:linux-amd64 . || exit 2
"${DOCKER_CMD}" save -o "${tmpdir}/alpine-linux-amd64.tar" local-alpine:linux-amd64 || exit 2
"${DOCKER_CMD}" build --platform linux/arm64 -t local-alpine:linux-arm64 . || exit 2
"${DOCKER_CMD}" save -o "${tmpdir}/alpine-linux-arm64.tar" local-alpine:linux-arm64 || exit 2

../../build/imgoin \
    --source image=docker-archive:"${tmpdir}/alpine-linux-amd64.tar" include-contents=y \
    --source image=docker-archive:"${tmpdir}/alpine-linux-arm64.tar" include-contents=y \
    --target image=oci-archive:"${tmpdir}/alpine-joined.tar" || exit 3

"${DOCKER_CMD}" load -i "${tmpdir}/alpine-joined.tar" || exit 4

if [ "$1" = "--force" ] ; then
    rm -rf "${tmpdir}" || true
else
    echo "Results saved off to ${tmpdir}"
fi
