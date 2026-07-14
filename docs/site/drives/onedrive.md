---
title: OneDrive
lang: en
translation_key: drive-onedrive
---

# OneDrive

The OneDrive Drive supports Microsoft's global and 21Vianet-operated services, personal and organizational accounts, and SharePoint sites.

## Register an application

Register a web application in the appropriate Microsoft Entra admin portal:

- Global: <https://portal.azure.com/>
- 21Vianet: <https://portal.azure.cn/>

Create a client secret and add a Web redirect URI. The default is:

```text
https://go-drive.top/oauth_callback
```

You can configure your own callback page in `config.yml`:

```yaml
oauth-redirect-uri: https://drive.example.com/oauth_callback
```

The URI in the portal must exactly match the configuration.

## Permissions

A normal personal or organizational drive requires these delegated permissions:

- `User.Read`
- `Files.ReadWrite`
- `offline_access`

For a SharePoint site, use `Files.ReadWrite.All` instead of `Files.ReadWrite` and obtain administrator consent as required by the organization's policy.

## go-drive fields

| Field | Description |
| --- | --- |
| Region | Select `global` for the global service or `china` for 21Vianet |
| Tenant | `common`, `organizations`, or `consumers`; must match the application's supported account types |
| Client ID | Application (client) ID |
| Client Secret | Client-secret value, not the secret ID |
| SharePoint site | Optional, for example `https://example.sharepoint.com/sites/team` |
| Proxy upload/download | Forces traffic through go-drive |
| Cache TTL | Directory-entry cache time; zero or below disables caching |

The 21Vianet service typically uses the `common` tenant. After saving, follow the interface to complete OAuth, select the Drive or SharePoint site to map, and finally reload the Drive.

Client secrets expire. Create a new secret in the portal and update the Drive before expiration. Reload the Drive and clear its old cache after changing the SharePoint site or drive selection.

> Microsoft may change the portal interface, but the permission names and go-drive fields above reflect the current implementation.
