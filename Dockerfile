FROM frolvlad/alpine-glibc

ARG ARCH=amd64

WORKDIR /app

COPY build/go-drive_linux_${ARCH}.tar.gz app.tar.gz

RUN tar xf app.tar.gz && \
    rm app.tar.gz && \
    mv go-drive_linux_${ARCH}/* . && \
    rmdir go-drive_linux_${ARCH} && \
    mkdir data

ENTRYPOINT ["/app/go-drive", "-d", "/app/data", "-s", "/app/web"]

EXPOSE 8089
