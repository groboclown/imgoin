# imgoin

**Container image join command.**

A tool to control the construction of manifest container images - an image that contains other images differentiated by local user characteristics.  Generally, this means allowing the construction of a single image that allows use by multiple operating systems and/or architectures.  The current landscape of tools makes construction and management of multi-architecture containers difficult to handle.  This tool aims to simplify that.

This fills a small gap in the [container tools](https://github.com/podman-container-tools) toolchain to allow creating a multi-architecture image with a single command.  This was based off of the [Skopeo](https://github.com/podman-container-tools/skopeo) `copy` command, combined with the `manifest` commands in [Podman](https://github.com/podman-container-tools/podman).  

Internally, we pronounce the tool like 'em join', a portmanteau of "image" and "join", even though it looks like a spelling of "I'm goin'".

## Usage

When you run the program, it takes the general form of:

```shell
imgoin --source IMAGE 1 SETTINGS --source IMAGE 2 SETTINGS --target TARGET SETTINGS
```

where each image's settings begin with either `--source` (at least 1) or `--target` (exactly 1).

The tool reads settings in the form `setting-name=setting_value`.  You can either specify them in that form, or you can include the argument `@FILENAME` to have the tool read the settings from that file.

### Shared Image Settings

* `image=IMAGE-LOCATION`
    * Location of the image, in the format 'transport:name' (see the list of supported transports below). Required for all images.  Aliases: 'img'.
* `auth-file=FILENAME`
    * Path of the registry credentials file. Default is `${XDG_RUNTIME_DIR}/containers/auth.json`.
* `credentials=USERNAME[:PASSWORD]`
    * Credentials for accessing the registry.  Cannot be used with the username setting.  Aliases: 'creds'.
* `username=USERNAME`
    * Username for accessing the registry.  Cannot be used with the credentials setting.  If given, you must also provide the password setting, even if empty.  Aliases: 'user'.
* `password=PASSWORD`
    * Password for accessing the registry.  Must be used with the 'username' setting.  Aliases: 'pass'.
* `anonymous=yes|no`
    * Set to 'yes' to force anonymous registry access.  This overrides username or credentials settings.  Aliases: 'no-credentials'.
* `registry-token=TOKEN`
    * Provides a bearer token for accessing the registry.  Aliases: 'token'.
* `certificate-dir=DIRNAME`
    * Directory containing *.{crt,cert,key} files for contacting the registry.  Aliases: 'cert-path'.
* `skip-tls-verify=yes|no`
    * Set to 'yes' to force ignoring HTTPS + certificate verification.
* `tls-details=FILENAME`
    * Path to a containers-tls-details.yaml(5) file.
* `user-agent-prefix=PREFIX`
    * Prefix to add to the user agent string.
* `registries-conf=FILENAME`
    * Path to the registries.conf file.
* `registries.d=DIR`
    * Use registry configuration files in 'DIR' (e.g. for container signature storage).
* `blobs-dir=DIRNAME`
    * DIR to use to share blobs across OCI repositories.
* `daemon-host=HOSTNAME[:PORT]`
    * Use docker daemon host at HOSTNAME (docker-daemon: transports only).
* `tmp-dir=DIR`
    * Path to use for big temporary files.

### Source Settings

* `include-contents=yes|no`
    * Set to 'yes' to have the target image save all the contents of this source image.  Without it, the target image will store just a reference.  Most multi-architecture images store just a reference.
* `policy=FILE`
    * Use a signature policy file at the given path.  Used for checking the source images.
* `insecure-policy=yes|no`
    * Set to 'yes' to force the program to ignore signatures on source images.
* `require-signed=yes|no`
    * Set to 'yes' for require all source images to have a valid signature.  Source images without a signature will cause a failure.

Additionally, you can pass image transport in the form 'image=sha256:ABC...', which allows you to explicitly declare the image reference information.

* `manifest-size=SIZE_IN_BYTES`
    * Number of bytes of the referenced image's manifest.
* `architecture=ARCH`
    * Name of the image's target architecture.  Aliases: 'arch'.
* `os=OS`
    * Name of the image's target operating system.
* `os-version=VERSION`
    * Version of the image's target operating system.
* `os-feature=FEATURE`
    * A feature required for the image's target operating system.  You may specify this more than once.
* `os-variant=VARIANT`
    * The image's target operating system variant.
* `annotation=KEY:VALUE`
    * Add a annotation to the image reference.  You may provide multiple of these.

### Target Settings

* `tag=TAG`
    * Give the constructed image a tag.  Only some output transports support this.  You may provide multiple of these.

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
# You can also have multiple settings on the same line.
name6=value name7=value
```

### Image Transports

This program relies on the [container-libs](https://github.com/podman-container-tools/container-libs/blob/main/image/transports/transports.go) transports to discover referenced images.  At the time of writing, this means that you can reference images at different locations by using a special format:

* Directory:
    * Format: `dir:DIR_NAME`
* Docker:
    * Format: `docker://[REGISTRY_NAME/]REPOSITORY_NAME[:(TAG | @INDEX)]`
    * Allows accessing a registry host.
* Local Podman Images:
    * Format: `containers-storage:REPOSITORY_NAME[:(TAG | @INDEX)]`
    * Allows for accessing images stored in the current user's Podman image list.
* Docker Archive:
    * Format: `docker-archive://FILENAME.tar[:(TAG | @INDEX)]`
    * The simple, single-image tar file format.
* OCI Archive:
    * Format: `oci-archive:FILENAME.tar[:image]`
    * Stores an image in a tar file, using the OCI manifest formats.  Allows for a file containing multiple images, artifacts, or references to images.
* OCI Layout:
    * Format: `oci:DIRNAME[:(TAG | @INDEX)]`
    * A local file layout for the OCI manifest format image.  Equivalent to an untarred `oci-archive` file.
* Openshift:
    * Format: `atomic:HOSTNAME/NAMESPACE/STREAM:TAG`
* SIF:
    * Format: `sif:ABSOLUTE_FILENAME`
* Tarball:
    * Format: `tarball:[FILENAME | -[:FILENAME | - ...]]`

## Examples

You can find a set of examples for using the tool in combination with other container tools in the [`examples`](examples) directory.  It shows how this tool replaces or enhances other tools' functionality.

## Limitations

This tool does not sign the generated (target) image.  Signing should happen through other tools, such as `skopeo`.


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

Some of the code comes from the [Skopeo](https://github.com/podman-container-tools/skopeo) project, which also was released under the Apache 2.0 license.
