---
title: Reverse Proxy and Subpath Deployment
lang: en
translation_key: reverse-proxy
---

# Reverse Proxy and Subpath Deployment

## Root-domain deployment

```nginx
server {
    listen 443 ssl;
    server_name drive.example.com;

    client_max_body_size 0;

    location / {
        proxy_set_header Host $host;
        proxy_set_header X-Real-IP $remote_addr;
        proxy_set_header X-Forwarded-For $proxy_add_x_forwarded_for;
        proxy_set_header X-Forwarded-Proto $scheme;
        proxy_request_buffering off;
        proxy_pass http://127.0.0.1:8089;
    }
}
```

Trust only the proxy that directly connects to the application:

```yaml
trusted-proxies:
  - 127.0.0.1
```

Do not trust `0.0.0.0/0` for convenience. If clients can bypass the proxy and reach go-drive directly, an attacker could forge `X-Forwarded-For`, corrupting logs and IP-based sign-in failure limits.

## Subpath deployment

To serve go-drive at `https://example.com/drive/`:

```yaml
api-path: /drive

web-dav:
  enabled: true
  prefix: /drive/dav
  allow-anonymous: false
  max-cache-items: 1000

trusted-proxies:
  - 127.0.0.1
```

```nginx
location /drive/ {
    proxy_set_header Host $host;
    proxy_set_header X-Real-IP $remote_addr;
    proxy_set_header X-Forwarded-For $proxy_add_x_forwarded_for;
    proxy_set_header X-Forwarded-Proto $scheme;
    proxy_request_buffering off;
    proxy_pass http://127.0.0.1:8089;
    client_max_body_size 0;
}
```

`api-path`, the WebDAV prefix, and the proxy location must agree. Restart go-drive after changing them.

## Upload considerations

- Nginx `client_max_body_size` must not be lower than the largest permitted upload.
- `proxy_request_buffering off` prevents the proxy from first writing the entire upload to its temporary directory.
- Proxy read/send timeouts must accommodate large transfers and ZIP downloads.
- When using a CDN, confirm it supports the required WebDAV methods or bypass the CDN for `/dav`.

## Diagnose the client IP

With the `GO_DRIVE_DEBUG` environment variable set, debug responses include information useful for confirming the client IP. Disable debug mode after diagnosis to avoid exposing extra implementation details.
