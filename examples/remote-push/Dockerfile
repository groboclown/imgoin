FROM public.ecr.aws/docker/library/alpine:3

# This does not perform 'RUN', as that requires a local qemu environment.
# Instead, this just sets the user + sets a command.  In this way, any
# build tool should construct a valid cross-platform image for any platform.

USER daemon
CMD [ "/bin/uname", "-a" ]
