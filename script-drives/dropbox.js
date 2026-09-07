// @name Dropbox
// @version 1.0.9
// @description Dropbox drive

/**
 * @param {RootConfig} config
 * @returns {OAuthRequest}
 */
function oauthReq(config) {
  return {
    endpoint: {
      authUrl:
        "https://www.dropbox.com/oauth2/authorize?token_access_type=offline",
      tokenUrl: "https://api.dropboxapi.com/oauth2/token",
    },
    redirectUrl: config.oauthRedirectURI,
    scopes: [
      "files.metadata.write",
      "files.metadata.read",
      "files.content.write",
      "files.content.read",
      "account_info.read",
    ],
    text: "Connect To Dropbox",
  };
}

defineDrive(
  {
    configForm: [
      {
        label: "Client ID",
        type: "text",
        field: "client_id",
        required: true,
      },
      {
        label: "Client Secret",
        type: "password",
        field: "client_secret",
        required: true,
      },
      entryCacheTTLFormItem("2h"),
    ],

    initConfig(config, utils) {
      const result = utils.oauthInitConfig(oauthReq(utils.config), {
        clientID: config.client_id,
        clientSecret: config.client_secret,
      });
      if (!result.oauthHolder) return result.config;

      const data = request(result.oauthHolder, "POST", "/users/get_current_account");
      return {
        configured: true,
        oauth: {
          url: result.config.oauth.url,
          text: result.config.oauth.text,
          principal: data.name.display_name + "<" + data.email + ">",
        },
        form: result.config.form,
        value: result.config.value,
      };
    },

    init(data, config, utils) {
      utils.oauthInit(data, oauthReq(utils.config), {
        clientID: config.client_id,
        clientSecret: config.client_secret,
      });
    },

    createInstance(config, utils) {
      return {
        entryCacheTTL: config.cache_ttl,
        oauth: utils.oauthLoad(oauthReq(utils.config), {
          clientID: config.client_id,
          clientSecret: config.client_secret,
        }),
      };
    },
  },
  {
    get(path) {
      console.debug("get", path);
      const data = request(this.oauth, "POST", "/files/get_metadata", null, {
        path: "/" + path,
      });
      return toEntry(data);
    },

    save(path, size, override, reader, onProgress) {
      if (size <= 150 * 1025 * 1024) {
        uploadSmall(this, "/" + path, size, reader);
      } else {
        uploadLarge(this, "/" + path, size, reader);
      }
    },

    makeDir(path) {
      request(this.oauth, "POST", "/files/create_folder_v2", null, {
        path: "/" + path,
      });
    },

    copy(from, to, override) {
      request(this.oauth, "POST", "/files/copy_v2", null, {
        from_path: "/" + from.path,
        to_path: "/" + to,
      });
    },

    move(from, to, override) {
      request(this.oauth, "POST", "/files/move_v2", null, {
        from_path: "/" + from.path,
        to_path: "/" + to,
      });
    },

    list(path) {
      console.debug("list", path);
      let hasMore = true;
      let cursor;
      const result = [];
      while (hasMore) {
        const data = cursor
          ? request(this.oauth, "POST", "/files/list_folder/continue", null, {
              cursor,
            })
          : request(this.oauth, "POST", "/files/list_folder", null, {
              path: path ? "/" + path : "",
            });
        hasMore = data.has_more;
        cursor = data.cursor;
        result.push(...data.entries.map(toEntry));
      }
      return result;
    },

    delete(path) {
      console.debug("delete", path);
      request(this.oauth, "POST", "/files/delete_v2", null, {
        path: "/" + path,
      });
    },

    getURL(entry) {
      console.debug("getURL", entry.path);
      const data = request(
        this.oauth,
        "POST",
        "/files/get_temporary_link",
        null,
        {
          path: "/" + entry.path,
        }
      );
      return { url: data.link };
    },

    getThumbnail(entry) {
      if (!dropboxCanThumbnail(entry)) {
        throw new UnsupportedError();
      }
      const resp = request(
        this.oauth,
        "POST",
        "/files/get_thumbnail_v2",
        {
          "Dropbox-API-Arg": JSON.stringify({
            format: "png",
            mode: "strict",
            resource: {
              ".tag": "path",
              path: "/" + entry.path,
            },
            size: "w256h256",
          }),
        },
        null,
        true
      );
      return resp.body;
    },
  }
);

/**
 * @param {{ oauth: OAuthHolder }} drive
 * @param {string} path
 * @param {number} size
 * @param {Reader} reader
 */
function uploadSmall(drive, path, size, reader) {
  request(
    drive.oauth,
    "POST",
    "/files/upload",
    {
      "Dropbox-API-Arg": JSON.stringify({
        path,
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
 * @param {string} path
 * @param {number} size
 * @param {Reader} reader
 */
function uploadLarge(drive, path, size, reader) {
  const sessionId = request(
    drive.oauth,
    "POST",
    "/files/upload_session/start",
    {
      "Dropbox-API-Arg": JSON.stringify({}),
      "Content-Type": "application/octet-stream",
    },
    null,
    true
  ).session_id;

  const chunkSize = 150 * 1024 * 1024;
  let offset = 0;
  while (offset < size) {
    const length = Math.min(chunkSize, size - offset);
    request(
      drive.oauth,
      "POST",
      "/files/upload_session/append_v2",
      {
        "Dropbox-API-Arg": JSON.stringify({
          cursor: {
            offset,
            session_id: sessionId,
          },
        }),
        "Content-Type": "application/octet-stream",
        "Content-Length": "" + length,
      },
      reader.limitReader(length),
      true
    );
    offset += length;
  }

  request(
    drive.oauth,
    "POST",
    "/files/upload_session/finish",
    {
      "Dropbox-API-Arg": JSON.stringify({
        commit: {
          path,
          mode: "overwrite",
          mute: true,
        },
        cursor: {
          offset,
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
  const isDir = data[".tag"] === "folder";
  const entry = {
    isDir,
    path: data.path_display.substring(1),
    size: isDir ? -1 : data.size,
    modTime: isDir ? -1 : dayjs(data.server_modified).toDate().getTime(),
  };
  if (dropboxCanThumbnail(entry)) {
    entry.meta = { readable: true, writable: true, selfThumbnail: true };
  }
  return entry;
}

function dropboxCanThumbnail(entry) {
  if (entry.isDir) return false;
  if (entry.size > 20 * 1024 * 1024) return false;
  const ext = pathUtils.ext(entry.path);
  return [
    "jpg",
    "jpeg",
    "png",
    "tiff",
    "tif",
    "gif",
    "webp",
    "ppm",
    "bmp",
  ].includes(ext);
}

/**
 * @param {OAuthHolder} oauthHolder
 * @param {HttpMethod} method
 * @param {string} [url]
 * @param {SM|null} [headers]
 * @param {any} [body]
 * @param {boolean} [contentApi]
 */
function request(oauthHolder, method, url, headers, body, contentApi) {
  const token = oauthHolder.token();
  /** @type {{ [key: string]: string }} */
  const reqHeaders = Object.assign(
    {
      Authorization: token.tokenType + " " + token.accessToken,
    },
    headers
  );

  if (!contentApi && body && typeof body === "object") {
    body = JSON.stringify(body);
    reqHeaders["Content-Type"] = "application/json";
  }

  const r = http(
    (contentApi
      ? "https://content.dropboxapi.com/2"
      : "https://api.dropboxapi.com/2") + url,
    contentApi
      ? { method, headers: reqHeaders, body, timeout: 0 }
      : { method, headers: reqHeaders, body }
  );
  const isJSON = r.headers
    .get("Content-Type")
    .toLowerCase()
    .includes("application/json");

  const dataStr = isJSON ? r.text() : undefined;

  console.debug("http response", method, url, r.status);

  let data;
  if (isJSON && dataStr) {
    try {
      data = JSON.parse(dataStr);
    } catch (e) {
      throw new RemoteApiError(500, "Failed to parse JSON: " + e);
    }
  }

  if (r.status < 200 || r.status >= 400) {
    r.dispose();
    const error = data?.error_summary;
    if (typeof error === "string") {
      if (error.includes("not_found")) {
        throw new NotFoundError();
      }
      if (error.includes("conflict")) {
        throw new NotAllowedError();
      }
    }

    throw new RemoteApiError(r.status, error || data);
  }

  return data || r;
}
