(function (global) {
  const defineFrozenGlobal = (name, value) => {
    if (
      value !== null &&
      (typeof value === "object" || typeof value === "function")
    ) {
      Object.freeze(value);
    }
    Object.defineProperty(global, name, {
      value,
      writable: false,
      configurable: false,
      enumerable: true,
    });
  };

  Object.defineProperty(global, "defineFrozenGlobal", {
    value: defineFrozenGlobal,
    writable: false,
    configurable: false,
    enumerable: false,
  });

  defineFrozenGlobal("ms", (ms) => ms * 1000000);

  const pathUtils = {
    clean(path) {
      if (!path) return "";
      const paths = [];
      for (const s of path.split("/").filter(Boolean)) {
        if (s === ".") continue;
        if (s === "..") paths.pop();
        else paths.push(s);
      }
      return paths.join("/");
    },
    join(...segments) {
      return pathUtils.clean(
        segments.filter(Boolean).join("/").replace(/\/+/g, "/")
      );
    },
    parent(path) {
      if (!path) return "";
      const idx = path.lastIndexOf("/");
      if (idx === -1) return "";
      return path.substring(0, idx);
    },
    base(path) {
      if (!path) return "";
      const idx = path.lastIndexOf("/");
      if (idx === -1) return path;
      return path.substring(idx + 1);
    },
    ext(path) {
      if (!path) return "";
      const idx = path.lastIndexOf(".");
      if (idx === -1) return "";
      return path.substring(idx + 1).toLowerCase();
    },
    isRoot(path) {
      return path === "";
    },
  };
  defineFrozenGlobal("pathUtils", pathUtils);
  defineFrozenGlobal("flattenEntriesTree", (root, deepFirst) => {
    const result = [];
    const active = new Set();
    const visit = (node) => {
      if (!node || typeof node !== "object" || !(node.entry instanceof Entry)) {
        throw new TypeError("not an EntryTreeNode");
      }
      if (active.has(node)) throw new TypeError("cyclic EntryTreeNode");
      active.add(node);
      if (!deepFirst && !node.excluded) result.push(node);
      if (node.children != null) {
        if (!Array.isArray(node.children)) {
          throw new TypeError("children must be an array");
        }
        for (const child of node.children) visit(child);
      }
      if (deepFirst && !node.excluded) result.push(node);
      active.delete(node);
    };
    visit(root);
    return result;
  });
  defineFrozenGlobal("dayjs", global.dayjs);
})(this);
