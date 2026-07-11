FROM public.ecr.aws/docker/library/alpine:3

# Setup the image
RUN set -x \
    && echo '#!/bin/sh' > /entrypoint \
    && echo 'echo "Hello, world."' >> /entrypoint \
    && chmod +x /entrypoint

USER daemon
CMD [ "/entrypoint" ]
