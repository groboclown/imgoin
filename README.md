# imgoin

**Container image join command.**

A tool to control the construction multi-arch container images.  The current landscape of tools makes construction and management of multi-architecture containers difficult to handle.  This tool aims to simplify that.

This was based off of the [Skopeo](https://github.com/podman-container-tools/skopeo) `copy` command, combined with the `manifest` commands in [Podman](https://github.com/podman-container-tools/podman).

## Transports

This program relies on the [container-libs](https://github.com/podman-container-tools/container-libs/blob/main/image/transports/transports.go) transports to discover referenced images.  At the time of writing, this means that you can reference images at different locations by using a special format:

* Directory:
    * Format: `dir:DIR_NAME`
* Docker:
    * Format: `docker://[REGISTRY_NAME/]REPOSITORY_NAME[:(TAG | @INDEX)]`
* Docker Archive:
    * Format: `docker-archive://FILENAME.tar[:(TAG | @INDEX)]`
* OCI Archive:
    * Format: `oci-archive:FILENAME.tar[:image]`
* OCI Layout:
    * Format: `oci:DIRNAME[:(TAG | @INDEX)]`
* Openshift:
    * Format: `atomic:HOSTNAME/NAMESPACE/STREAM:TAG`
* SIF:
    * Format: `sif:ABSOLUTE_FILENAME`
* Tarball:
    * Format: `tarball:[FILENAME | -[:FILENAME | - ...]]`

## A Refresher: Manifest Images

There exist at least two "standards" for the contents of a container image - the OCI and Docker v2 formats.  These consist of a primary index, which references all the contents of the container.

...

Note that you can get a bit of this functionality through `skopeo` by joining the images together into a single one:

```shell
skopeo copy --all docker://public.ecr.aws/docker/library/alpine:3 oci-archive:joined-image.tar
```


## License

Released under the [Apache 2.0 license](LICENSE).
