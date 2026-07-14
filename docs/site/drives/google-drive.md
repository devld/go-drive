---
title: Google Drive
lang: en
translation_key: drive-google-drive
---

# Google Drive

## Create an OAuth application

1. Create a project in the [Google Cloud Console](https://console.cloud.google.com/).
2. Enable the Google Drive API.
3. Configure the OAuth consent screen.
4. Add these scopes:
   - `https://www.googleapis.com/auth/drive`
   - `https://www.googleapis.com/auth/userinfo.profile`
5. Create a **Web application** OAuth client.
6. Add the redirect URI. The default is `https://go-drive.top/oauth_callback`.

For an external application in testing status, refresh tokens may be restricted by Google's testing-app policy. Before long-term operation, publish the application appropriately for its account type and organizational policy.

You can use your own callback page:

```yaml
oauth-redirect-uri: https://drive.example.com/oauth_callback
```

The URI in Google Cloud Console must exactly match the configuration.

## Add the Drive

Enter the Client ID, Client Secret, cache TTL, and **Proxy thumbnails** setting. After OAuth completes, select a personal or shared drive, save, and reload the Drive.

- Cache TTL defaults to `4h`; clear the cache after files are changed directly outside go-drive.
- Proxy thumbnails is enabled by default and is useful when browsers cannot access Google thumbnail URLs directly.

The Google Drive API does not have a traditional path model and allows duplicate names in one directory. When go-drive encounters duplicate entries, it appends the first six characters of the file ID to the name.

## Exporting native Google files

| Google type | Download format |
| --- | --- |
| Docs document | `.docx` |
| Sheets spreadsheet | `.xlsx` |
| Slides presentation | `.pptx` |
| Drawing | `.svg` |
| Apps Script | `.json` |

Google Drive does not support native folder copies. go-drive performs directory copies recursively.

> Google may change the Cloud Console interface, but the scopes and go-drive fields above reflect the current implementation.
