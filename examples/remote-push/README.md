# Push Multi-Arch Images into a Remote Repository

The concept of a container image repository has several parts:

* A *registry* host that provides access checks and access to repositories.  Examples of public registries include `docker.io` and `public.ecr.aws`.
* A *repository* resides within a hosting *registry*.  This has a name, such as `alpine`.
* A repository generally has multiple *tagged* images.  Each image has a digest (like the SHA256 hash), and the tag works like an alias for that digest.  This allows multiple tags to reference the same digest.

In this example, it builds two images for different architectures, pushes each one to a private registry using the same repository name.  Because they were build for different architectures, their contents differ, so they therefore have different digests.

This includes an example registry in `Dockerfile.registry`, which exists *only for example purposes and must never be used for production or anything outside a simple runtime environment.*  Because that uses a self-signed and self-contained certificate, interacting with it requres passing the `--tls-verify=false` argument with `podman` or `buildah`, and the `skip-tls-verify=yes` argument to `imgoin`.
