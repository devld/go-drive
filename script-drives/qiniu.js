// @name Qiniu
// @version 1.0.10
// @uploader qiniu-uploader.js
// @description Qiniu Kodo

const utcOffset = dayjs().utcOffset();

const baseURLRegex = /^https?:\/\/([^/]+)/i;

defineDrive(
  {
    configForm: [
      { label: "Bucket", field: "bucket", type: "text", required: true },
      { label: "AccessKey", field: "ak", type: "text", required: true },
      { label: "SecretKey", field: "sk", type: "password", required: true },
      {
        label: "Upload URL",
        description:
          "See https://developer.qiniu.com/kodo/1671/region-endpoint-fq",
        field: "uploadURL",
        type: "text",
        required: true,
      },
      {
        label: "Download Base URL",
        description:
          "The domain name bound to the bucket must starts with http or https and cannot end with /. For example https://example.com",
        field: "downloadBaseURL",
        type: "text",
        required: true,
      },
      entryCacheTTLFormItem("2h"),
    ],

    validateConfig(config) {
      if (
        config.downloadBaseURL &&
        !/^https?:\/\/[^/]+$/i.test(config.downloadBaseURL)
      ) {
        throw new BadRequestError("invalid Download Base URL");
      }
    },

    createInstance(config) {
      return {
        entryCacheTTL: config.cache_ttl,
        ak: config.ak,
        sk: config.sk,
        bucket: config.bucket,
        downloadBaseURL: config.downloadBaseURL,
        uploadURL: config.uploadURL,
      };
    },
  },
  {
    get(path) {
      let entry;
      try {
        const data = request(
          this,
          "GET",
          "https://rs.qiniu.com/stat/" + buildURI(this.bucket, path)
        );
        entry = toEntry(data, path);
      } catch (e) {
        if (!(e instanceof NotFoundError)) throw e;
        const found = this.list(pathUtils.parent(path)).find(
          (item) => item.path === path
        );
        if (!found) throw new NotFoundError();
        entry = found;
      }
      return entry;
    },

    save(path, size, override, reader, progress) {
      saveSmall(this, path, reader.withProgress(progress));
    },

    makeDir(path) {
      saveSmall(this, path + "/", "");
    },

    copy(from, to, override, progress) {
      if (from.isDir) throw new UnsupportedError();
      progress.addTotal(Math.max(from.size, 0));
      request(
        this,
        "POST",
        "https://rs.qiniuapi.com/copy/" +
          buildURI(this.bucket, from.path) +
          "/" +
          buildURI(this.bucket, to) +
          "/force/" +
          !!override,
        null,
        null
      );
      progress.addLoaded(Math.max(from.size, 0));
    },

    move(from, to, override, progress) {
      if (from.isDir) throw new UnsupportedError();
      progress.addTotal(Math.max(from.size, 0));
      request(
        this,
        "POST",
        "https://rs.qiniuapi.com/move/" +
          buildURI(this.bucket, from.path) +
          "/" +
          buildURI(this.bucket, to) +
          "/force/" +
          !!override,
        null,
        null
      );
      progress.addLoaded(Math.max(from.size, 0));
    },

    list(path) {
      const entries = [];
      let marker;
      do {
        const data = request(
          this,
          "GET",
          "https://rsf.qiniu.com/list?delimiter=%2F&bucket=" +
            encodeURIComponent(this.bucket) +
            (path ? "&prefix=" + encodeURIComponent(path + "/") : "") +
            (marker ? "&marker=" + encodeURIComponent(marker) : "")
        );
        if (data.commonPrefixes) {
          entries.push(...data.commonPrefixes.map((k) => toEntry(k)));
        }
        if (data.items) {
          entries.push(
            ...data.items
              .filter((item) => item.key !== path + "/")
              .map((item) => toEntry(item))
          );
        }
        marker = data.marker;
      } while (marker);

      return entries;
    },

    delete(path, progress) {
      const entry = selfDrive.get(path);
      const entries = flattenEntriesTree(
        buildEntriesTree(
          entry,
          false,
          progress.derive({ loaded: false, total: true })
        )
      );
      const payload = entries
        .map(
          (e) =>
            "op=/delete/" +
            buildURI(
              this.bucket,
              e.entry.path + (e.entry.type === "dir" ? "/" : "")
            )
        )
        .join("&");
      request(this, "POST", "https://rs.qiniuapi.com/batch", null, payload);
      progress.addLoaded(entries.length);
    },

    upload(path, size, override, config) {
      if (config && config.action === "Completed") return;
      return useCustomProvider({
        baseURL: this.uploadURL,
        key: path,
        bucket: this.bucket,
        encodedKey: Bytes.fromString(path).toString("base64url"),
        token: getUploadSignature(this.ak, this.sk, this.bucket, path),
      });
    },

    getURL(entry) {
      const url = getDownloadURL(
        this.downloadBaseURL,
        entry.path,
        this.ak,
        this.sk
      );
      return { url };
    },
  }
);

/**
 * @param {{ ak: string, sk: string, bucket: string, uploadURL: string }} drive
 * @param {string} path
 * @param {Exclude<HttpBody, HttpFormData>} reader
 */
function saveSmall(drive, path, reader) {
  const data = new HttpFormData();
  data.appendField("key", path);
  data.appendField(
    "token",
    getUploadSignature(drive.ak, drive.sk, drive.bucket, path)
  );
  data.appendFile("file", pathUtils.base(path), reader);

  const resp = http(drive.uploadURL, { method: "POST", body: data, timeout: 0 });
  const respStr = resp.text();
  let respData;
  try {
    respData = JSON.parse(respStr);
  } catch (e) {
    // ignore
  }
  if (resp.status !== 200) {
    throw new RemoteApiError(resp.status, respData?.error || respData);
  }
}

function getDownloadURL(baseURL, key, ak, sk) {
  const e = Math.round(Date.now() / 1000) + 2 * 60 * 60; // two hours
  const url = `${baseURL}/${key}?e=${e}`;

  const sign =
    ak +
    ":" +
    new Hmac("sha1", Bytes.fromString(sk)).write(Bytes.fromString(url)).sum().toString("base64url");

  return url + "&token=" + encodeURIComponent(sign);
}

function toEntry(data, path) {
  if (typeof data === "string") {
    return {
      isDir: true,
      path: data.substring(0, data.length - 1), // remove suffix /
      size: -1,
      modTime: -1,
    };
  }
  return {
    isDir: false,
    path: data.key || path,
    size: data.fsize,
    modTime: dayjs(data.putTime / 10000).toDate().getTime(),
  };
}

/**
 * @param {{ ak: string, sk: string }} drive
 * @param {HttpMethod} method
 * @param {string} url
 * @param {Record<string, string>|null} [headers]
 * @param {Exclude<HttpBody, HttpFormData>|null} [body]
 */
function request(drive, method, url, headers, body) {
  headers = Object.assign({}, headers, {
    "X-Qiniu-Date":
      dayjs().subtract(utcOffset, "minute").format("YYYYMMDDTHHmmss") + "Z",
    "Content-Type": "application/x-www-form-urlencoded",
  });

  const urlParts = baseURLRegex.exec(url);
  if (!urlParts) throw new BadRequestError("invalid URL");

  const signature = getManagementSignature(
    drive.ak,
    drive.sk,
    urlParts[1],
    method,
    url.substring(urlParts[0].length),
    headers,
    body
  );
  headers["Authorization"] = "Qiniu " + signature;
  console.debug("http request", method, url);
  const r = http(url, { method, headers, body: body || undefined });

  const isJSON = r.headers
    .get("Content-Type")
    .toLowerCase()
    .includes("application/json");
  const dataStr = isJSON ? r.text() : undefined;
  console.debug("http response", r.status);
  let data;
  if (isJSON) {
    try {
      if (dataStr) {
        // qiniu may return empty body with Content-Type application/json
        data = JSON.parse(dataStr);
      }
    } catch (e) {
      throw new RemoteApiError(500, "Failed to parse JSON: " + e);
    }
  }
  if (r.status < 200 || r.status >= 400) {
    r.dispose();
    if (r.status === 404 || r.status === 612) throw new NotFoundError();
    throw new RemoteApiError(r.status, data?.error || data);
  }
  return data;
}

/**
 * @param {string} ak
 * @param {string} sk
 * @param {string} bucket
 * @param {string} key
 * @param {string} [returnBody]
 */
function getUploadSignature(ak, sk, bucket, key, returnBody) {
  const putPolicy = JSON.stringify({
    scope: bucket + ":" + key,
    deadline: Math.round(Date.now() / 1000) + 3 * 24 * 3600, // three days
    returnBody,
  });
  const encodedPutPolicy = Bytes.fromString(putPolicy).toString("base64url");
  const sign = new Hmac("sha1", Bytes.fromString(sk)).write(Bytes.fromString(encodedPutPolicy)).sum().toString("base64url");
  return ak + ":" + sign + ":" + encodedPutPolicy;
}

/**
 * @param {string} ak
 * @param {string} sk
 * @param {string} host
 * @param {HttpMethod} method
 * @param {string} url
 * @param {SM} headers
 */
function getManagementSignature(ak, sk, host, method, url, headers, bodyStr) {
  let payload = method + " " + url; // url with or without query
  payload += "\nHost: " + host;
  if (headers) {
    payload += "\nContent-Type: " + headers["Content-Type"];
    Object.keys(headers)
      .filter((key) => key.startsWith("X-Qiniu-"))
      .map((key) => ({ key, value: headers[key] }))
      .sort((a, b) => a.key.localeCompare(b.key))
      .forEach((v) => {
        payload += "\n" + v.key + ": " + v.value;
      });
  }
  payload += "\n\n";
  if (bodyStr) {
    payload += bodyStr;
  }
  return (
    ak +
    ":" +
    new Hmac("sha1", Bytes.fromString(sk)).write(Bytes.fromString(payload)).sum().toString("base64url")
  );
}

/**
 * @param {string} bucket
 * @param {string} key
 */
function buildURI(bucket, key) {
  return Bytes.fromString(bucket + ":" + key).toString("base64url");
}
