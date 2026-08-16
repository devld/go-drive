var global = this;

function __requireFunction__(name, e) {
  if (typeof e !== "function") {
    throw new Error(name + " is required to be implemented");
  }
}

function formFields(form) {
  var keys = [];
  if (!form) return keys;
  for (var i = 0; i < form.length; i++) {
    if (form[i] && form[i].Field) keys.push(form[i].Field);
  }
  return keys;
}

function loadFormData(utils, form) {
  var keys = formFields(form);
  if (!keys.length) return {};
  return utils.Data.Load.apply(utils.Data, keys);
}

function formConfigured(form, data) {
  if (!form || !form.length) return false;
  for (var i = 0; i < form.length; i++) {
    var item = form[i];
    if (item && item.Required && !data[item.Field]) return false;
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
        return getData(key);
      },
      set: function (v) {
        var dat = {};
        dat[key] = v;
        setData(dat);
      },
      enumerable: true,
    };
    values[key] = drive[key];
  });
  setData(values);
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
    "hasThumbnail",
  ];
  for (var i = 0; i < names.length; i++) {
    var name = names[i];
    if (typeof methods[name] !== "function") continue;
    drive[name] = methods[name];
    global["__drive_" + name] = methods[name].bind(drive);
  }
}

function oauthPair(setup, config, data) {
  if (typeof setup.oauthRequest !== "function") return null;
  var pair = setup.oauthRequest(config, data);
  if (!pair || !pair.request || !pair.credentials) {
    throw new Error("oauthRequest must return { request, credentials }");
  }
  return pair;
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

  global.__driveInitConfig = function (ctx, config, utils) {
    var data = loadFormData(utils, form);
    var formReady = formConfigured(form, data);
    var pair = oauthPair(setup, utils.Config, data);

    if (pair) {
      if (!formReady) {
        return { Configured: false, Form: form, Value: data };
      }
      var result = utils.OAuthInitConfig(pair.request, pair.credentials);
      var initConfig = result.Config;
      var oauthResp = result.Response;
      if (!oauthResp) {
        return {
          Configured: false,
          Form: form,
          Value: data,
          OAuth: initConfig.OAuth,
        };
      }
      var oauth = initConfig.OAuth || {};
      if (typeof setup.oauthPrincipal === "function") {
        oauth.Principal = setup.oauthPrincipal(ctx, oauthResp);
      }
      return {
        Configured: true,
        Form: form,
        Value: data,
        OAuth: oauth,
      };
    }

    return { Configured: formReady, Form: form, Value: data };
  };

  global.__driveInit = function (ctx, data, config, utils) {
    if (typeof setup.validateConfig === "function") {
      setup.validateConfig(data);
    }
    var fields = formFields(form);
    var toSave = {};
    var i;
    for (i = 0; i < fields.length; i++) {
      var key = fields[i];
      if (data[key]) toSave[key] = data[key];
    }
    if (Object.keys(toSave).length) {
      utils.Data.Save(toSave);
    }

    var saved = loadFormData(utils, form);
    var pair = oauthPair(setup, utils.Config, saved);
    if (!pair) return;
    if (!formConfigured(form, saved)) return;
    utils.OAuthInit(ctx, data, pair.request, pair.credentials);
  };

  global.__driveCreate = function (ctx, config, utils) {
    var data = loadFormData(utils, form);
    if (form.length && !formConfigured(form, data)) {
      throw ErrNotAllowed("drive not configured");
    }

    var drive = setup.createInstance(data, utils);
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
