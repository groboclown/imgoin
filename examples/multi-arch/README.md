# Construct a Multi-Arch Index File

This example contains a [`Dockerfile`](Dockerfile) that constructs an image that, when run, outputs `Hello, world.` by running a simple C program.  The image only contains the static program, so is very light weight.

The [`run.sh`](run.sh) script generates the container image for both AMD x64 and ARM64 processors.  It then runs the `imgoin` program to create a manifest image that contains the full image of the two.

## Local Configuration

To run this, you'll need to have an environment that allows you to run both AMD64 and ARM64 instructions.

### Ubuntu x64

If you're on Ubuntu with an AMD64 (x64, x86_64) processor, you can configure your system like so:

```sh
DEBIAN_FRONTEND=noninteractive apt-get install -y \
  qemu-user-static binfmt-support podman
update-binfmts --enable qemu-aarch64
```

This will install:

* `qemu` to emulate running on ARM64 processors;
* `podman` to execute the image construction commands;
* `binfmt-support` to allow the Linux kernel to run `qemu` for emulation when asked to run an ARM64 program.

Then it will add into the kernel the support to run `qemu` for ARM64 programs.
