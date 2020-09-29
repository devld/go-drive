FROM frolvlad/alpine-glibc

ARG ARCH=amd64

WORKDIR /app

COPY build/go-drive_linux_${ARCH}.tar.gz app.tar.gz
COPY docs/init.sql init.sql

RUN apk add sqlite && \
        mkdir data && \
        cat init.sql | sqlite3 data/data.db && \
        rm init.sql && \
        tar xf app.tar.gz && \
        rm app.tar.gz && \
        mv go-drive_linux_${ARCH}/* . && \
        rmdir go-drive_linux_${ARCH}

ENTRYPOINT ["/app/go-drive", "-d", "/app/data", "-s", "/app/web"]

EXPOSE 8089
