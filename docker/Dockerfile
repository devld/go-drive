FROM golang:1.25.6-alpine AS golang

FROM alpine AS builder

ARG REF=heads/master
ARG BUILD_VERSION

WORKDIR /work

RUN apk update && \
    apk add git curl tar nodejs npm build-base
# Setup golang
COPY --from=golang /usr/local/go /usr/local/go
ENV PATH="/usr/local/go/bin:${PATH}"
# musl (Alpine 3.19+) no longer exposes the LFS64 symbols (pread64/pwrite64/off64_t)
# used by go-sqlite3 unless _LARGEFILE64_SOURCE is defined.
ENV CGO_CFLAGS="-D_LARGEFILE64_SOURCE"

ADD https://api.github.com/repos/devld/go-drive/git/refs/${REF} /tmp/ref-info

RUN git clone https://github.com/devld/go-drive.git && \
    cd go-drive && \
    git reset --hard ${REF} && \
    make target_name=go-drive_docker BUILD_VERSION=${BUILD_VERSION} all

FROM alpine

WORKDIR /app

RUN apk add --no-cache ffmpeg vips-tools

COPY --from=builder /work/go-drive/docker/setup-config.sh /usr/local/bin/setup-config

COPY --from=builder /work/go-drive/build/go-drive_docker.tar.gz .

RUN tar xf go-drive_docker.tar.gz && \
    rm go-drive_docker.tar.gz && \
    mv go-drive_docker/* . && \
    rmdir go-drive_docker && \
    mkdir data && \
    sh /usr/local/bin/setup-config config.yml

ENTRYPOINT ["/app/go-drive", "-c", "/app/config.yml"]

EXPOSE 8089
