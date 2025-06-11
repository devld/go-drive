FROM golang:1.24.4-alpine AS golang

FROM alpine AS builder

ARG REF=heads/master
ARG BUILD_VERSION

WORKDIR /work

RUN apk update && \
    apk add git curl tar nodejs npm build-base
# Setup golang
COPY --from=golang /usr/local/go /usr/local/go
ENV PATH="/usr/local/go/bin:${PATH}"

ADD https://api.github.com/repos/devld/go-drive/git/refs/${REF} /tmp/ref-info

RUN git clone https://github.com/devld/go-drive.git && \
    cd go-drive && \
    git reset --hard ${REF} && \
    make target_name=go-drive_docker BUILD_VERSION=${BUILD_VERSION} all

FROM alpine

WORKDIR /app

COPY --from=builder /work/go-drive/build/go-drive_docker.tar.gz .

RUN tar xf go-drive_docker.tar.gz && \
    rm go-drive_docker.tar.gz && \
    mv go-drive_docker/* . && \
    rmdir go-drive_docker && \
    mkdir data && \
    sed 's/data-dir: .\//data-dir: \/app\/data/; s/web-dir: .\/web/web-dir: \/app\/web/; s/lang-dir: .\/lang/lang-dir: \/app\/lang/' -i config.yml

ENTRYPOINT ["/app/go-drive", "-c", "/app/config.yml"]

EXPOSE 8089
