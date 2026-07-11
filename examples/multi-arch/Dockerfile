FROM public.ecr.aws/docker/library/alpine:3 as build

# Add dependencies.
RUN set -x \
    && apk add build-base

COPY hello.c /code/hello.c

WORKDIR /code

# Compile and test.
RUN set -x \
    && gcc -static hello.c -o hello \
    && ./hello

# ------------------
FROM scratch

COPY --from=build /code/hello /hello

ENTRYPOINT [ "/hello" ]
