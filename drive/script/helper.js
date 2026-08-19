var global = this;

function __requireFunction__(name, e) {
  if (typeof e !== "function") {
    throw new Error(name + " is required to be implemented");
  }
}

function validateForm(form, name) {
  if (!Array.isArray(form)) {
    throw new Error(name + " must be an array");
  }
  for (var i = 0; i < form.length; i++) {
    var field = form[i] && form[i].Field;
    if (field && field.charAt(0) === "_") {
      throw ErrBadRequest("script form fields must not start with '_': " + field);
    }
  }
}

function formConfigured(form, data) {
  if (!form || !form.length) return true;
  for (var i = 0; i < form.length; i++) {
    var item = form[i];
    if (
      item &&
      item.Required &&
      (!data || data[item.Field] === undefined || data[item.Field] === null || data[item.Field] === "")
    ) {
      return false;
    }
  }
  return true;
}

function bindSharedState(drive) {
  var props = Object.keys(drive).filter(function (key) {
    return key[0] === "$";
  });
  if (!props.length) return;
  var descriptors = {};
  var values = {};
  props.forEach(function (key) {
    descriptors[key] = {
      configurable: false,
      get: function () {
        return __getData(key);
      },
      set: function (v) {
        var dat = {};
        dat[key] = v;
        __setData(dat);
      },
      enumerable: true,
    };
    values[key] = drive[key];
  });
  __setData(values);
  Object.defineProperties(drive, descriptors);
}

function bindDriveMethods(drive, methods) {
  var names = [
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
  ];
  for (var i = 0; i < names.length; i++) {
    var name = names[i];
    if (typeof methods[name] !== "function") continue;
    drive[name] = methods[name];
    global["__drive_" + name] = methods[name].bind(drive);
  }
}

function defineDrive(setup, methods) {
  if (!setup || typeof setup !== "object") {
    throw new Error("defineDrive setup is required");
  }
  if (!methods || typeof methods !== "object") {
    throw new Error("defineDrive methods are required");
  }
  __requireFunction__("get", methods.get);
  __requireFunction__("list", methods.list);
  if (typeof setup.createInstance !== "function") {
    throw new Error("createInstance is required");
  }
  if (
    typeof methods.getReader !== "function" &&
    typeof methods.getURL !== "function"
  ) {
    throw new Error("getReader or getURL is required");
  }

  var form = setup.configForm || [];
  validateForm(form, "configForm");
  global.__driveConfigForm = form;

  if (typeof setup.initConfig === "function") {
    global.__driveInitConfig = function (ctx, config, utils) {
      if (!formConfigured(form, config)) {
        return { Configured: false };
      }
      var result = setup.initConfig(ctx, config, utils);
      if (result && result.Form) validateForm(result.Form, "initConfig form");
      return result === undefined ? null : result;
    };
  } else {
    global.__driveInitConfig = null;
  }

  if (typeof setup.init === "function") {
    global.__driveInit = function (ctx, data, config, utils) {
      setup.init(ctx, data, config, utils);
    };
  } else {
    global.__driveInit = null;
  }

  global.__driveCreate = function (ctx, config, utils) {
    if (!formConfigured(form, config)) {
      throw ErrNotAllowed("drive not configured");
    }
    if (typeof setup.validateConfig === "function") {
      setup.validateConfig(config);
    }

    var drive = setup.createInstance(ctx, config, utils);
    if (!drive || typeof drive !== "object") {
      throw new Error("createInstance must return an object");
    }
    drive.cache = utils.CreateCache();
    drive.own = function (from) {
      return __ownEntry(from);
    };

    bindDriveMethods(drive, methods);
    bindSharedState(drive);
    Object.freeze(drive);

    return {
      Writable: drive.writable !== false,
      EntryCacheTTL: drive.entryCacheTTL || "",
    };
  };
}

function entryCacheTTLFormItem(defaultValue) {
  var item = {
    Label: "CacheTTL",
    Field: "cache_ttl",
    Type: "text",
    Description:
      "Cache time to live, if omitted, no cache. Valid time units are 'ms', 's', 'm', 'h'.",
  };
  if (defaultValue) item.DefaultValue = String(defaultValue);
  return item;
}

var LocalProviderChunkSize = 5 * 1024 * 1024;

function useLocalProvider(size) {
  if (size <= LocalProviderChunkSize) {
    return { Provider: "local" };
  }
  return { Provider: "localChunk" };
}

function useCustomProvider(data) {
  if (typeof data === "string") {
    throw new Error("useCustomProvider(config) does not take an uploader name");
  }
  return {
    Provider: "custom",
    Config: Object.assign({}, data, {
      uploader: __driveUploaderName,
      uploaderVersion: __driveScriptVersion,
    }),
  };
}
