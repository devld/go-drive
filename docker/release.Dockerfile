
# Used for docker buildx

FROM alpine:3.16

ARG TAG
ARG TARGETPLATFORM

WORKDIR /app

RUN set -e; \
    case "${TARGETPLATFORM}" in \
        linux/arm64*) \
            arch=arm64; \
            ;; \
        linux/arm*) \
            arch=arm; \
            ;; \
        linux/amd64) \
            arch=amd64; \
            ;; \
        *) \
            echo 'unsupported TARGETPLATFORM: ' ${TARGETPLATFORM}; \
            return 4 \
            ;;\
    esac; \
    name=go-drive_linux_musl_${arch}; \
    file=/tmp/${name}.tar.gz; \
    url=https://github.com/devld/go-drive/releases/download/${TAG}/${name}.tar.gz; \
    echo downloading ${url} ; \
    wget -q -O ${file} ${url} && \
    tar xf ${file} && \
    rm ${file} && \
    mv ${name}/* . && \
    rmdir ${name} && \
    mkdir data && \
    sed 's/data-dir: .\//data-dir: \/app\/data/; s/web-dir: .\/web/web-dir: \/app\/web/; s/lang-dir: .\/lang/lang-dir: \/app\/lang/' -i config.yml

ENTRYPOINT ["/app/go-drive", "-c", "/app/config.yml"]

EXPOSE 8089
