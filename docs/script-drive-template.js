// @name My Drive
// @version 1.0.2
// @uploader my-drive-uploader.js
// @description
// > Here and below is the drive's description
// > It supports `markdown`
// > It will be shown above the configuration form
//
// This template demonstrates both static configuration and a dynamic OAuth
// initialization step. Replace the example OAuth endpoints and scopes with
// the provider-specific values.

/// <reference path="./scripts/env/drive.d.ts"/>

function oauthRequest(utils) {
  return {
    endpoint: {
      authUrl: "https://example.com/oauth/authorize",
      tokenUrl: "https://example.com/oauth/token",
    },
    redirectUrl: utils.config.oauthRedirectURI,
    scopes: ["files.read"],
    text: "Authorize Example Cloud",
  };
}

function oauthCredentials(config) {
  return {
    clientID: config.client_id,
    clientSecret: config.client_secret,
  };
}

defineDrive(
  {
    // Static fields are saved as the Drive configuration before initConfig
    // runs. Field names beginning with '_' are reserved by the runtime.
    configForm: [
      {
        label: "Client ID",
        description: "The OAuth application's client ID.",
        type: "text",
        field: "client_id",
        required: true,
      },
      {
        label: "Client Secret",
        description: "The OAuth application's client secret.",
        type: "password",
        field: "client_secret",
        required: true,
      },
      entryCacheTTLFormItem("2h"),
    ],

    initConfig(config, utils) {
      const result = utils.oauthInitConfig(
        oauthRequest(utils),
        oauthCredentials(config)
      );
      return result.config;
    },

    init(data, config, utils) {
      utils.oauthInit(
        data,
        oauthRequest(utils),
        oauthCredentials(config)
      );
    },

    createInstance(config, utils) {
      return {
        entryCacheTTL: config.cache_ttl,
        oauth: utils.oauthLoad(oauthRequest(utils), oauthCredentials(config)),
      };
    },
  },
  {
    get(path) {
      console.debug("get", path);
      // TODO request
      return {
        path,
        isDir: false,
        size: -1,
        modTime: -1,
      };
    },

    list(path) {
      console.debug("list", path);
      // TODO request
      return [];
    },

    save(path, size, override, reader, progress) {
      // TODO upload reader.withProgress(progress)
    },

    makeDir(path) {
      // TODO request
    },

    copy(from, to, override, progress) {
      throw new UnsupportedError();
    },

    move(from, to, override, progress) {
      throw new UnsupportedError();
    },

    delete(path, progress) {
      console.debug("delete", path);
      // TODO request
    },

    getURL(entry) {
      console.debug("getURL", entry.path);
      // TODO
    },
  }
);
