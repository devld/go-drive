FROM alpine:latest AS builder

WORKDIR /work

RUN apk update && \
    apk add git tar go nodejs npm make

ARG BRANCH=master
ARG BUILD_VERSION

ADD https://api.github.com/repos/devld/go-drive/git/refs/heads/${BRANCH} /tmp/branch-info

RUN git clone https://github.com/devld/go-drive.git && \
    cd go-drive && \
    git checkout ${BRANCH} && \
    make target_name=go-drive_docker BUILD_VERSION=${BUILD_VERSION} all

FROM alpine:latest

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
