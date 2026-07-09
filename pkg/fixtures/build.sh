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
if [ -z "${JQ_CMD}" ] ; then
  if which jq >/dev/null 2>&1 ; then
    JQ_CMD=jq
  else
    2> echo "ERROR could not find jq; set JQ_CMD to force a JSON file parser."
    exit 1
  fi
fi

cd $( dirname "$0" ) || exit 1
tag=local-$$
tmpdir=$(mktemp -d)

for plat in arm64 amd64 ; do
  echo "Building ${plat} version"
  "${DOCKER_CMD}" build --platform linux/${plat} -t local-path/test-data:${tag} . || exit 2
  "${DOCKER_CMD}" save -o "${tmpdir}/t.tar" local-path/test-data:${tag} || exit 2
  "${DOCKER_CMD}" rmi local-path/test-data:${tag} || exit 2

  # The saved output is in a docker-archive format.  Make it explicitly clear.
  test -f docker-linux-${plat}.tar && rm docker-linux-${plat}.tar || true
  "${SKOPEO_CMD}" copy docker-archive://"${tmpdir}/t.tar" docker-archive://$(pwd)/docker-linux-${plat}.tar || exit 2
  test -f oci-linux-${plat}.tar && rm oci-linux-${plat}.tar || true
  "${SKOPEO_CMD}" copy docker-archive://"${tmpdir}/t.tar" oci-archive://$(pwd)/oci-linux-${plat}.tar || exit 2

  # Save off docker-archive information.
  ( cd "${tmpdir}" && tar xf t.tar ) || exit 2
  config=$( "${JQ_CMD}" -r '.[0].Config' "${tmpdir}/manifest.json" ) || exit 2
  tags=$( "${JQ_CMD}" -r '.[0].RepoTags.[0]' "${tmpdir}/manifest.json" ) || exit 2
  echo "${tags}" > linux-${plat}-tags.txt
  cp "${tmpdir}/${config}" linux-${plat}-manifest.json || exit 2
  chmod +w linux-${plat}-manifest.json || exit 2
  rm -rf "${tmpdir}/"* >/dev/null 2>&1 || true
done

test -f oci-indexed.tar && rm oci-indexed.tar || true
"${SKOPEO_CMD}" copy --multi-arch index-only docker://public.ecr.aws/docker/library/alpine:3.22.5 oci-archive:oci-indexed.tar || exit 3
cp oci-indexed.tar "${tmpdir}/t.tar" || exit 3
( cd "${tmpdir}" && tar xf t.tar ) || exit 3
digest=$( "${JQ_CMD}" -r '.manifests.[0].digest' "${tmpdir}/index.json" ) || exit 3
digest=$( echo "${digest}" | cut -f 2 -d : ) || exit 3
cp "${tmpdir}/index.json" oci-indexed-index.json || exit 3
chmod +w oci-indexed-index.json || exit 3
cp "${tmpdir}/blobs/sha256/${digest}" oci-indexed-manifest.json || exit 3
chmod +w oci-indexed-manifest.json || exit 3

# Not performed because the file is quite large for source control.
test -f oci-joined.tar && rm oci-joined.tar || true
"${SKOPEO_CMD}" copy --all docker://public.ecr.aws/docker/library/alpine:3.22.5 oci-archive:oci-joined.tar || exit 3

rm -rf "${tmpdir}" || true
