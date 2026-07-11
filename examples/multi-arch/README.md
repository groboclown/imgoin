# Construct a Multi-Arch Index File

This example contains a [`Dockerfile`](Dockerfile) that constructs an image that, when run, outputs `Hello, world.` by running a simple C program.  The image only contains the static program, so is very light weight.

The [`run.sh`](run.sh) script generates the container image for both AMD x64 and ARM64 processors.  It then runs the `imgoin` program to create a manifest image that contains the full image of the two.

The script includes running `imgoin` with the `include-contents=y` setting for the source images.  That means the target image contains the contents of the source images, so you only need to push the one image.
