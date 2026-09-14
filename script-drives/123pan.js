// @name 123Pan Open
// @version 1.0.0
// @description 123Pan official Open API drive.
//
// Use an access token, or configure the Open API client ID and client secret.

const PAN123_DEFAULT_API_URL = "https://open-api.123pan.com";
const PAN123_PLATFORM = "open_platform";

defineDrive(
  {
    configForm: [
      {
        label: "API URL",
        field: "api_url",
        type: "text",
        defaultValue: PAN123_DEFAULT_API_URL,
      },
      { label: "Client ID", field: "client_id", type: "text" },
      { label: "Client Secret", field: "client_secret", type: "password" },
      {
        label: "Access Token",
        description:
          "Use this instead of Client ID and Client Secret when an access token is already available.",
        field: "access_token",
        type: "password",
      },
      {
        label: "Root Folder ID",
        description: "123Pan root is 0; use another folder ID to mount a subfolder.",
        field: "root_id",
        type: "text",
        defaultValue: "0",
      },
      entryCacheTTLFormItem("2h"),
    ],

    validateConfig(config) {
      if (
        config.api_url &&
        !/^https?:\/\/[^/]+(?:\/.*)?$/i.test(config.api_url)
      ) {
        throw new BadRequestError("invalid 123Pan API URL");
      }
      if (
        (config.client_id && !config.client_secret) ||
        (!config.client_id && config.client_secret)
      ) {
        throw new BadRequestError(
          "Client ID and Client Secret must be configured together"
        );
      }
      if (!config.access_token && !(config.client_id && config.client_secret)) {
        throw new BadRequestError(
          "configure an Access Token or both Client ID and Client Secret"
        );
      }
      if (config.root_id && !/^\d+$/.test(config.root_id)) {
        throw new BadRequestError("Root Folder ID must be a number");
      }
    },

    createInstance(config) {
      return {
        entryCacheTTL: config.cache_ttl,
        baseURL: trimPan123URL(config.api_url),
        clientID: config.client_id || "",
        clientSecret: config.client_secret || "",
        accessToken: config.access_token || "",
        rootID: config.root_id || "0",
        $accessToken: "",
        $accessTokenExpiry: 0,
      };
    },
  },
  {
    get(path) {
      if (!path) {
        return { isDir: true, path: "", size: -1, modTime: -1 };
      }
      return resolvePan123Entry(this, path).entry;
    },

    list(path) {
      const folder = resolvePan123Folder(this, path);
      return listPan123Children(this, folder.id, path);
    },

    save(path, size, override, reader, progress) {
      const temp = new TempFile();
      try {
        const parent = resolvePan123Parent(this, path);
        const name = pathUtils.base(path);
        temp.copyFrom(reader);
        const actualSize = temp.size();
        if (size >= 0 && size !== actualSize) {
          throw new BadRequestError("upload size does not match the declared size");
        }
        if (!override && pan123FindChild(this, parent.id, pathUtils.parent(path), name)) {
          throw new NotAllowedError("destination already exists");
        }

        const sha1 = hashPan123(temp, "sha1");
        const reused = requestPan123(this, "POST", "/api/v2/file/sha1_reuse", null, {
          parentFileID: parent.id,
          filename: name,
          sha1,
          size: actualSize,
          duplicate: 2,
        });
        if (
          reused.data?.reuse &&
          reused.data.fileID !== undefined &&
          Number(reused.data.fileID) !== 0
        ) {
          progress.addLoaded(actualSize);
          return;
        }

        const md5 = hashPan123(temp, "md5");
        const created = requestPan123(this, "POST", "/upload/v2/file/create", null, {
          parentFileId: parent.id,
          filename: name,
          etag: md5,
          size: actualSize,
          duplicate: 2,
          containDir: false,
        });
        const upload = created.data || {};
        if (
          upload.reuse &&
          upload.fileID !== undefined &&
          Number(upload.fileID) !== 0
        ) {
          progress.addLoaded(actualSize);
          return;
        }
        if (!upload.preuploadID || !upload.servers?.length) {
          throw new RemoteApiError(502, "123Pan did not return an upload session");
        }

        const sliceSize = Number(upload.sliceSize);
        if (!Number.isFinite(sliceSize) || sliceSize <= 0) {
          throw new RemoteApiError(502, "123Pan returned an invalid slice size");
        }
        const server = String(upload.servers[0]).replace(/\/+$/, "");
        let offset = 0;
        let sliceNo = 1;
        while (offset < actualSize) {
          const length = Math.min(sliceSize, actualSize - offset);
          temp.seekTo(offset, SEEK_START);
          const sliceMD5 = new Hash("md5")
            .writeFrom(temp.limitReader(length))
            .sum()
            .toString("hex");
          temp.seekTo(offset, SEEK_START);
          const slice = temp.limitReader(length).withProgress(progress);
          uploadPan123Slice(
            this,
            server,
            upload.preuploadID,
            sliceNo,
            sliceMD5,
            name,
            slice
          );
          offset += length;
          sliceNo += 1;
        }

        completePan123Upload(this, upload.preuploadID);
      } finally {
        temp.close();
      }
    },

    makeDir(path) {
      const parent = resolvePan123Parent(this, path);
      requestPan123(this, "POST", "/upload/v1/file/mkdir", null, {
        parentID: parent.id,
        name: pathUtils.base(path),
      });
    },

    copy(from, to, override, progress) {
      if (from.isDir) throw new UnsupportedError();
      let source = from;
      if (!source.data?.etag) {
        source = resolvePan123Entry(this, from.path).entry;
      }
      if (!source.data?.etag) throw new UnsupportedError();
      const parent = resolvePan123Parent(this, to);
      const name = pathUtils.base(to);
      if (!override) {
        const existing = pan123FindChild(this, parent.id, pathUtils.parent(to), name);
        if (existing && existing.data?.id !== source.data.id) {
          throw new NotAllowedError("destination already exists");
        }
      }
      const amount = Math.max(source.size, 0);
      const created = requestPan123(this, "POST", "/upload/v2/file/create", null, {
        parentFileId: parent.id,
        filename: name,
        etag: source.data.etag,
        size: amount,
        duplicate: 2,
        containDir: false,
      });
      if (created.data?.reuse && created.data.fileID !== undefined &&
          Number(created.data.fileID) !== 0) {
        progress.addTotal(amount);
        progress.addLoaded(amount);
        return;
      }
      throw new UnsupportedError();
    },

    move(from, to, override, progress) {
      const amount = from.isDir ? 1 : Math.max(from.size, 0);
      const source = from.data?.id
        ? { id: from.data.id }
        : resolvePan123Entry(this, from.path);
      const parent = resolvePan123Parent(this, to);
      const name = pathUtils.base(to);
      if (!override) {
        const existing = pan123FindChild(
          this,
          parent.id,
          pathUtils.parent(to),
          name
        );
        if (existing && existing.data?.id !== String(source.id)) {
          throw new NotAllowedError("destination already exists");
        }
      }

      progress.addTotal(amount);
      requestPan123(this, "POST", "/api/v1/file/move", null, {
        fileIDs: [Number(source.id)],
        toParentFileID: Number(parent.id),
      });
      if (pathUtils.base(from.path) !== name) {
        requestPan123(this, "PUT", "/api/v1/file/name", null, {
          fileId: Number(source.id),
          fileName: name,
        });
      }
      progress.addLoaded(amount);
    },

    delete(path, progress) {
      const entry = selfDrive.get(path);
      const entries = flattenEntriesTree(
        buildEntriesTree(
          entry,
          false,
          progress.derive({ loaded: false, total: true })
        ),
        true
      );
      for (const item of entries) {
        const id = item.entry.data?.id;
        if (!id) throw new RemoteApiError(502, "123Pan entry has no file ID");
        requestPan123(this, "POST", "/api/v1/file/trash", null, {
          fileIDs: [Number(id)],
        });
      }
      progress.addLoaded(entries.length);
    },

    getURL(entry) {
      const id = entry.data?.id || resolvePan123Entry(this, entry.path).entry.data.id;
      const data = requestPan123(
        this,
        "GET",
        "/api/v1/file/download_info",
        { fileId: id },
        undefined
      );
      if (!data.data?.downloadUrl) {
        throw new RemoteApiError(502, "123Pan did not return a download URL");
      }
      return { url: data.data.downloadUrl };
    },
  }
);

function trimPan123URL(url) {
  return String(url || PAN123_DEFAULT_API_URL).replace(/\/+$/, "");
}

function resolvePan123Folder(drive, path) {
  if (!path) return { id: drive.rootID };
  const resolved = resolvePan123Entry(drive, path);
  if (!resolved.entry.isDir) throw new NotFoundError();
  return { id: resolved.entry.data.id };
}

function resolvePan123Parent(drive, path) {
  const parentPath = pathUtils.parent(path);
  return {
    id: resolvePan123Folder(drive, parentPath).id,
    path: parentPath,
  };
}

function resolvePan123Entry(drive, path) {
  let parentID = drive.rootID;
  let parentPath = "";
  const parts = path.split("/").filter((part) => part !== "");
  for (let i = 0; i < parts.length; i++) {
    const name = parts[i];
    const entry = listPan123Children(drive, parentID, parentPath).find(
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

function pan123FindChild(drive, parentID, parentPath, name) {
  return listPan123Children(drive, parentID, parentPath).find(
    (item) => pathUtils.base(item.path) === name
  );
}

function listPan123Children(drive, parentID, parentPath) {
  const result = [];
  let lastFileID = 0;
  while (true) {
    const data = requestPan123(
      drive,
      "GET",
      "/api/v2/file/list",
      {
        parentFileId: parentID,
        limit: 100,
        lastFileId: lastFileID,
        trashed: "false",
        searchMode: "",
        searchData: "",
      },
      undefined
    );
    const page = data.data || {};
    const files = page.fileList || [];
    result.push(
      ...files
        .filter((file) => Number(file.trashed || 0) === 0)
        .map((file) => pan123Entry(file, parentPath))
    );
    if (page.lastFileId === undefined || page.lastFileId === null) break;
    const next = Number(page.lastFileId);
    if (next === -1) break;
    if (!Number.isFinite(next) || next === lastFileID) {
      throw new RemoteApiError(502, "123Pan returned an invalid list cursor");
    }
    lastFileID = next;
  }
  return result;
}

function pan123Entry(file, parentPath) {
  const isDir = Number(file.type) === 1;
  const id = file.fileId;
  return {
    isDir,
    path: pathUtils.join(parentPath, file.filename),
    size: isDir ? -1 : Number(file.size),
    modTime: parsePan123Time(file.updateAt || file.createAt),
    data: {
      id: String(id),
      kind: isDir ? "dir" : "file",
      etag: file.etag ? String(file.etag) : "",
    },
  };
}

function parsePan123Time(value) {
  if (!value) return -1;
  const text = String(value);
  const parsed = Date.parse(
    text.includes("T") ? text : text.replace(" ", "T") + "+08:00"
  );
  return Number.isFinite(parsed) ? parsed : -1;
}

function hashPan123(temp, algorithm) {
  temp.seekTo(0, SEEK_START);
  return new Hash(algorithm).writeFrom(temp).sum().toString("hex");
}

function appendPan123Query(url, params) {
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

function responsePan123(resp) {
  const text = resp.text();
  if (!text) return {};
  try {
    return JSON.parse(text);
  } catch (e) {
    throw new RemoteApiError(502, "123Pan returned invalid JSON");
  }
}

function pan123Message(data, fallback) {
  return String(data?.message || data?.msg || data?.error || fallback);
}

function tokenPan123(drive, force) {
  if (!drive.clientID || !drive.clientSecret) {
    if (!drive.accessToken) throw new NotAllowedError("123Pan access token is missing");
    return drive.accessToken;
  }
  if (
    !force &&
    drive.$accessToken &&
    drive.$accessTokenExpiry > Date.now() + 60 * 1000
  ) {
    return drive.$accessToken;
  }

  const resp = http(drive.baseURL + "/api/v1/access_token", {
    method: "POST",
    headers: {
      platform: PAN123_PLATFORM,
      "Content-Type": "application/json",
      Accept: "application/json",
    },
    body: JSON.stringify({
      clientID: drive.clientID,
      clientSecret: drive.clientSecret,
    }),
  });
  const data = responsePan123(resp);
  if (resp.status < 200 || resp.status >= 300 || Number(data.code) !== 0) {
    if (resp.status === 401 || resp.status === 403) {
      throw new NotAllowedError(pan123Message(data, "123Pan authentication failed"));
    }
    throw new RemoteApiError(
      resp.status || 502,
      pan123Message(data, "123Pan token request failed")
    );
  }
  const accessToken = data.data?.accessToken;
  if (!accessToken) throw new RemoteApiError(502, "123Pan token response is incomplete");
  drive.$accessToken = String(accessToken);
  const expiredAt = data.data.expiredAt;
  const expiry =
    typeof expiredAt === "number"
      ? (expiredAt < 100000000000 ? expiredAt * 1000 : expiredAt)
      : Date.parse(String(expiredAt || ""));
  drive.$accessTokenExpiry = Number.isFinite(expiry)
    ? expiry
    : Date.now() + 5 * 60 * 1000;
  return drive.$accessToken;
}

function requestPan123(drive, method, route, params, body, retried) {
  const token = tokenPan123(drive, false);
  const headers = {
    authorization: "Bearer " + token,
    platform: PAN123_PLATFORM,
    Accept: "application/json",
  };
  if (body !== undefined && body !== null) {
    headers["Content-Type"] = "application/json";
  }
  const resp = http(
    appendPan123Query(drive.baseURL + route, params),
    {
      method,
      headers,
      body: body === undefined ? undefined : JSON.stringify(body),
      timeout: "30s",
    }
  );
  const data = responsePan123(resp);
  const code = Number(data.code);
  if (
    !retried &&
    (resp.status === 401 || code === 401) &&
    drive.clientID &&
    drive.clientSecret
  ) {
    drive.$accessToken = "";
    drive.$accessTokenExpiry = 0;
    return requestPan123(drive, method, route, params, body, true);
  }
  if (resp.status === 404 || code === 404) throw new NotFoundError();
  if (resp.status === 401 || resp.status === 403 || code === 401 || code === 403) {
    throw new NotAllowedError(pan123Message(data, "123Pan request is not allowed"));
  }
  if (resp.status < 200 || resp.status >= 300) {
    throw new RemoteApiError(
      resp.status || 502,
      pan123Message(data, "123Pan request failed")
    );
  }
  if (data.code !== undefined && code !== 0) {
    throw new RemoteApiError(
      resp.status || 502,
      pan123Message(data, "123Pan request failed")
    );
  }
  return data;
}

function uploadPan123Slice(
  drive,
  server,
  preuploadID,
  sliceNo,
  sliceMD5,
  filename,
  reader
) {
  const form = new HttpFormData();
  form.appendField("preuploadID", String(preuploadID));
  form.appendField("sliceNo", String(sliceNo));
  form.appendField("sliceMD5", sliceMD5);
  form.appendFile("slice", filename + ".part" + sliceNo, reader);
  const token = tokenPan123(drive, false);
  const resp = http(server + "/upload/v2/file/slice", {
    method: "POST",
    headers: {
      authorization: "Bearer " + token,
      platform: PAN123_PLATFORM,
    },
    body: form,
    timeout: 0,
  });
  const data = responsePan123(resp);
  if (resp.status === 401 || resp.status === 403) {
    throw new NotAllowedError(pan123Message(data, "123Pan slice upload is not allowed"));
  }
  if (resp.status < 200 || resp.status >= 300 || (data.code !== undefined && Number(data.code) !== 0)) {
    throw new RemoteApiError(
      resp.status || 502,
      pan123Message(data, "123Pan slice upload failed")
    );
  }
}

function completePan123Upload(drive, preuploadID) {
  for (let attempt = 0; attempt < 60; attempt++) {
    const data = requestPan123(
      drive,
      "POST",
      "/upload/v2/file/upload_complete",
      null,
      { preuploadID: String(preuploadID) }
    );
    const state = data.data || {};
    if (
      (state.completed === true || Number(state.completed) === 1) &&
      state.fileID !== undefined &&
      Number(state.fileID) !== 0
    ) {
      return;
    }
    if (attempt === 59) {
      throw new RemoteApiError(504, "123Pan upload did not complete in time");
    }
    sleep("1s");
  }
}
