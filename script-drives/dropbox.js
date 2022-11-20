// Dropbox
// Dropbox drive

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

/**
 * @returns {FormItem[]}
 */
function initForm() {
  return [
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
  ];
}

defineCreate(function (ctx, config, utils) {
  var cred = utils.Data.Load("client_id", "client_secret", "_id");
  var resp = utils.OAuthGet(oauthReq(utils.Config), {
    ClientID: cred.client_id,
    ClientSecret: cred.client_secret,
  });
  return new DriveHolder(cred._id, resp, utils.CreateCache());
});

defineInitConfig(function (ctx, config, utils) {
  var cred = utils.Data.Load("client_id", "client_secret");

  var form = initForm();

  if (!cred.client_id || !cred.client_secret) {
    return {
      Configured: false,
      Form: form,
      Value: cred,
    };
  }

  var result = utils.OAuthInitConfig(oauthReq(utils.Config), {
    ClientID: cred.client_id,
    ClientSecret: cred.client_secret,
  });
  var initConfig = result.Config;
  var oauthResp = result.Response;
  if (!oauthResp) {
    result.Config.Form = form;
    result.Config.Value = cred;
    return result.Config;
  }

  var data = request(oauthResp, ctx, "POST", "/users/get_current_account");

  return {
    Configured: true,
    Form: form,
    Value: cred,
    OAuth: Object.assign(initConfig.OAuth, {
      Principal: data.name.display_name + "<" + data.email + ">",
    }),
  };
});

defineInit(function (ctx, data, config, utils) {
  var cred = utils.Data.Load("client_id", "client_secret", "_id");
  if (!cred.client_id || !cred.client_secret) {
    utils.Data.Save(data);
    return;
  }

  if (!cred._id) {
    var id = (Math.random() * 1000000).toFixed(0);
    utils.Data.Save({ _id: id });
  }

  utils.OAuthInit(ctx, data, oauthReq(utils.Config), {
    ClientID: cred.client_id,
    ClientSecret: cred.client_secret,
  });
});

/**
 * @typedef DriveDataHolder
 * @property {DriveCache} _cache
 * @property {OAuthResponse} _oauth
 * @property {Duration} _cacheTTL
 * @property {string} _rid
 */

/**
 * @typedef {Drive & DriveDataHolder} DropboxDrive
 */

/**
 * @type {DropboxDrive}
 */
var DriveImpl = {
  meta: function (ctx) {},
  get: function (ctx, path) {
    if (DEBUG) console.log("get", path);
    if (!path) {
      return {
        Path: "",
        IsDir: true,
        Size: -1,
        ModTime: -1,
      };
    }
    var cachedItem = this._cache.GetEntry(path);
    if (cachedItem) {
      return cacheItemToEntry(cachedItem);
    }
    var data = request(this._oauth, ctx, "POST", "/files/get_metadata", null, {
      path: "/" + path,
    });
    var entry = toEntry(this, data);
    this._cache.PutEntry(entry, this._cacheTTL);
    return entry;
  },
  save: function (ctx, path, size, override, reader) {
    if (size <= 150 * 1025 * 1024) {
      ctx.Total(size, true);
      uploadSmall(this, ctx, "/" + path, size, reader.ProgressReader(ctx));
    } else {
      uploadLarge(this, ctx, "/" + path, size, reader);
    }
    this._cache.Evict(path, false);
    this._cache.Evict(pathUtils.parent(path), false);
    return this.get(ctx, path);
  },
  makeDir: function (ctx, path) {
    request(this._oauth, ctx, "POST", "/files/create_folder_v2", null, {
      path: "/" + path,
    });
    this._cache.Evict(pathUtils.parent(path), false);
    return this.get(ctx, path);
  },
  copy: function (ctx, from, to, override) {
    var dat = from.Data();
    if (!dat || dat.d !== this._rid) {
      throw ErrUnsupported();
    }
    request(this._oauth, ctx, "POST", "/files/copy_v2", null, {
      from_path: "/" + from.Unwrap().Path(),
      to_path: "/" + to,
    });
    this._cache.Evict(to, true);
    this._cache.Evict(pathUtils.pa(to), false);
    return this.get(ctx, to);
  },
  move: function (ctx, from, to, override) {
    from = from.Unwrap();
    var dat = from.Data();
    if (!dat || dat.d !== this._rid) {
      throw ErrUnsupported();
    }
    request(this._oauth, ctx, "POST", "/files/move_v2", null, {
      from_path: "/" + from.Unwrap().Path(),
      to_path: "/" + to,
    });
    this._cache.Evict(to, true);
    this._cache.Evict(pathUtils.parent(to), false);
    this._cache.Evict(from.Path, true);
    this._cache.Evict(pathUtils.parent(from.Path()), false);
    return this.get(ctx, to);
  },
  list: function (ctx, path) {
    if (DEBUG) console.log("list", path);
    var _this = this;
    var cachedItems = this._cache.GetChildren(path);
    if (cachedItems) {
      return cachedItems.map(cacheItemToEntry);
    }

    var hasMore = true;
    var cursor;
    var result = [];
    while (hasMore) {
      var data = cursor
        ? request(
            this._oauth,
            ctx,
            "POST",
            "/files/list_folder/continue",
            null,
            {
              cursor: cursor,
            }
          )
        : request(this._oauth, ctx, "POST", "/files/list_folder", null, {
            path: path ? "/" + path : "",
          });
      hasMore = data.has_more;
      cursor = data.cursor;
      result = result.concat(
        data.entries.map(function (e) {
          return toEntry(_this, e);
        })
      );
    }

    this._cache.PutChildren(path, result, this._cacheTTL);
    return result;
  },
  delete: function (ctx, path) {
    if (DEBUG) console.log("delete", path);
    request(this._oauth, ctx, "POST", "/files/delete_v2", null, {
      path: "/" + path,
    });
    this._cache.Evict(pathUtils.parent(path), false);
    this._cache.Evict(path, true);
  },
  upload: function (ctx, path, size, override, config) {
    return useLocalProvider(size);
  },
  getReader: function (ctx, entry) {
    throw ErrUnsupported();
  },
  getURL: function (ctx, entry) {
    if (DEBUG) console.log("getURL", entry.Path);
    var data = request(
      this._oauth,
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
    // https://www.dropbox.com/developers/documentation/http/documentation#files-get_thumbnail
    if (entry.Size > 20 * 1024 * 1024) return false;
    var ext = pathUtils.ext(entry.Path);
    if (
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
      ].indexOf(ext) === -1
    ) {
      return false;
    }
    return true;
  },
  getThumbnail: function (ctx, entry) {
    var resp = request(
      this._oauth,
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
};

/**
 * @param {DropboxDrive} drive
 * @param {TaskCtx} ctx
 * @param {number} size
 * @param {Reader} reader
 */
function uploadSmall(drive, ctx, path, size, reader) {
  request(
    drive._oauth,
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
 * @param {DropboxDrive} drive
 * @param {TaskCtx} ctx
 * @param {number} size
 * @param {Reader} reader
 */
function uploadLarge(drive, ctx, path, size, reader) {
  var sessionId = request(
    drive._oauth,
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
      drive._oauth,
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
    drive._oauth,
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

/**
 * @param {DropboxDrive} drive
 */
function toEntry(drive, data) {
  return {
    IsDir: data[".tag"] === "folder",
    Path: data.path_display.substring(1),
    Size: data[".tag"] === "folder" ? -1 : data.size,
    ModTime:
      data[".tag"] === "folder"
        ? -1
        : dayjs(data.server_modified).toDate().getTime(),
    Data: { d: drive._rid },
  };
}

/**
 * @param {OAuthResponse} resp
 * @param {Context} ctx
 * @param {HttpMethod} method
 * @param {string} [url]
 * @param {SM} [headers]
 * @param {any} [body]
 * @param {boolean} [contentApi]
 */
function request(resp, ctx, method, url, headers, body, contentApi) {
  var token = resp.Token();
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

/**
 * @param {DriveCacheItem} item
 * @returns {Entry}
 */
function cacheItemToEntry(item) {
  return {
    IsDir: item.Type === "dir",
    Path: item.Path,
    Size: item.Size,
    ModTime: item.ModTime,
    Data: item.Data,
  };
}

/**
 * @constructor
 * @param {string} rid
 * @param {OAuthResponse} oauthResp
 * @param {DriveCache} cache
 */
function DriveHolder(rid, oauthResp, cache) {
  this._rid = rid;
  this._oauth = oauthResp;
  this._cache = cache;
  this._cacheTTL = ms(2 * 60 * 60 * 1000);
}

Object.keys(DriveImpl).forEach(function (fn) {
  DriveHolder.prototype[fn] = DriveImpl[fn];
});
