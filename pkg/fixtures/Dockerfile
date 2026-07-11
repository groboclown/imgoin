FROM scratch

COPY empty /empty

# This will just fail because empty isn't an executable.
# But that's fine - this image is for testing the generated output, not
# for running.
ENTRYPOINT [ "/empty" ]
