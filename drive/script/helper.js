(function (global, bridge) {
  const requireFunction = (name, e) => {
    if (typeof e !== "function") {
      throw new Error(name + " is required to be implemented");
    }
  };

  const validateForm = (form, name) => {
    if (!Array.isArray(form)) {
      throw new Error(name + " must be an array");
    }
    for (const item of form) {
      const field = item?.field;
      if (field && field.startsWith("_")) {
        throw new BadRequestError(
          "script form fields must not start with '_': " + field
        );
      }
    }
  };

  const formConfigured = (form, data) => {
    if (!form?.length) return true;
    return form.every((item) => {
      if (!item?.required) return true;
      const v = data?.[item.field];
      return v !== undefined && v !== null && v !== "";
    });
  };

  const bindSharedState = (drive) => {
    const props = Object.keys(drive).filter((key) => key.startsWith("$"));
    if (!props.length) return;
    const descriptors = {};
    const values = {};
    for (const key of props) {
      descriptors[key] = {
        configurable: false,
        get: () => bridge.getData(key),
        set: (v) => {
          bridge.setData({ [key]: v });
        },
        enumerable: true,
      };
      values[key] = drive[key];
    }
    bridge.initData(values);
    Object.defineProperties(drive, descriptors);
  };

  const bindDriveMethods = (drive, methods) => {
    const names = [
      "meta",
      "get",
      "list",
      "getReader",
      "save",
      "makeDir",
      "copy",
      "move",
      "delete",
      "upload",
      "getURL",
      "getThumbnail",
      "onInterval",
    ];
    for (const name of names) {
      if (typeof methods[name] !== "function") continue;
      drive[name] = methods[name];
      global["__drive_" + name] = methods[name].bind(drive);
    }
  };

  const rejectPromise = (name, value) => {
    if (value instanceof Promise) {
      throw new Error(name + " must not return a Promise");
    }
    return value;
  };

  const defineDrive = (setup, methods) => {
    if (!setup || typeof setup !== "object") {
      throw new Error("defineDrive setup is required");
    }
    if (!methods || typeof methods !== "object") {
      throw new Error("defineDrive methods are required");
    }
    requireFunction("get", methods.get);
    requireFunction("list", methods.list);
    if (typeof setup.createInstance !== "function") {
      throw new Error("createInstance is required");
    }
    if (
      typeof methods.getReader !== "function" &&
      typeof methods.getURL !== "function"
    ) {
      throw new Error("getReader or getURL is required");
    }

    const form = setup.configForm || [];
    validateForm(form, "configForm");
    global.__driveConfigForm = form;

    if (typeof setup.initConfig === "function") {
      global.__driveInitConfig = (config, utils) => {
        if (!formConfigured(form, config)) {
          return { configured: false };
        }
        const result = rejectPromise(
          "initConfig",
          setup.initConfig(config, utils)
        );
        if (result?.form) validateForm(result.form, "initConfig form");
        return result === undefined ? null : result;
      };
    } else {
      global.__driveInitConfig = null;
    }

    if (typeof setup.init === "function") {
      global.__driveInit = (data, config, utils) => {
        rejectPromise("init", setup.init(data, config, utils));
      };
    } else {
      global.__driveInit = null;
    }

    global.__driveCreate = (config, utils) => {
      if (!formConfigured(form, config)) {
        throw new NotAllowedError("drive not configured");
      }
      if (typeof setup.validateConfig === "function") {
        rejectPromise("validateConfig", setup.validateConfig(config));
      }

      const drive = rejectPromise(
        "createInstance",
        setup.createInstance(config, utils)
      );
      if (!drive || typeof drive !== "object") {
        throw new Error("createInstance must return an object");
      }
      drive.cache = utils.createCache();

      bindDriveMethods(drive, methods);
      bindSharedState(drive);
      Object.freeze(drive);

      return {
        writable: drive.writable !== false,
        entryCacheTTL: drive.entryCacheTTL,
        intervals: normalizeIntervals(drive.intervals),
      };
    };
  };

  const normalizeIntervals = (raw) => {
    if (!raw) return [];
    if (!Array.isArray(raw)) {
      throw new Error("intervals must be an array");
    }
    return raw.map((item) => {
      const { name = "", interval, timeout, immediately } = item || {};
      return {
        name,
        interval,
        timeout,
        immediately: immediately === true,
      };
    });
  };

  const entryCacheTTLFormItem = (defaultValue) => {
    const item = {
      label: "Cache TTL",
      field: "cache_ttl",
      type: "text",
      description:
        "Cache time to live, if omitted, no cache. Valid time units are 'ms', 's', 'm', 'h'.",
    };
    if (defaultValue) item.defaultValue = String(defaultValue);
    return item;
  };

  const LOCAL_PROVIDER_CHUNK_SIZE = 5 * 1024 * 1024;

  const useLocalProvider = (size) => {
    if (size <= LOCAL_PROVIDER_CHUNK_SIZE) {
      return { provider: "local" };
    }
    return { provider: "localChunk" };
  };

  const useCustomProvider = (data) => {
    if (typeof data === "string") {
      throw new Error(
        "useCustomProvider(config) does not take an uploader name"
      );
    }
    return {
      provider: "custom",
      config: Object.assign({}, data, {
        uploader: bridge.name,
        uploaderVersion: bridge.version,
      }),
    };
  };

  defineFrozenGlobal("defineDrive", defineDrive);
  defineFrozenGlobal("entryCacheTTLFormItem", entryCacheTTLFormItem);
  defineFrozenGlobal("LOCAL_PROVIDER_CHUNK_SIZE", LOCAL_PROVIDER_CHUNK_SIZE);
  defineFrozenGlobal("useLocalProvider", useLocalProvider);
  defineFrozenGlobal("useCustomProvider", useCustomProvider);
})(this, __goDrive_bridge__);
