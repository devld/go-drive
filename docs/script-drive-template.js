// @name My Drive
// @version 1.0.0
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
    Endpoint: {
      AuthURL: "https://example.com/oauth/authorize",
      TokenURL: "https://example.com/oauth/token",
    },
    RedirectURL: utils.Config.OAuthRedirectURI,
    Scopes: ["files.read"],
    Text: "Authorize Example Cloud",
  };
}

function oauthCredentials(config) {
  return {
    ClientID: config.client_id,
    ClientSecret: config.client_secret,
  };
}

defineDrive(
  {
    // Static fields are saved as the Drive configuration before initConfig
    // runs. Field names beginning with '_' are reserved by the runtime.
    configForm: [
      {
        Label: "Client ID",
        Description: "The OAuth application's client ID.",
        Type: "text",
        Field: "client_id",
        Required: true,
      },
      {
        Label: "Client Secret",
        Description: "The OAuth application's client secret.",
        Type: "password",
        Field: "client_secret",
        Required: true,
      },
      entryCacheTTLFormItem("2h"),
    ],

    initConfig: function (ctx, config, utils) {
      var result = utils.OAuthInitConfig(
        oauthRequest(utils),
        oauthCredentials(config)
      );
      return result.Config;
    },

    init: function (ctx, data, config, utils) {
      utils.OAuthInit(
        ctx,
        data,
        oauthRequest(utils),
        oauthCredentials(config)
      );
    },

    createInstance: function (ctx, config, utils) {
      return {
        entryCacheTTL: config.cache_ttl,
        oauth: utils.OAuthLoad(oauthRequest(utils), oauthCredentials(config)),
      };
    },
  },
  {
    get: function (ctx, path) {
      if (DEBUG) console.log("get", path);
      // TODO request
      return {
        Path: path,
        IsDir: false,
        Size: -1,
        ModTime: -1,
      };
    },

    list: function (ctx, path) {
      if (DEBUG) console.log("list", path);
      // TODO request
      return [];
    },

    save: function (ctx, path, size, override, reader) {
      // TODO upload
    },

    makeDir: function (ctx, path) {
      // TODO request
    },

    copy: function (ctx, from, to, override) {
      throw ErrUnsupported();
    },

    move: function (ctx, from, to, override) {
      throw ErrUnsupported();
    },

    delete: function (ctx, path) {
      if (DEBUG) console.log("delete", path);
      // TODO request
    },

    getURL: function (ctx, entry) {
      if (DEBUG) console.log("getURL", entry.Path);
      // TODO
    },
  }
);
