// @name Yandex Disk
// @version 1.0.0
// @description Yandex Disk REST API drive.
//
// Configure an OAuth application with cloud_api:disk.read and cloud_api:disk.write.

const YANDEX_API_URL = "https://cloud-api.yandex.net/v1/disk";

/**
 * @param {RootConfig} config
 * @returns {OAuthRequest}
 */
function yandexOAuthRequest(config) {
  return {
    endpoint: {
      authUrl: "https://oauth.yandex.com/authorize",
      tokenUrl: "https://oauth.yandex.com/token",
    },
    redirectUrl: config.oauthRedirectURI,
    scopes: ["cloud_api:disk.read", "cloud_api:disk.write"],
    text: "Connect To Yandex Disk",
  };
}

defineDrive(
  {
    configForm: [
      { label: "Client ID", field: "client_id", type: "text", required: true },
      {
        label: "Client Secret",
        field: "client_secret",
        type: "password",
        required: true,
      },
      {
        label: "Root Path",
        description:
          "Yandex Disk path to mount. Leave as / to mount the account root.",
        field: "root_path",
        type: "text",
        defaultValue: "/",
      },
      entryCacheTTLFormItem("2h"),
    ],

    initConfig(config, utils) {
      const result = utils.oauthInitConfig(yandexOAuthRequest(utils.config), {
        clientID: config.client_id,
        clientSecret: config.client_secret,
      });
      if (!result.oauthHolder) return result.config;

      const info = requestYandexWithHolder(
        result.oauthHolder,
        "GET",
        "",
        undefined
      );
      const user = info.user || {};
      const principal = user.login || user.display_name || "Yandex Disk";
      return {
        configured: true,
        oauth: {
          url: result.config.oauth.url,
          text: result.config.oauth.text,
          principal: String(principal),
        },
        form: result.config.form,
        value: result.config.value,
      };
    },

    init(data, config, utils) {
      utils.oauthInit(data, yandexOAuthRequest(utils.config), {
        clientID: config.client_id,
        clientSecret: config.client_secret,
      });
    },

    validateConfig(config) {
      if (!config.client_id || !config.client_secret) {
        throw new BadRequestError("Yandex Client ID and Client Secret are required");
      }
      normalizeYandexRoot(config.root_path);
    },

    createInstance(config, utils) {
      return {
        entryCacheTTL: config.cache_ttl,
        rootPath: normalizeYandexRoot(config.root_path),
        oauth: utils.oauthLoad(yandexOAuthRequest(utils.config), {
          clientID: config.client_id,
          clientSecret: config.client_secret,
        }),
      };
    },
  },
  {
    get(path) {
      if (!path) {
        return { isDir: true, path: "", size: -1, modTime: -1 };
      }
      const data = requestYandex(this, "GET", "/resources", {
        path: yandexRemotePath(this, path),
      });
      return yandexEntry(data, path);
    },

    list(path) {
      let offset = 0;
      const result = [];
      while (true) {
        const data = requestYandex(this, "GET", "/resources", {
          path: yandexRemotePath(this, path),
          limit: 100,
          offset,
        });
        const embedded = data._embedded || {};
        const items = embedded.items || [];
        result.push(...items.map((item) => yandexEntry(item, path)));
        offset += items.length;
        const total = Number(embedded.total);
        if (
          items.length === 0 ||
          (Number.isFinite(total) && offset >= total) ||
          items.length < 100
        ) {
          break;
        }
      }
      return result;
    },

    save(path, size, override, reader, progress) {
      const data = requestYandex(this, "GET", "/resources/upload", {
        path: yandexRemotePath(this, path),
        overwrite: override ? "true" : "false",
      });
      if (!data.href) {
        throw new RemoteApiError(502, "Yandex Disk did not return an upload URL");
      }
      const headers = { "Content-Type": "application/octet-stream" };
      if (size >= 0) headers["Content-Length"] = String(size);
      const resp = http(data.href, {
        method: data.method || "PUT",
        headers,
        body: reader.withProgress(progress),
        timeout: 0,
      });
      const text = resp.text();
      if (resp.status < 200 || resp.status >= 300) {
        throw new RemoteApiError(resp.status || 502, yandexErrorMessage(text));
      }
    },

    makeDir(path) {
      requestYandex(this, "PUT", "/resources", {
        path: yandexRemotePath(this, path),
      });
    },

    copy(from, to, override, progress) {
      const amount = from.isDir ? 1 : Math.max(from.size, 0);
      progress.addTotal(amount);
      requestYandex(this, "POST", "/resources/copy", {
        from: yandexRemotePath(this, from.path),
        path: yandexRemotePath(this, to),
        overwrite: override ? "true" : "false",
      });
      progress.addLoaded(amount);
    },

    move(from, to, override, progress) {
      const amount = from.isDir ? 1 : Math.max(from.size, 0);
      progress.addTotal(amount);
      requestYandex(this, "POST", "/resources/move", {
        from: yandexRemotePath(this, from.path),
        path: yandexRemotePath(this, to),
        overwrite: override ? "true" : "false",
      });
      progress.addLoaded(amount);
    },

    delete(path, progress) {
      progress.addTotal(1);
      requestYandex(this, "DELETE", "/resources", {
        path: yandexRemotePath(this, path),
      });
      progress.addLoaded(1);
    },

    getURL(entry) {
      const data = requestYandex(this, "GET", "/resources/download", {
        path: yandexRemotePath(this, entry.path),
      });
      if (!data.href) {
        throw new RemoteApiError(502, "Yandex Disk did not return a download URL");
      }
      return { url: data.href };
    },
  }
);

function normalizeYandexRoot(value) {
  let path = String(value || "/").trim();
  if (!path.startsWith("/")) path = "/" + path;
  path = path.replace(/\/+/g, "/");
  if (path.length > 1) path = path.replace(/\/+$/, "");
  if (path.split("/").some((part) => part === "." || part === "..")) {
    throw new BadRequestError("Root Path must not contain dot segments");
  }
  return path || "/";
}

function yandexRemotePath(drive, path) {
  const root = drive.rootPath === "/" ? "" : drive.rootPath;
  return root + (path ? "/" + path : "") || "/";
}

function appendYandexQuery(url, params) {
  const parts = urlUtils.parse(url);
  /** @type {{ [key: string]: string[] }} */
  const searchParams = {};
  for (const key of Object.keys(parts.searchParams)) {
    searchParams[key] = parts.searchParams[key].slice();
  }
  for (const key of Object.keys(params || {})) {
    const value = params[key];
    if (value === undefined || value === null) continue;
    const values = Array.isArray(value) ? value : [value];
    searchParams[key] = values.map((item) => String(item));
  }
  return urlUtils.build(Object.assign({}, parts, { searchParams }));
}

function yandexErrorMessage(text) {
  if (!text) return "Yandex Disk request failed";
  try {
    const data = JSON.parse(text);
    return String(data.message || data.description || "Yandex Disk request failed");
  } catch (e) {
    return "Yandex Disk request failed";
  }
}

function requestYandexWithHolder(holder, method, route, params) {
  const token = holder.token();
  const resp = http(
    appendYandexQuery(YANDEX_API_URL + route, params),
    {
      method,
      headers: {
        Authorization: "OAuth " + token.accessToken,
        Accept: "application/json",
      },
      timeout: "30s",
    }
  );
  const text = resp.text();
  if (resp.status === 404) throw new NotFoundError();
  if (resp.status === 401 || resp.status === 403) {
    throw new NotAllowedError(yandexErrorMessage(text));
  }
  if (resp.status < 200 || resp.status >= 300) {
    throw new RemoteApiError(resp.status || 502, yandexErrorMessage(text));
  }
  if (!text) return {};
  try {
    return JSON.parse(text);
  } catch (e) {
    throw new RemoteApiError(502, "Yandex Disk returned invalid JSON");
  }
}

function requestYandex(drive, method, route, params) {
  return requestYandexWithHolder(drive.oauth, method, route, params);
}

function yandexEntry(item, parentPath) {
  const isDir = item.type === "dir";
  const size = Number(item.size);
  const modTime = item.modified ? Date.parse(String(item.modified)) : -1;
  return {
    isDir,
    path: pathUtils.join(parentPath, item.name),
    size: isDir ? -1 : Number.isFinite(size) ? size : -1,
    modTime: Number.isFinite(modTime) ? modTime : -1,
    data: {
      path: String(item.path || ""),
      kind: isDir ? "dir" : "file",
    },
  };
}
