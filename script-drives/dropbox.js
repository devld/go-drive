// @name Dropbox
// @version 1.0.2
// @description Dropbox drive

/// <reference path="../docs/scripts/env/drive.d.ts"/>

/**
 * @param {RootConfig} config
 * @returns {OAuthRequest}
 */
function oauthReq(config) {
  return {
    Endpoint: {
      AuthURL:
        "https://www.dropbox.com/oauth2/authorize?token_access_type=offline",
      TokenURL: "https://api.dropboxapi.com/oauth2/token",
    },
    RedirectURL: config.OAuthRedirectURI,
    Scopes: [
      "files.metadata.write",
      "files.metadata.read",
      "files.content.write",
      "files.content.read",
      "account_info.read",
    ],
    Text: "Connect To Dropbox",
  };
}

defineDrive(
  {
    configForm: [
      {
        Label: "Client ID",
        Type: "text",
        Field: "client_id",
        Required: true,
      },
      {
        Label: "Client Secret",
        Type: "password",
        Field: "client_secret",
        Required: true,
      },
      entryCacheTTLFormItem("2h"),
    ],

    initConfig: function (ctx, config, utils) {
      var result = utils.OAuthInitConfig(oauthReq(utils.Config), {
        ClientID: config.client_id,
        ClientSecret: config.client_secret,
      });
      if (!result.OAuthHolder) return result.Config;

      var oauth = result.Config.OAuth || {};
      var data = request(result.OAuthHolder, ctx, "POST", "/users/get_current_account");
      oauth.Principal = data.name.display_name + "<" + data.email + ">";
      result.Config.OAuth = oauth;
      result.Config.Configured = true;
      return result.Config;
    },

    init: function (ctx, data, config, utils) {
      utils.OAuthInit(ctx, data, oauthReq(utils.Config), {
        ClientID: config.client_id,
        ClientSecret: config.client_secret,
      });
    },

    createInstance: function (ctx, config, utils) {
      return {
        entryCacheTTL: config.cache_ttl,
        oauth: utils.OAuthLoad(oauthReq(utils.Config), {
          ClientID: config.client_id,
          ClientSecret: config.client_secret,
        }),
      };
    },
  },
  {
    get: function (ctx, path) {
      if (DEBUG) console.log("get", path);
      var data = request(this.oauth, ctx, "POST", "/files/get_metadata", null, {
        path: "/" + path,
      });
      return toEntry(data);
    },

    save: function (ctx, path, size, override, reader) {
      if (size <= 150 * 1025 * 1024) {
        ctx.Total(size, true);
        uploadSmall(this, ctx, "/" + path, size, reader.ProgressReader(ctx));
      } else {
        uploadLarge(this, ctx, "/" + path, size, reader);
      }
    },

    makeDir: function (ctx, path) {
      request(this.oauth, ctx, "POST", "/files/create_folder_v2", null, {
        path: "/" + path,
      });
    },

    copy: function (ctx, from, to, override) {
      request(this.oauth, ctx, "POST", "/files/copy_v2", null, {
        from_path: "/" + from.Path,
        to_path: "/" + to,
      });
    },

    move: function (ctx, from, to, override) {
      request(this.oauth, ctx, "POST", "/files/move_v2", null, {
        from_path: "/" + from.Path,
        to_path: "/" + to,
      });
    },

    list: function (ctx, path) {
      if (DEBUG) console.log("list", path);
      var hasMore = true;
      var cursor;
      var result = [];
      while (hasMore) {
        var data = cursor
          ? request(
              this.oauth,
              ctx,
              "POST",
              "/files/list_folder/continue",
              null,
              {
                cursor: cursor,
              }
            )
          : request(this.oauth, ctx, "POST", "/files/list_folder", null, {
              path: path ? "/" + path : "",
            });
        hasMore = data.has_more;
        cursor = data.cursor;
        result = result.concat(
          data.entries.map(function (e) {
            return toEntry(e);
          })
        );
      }
      return result;
    },

    delete: function (ctx, path) {
      if (DEBUG) console.log("delete", path);
      request(this.oauth, ctx, "POST", "/files/delete_v2", null, {
        path: "/" + path,
      });
    },

    getURL: function (ctx, entry) {
      if (DEBUG) console.log("getURL", entry.Path);
      var data = request(
        this.oauth,
        ctx,
        "POST",
        "/files/get_temporary_link",
        null,
        {
          path: "/" + entry.Path,
        }
      );
      return { URL: data.link };
    },

    hasThumbnail: function (entry) {
      if (entry.IsDir) return false;
      if (entry.Size > 20 * 1024 * 1024) return false;
      var ext = pathUtils.ext(entry.Path);
      return (
        [
          "jpg",
          "jpeg",
          "png",
          "tiff",
          "tif",
          "gif",
          "webp",
          "ppm",
          "bmp",
        ].indexOf(ext) !== -1
      );
    },

    getThumbnail: function (ctx, entry) {
      var resp = request(
        this.oauth,
        ctx,
        "POST",
        "/files/get_thumbnail_v2",
        {
          "Dropbox-API-Arg": JSON.stringify({
            format: "png",
            mode: "strict",
            resource: {
              ".tag": "path",
              path: "/" + entry.Path,
            },
            size: "w256h256",
          }),
        },
        null,
        true
      );
      return resp.Body;
    },
  }
);

/**
 * @param {{ oauth: OAuthHolder }} drive
 * @param {TaskCtx} ctx
 * @param {number} size
 * @param {Reader} reader
 */
function uploadSmall(drive, ctx, path, size, reader) {
  request(
    drive.oauth,
    ctx,
    "POST",
    "/files/upload",
    {
      "Dropbox-API-Arg": JSON.stringify({
        path: path,
        mode: "overwrite",
        mute: true,
      }),
      "Content-Type": "application/octet-stream",
    },
    reader,
    true
  );
}

/**
 * @param {{ oauth: OAuthHolder }} drive
 * @param {TaskCtx} ctx
 * @param {number} size
 * @param {Reader} reader
 */
function uploadLarge(drive, ctx, path, size, reader) {
  var sessionId = request(
    drive.oauth,
    ctx,
    "POST",
    "/files/upload_session/start",
    {
      "Dropbox-API-Arg": JSON.stringify({}),
      "Content-Type": "application/octet-stream",
    },
    null,
    true
  ).session_id;

  var chunkSize = 150 * 1024 * 1024;
  var offset = 0;
  while (offset < size) {
    var length = Math.min(chunkSize, size - offset);
    request(
      drive.oauth,
      ctx,
      "POST",
      "/files/upload_session/append_v2",
      {
        "Dropbox-API-Arg": JSON.stringify({
          cursor: {
            offset: offset,
            session_id: sessionId,
          },
        }),
        "Content-Type": "application/octet-stream",
        "Content-Length": "" + length,
      },
      reader.LimitReader(length).ProgressReader(ctx),
      true
    );
    offset += length;
  }

  request(
    drive.oauth,
    ctx,
    "POST",
    "/files/upload_session/finish",
    {
      "Dropbox-API-Arg": JSON.stringify({
        commit: {
          path: path,
          mode: "overwrite",
          mute: true,
        },
        cursor: {
          offset: offset,
          session_id: sessionId,
        },
      }),
      "Content-Type": "application/octet-stream",
    },
    null,
    true
  );
}

function toEntry(data) {
  return {
    IsDir: data[".tag"] === "folder",
    Path: data.path_display.substring(1),
    Size: data[".tag"] === "folder" ? -1 : data.size,
    ModTime:
      data[".tag"] === "folder"
        ? -1
        : dayjs(data.server_modified).toDate().getTime(),
  };
}

/**
 * @param {OAuthHolder} oauthHolder
 * @param {Context} ctx
 * @param {HttpMethod} method
 * @param {string} [url]
 * @param {SM} [headers]
 * @param {any} [body]
 * @param {boolean} [contentApi]
 */
function request(oauthHolder, ctx, method, url, headers, body, contentApi) {
  var token = oauthHolder.Token(ctx);
  headers = Object.assign(
    {
      Authorization: token.TokenType + " " + token.AccessToken,
    },
    headers
  );

  if (!contentApi && body && typeof body === "object") {
    body = JSON.stringify(body);
    headers["Content-Type"] = "application/json";
  }

  var r = http(
    ctx,
    method,
    (contentApi
      ? "https://content.dropboxapi.com/2"
      : "https://api.dropboxapi.com/2") + url,
    headers,
    body
  );
  var isJSON =
    r.Headers.Get("Content-Type").toLowerCase().indexOf("application/json") >=
    0;

  var data = isJSON ? r.Text() : undefined;

  if (DEBUG) {
    console.log("http", method, url, body, r.Status, data);
  }

  if (isJSON) {
    try {
      data = JSON.parse(data);
    } catch (e) {
      throw ErrRemoteApi(500, "Failed to parse JSON: " + e);
    }
  }

  if (r.Status < 200 || r.Status >= 400) {
    r.Dispose();
    var error = data && data.error_summary;
    if (typeof error === "string") {
      if (error.indexOf("not_found") >= 0) {
        throw ErrNotFound();
      }
      if (error.indexOf("conflict") >= 0) {
        throw ErrNotAllowed();
      }
    }

    throw ErrRemoteApi(r.Status, error || data);
  }

  return data || r;
}
