# Contributing to the Project

All contributors must add their software under the [Apache 2.0 license](LICENSE).


## Build

### Setup Requirements

To build, you'll need the [Go toolchain](https://go.dev/) installed and a GNU-compatible `make`.  You'll also need the source code downloaded.

Once you have that, you will most likely need to install some additional Go tools used by the build:

```shell
make go-dependencies
```

### Linux Requirements

Additionally, because this depends upon the container library, you'll need these installed when building on Linux:

* btrfs headers
* gpgme headers (not 100% required, but the vulnerability check will fail without it)

For Ubuntu, you can install it with:

```shell
apt-get install libbtrfs-dev libassuan-dev libgpgme-dev
```

For Arch Linux, you can install it with:

```shell
pacman -S btrfs-progs gpgme
```

For Fedora Linux, you can install it with:

```shell
dnf install libbtrfs-devel gpgme-devel libassuan-devel
```

### Run

While developing, you can run:

```shell
make
```

to run the standard developer tools to format the code, test it, and create the executable.

If you're looking just to compile the executable, you can run:

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
  1. Run `make distribution` to generate the distribution files in the `build/distribution` directory.
  2. Create the release in GitHub off the main branch, using the version number as the tag (in the form `v` + contents of `version.txt`).  The description should contain the `CHANGELOG.md` for this release.  The files should contain *all* the files in the `build/distribution` directory.
