# Contributing to the Project

All contributors must add their software under the [Apache 2.0 license](LICENSE).


## Build

### Setup Requirements

To build, you'll need the [Go toolchain](https://go.dev/) installed and a GNU-compatible `make`.  You'll also need the source code downloaded.

Once you have that, you will most likely need to install some additional Go tools used by the build:

```shell
make go-dependencies
```

### Linux + BTRFS Requirements

If you're running on Linux, the build will try to the BTRFS library to support that storage type (the code inherits this behavior from the podman 'storage' library) if you have the library installed.  To install it, you'll need to install the `btrfs` headers.

For Ubuntu, you can install it with:

```shell
apt-get install libbtrfs-dev
```

For Arch Linux, you can install it with:

```shell
pacman -S btrfs-progs
```

For Fedora Linux, you can install it with:

```shell
dnf install libbtrfs-devel
```

### Run

While developing, you can run:

```shell
make
```

to run the standard developer tools to format the code, test it, and create the executable.

For the full list of usable build targets, you can run:

```shell
make help
```

To just compile the executable, you can run:

```shell
make build
```

This will compile the executable into the file `imgoin`.


## Release

To release the product, you need to follow these steps:

1. Ensure all dependencies are up-to-date, while keeping the project at the current Go toolchain version (1.26):
   ```shell
   go get -t -u go@1.26 .
   ```
2. In the `dev` branch, which should contain the pending changes:
  1. Bump the version number in the [`version.txt`](version.txt) based on semantic versioning.
  2. Update the [`CHANGELOG.md`](CHANGELOG.md) to include the commit IDs into main, along with a high level description of the release.
  3. Create a merge pull request (PR) from `dev` into `main`.  The builds must pass.  After all checks pass, merge the PR.
3. Manually release.  *Note: in the future, we should aim to automate this.*
  1. Run `make distribution` to generate the distribution files in the `build/distribution` directory.  This can take a while; you probably want to run `make distribution -j16`
  2. Create the release in GitHub off the main branch, using the version number as the tag (in the form `v` + contents of `version.txt`).  The description should contain the `CHANGELOG.md` for this release.  The files should contain *all* the files in the `build/distribution` directory.
