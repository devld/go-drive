
export default {
  fs: {
    name: 'File System',
    description: 'Local file system drive',
    configForm: [
      { field: 'path', label: 'Root', type: 'text', description: 'The path of root', required: true }
    ]
  },
  webdav: {
    name: 'WebDAV',
    description: 'WebDAV protocol drive',
    configForm: [
      { field: 'url', label: 'URL', type: 'text', description: 'The base URL', required: true },
      { field: 'username', label: 'Username', type: 'text', description: 'The username, if omitted, no authorization is required' },
      { field: 'password', label: 'Password', type: 'password' },
      { field: 'cache_ttl', label: 'CacheTTL', type: 'text', description: 'Cache time to live, if omitted, no cache. Valid time units are "ms", "s", "m", "h".' }
    ]
  },
  s3: {
    name: 'S3',
    description: 'S3 compatible storage',
    configForm: [
      { field: 'id', label: 'AccessKey', type: 'text', required: true },
      { field: 'secret', label: 'SecretKey', type: 'password', required: true },
      { field: 'bucket', label: 'Bucket', type: 'text', required: true },
      { field: 'path_style', label: 'PathStyle', type: 'checkbox', description: 'Force use path style api' },
      { field: 'region', label: 'Region', type: 'text' },
      { field: 'endpoint', label: 'Endpoint', type: 'text', description: 'The S3 api endpoint' },
      { field: 'proxy_upload', label: 'ProxyIn', type: 'checkbox', description: 'Upload files to server proxy' },
      { field: 'proxy_download', label: 'ProxyOut', type: 'checkbox', description: 'Download files from server proxy' },
      { field: 'cache_ttl', label: 'CacheTTL', type: 'text', description: 'Cache time to live, if omitted, no cache. Valid time units are "ms", "s", "m", "h".' }
    ]
  },
  onedrive: {
    name: 'OneDrive',
    description: 'OneDrive',
    configForm: [
      { field: 'client_id', label: 'Client Id', type: 'text', required: true },
      { field: 'client_secret', label: 'Client Secret', type: 'password', required: true },
      { field: 'proxy_upload', label: 'ProxyIn', type: 'checkbox', description: 'Upload files to server proxy' },
      { field: 'proxy_download', label: 'ProxyOut', type: 'checkbox', description: 'Download files from server proxy' },
      { field: 'cache_ttl', label: 'CacheTTL', type: 'text', description: 'Cache time to live, if omitted, no cache. Valid time units are "ms", "s", "m", "h".' }
    ]
  }

}
