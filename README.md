# imgoin

**Container image join command.**

A tool to control the construction of manifest container images - an image that contains other images differentiated by local user characteristics.  Generally, this means allowing the construction of a single image that allows use by multiple operating systems and/or architectures.  The current landscape of tools makes construction and management of multi-architecture containers difficult to handle.  This tool aims to simplify that.

This fills a small gap in the [container tools](https://github.com/podman-container-tools) toolchain to allow creating a multi-architecture image with a single command.  This was based off of the [Skopeo](https://github.com/podman-container-tools/skopeo) `copy` command, combined with the `manifest` commands in [Podman](https://github.com/podman-container-tools/podman).  

Internally, we pronounce the tool like 'im join', a portmanteau of "image" and "join", even though it looks like a spelling of "I'm goin'".

## Usage

When you run the program, it takes the general form of:

```shell
imgoin --source IMAGE 1 SETTINGS --source IMAGE 2 SETTINGS --target TARGET SETTINGS
```

where each image's settings begin with either `--source` (at least 1) or `--target` (exactly 1).

The tool reads settings in the form `setting-name=setting_value`.  You can either specify them in that form, or you can include the argument `@FILENAME` to have the tool read the settings from that file.

### Shared Image Settings

### Source Settings

### Target Settings

### Storing Settings in a File

Say you run the program like:

```shell
imgoin --source image=oci-archive:a.tar @source.txt --target oci-archive:b.tar
```

This will read settings for the source image from the file `source.txt`.  Note that, because this argument comes *after* the `image=` argument, the settings inside `source.txt` can overwrite the `image=` argument.  Likewise, if the argument ordering was swapped (`--source @source.txt image=oci-archive:a.tar`), then the `image=` argument would overwrite that setting from the `source.txt` file.

The format for the file looks like this:

```
name1=value
name2="A value with spaces or special characters like ', but \" needs escaping."
name3='Single quoted text works the same as ", but requires escaping \' characters."
name4=Or\ escape\ using\ backslashes\ rather\ than\ \"\ or\ \'.
# A comment
name5=value # or end a line with a comment.
# You can also have values on the same line.
name6=value name7=value
```

### Image Transports

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

## Examples

You can find a set of examples for using the tool in combination with other container tools in the [`examples`] directory.




Note that you can get a bit of this functionality through `skopeo` by joining the images together into a single one:

```shell
skopeo copy --all docker://public.ecr.aws/docker/library/alpine:3 oci-archive:joined-image.tar
```

## A Refresher: Manifest Images

There exist several formats for container images, with the primary types of the OCI and Docker v2 formats.  They have a general form of a main manifest file that points to internally stored "blobs" by their hashes.

Some variants of these formats allows for "manifest lists", meaning that the image stores one or more "things" (such as images, image references, or artifacts like bill of materials).  These images have a primary manifest which points at the index, and the index lists all the contents.

The items in the index list includes classifiers that allows the container tools to select the appropriate image for the current environment.  For example, the Linux Alpine 3 index contains, among its many entries, one entry with:

```json
{
    "platform": {
        "architecture": "amd64",
        "os": "linux"
    }
}
```

and another with:

```json
{
    "platform": {
        "architecture": "arm",
        "os": "linux",
        "variant": "v6"
    }
}
```


## License

Released under the [Apache 2.0 license](LICENSE).
