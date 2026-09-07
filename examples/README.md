# Examples

Here you'll find examples of using the `imgoin` tool.

* [multi-arch](multi-arch/README.md) contains a full example of a container that runs a native program.  The example script builds the container locally for each supported architecture then joins them into a single image.
* [remote-push](remote-push/README.md) shows how to use a container image registry host to store an image in the host using the equivalent of other commands.


## Multi-Architecture Local Configuration

Many of these examples require running them on a computer that has a configuration that allows for creating images with multiple architectures.  Not all systems support this, and, for those that do, they require some unusual configuration.

### Ubuntu x64

If you're on Ubuntu with an AMD64 (x64, x86_64) processor, you can configure your system like so:

```sh
DEBIAN_FRONTEND=noninteractive apt-get install -y \
  qemu-user-binfmt binfmt-support podman
update-binfmts --enable qemu-aarch64
```

This will install:

* `qemu` to emulate running on ARM64 processors;
* `podman` to execute the image construction commands;
* `binfmt-support` to allow the Linux kernel to run `qemu` for emulation when asked to run an ARM64 program.

Then it will add into the kernel the support to run `qemu` for ARM64 programs.

### Amazon Linux 2023

Amazon Linux 2023 does not ship with Podman, but it does have [Buildah](https://github.com/podman-container-tools/buildah), the build-side of the Podman tools.

```sh
dnf install buildah qemu-user-static qemu-user-binfmt
systemctl start systemd-binfmt
```

With the `buildah` tool, you'll want to *build* the container using the same arguments as you would with `podman` or `docker`.  Instead of `buildah save -o FILENAME IMAGE_TAG`, you'd use `buildah push IMAGE_TAG oci-archive:FILENAME`

### Arch Linux x64

```sh
pacman -Sy \
  qemu-user-static binfmt-support podman
update-binfmts --enable qemu-aarch64
```

### Others

Do you know of a way to configure this on other OSes or architectures?  Open a ticket on this project with a suggestion!
