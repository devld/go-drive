// @name GitHub
// @version 1.0.4
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

const API_BASE = "https://api.github.com";
const RAW_BASE = "https://raw.githubusercontent.com";
const READONLY_META = { readable: true, writable: false };
const UNREADABLE_META = { readable: false, writable: false };

/**
 * @typedef {DriveAdapterState & {
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
        label: "Repository Owner",
        description: "The owner of the GitHub repository (e.g. 'devld' or 'facebook')",
        type: "text",
        field: "owner",
        required: true,
      },
      {
        label: "Repository Name",
        description: "The name of the GitHub repository (e.g. 'go-drive' or 'react')",
        type: "text",
        field: "repo",
        required: true,
      },
      {
        label: "Branch",
        description: "Branch, tag, or commit SHA. Leave empty to use the repository default branch.",
        type: "text",
        field: "branch",
        required: false,
      },
      {
        label: "GitHub Token",
        description: "Personal access token for private repositories and higher rate limits",
        type: "password",
        field: "token",
        required: false,
      },
      entryCacheTTLFormItem("5m"),
    ],

    validateConfig(config) {
      if (config.owner && /[/\s]/.test(config.owner)) {
        throw new BadRequestError("invalid repository owner");
      }
      if (config.repo && /[/\s]/.test(config.repo)) {
        throw new BadRequestError("invalid repository name");
      }
    },

    /**
     * @param {SM} config
     * @returns {GitHubDriveState}
     */
    createInstance(config) {
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
    get(path) {
      console.debug("get", path);
      const result = getContents(this, path);
      if (result.tooMany) {
        return makeEntry(true, path, -1, true);
      }
      return toEntry(result.data, path);
    },

    list(path) {
      console.debug("list", path);
      const result = getContents(this, path);
      if (result.tooMany) {
        return listViaTree(this, path);
      }
      if (!Array.isArray(result.data)) {
        throw new NotFoundError();
      }
      return result.data.map((item) => toEntry(item));
    },

    getURL(entry) {
      console.debug("getURL", entry.path);
      if (entry.isDir || entry.meta?.readable === false) {
        throw new UnsupportedError();
      }
      const ref = rawRef(this);
      const url =
        `${RAW_BASE}/${encodeURIComponent(this.owner)}/${encodeURIComponent(this.repo)}/${ref}/${encodePath(entry.path)}`;
      if (!this.token) {
        return { url };
      }
      return {
        url,
        header: {
          Authorization: "Bearer " + this.token,
          "User-Agent": "go-drive",
        },
        proxy: true,
      };
    },
  }
);

/**
 * @param {GitHubDriveState} drive
 * @returns {string}
 */
function repoAPI(drive) {
  return `/repos/${encodeURIComponent(drive.owner)}/${encodeURIComponent(drive.repo)}`;
}

/**
 * @param {GitHubDriveState} drive
 * @returns {string}
 */
function branchRef(drive) {
  if (drive.branch) return drive.branch;
  if (drive.$resolvedBranch) return drive.$resolvedBranch;
  const repo = requestJSON(drive, "GET", repoAPI(drive));
  const resolved = repo.default_branch || "main";
  drive.$resolvedBranch = resolved;
  return resolved;
}

/**
 * raw.githubusercontent.com treats `/` in the ref as extra path segments, so
 * refs that contain slashes are resolved to a commit SHA.
 * @param {GitHubDriveState} drive
 * @returns {string}
 */
function rawRef(drive) {
  const ref = branchRef(drive);
  if (!ref.includes("/")) {
    return encodeURIComponent(ref);
  }
  const commit = requestJSON(
    drive,
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
  return path.split("/").map(encodeURIComponent).join("/");
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
 * @returns {EntryRecord}
 */
function makeEntry(isDir, path, size, readable) {
  return {
    isDir,
    path: pathUtils.clean(path || ""),
    size: isDir ? -1 : size,
    modTime: -1,
    meta: readable ? READONLY_META : UNREADABLE_META,
  };
}

/**
 * @param {GitHubDriveState} drive
 * @param {string} path
 */
function contentsPath(drive, path) {
  let url = repoAPI(drive) + "/contents";
  if (path) url += "/" + encodePath(path);
  return url + "?ref=" + encodeURIComponent(branchRef(drive));
}

/**
 * @param {GitHubDriveState} drive
 * @param {string} path
 * @returns {{ tooMany?: boolean, data?: any }}
 */
function getContents(drive, path) {
  const parsed = parseGitHubResponse(
    request(drive, "GET", contentsPath(drive, path))
  );
  if (parsed.status === 403 && isTooManyFiles(parsed.data)) {
    return { tooMany: true };
  }
  throwIfGitHubError(parsed);
  return { data: parsed.data };
}

/**
 * @param {GitHubDriveState} drive
 * @param {string} path
 * @returns {EntryRecord[]}
 */
function listViaTree(drive, path) {
  const tree =     requestJSON(
    drive,
    "GET",
    repoAPI(drive) + "/git/trees/" + encodeURIComponent(treeShaAt(drive, path))
  );
  if (tree.truncated) {
    throw new RemoteApiError(403, "GitHub tree listing is truncated for this directory");
  }
  return (tree.tree || []).map((item) => {
    if (item.type === "commit") {
      return makeEntry(false, path ? pathUtils.join(path, item.path) : item.path, -1, false);
    }
    const isDir = item.type === "tree";
    return makeEntry(
      isDir,
      path ? pathUtils.join(path, item.path) : item.path,
      fileSize(item.size),
      true
    );
  });
}

/**
 * @param {GitHubDriveState} drive
 * @param {string} path
 * @returns {string}
 */
function treeShaAt(drive, path) {
  let current = branchRef(drive);
  if (!path) {
    const root =     requestJSON(
    drive,
      "GET",
      repoAPI(drive) + "/git/trees/" + encodeURIComponent(current)
    );
    return root.sha;
  }
  for (const part of path.split("/")) {
    const tree =     requestJSON(
    drive,
      "GET",
      repoAPI(drive) + "/git/trees/" + encodeURIComponent(current)
    );
    const found = (tree.tree || []).find((item) => item.path === part && item.type === "tree");
    if (!found) throw new NotFoundError();
    current = found.sha;
  }
  return current;
}

/**
 * @param {any} data
 * @param {string} [path]
 * @returns {EntryRecord}
 */
function toEntry(data, path) {
  if (Array.isArray(data)) {
    return makeEntry(true, path || "", -1, true);
  }
  const entryPath = data.path || path || "";
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
  return !!data?.message && String(data.message).includes("too many files");
}

/**
 * @param {GitHubDriveState} drive
 * @param {HttpMethod} method
 * @param {string} path
 * @returns {any}
 */
function requestJSON(drive, method, path) {
  const parsed = parseGitHubResponse(request(drive, method, path));
  throwIfGitHubError(parsed);
  return parsed.data;
}

/**
 * @param {GitHubDriveState} drive
 * @param {HttpMethod} method
 * @param {string} path
 */
function request(drive, method, path) {
  const headers = {
    Accept: "application/vnd.github+json",
    "User-Agent": "go-drive",
    "X-GitHub-Api-Version": "2022-11-28",
  };
  if (drive.token) {
    headers.Authorization = "Bearer " + drive.token;
  }
  console.debug("http", method, path);
  return http(API_BASE + path, { method, headers });
}

/**
 * @param {HttpResponse} resp
 */
function parseGitHubResponse(resp) {
  const status = resp.status;
  const remaining = resp.headers.get("X-RateLimit-Remaining");
  const text = resp.text();
  let data = null;
  if (text) {
    try {
      data = JSON.parse(text);
    } catch (e) {
      throw new RemoteApiError(status, "Failed to parse JSON");
    }
  }
  return { status, remaining, data };
}

/**
 * @param {{ status: number, remaining: string, data: any }} parsed
 */
function throwIfGitHubError(parsed) {
  const { status, data } = parsed;
  const message = data?.message || "GitHub API error";
  if (status === 404) throw new NotFoundError();
  if (status === 401) throw new NotAllowedError("Invalid GitHub token");
  if (status === 403) {
    if (parsed.remaining === "0" || /rate limit/i.test(message)) {
      throw new RemoteApiError(
        403,
        "GitHub API rate limit exceeded. Provide a GitHub token for higher limits."
      );
    }
    throw new NotAllowedError(message);
  }
  if (status < 200 || status >= 400) {
    throw new RemoteApiError(status, message);
  }
}
