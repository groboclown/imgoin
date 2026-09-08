#!/bin/sh

set -e

# This script generates images for multiple architectures and push them into the sample registry.
# Because it uses the sample registry, which contains self-signed certificates, this must explicitly
# tell the podman and imgoin tools to ignore TLS issues.
cd "$( dirname "$0" )" || exit 1

REMOTE_REGISTRY=localhost:39443
REMOTE_REPOSITORY=hello-world
SELF_SIGNED_CERT="--tls-verify=false"
SELF_SIGNED_CERT_IMGOIN="skip-tls-verify=yes"

# First up, start and create the local registry.
podman build -t imgoin-local/registry -f Dockerfile.registry .
registry_container=$( podman run --rm -d -p 39443:39443 imgoin-local/registry )
echo "Follow along by running 'podman logs -f ${registry_container}'"
podman login $SELF_SIGNED_CERT --username that --password thing $REMOTE_REGISTRY

# Create the local image for amd64.
podman build \
  -t $REMOTE_REGISTRY/$REMOTE_REPOSITORY:v1-amd64 \
  --platform linux/amd64 \
  .

# Create the local image for arm64.
podman build \
  -t $REMOTE_REGISTRY/$REMOTE_REPOSITORY:v1-arm64 \
  --platform linux/arm64 \
  .

# Push the images into the registry.
podman push $SELF_SIGNED_CERT $REMOTE_REGISTRY/$REMOTE_REPOSITORY:v1-amd64
podman push $SELF_SIGNED_CERT $REMOTE_REGISTRY/$REMOTE_REPOSITORY:v1-arm64


# Example 1:
# Construct the manifest image to an archive file using imgoin, pulling from podman's image store.
imgoin \
    --source image=containers-storage:$REMOTE_REGISTRY/$REMOTE_REPOSITORY:v1-amd64 \
    --source image=containers-storage:$REMOTE_REGISTRY/$REMOTE_REPOSITORY:v1-arm64 \
    --target image=oci-archive:out.tar

# Example 2:
# Construct the manifest image in the remote registry, pulling from the remote registry.
imgoin \
    --source $SELF_SIGNED_CERT_IMGOIN image=docker://$REMOTE_REGISTRY/$REMOTE_REPOSITORY:v1-amd64 \
    --source $SELF_SIGNED_CERT_IMGOIN image=docker://$REMOTE_REGISTRY/$REMOTE_REPOSITORY:v1-arm64 \
    --target $SELF_SIGNED_CERT_IMGOIN image=docker://$REMOTE_REGISTRY/$REMOTE_REPOSITORY:v1

# Example 3:
# Construct the manifest image locally with Podman using the local image stores.
podman manifest create $SELF_SIGNED_CERT $REMOTE_REGISTRY/$REMOTE_REPOSITORY:v1
podman manifest add $SELF_SIGNED_CERT $REMOTE_REGISTRY/$REMOTE_REPOSITORY:v1-amd64 $REMOTE_REGISTRY/$REMOTE_REPOSITORY:v1
podman manifest add $SELF_SIGNED_CERT $REMOTE_REGISTRY/$REMOTE_REPOSITORY:v1-arm64 $REMOTE_REGISTRY/$REMOTE_REPOSITORY:v1

# Save the manifest image to a local file.
podman push localhost/$REMOTE_REGISTRY/$REMOTE_REPOSITORY:v1 oci-archive:out.tar

# Push the image into the remote store.
podman push $SELF_SIGNED_CERT $REMOTE_REGISTRY/$REMOTE_REPOSITORY:v1

podman kill $registry_container
