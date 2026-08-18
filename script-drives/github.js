// @name GitHub
// @version 1.0.1
// @description Map a GitHub repository branch as a read-only virtual drive.
//
// Configure the repository owner, repository name, and optionally a branch
// and GitHub token. An empty branch uses the repository default branch.
//
// A token is recommended for private repositories and higher API rate limits.
// Fine-grained tokens need Contents: Read; classic tokens need the `repo` scope
// for private repositories.
//
// Git LFS objects and git submodules are not downloaded. GitHub also rejects
// files larger than 100 MB outside LFS.

/// <reference path="../docs/scripts/env/drive.d.ts"/>

var API_BASE = "https://api.github.com";
var RAW_BASE = "https://raw.githubusercontent.com";
var READONLY_META = { Readable: true, Writable: false };
var UNREADABLE_META = { Readable: false, Writable: false };

/**
 * @typedef {DriveInstanceState & {
 *   owner: string,
 *   repo: string,
 *   branch: string,
 *   token: string,
 *   $resolvedBranch: string
 * }} GitHubDriveState
 */

defineDrive(
  {
    configForm: [
      {
        Label: "Repository Owner",
        Description: "The owner of the GitHub repository (e.g. 'devld' or 'facebook')",
        Type: "text",
        Field: "owner",
        Required: true,
      },
      {
        Label: "Repository Name",
        Description: "The name of the GitHub repository (e.g. 'go-drive' or 'react')",
        Type: "text",
        Field: "repo",
        Required: true,
      },
      {
        Label: "Branch",
        Description: "Branch, tag, or commit SHA. Leave empty to use the repository default branch.",
        Type: "text",
        Field: "branch",
        Required: false,
      },
      {
        Label: "GitHub Token",
        Description: "Personal access token for private repositories and higher rate limits",
        Type: "password",
        Field: "token",
        Required: false,
      },
      entryCacheTTLFormItem("5m"),
    ],

    validateConfig: function (config) {
      if (config.owner && /[/\s]/.test(config.owner)) {
        throw ErrBadRequest("invalid repository owner");
      }
      if (config.repo && /[/\s]/.test(config.repo)) {
        throw ErrBadRequest("invalid repository name");
      }
    },

    /**
     * @param {Context} ctx
     * @param {SM} config
     * @returns {GitHubDriveState}
     */
    createInstance: function (ctx, config) {
      return {
        entryCacheTTL: config.cache_ttl,
        writable: false,
        owner: config.owner,
        repo: config.repo,
        branch: config.branch || "",
        token: config.token || "",
        $resolvedBranch: "",
      };
    },
  },
  {
    get: function (ctx, path) {
      if (DEBUG) console.log("get", path);
      var result = getContents(this, ctx, path);
      if (result.tooMany) {
        return makeEntry(true, path, -1, true);
      }
      return toEntry(result.data, path);
    },

    list: function (ctx, path) {
      if (DEBUG) console.log("list", path);
      var result = getContents(this, ctx, path);
      if (result.tooMany) {
        return listViaTree(this, ctx, path);
      }
      if (!Array.isArray(result.data)) {
        throw ErrNotFound();
      }
      var entries = [];
      for (var i = 0; i < result.data.length; i++) {
        ctx.Err();
        entries.push(toEntry(result.data[i]));
      }
      return entries;
    },

    getURL: function (ctx, entry) {
      if (DEBUG) console.log("getURL", entry.Path);
      if (entry.IsDir) return undefined;
      if (entry.Meta && entry.Meta.Readable === false) {
        throw ErrUnsupported();
      }
      var ref = rawRef(this, ctx);
      var url =
        RAW_BASE +
        "/" +
        encodeURIComponent(this.owner) +
        "/" +
        encodeURIComponent(this.repo) +
        "/" +
        ref +
        "/" +
        encodePath(entry.Path);
      if (!this.token) {
        return { URL: url };
      }
      return {
        URL: url,
        Header: {
          Authorization: "Bearer " + this.token,
          "User-Agent": "go-drive",
        },
        Proxy: true,
      };
    },
  }
);

/**
 * @param {GitHubDriveState} drive
 * @returns {string}
 */
function repoAPI(drive) {
  return (
    "/repos/" +
    encodeURIComponent(drive.owner) +
    "/" +
    encodeURIComponent(drive.repo)
  );
}

/**
 * @param {GitHubDriveState} drive
 * @param {Context} ctx
 * @returns {string}
 */
function branchRef(drive, ctx) {
  if (drive.branch) return drive.branch;
  if (drive.$resolvedBranch) return drive.$resolvedBranch;
  var repo = requestJSON(drive, ctx, "GET", repoAPI(drive));
  var resolved = repo.default_branch || "main";
  drive.$resolvedBranch = resolved;
  return resolved;
}

/**
 * raw.githubusercontent.com treats `/` in the ref as extra path segments, so
 * refs that contain slashes are resolved to a commit SHA.
 * @param {GitHubDriveState} drive
 * @param {Context} ctx
 * @returns {string}
 */
function rawRef(drive, ctx) {
  var ref = branchRef(drive, ctx);
  if (ref.indexOf("/") < 0) {
    return encodeURIComponent(ref);
  }
  var commit = requestJSON(
    drive,
    ctx,
    "GET",
    repoAPI(drive) + "/commits/" + encodeURIComponent(ref)
  );
  return commit.sha;
}

/**
 * @param {string} path
 * @returns {string}
 */
function encodePath(path) {
  if (!path) return "";
  return path
    .split("/")
    .map(function (segment) {
      return encodeURIComponent(segment);
    })
    .join("/");
}

/**
 * @param {number} n
 * @returns {number}
 */
function fileSize(n) {
  return typeof n === "number" ? n : -1;
}

/**
 * @param {boolean} isDir
 * @param {string} path
 * @param {number} size
 * @param {boolean} readable
 * @returns {Entry}
 */
function makeEntry(isDir, path, size, readable) {
  return {
    IsDir: isDir,
    Path: pathUtils.clean(path || ""),
    Size: isDir ? -1 : size,
    ModTime: -1,
    Meta: readable ? READONLY_META : UNREADABLE_META,
  };
}

/**
 * @param {GitHubDriveState} drive
 * @param {Context} ctx
 * @param {string} path
 */
function contentsPath(drive, ctx, path) {
  var url = repoAPI(drive) + "/contents";
  if (path) url += "/" + encodePath(path);
  return url + "?ref=" + encodeURIComponent(branchRef(drive, ctx));
}

/**
 * @param {GitHubDriveState} drive
 * @param {Context} ctx
 * @param {string} path
 * @returns {{ tooMany?: boolean, data?: any }}
 */
function getContents(drive, ctx, path) {
  var parsed = parseGitHubResponse(
    request(drive, ctx, "GET", contentsPath(drive, ctx, path))
  );
  if (parsed.status === 403 && isTooManyFiles(parsed.data)) {
    return { tooMany: true };
  }
  throwIfGitHubError(parsed);
  return { data: parsed.data };
}

/**
 * @param {GitHubDriveState} drive
 * @param {Context} ctx
 * @param {string} path
 * @returns {Entry[]}
 */
function listViaTree(drive, ctx, path) {
  var tree = requestJSON(
    drive,
    ctx,
    "GET",
    repoAPI(drive) + "/git/trees/" + encodeURIComponent(treeShaAt(drive, ctx, path))
  );
  if (tree.truncated) {
    throw ErrRemoteApi(403, "GitHub tree listing is truncated for this directory");
  }
  var entries = [];
  var items = tree.tree || [];
  for (var i = 0; i < items.length; i++) {
    ctx.Err();
    var item = items[i];
    if (item.type === "commit") {
      entries.push(
        makeEntry(false, path ? pathUtils.join(path, item.path) : item.path, -1, false)
      );
      continue;
    }
    var isDir = item.type === "tree";
    entries.push(
      makeEntry(
        isDir,
        path ? pathUtils.join(path, item.path) : item.path,
        fileSize(item.size),
        true
      )
    );
  }
  return entries;
}

/**
 * @param {GitHubDriveState} drive
 * @param {Context} ctx
 * @param {string} path
 * @returns {string}
 */
function treeShaAt(drive, ctx, path) {
  var current = branchRef(drive, ctx);
  if (!path) {
    var root = requestJSON(
      drive,
      ctx,
      "GET",
      repoAPI(drive) + "/git/trees/" + encodeURIComponent(current)
    );
    return root.sha;
  }
  var parts = path.split("/");
  for (var i = 0; i < parts.length; i++) {
    ctx.Err();
    var tree = requestJSON(
      drive,
      ctx,
      "GET",
      repoAPI(drive) + "/git/trees/" + encodeURIComponent(current)
    );
    var found = null;
    var items = tree.tree || [];
    for (var j = 0; j < items.length; j++) {
      if (items[j].path === parts[i] && items[j].type === "tree") {
        found = items[j].sha;
        break;
      }
    }
    if (!found) throw ErrNotFound();
    current = found;
  }
  return current;
}

/**
 * @param {any} data
 * @param {string} [path]
 * @returns {Entry}
 */
function toEntry(data, path) {
  if (Array.isArray(data)) {
    return makeEntry(true, path, -1, true);
  }
  var entryPath = data.path || path;
  if (data.type === "dir") {
    return makeEntry(true, entryPath, -1, true);
  }
  if (data.type === "submodule") {
    return makeEntry(false, entryPath, -1, false);
  }
  return makeEntry(false, entryPath, fileSize(data.size), true);
}

/**
 * @param {any} data
 * @returns {boolean}
 */
function isTooManyFiles(data) {
  return !!(
    data &&
    data.message &&
    String(data.message).indexOf("too many files") >= 0
  );
}

/**
 * @param {GitHubDriveState} drive
 * @param {Context} ctx
 * @param {HttpMethod} method
 * @param {string} path
 * @returns {any}
 */
function requestJSON(drive, ctx, method, path) {
  var parsed = parseGitHubResponse(request(drive, ctx, method, path));
  throwIfGitHubError(parsed);
  return parsed.data;
}

/**
 * @param {GitHubDriveState} drive
 * @param {Context} ctx
 * @param {HttpMethod} method
 * @param {string} path
 */
function request(drive, ctx, method, path) {
  var headers = {
    Accept: "application/vnd.github+json",
    "User-Agent": "go-drive",
    "X-GitHub-Api-Version": "2022-11-28",
  };
  if (drive.token) {
    headers.Authorization = "Bearer " + drive.token;
  }
  if (DEBUG) console.log("http", method, path);
  return http(ctx, method, API_BASE + path, headers, null);
}

/**
 * @param {HttpResponse} resp
 */
function parseGitHubResponse(resp) {
  var status = resp.Status;
  var remaining = resp.Headers.Get("X-RateLimit-Remaining");
  var text = resp.Text();
  var data = null;
  if (text) {
    try {
      data = JSON.parse(text);
    } catch (e) {
      throw ErrRemoteApi(status, "Failed to parse JSON");
    }
  }
  return { status: status, remaining: remaining, data: data };
}

/**
 * @param {{ status: number, remaining: string, data: any }} parsed
 */
function throwIfGitHubError(parsed) {
  var status = parsed.status;
  var data = parsed.data;
  var message = (data && data.message) || "GitHub API error";
  if (status === 404) throw ErrNotFound();
  if (status === 401) throw ErrNotAllowed("Invalid GitHub token");
  if (status === 403) {
    if (parsed.remaining === "0" || /rate limit/i.test(message)) {
      throw ErrRemoteApi(
        403,
        "GitHub API rate limit exceeded. Provide a GitHub token for higher limits."
      );
    }
    throw ErrNotAllowed(message);
  }
  if (status < 200 || status >= 400) {
    throw ErrRemoteApi(status, message);
  }
}
