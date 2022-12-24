// Qiniu
// Qiniu Kodo

/// <reference path="../docs/scripts/env/drive.d.ts"/>

var utcOffset = dayjs().utcOffset();

var baseURLRegex = /^https?:\/\/([^/]+)/i;

defineCreate(function (ctx, config, utils) {
  var data = utils.Data.Load(
    "bucket",
    "ak",
    "sk",
    "_id",
    "downloadBaseURL",
    "uploadURL"
  );
  if (
    !data.bucket ||
    !data.ak ||
    !data.sk ||
    !data.downloadBaseURL ||
    !data.uploadURL
  ) {
    throw ErrNotAllowed("drive not configured");
  }
  return new DriveHolder(data._id, data, utils.CreateCache());
});

defineInitConfig(function (ctx, config, utils) {
  var data = utils.Data.Load(
    "bucket",
    "ak",
    "sk",
    "downloadBaseURL",
    "uploadURL"
  );
  return {
    Configured: !!(
      data.bucket &&
      data.ak &&
      data.sk &&
      data.downloadBaseURL &&
      data.uploadURL
    ),
    Form: [
      { Label: "Bucket", Field: "bucket", Type: "text", Required: true },
      { Label: "AccessKey", Field: "ak", Type: "text", Required: true },
      { Label: "SecretKey", Field: "sk", Type: "password", Required: true },
      {
        Label: "Upload URL",
        Description:
          "See https://developer.qiniu.com/kodo/1671/region-endpoint-fq",
        Field: "uploadURL",
        Type: "text",
        Required: true,
      },
      {
        Label: "Download Base URL",
        Description:
          "The domain name bound to the bucket must starts with http or https and cannot end with /. For example https://example.com",
        Field: "downloadBaseURL",
        Type: "text",
        Required: true,
      },
    ],
    Value: data,
  };
});

defineInit(function (ctx, data, config, utils) {
  if (
    data.downloadBaseURL &&
    !/^https?:\/\/[^/]+$/i.test(data.downloadBaseURL)
  ) {
    throw ErrBadRequest("invalid Download Base URL");
  }

  var idm = utils.Data.Load("_id");
  if (!idm._id) {
    data._id = (Math.random() * 1000000).toFixed(0);
  }
  utils.Data.Save(data);
});

/**
 * @type {QiniuDrive}
 */
var DriveImpl = {
  meta: function (ctx) {},
  get: function (ctx, path) {
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
    var entry;
    try {
      var data = request(
        this,
        ctx,
        "GET",
        "https://rs.qiniu.com/stat/" + buildURI(this._bucket, path)
      );
      entry = toEntry(this, data, path);
    } catch (e) {
      if (!isNotFoundErr(e)) throw e;
      var entries = this.list(ctx, pathUtils.parent(path)).filter(function (
        entry
      ) {
        return entry.Path === path;
      });
      if (entries.length === 0) throw ErrNotFound();
      entry = entries[0];
    }
    this._cache.PutEntry(entry, this._cacheTTL);
    return entry;
  },
  save: function (ctx, path, size, override, reader) {
    saveSmall(this, ctx, path, reader);
    this._cache.Evict(path, false);
    this._cache.Evict(pathUtils.parent(path), false);
    return this.get(ctx, path);
  },
  makeDir: function (ctx, path) {
    saveSmall(this, ctx, path + "/", "");
    this._cache.Evict(path, false);
    this._cache.Evict(pathUtils.parent(path), false);
    return this.get(ctx, path);
  },
  copy: function (ctx, from, to, override) {
    from = from.Unwrap();
    var dat = from.Data();
    if (!dat || dat.d !== this._rid || from.Type() === "dir") {
      throw ErrUnsupported();
    }
    request(
      this,
      ctx,
      "POST",
      "https://rs.qiniuapi.com/copy/" +
        buildURI(this._bucket, from.Path()) +
        "/" +
        buildURI(this._bucket, to) +
        "/force/" +
        !!override,
      null,
      null
    );
    this._cache.Evict(to, true);
    this._cache.Evict(pathUtils.parent(to), false);
    return this.get(ctx, to);
  },
  move: function (ctx, from, to, override) {
    from = from.Unwrap();
    var dat = from.Data();
    if (!dat || dat.d !== this._rid || from.Type() === "dir") {
      throw ErrUnsupported();
    }
    request(
      this,
      ctx,
      "POST",
      "https://rs.qiniuapi.com/move/" +
        buildURI(this._bucket, from.Path()) +
        "/" +
        buildURI(this._bucket, to) +
        "/force/" +
        !!override,
      null,
      null
    );
    this._cache.Evict(to, true);
    this._cache.Evict(pathUtils.parent(to), false);
    this._cache.Evict(from.Path, true);
    this._cache.Evict(pathUtils.parent(from.Path()), false);
    return this.get(ctx, to);
  },
  list: function (ctx, path) {
    var this_ = this;
    var cachedItems = this._cache.GetChildren(path);
    if (cachedItems) {
      return cachedItems.map(cacheItemToEntry);
    }

    var entries = [];
    var marker;
    do {
      var data = request(
        this,
        ctx,
        "GET",
        "https://rsf.qiniu.com/list?delimiter=%2F&bucket=" +
          encodeURIComponent(this._bucket) +
          (path ? "&prefix=" + encodeURIComponent(path + "/") : "")
      );
      if (data.commonPrefixes) {
        data.commonPrefixes.forEach(function (k) {
          entries.push(toEntry(this_, k));
        });
      }
      if (data.items) {
        data.items.forEach(function (item) {
          if (item.key === path + "/") return;
          entries.push(toEntry(this_, item));
        });
      }
      marker = data.marker;
    } while (marker);

    this._cache.PutChildren(path, entries, this._cacheTTL);
    return entries;
  },
  delete: function (ctx, path) {
    var this_ = this;
    var entry = selfDrive.Get(ctx, path);
    var entries = flattenEntriesTree(buildEntriesTree(ctx, entry));
    var payload = entries
      .map(function (e) {
        return (
          "op=/delete/" +
          buildURI(
            this_._bucket,
            e.Entry.Path() + (e.Entry.Type() === "dir" ? "/" : "")
          )
        );
      })
      .join("&");
    request(this, ctx, "POST", "https://rs.qiniuapi.com/batch", null, payload);
    this._cache.Evict(pathUtils.parent(path), false);
    this._cache.Evict(path, true);
  },
  upload: function (ctx, path, size, override, config) {
    switch (config.action) {
      case "Completed":
        this._cache.Evict(path, false);
        this._cache.Evict(pathUtils.parent(path), false);
        return;
    }

    return useCustomProvider("qiniu", {
      baseURL: this._uploadURL,
      key: path,
      bucket: this._bucket,
      encodedKey: encUtils.urlBase64Encode(newBytes(path)),
      token: getUploadSignature(this._ak, this._sk, this._bucket, path),
    });
  },
  getReader: function (ctx, entry, start, size) {
    throw ErrUnsupported();
  },
  getURL: function (ctx, entry) {
    var url = getDownloadURL(
      this._downloadBaseURL,
      entry.Path,
      this._ak,
      this._sk
    );
    return { URL: url };
  },
  hasThumbnail: function (entry) {
    return false;
  },
  getThumbnail: function (ctx, entry) {
    throw ErrUnsupported();
  },
};

/**
 * @param {QiniuDrive} drive
 * @param {Context} ctx
 * @param {string} path
 * @param {HttpBody} reader
 */
function saveSmall(drive, ctx, path, reader) {
  var data = newFormData();
  data.AppendField("key", path);
  data.AppendField(
    "token",
    getUploadSignature(drive._ak, drive._sk, drive._bucket, path)
  );
  data.AppendFile("file", pathUtils.base(path), reader);

  var resp = http(ctx, "POST", drive._uploadURL, null, data);
  var respData = resp.Text();
  try {
    respData = JSON.parse(respData);
  } catch (e) {
    // ignore
  }
  if (resp.Status !== 200) {
    throw ErrRemoteApi(resp.Status, (respData && respData.error) || respData);
  }
}

function getDownloadURL(baseURL, key, ak, sk) {
  var url = baseURL + "/" + key;

  var e = Math.round(Date.now() / 1000) + 2 * 60 * 60; // two hours
  url += "?e=" + e;

  var sign =
    ak +
    ":" +
    encUtils.urlBase64Encode(
      encUtils.hmac(HASH.SHA1, newBytes(url), newBytes(sk))
    );

  return url + "&token=" + encodeURIComponent(sign);
}

/**
 * @param {QiniuDrive} drive
 * @returns {Entry}
 */
function toEntry(drive, data, path) {
  if (typeof data === "string") {
    return {
      IsDir: true,
      Path: data.substring(0, data.length - 1), // remove suffix /
      Size: -1,
      ModTime: -1,
      Data: { d: drive._rid },
    };
  }
  return {
    IsDir: false,
    Path: data.key || path,
    Size: data.fsize,
    ModTime: dayjs(data.putTime / 10000)
      .toDate()
      .getTime(),
    Data: { d: drive._rid },
  };
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
 * @param {QiniuDrive} drive
 * @param {Context} ctx
 * @param {HttpMethod} method
 * @param {string} url
 * @param {SM} headers
 */
function request(drive, ctx, method, url, headers, body) {
  headers = Object.assign({}, headers, {
    "X-Qiniu-Date":
      dayjs().subtract(utcOffset, "minute").format("YYYYMMDDTHHmmss") + "Z",
    "Content-Type": "application/x-www-form-urlencoded",
  });

  var urlParts = baseURLRegex.exec(url);

  var signature = getManagementSignature(
    drive._ak,
    drive._sk,
    urlParts[1],
    method,
    url.substring(urlParts[0].length),
    headers,
    body
  );
  headers["Authorization"] = "Qiniu " + signature;
  if (DEBUG) {
    console.log("[HTTP REQ]", method, url, JSON.stringify(headers), body);
  }
  var r = http(ctx, method, url, headers, body);

  var isJSON =
    r.Headers.Get("Content-Type").toLowerCase().indexOf("application/json") >=
    0;
  var data = isJSON ? r.Text() : undefined;
  if (DEBUG) {
    console.log("[HTTP RES]", r.Status, data);
  }
  if (isJSON) {
    try {
      if (data) {
        // qiniu may return empty body with Content-Type application/json
        data = JSON.parse(data);
      }
    } catch (e) {
      throw ErrRemoteApi(500, "Failed to parse JSON: " + e);
    }
  }
  if (r.Status < 200 || r.Status >= 400) {
    r.Dispose();
    if (r.Status === 404 || r.Status === 612) throw ErrNotFound();
    throw ErrRemoteApi(r.Status, (data && data.error) || data);
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
  var putPolicy = JSON.stringify({
    scope: bucket + ":" + key,
    deadline: Math.round(Date.now() / 1000) + 3 * 24 * 3600, // three days
    returnBody: returnBody,
  });
  var encodedPutPolicy = encUtils.urlBase64Encode(newBytes(putPolicy));
  var sign = encUtils.urlBase64Encode(
    encUtils.hmac(HASH.SHA1, newBytes(encodedPutPolicy), newBytes(sk))
  );
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
  var payload = method + " " + url; // url with or without query
  payload += "\nHost: " + host;
  if (headers) {
    payload += "\nContent-Type: " + headers["Content-Type"];
    Object.keys(headers)
      .filter(function (key) {
        return key.indexOf("X-Qiniu-") === 0;
      })
      .map(function (key) {
        return { key: key, value: headers[key] };
      })
      .sort(function (a, b) {
        return a.key.localeCompare(b.key);
      })
      .forEach(function (v) {
        payload += "\n" + v.key + ": " + v.value;
      });
  }
  payload += "\n\n";
  if (bodyStr) {
    payload += bodyStr;
  }
  var s =
    ak +
    ":" +
    encUtils.urlBase64Encode(
      encUtils.hmac(HASH.SHA1, newBytes(payload), newBytes(sk))
    );
  return s;
}

/**
 * @param {string} bucket
 * @param {string} key
 */
function buildURI(bucket, key) {
  return encUtils.urlBase64Encode(newBytes(bucket + ":" + key));
}

/**
 * @typedef DriveDataHolder
 * @property {DriveCache} _cache
 * @property {Duration} _cacheTTL
 * @property {string} _rid
 * @property {string} _ak
 * @property {string} _sk
 * @property {string} _bucket
 * @property {string} _downloadBaseURL
 * @property {string} _uploadURL
 */

/**
 * @typedef {Drive & DriveDataHolder} QiniuDrive
 */

/**
 * @constructor
 * @param {string} rid
 * @param {any} data
 * @param {DriveCache} cache
 */
function DriveHolder(rid, data, cache) {
  this._rid = rid;
  this._cache = cache;
  this._cacheTTL = ms(2 * 60 * 60 * 1000);

  this._ak = data.ak;
  this._sk = data.sk;
  this._bucket = data.bucket;
  this._downloadBaseURL = data.downloadBaseURL;
  this._uploadURL = data.uploadURL;
}

Object.keys(DriveImpl).forEach(function (fn) {
  DriveHolder.prototype[fn] = DriveImpl[fn];
});
