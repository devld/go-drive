// @name Qiniu
// @version 1.0.1
// @uploader qiniu-uploader.js
// @description Qiniu Kodo

/// <reference path="../docs/scripts/env/drive.d.ts"/>

var utcOffset = dayjs().utcOffset();

var baseURLRegex = /^https?:\/\/([^/]+)/i;

defineDrive(
  {
    configForm: [
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
      entryCacheTTLFormItem("2h"),
    ],

    validateConfig: function (config) {
      if (
        config.downloadBaseURL &&
        !/^https?:\/\/[^/]+$/i.test(config.downloadBaseURL)
      ) {
        throw ErrBadRequest("invalid Download Base URL");
      }
    },

    createInstance: function (ctx, config) {
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
    get: function (ctx, path) {
      var entry;
      try {
        var data = request(
          this,
          ctx,
          "GET",
          "https://rs.qiniu.com/stat/" + buildURI(this.bucket, path)
        );
        entry = toEntry(data, path);
      } catch (e) {
        if (!isNotFoundErr(e)) throw e;
        var entries = this.list(ctx, pathUtils.parent(path)).filter(function (
          item
        ) {
          return item.Path === path;
        });
        if (entries.length === 0) throw ErrNotFound();
        entry = entries[0];
      }
      return entry;
    },

    save: function (ctx, path, size, override, reader) {
      saveSmall(this, ctx, path, reader);
    },

    makeDir: function (ctx, path) {
      saveSmall(this, ctx, path + "/", "");
    },

    copy: function (ctx, from, to, override) {
      if (from.IsDir) throw ErrUnsupported();
      request(
        this,
        ctx,
        "POST",
        "https://rs.qiniuapi.com/copy/" +
          buildURI(this.bucket, from.Path) +
          "/" +
          buildURI(this.bucket, to) +
          "/force/" +
          !!override,
        null,
        null
      );
    },

    move: function (ctx, from, to, override) {
      if (from.IsDir) throw ErrUnsupported();
      request(
        this,
        ctx,
        "POST",
        "https://rs.qiniuapi.com/move/" +
          buildURI(this.bucket, from.Path) +
          "/" +
          buildURI(this.bucket, to) +
          "/force/" +
          !!override,
        null,
        null
      );
    },

    list: function (ctx, path) {
      var entries = [];
      var marker;
      do {
        var data = request(
          this,
          ctx,
          "GET",
          "https://rsf.qiniu.com/list?delimiter=%2F&bucket=" +
            encodeURIComponent(this.bucket) +
            (path ? "&prefix=" + encodeURIComponent(path + "/") : "")
        );
        if (data.commonPrefixes) {
          data.commonPrefixes.forEach(function (k) {
            entries.push(toEntry(k));
          });
        }
        if (data.items) {
          data.items.forEach(function (item) {
            if (item.key === path + "/") return;
            entries.push(toEntry(item));
          });
        }
        marker = data.marker;
      } while (marker);

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
              this_.bucket,
              e.Entry.Path() + (e.Entry.Type() === "dir" ? "/" : "")
            )
          );
        })
        .join("&");
      request(
        this,
        ctx,
        "POST",
        "https://rs.qiniuapi.com/batch",
        null,
        payload
      );
    },

    upload: function (ctx, path, size, override, config) {
      if (config && config.action === "Completed") return;
      return useCustomProvider({
        baseURL: this.uploadURL,
        key: path,
        bucket: this.bucket,
        encodedKey: encUtils.urlBase64Encode(newBytes(path)),
        token: getUploadSignature(this.ak, this.sk, this.bucket, path),
      });
    },

    getURL: function (ctx, entry) {
      var url = getDownloadURL(
        this.downloadBaseURL,
        entry.Path,
        this.ak,
        this.sk
      );
      return { URL: url };
    },
  }
);

/**
 * @param {{ ak: string, sk: string, bucket: string, uploadURL: string }} drive
 * @param {Context} ctx
 * @param {string} path
 * @param {HttpBody} reader
 */
function saveSmall(drive, ctx, path, reader) {
  var data = newFormData();
  data.AppendField("key", path);
  data.AppendField(
    "token",
    getUploadSignature(drive.ak, drive.sk, drive.bucket, path)
  );
  data.AppendFile("file", pathUtils.base(path), reader);

  var resp = http(ctx, "POST", drive.uploadURL, null, data);
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

function toEntry(data, path) {
  if (typeof data === "string") {
    return {
      IsDir: true,
      Path: data.substring(0, data.length - 1), // remove suffix /
      Size: -1,
      ModTime: -1,
    };
  }
  return {
    IsDir: false,
    Path: data.key || path,
    Size: data.fsize,
    ModTime: dayjs(data.putTime / 10000)
      .toDate()
      .getTime(),
  };
}

/**
 * @param {{ ak: string, sk: string }} drive
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
    drive.ak,
    drive.sk,
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
