---
title: 反向代理和子路径部署
description: 将 go-drive 部署到 Nginx 等反向代理之后，正确配置 HTTPS 请求头、WebSocket 转发和子路径访问。
lang: zh-CN
translation_key: reverse-proxy
source_hash: d80e2a9b38d4a5967045870777536bb277340beacc4f1d7c7d9cf26035faeb19
---

# 反向代理和子路径部署

## 根域名部署

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

对应配置只信任直接连接应用的代理：

```yaml
trusted-proxies:
  - 127.0.0.1
```

不要为了省事信任 `0.0.0.0/0`。如果应用能被绕过代理直接访问，攻击者就可以伪造 `X-Forwarded-For`，影响日志和按 IP 的登录失败限制。

## 子路径部署

要通过 `https://example.com/drive/` 访问：

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

`api-path`、WebDAV 前缀和代理 location 必须保持一致。修改后重启 go-drive。

## 上传相关

- Nginx 的 `client_max_body_size` 不应小于允许上传的最大文件。
- `proxy_request_buffering off` 可避免代理先把整个大文件写入临时目录。
- 确保代理读取/发送超时足以覆盖大文件和 ZIP 下载。
- 如果使用 CDN，确认它支持需要的 WebDAV 方法，或让 `/dav` 绕过 CDN。

## 排查真实 IP

设置环境变量 `GO_DRIVE_DEBUG` 后，调试响应会包含帮助确认客户端 IP 的信息。排查完成后关闭调试模式，避免长期暴露额外实现信息。
