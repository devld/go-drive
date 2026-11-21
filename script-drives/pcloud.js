// @name pCloud
// @version 1.0.0
// @description pCloud drive using the official HTTP API.
//
// Create an access token with file read and write permissions and paste it below.

const PCLOUD_US_API_URL = "https://api.pcloud.com";
const PCLOUD_EU_API_URL = "https://eapi.pcloud.com";

defineDrive(
  {
    configForm: [
      {
        label: "Region",
        field: "region",
        type: "select",
        required: true,
        defaultValue: "us",
        options: [
          { name: "US", value: "us" },
          { name: "Europe", value: "eu" },
        ],
      },
      {
        label: "Access Token",
        description:
          "Create one through pCloud OAuth and grant file read/write permissions.",
        field: "access_token",
        type: "password",
        required: true,
      },
      {
        label: "Root Folder ID",
        description: "Use 0 for the account root, or a numeric folder ID.",
        field: "root_folder_id",
        type: "text",
        defaultValue: "0",
      },
      entryCacheTTLFormItem("2h"),
    ],

    validateConfig(config) {
      if (config.region !== "us" && config.region !== "eu") {
        throw new BadRequestError("Region must be us or eu");
      }
      if (!config.access_token) {
        throw new BadRequestError("pCloud access token is required");
      }
      if (
        config.root_folder_id &&
        !/^\d+$/.test(config.root_folder_id)
      ) {
        throw new BadRequestError("Root Folder ID must be a number");
      }
    },

    createInstance(config) {
      return {
        entryCacheTTL: config.cache_ttl,
        apiURL: config.region === "eu" ? PCLOUD_EU_API_URL : PCLOUD_US_API_URL,
        accessToken: config.access_token,
        rootFolderID: config.root_folder_id || "0",
      };
    },
  },
  {
    get(path) {
      if (!path) {
        return { isDir: true, path: "", size: -1, modTime: -1 };
      }
      return resolvePCloudEntry(this, path).entry;
    },

    list(path) {
      const folder = resolvePCloudFolder(this, path);
      return listPCloudChildren(this, folder.id, path);
    },

    save(path, size, override, reader, progress) {
      const parent = resolvePCloudParent(this, path);
      const name = pathUtils.base(path);
      if (!override && findPCloudChild(this, parent.id, name)) {
        throw new NotAllowedError("destination already exists");
      }

      const form = new HttpFormData();
      form.appendFile(name, name, reader.withProgress(progress));
      const data = requestPCloudUpload(this, {
        folderid: parent.id,
        filename: name,
        nopartial: 1,
      }, form);
      if (data.result !== undefined && Number(data.result) !== 0) {
        throwPCloudError(data, 502);
      }
    },

    makeDir(path) {
      const parent = resolvePCloudParent(this, path);
      requestPCloud(this, "createfolderifnotexists", "GET", {
        folderid: parent.id,
        name: pathUtils.base(path),
      });
    },

    copy(from, to, override, progress) {
      const parent = resolvePCloudParent(this, to);
      const source =
        from.data?.id || resolvePCloudEntry(this, from.path).entry.data.id;
      const params = {
        tofolderid: parent.id,
        toname: pathUtils.base(to),
      };
      if (from.isDir) {
        params.folderid = source;
      } else {
        params.fileid = source;
      }
      if (!override) params.noover = 1;
      const amount = from.isDir ? 1 : Math.max(from.size, 0);
      progress.addTotal(amount);
      requestPCloud(this, from.isDir ? "copyfolder" : "copyfile", "GET", params);
      progress.addLoaded(amount);
    },

    move(from, to, override, progress) {
      const parent = resolvePCloudParent(this, to);
      const name = pathUtils.base(to);
      const source = from.data?.id || resolvePCloudEntry(this, from.path).entry.data.id;
      if (!override) {
        const existing = findPCloudChild(this, parent.id, name);
        if (existing && existing.data?.id !== String(source)) {
          throw new NotAllowedError("destination already exists");
        }
      }

      const amount = from.isDir ? 1 : Math.max(from.size, 0);
      progress.addTotal(amount);
      requestPCloud(
        this,
        from.isDir ? "renamefolder" : "renamefile",
        "GET",
        from.isDir
          ? {
              folderid: source,
              tofolderid: parent.id,
              toname: name,
            }
          : {
              fileid: source,
              tofolderid: parent.id,
              toname: name,
            }
      );
      progress.addLoaded(amount);
    },

    delete(path, progress) {
      const entry = selfDrive.get(path);
      const id = entry.data?.id || resolvePCloudEntry(this, path).entry.data.id;
      progress.addTotal(1);
      requestPCloud(
        this,
        entry.type === "dir" ? "deletefolderrecursive" : "deletefile",
        "GET",
        entry.type === "dir" ? { folderid: id } : { fileid: id }
      );
      progress.addLoaded(1);
    },

    getURL(entry) {
      const id = entry.data?.id || resolvePCloudEntry(this, entry.path).entry.data.id;
      const data = requestPCloud(this, "getfilelink", "GET", { fileid: id });
      const host = data.hosts?.[0];
      if (!host || !data.path) {
        throw new RemoteApiError(502, "pCloud did not return a download URL");
      }
      return {
        url: (String(host).startsWith("http") ? String(host) : "https://" + host) +
          String(data.path),
      };
    },
  }
);

function pCloudURL(drive, methodName, params) {
  const url = urlUtils.parse(drive.apiURL + "/" + methodName);
  /** @type {{ [key: string]: string[] }} */
  const searchParams = {};
  for (const key of Object.keys(url.searchParams)) {
    searchParams[key] = url.searchParams[key].slice();
  }
  searchParams.auth = [drive.accessToken];
  for (const key of Object.keys(params || {})) {
    const value = params[key];
    if (value === undefined || value === null) continue;
    const values = Array.isArray(value) ? value : [value];
    searchParams[key] = values.map((item) => String(item));
  }
  return urlUtils.build(Object.assign({}, url, { searchParams }));
}

function parsePCloudResponse(resp) {
  const text = resp.text();
  if (!text) return {};
  try {
    return JSON.parse(text);
  } catch (e) {
    throw new RemoteApiError(502, "pCloud returned invalid JSON");
  }
}

function pCloudMessage(data, fallback) {
  return String(data?.error || data?.message || fallback);
}

function throwPCloudError(data, status) {
  const result = Number(data?.result);
  if (result === 2005 || result === 2009 || status === 404) {
    throw new NotFoundError();
  }
  if (
    status === 401 ||
    status === 403 ||
    result === 1000 ||
    result === 2000 ||
    result === 2003 ||
    result === 2004 ||
    result === 2006
  ) {
    throw new NotAllowedError(pCloudMessage(data, "pCloud request is not allowed"));
  }
  throw new RemoteApiError(
    status || 502,
    pCloudMessage(data, "pCloud request failed")
  );
}

function requestPCloud(drive, methodName, method, params) {
  const resp = http(pCloudURL(drive, methodName, params), {
    method,
    timeout: "30s",
  });
  const data = parsePCloudResponse(resp);
  if (resp.status < 200 || resp.status >= 300) {
    throwPCloudError(data, resp.status);
  }
  if (data.result !== undefined && Number(data.result) !== 0) {
    throwPCloudError(data, resp.status);
  }
  return data;
}

function requestPCloudUpload(drive, params, form) {
  const resp = http(pCloudURL(drive, "uploadfile", params), {
    method: "POST",
    body: form,
    timeout: 0,
  });
  const data = parsePCloudResponse(resp);
  if (resp.status < 200 || resp.status >= 300) {
    throwPCloudError(data, resp.status);
  }
  if (data.result !== undefined && Number(data.result) !== 0) {
    throwPCloudError(data, resp.status);
  }
  return data;
}

function resolvePCloudFolder(drive, path) {
  if (!path) return { id: drive.rootFolderID };
  const resolved = resolvePCloudEntry(drive, path);
  if (!resolved.entry.isDir) throw new NotFoundError();
  return { id: resolved.entry.data.id };
}

function resolvePCloudParent(drive, path) {
  const parentPath = pathUtils.parent(path);
  return {
    id: resolvePCloudFolder(drive, parentPath).id,
    path: parentPath,
  };
}

function resolvePCloudEntry(drive, path) {
  let parentID = drive.rootFolderID;
  let parentPath = "";
  const parts = path.split("/").filter((part) => part !== "");
  for (let i = 0; i < parts.length; i++) {
    const name = parts[i];
    const entry = listPCloudChildren(drive, parentID, parentPath).find(
      (item) => pathUtils.base(item.path) === name
    );
    if (!entry) throw new NotFoundError();
    if (i !== parts.length - 1 && !entry.isDir) throw new NotFoundError();
    if (i === parts.length - 1) {
      return { id: entry.data.id, entry };
    }
    parentID = entry.data.id;
    parentPath = entry.path;
  }
  throw new NotFoundError();
}

function findPCloudChild(drive, parentID, name) {
  return listPCloudChildren(drive, parentID, "").find(
    (item) => pathUtils.base(item.path) === name
  );
}

function listPCloudChildren(drive, folderID, parentPath) {
  const data = requestPCloud(drive, "listfolder", "GET", {
    folderid: folderID,
    nofiles: 0,
  });
  return (data.metadata?.contents || []).map((item) =>
    pCloudEntry(item, parentPath)
  );
}

function pCloudEntry(item, parentPath) {
  const isDir = Number(item.isfolder) === 1 || item.isfolder === true;
  const numericSize = Number(item.size);
  return {
    isDir,
    path: pathUtils.join(parentPath, item.name),
    size: isDir ? -1 : Number.isFinite(numericSize) ? numericSize : -1,
    modTime: parsePCloudTime(item.modified || item.created),
    data: {
      id: String(isDir ? item.folderid : item.fileid),
      kind: isDir ? "dir" : "file",
    },
  };
}

function parsePCloudTime(value) {
  if (!value) return -1;
  const parsed = Date.parse(String(value));
  return Number.isFinite(parsed) ? parsed : -1;
}
