function __consoleWrite__(level, args) {
  var argsArr = [level];
  for (var i = 0; i < args.length; i++) {
    argsArr.push(args[i]);
  }
  consoleWrite.apply(undefined, argsArr);
}

var console = {
  debug: function () {
    __consoleWrite__("DEBUG", arguments);
  },
  error: function () {
    __consoleWrite__("ERROR", arguments);
  },
  info: function () {
    __consoleWrite__(" INFO", arguments);
  },
  log: function () {
    __consoleWrite__("  LOG", arguments);
  },
  warn: function () {
    __consoleWrite__(" WARN", arguments);
  },
};

function ms(ms) {
  return ms * 1000000;
}

function toDate(goTime) {
  return new Date(goTime.UnixMilli());
}

var __dayjs__ = dayjs;
dayjs = function () {
  if (
    arguments.length === 1 &&
    arguments[0] &&
    typeof arguments[0] === "object" &&
    typeof arguments[0].UnixMilli === "function"
  ) {
    return __dayjs__(toDate(arguments[0]));
  }
  return __dayjs__.apply(this, arguments);
};

// from https://github.com/christiansany/object-assign-polyfill
if (typeof Object.assign != "function") {
  Object.assign = function (target) {
    // .length of function is 2
    "use strict";
    if (target == null) {
      // TypeError if undefined or null
      throw new TypeError("Cannot convert undefined or null to object");
    }

    var to = Object(target);

    for (var index = 1; index < arguments.length; index++) {
      var nextSource = arguments[index];

      if (nextSource != null) {
        // Skip over if undefined or null
        for (var nextKey in nextSource) {
          // Avoid bugs when hasOwnProperty is shadowed
          if (Object.prototype.hasOwnProperty.call(nextSource, nextKey)) {
            to[nextKey] = nextSource[nextKey];
          }
        }
      }
    }
    return to;
  };
}

function __newGoError__(type, status, msg) {
  return new Error("E:" + type + ":" + (status || 0) + ":" + (msg || ""));
}

function __isTypeOfErr__(e, type) {
  var message = e && e.message;
  return typeof message === "string" && message.indexOf("E:" + type) === 0;
}

function ErrBadRequest(msg) {
  return __newGoError__("BAD_REQUEST", 0, msg);
}

function isBadRequestErr(e) {
  return __isTypeOfErr__(e, "BAD_REQUEST");
}

function ErrNotFound(msg) {
  return __newGoError__("NOT_FOUND", 0, msg);
}

function isNotFoundErr(e) {
  return __isTypeOfErr__(e, "NOT_FOUND");
}

function ErrNotAllowed(msg) {
  return __newGoError__("NOT_ALLOWED", 0, msg);
}

function isNotAllowedErr(e) {
  return __isTypeOfErr__(e, "NOT_ALLOWED");
}

function ErrUnsupported(msg) {
  return __newGoError__("UNSUPPORTED", 0, msg);
}

function isUnsupportedErr(e) {
  return __isTypeOfErr__(e, "UNSUPPORTED");
}

function ErrRemoteApi(status, msg) {
  return __newGoError__("REMOTE_API", status, msg);
}

function isRemoteApiErr(e) {
  return __isTypeOfErr__(e, "REMOTE_API");
}

var pathUtils = Object.freeze({
  clean: function (path) {
    if (!path) return "";
    var segments = path.split("/").filter(Boolean);
    var paths = [];
    segments.forEach(function (s) {
      if (s === ".") return;
      if (s === "..") paths.pop();
      else paths.push(s);
    });
    return paths.join("/");
  },
  join: function () {
    var segments = [];
    for (var i = 0; i < arguments.length; i++) {
      segments.push(arguments[i]);
    }
    return pathUtils.clean(
      segments.filter(Boolean).join("/").replace(/\/+/g, "/")
    );
  },
  parent: function (path) {
    if (!path) return "";
    var i = path.lastIndexOf("/");
    if (i === -1) return "";
    return path.substring(0, i);
  },
  base: function (path) {
    if (!path) return "";
    var i = path.lastIndexOf("/");
    if (i === -1) return path;
    return path.substring(i + 1);
  },
  ext: function (path) {
    if (!path) return "";
    var i = path.lastIndexOf(".");
    if (i === -1) return "";
    return path.substring(i + 1).toLowerCase();
  },
  isRoot: function (path) {
    return path === "";
  },
});

var HASH = Object.freeze({
  MD5: 1,
  SHA1: 2,
  SHA256: 3,
  SHA512: 4,
});

var encUtils = Object.freeze({
  toHex: __encToHex__,
  fromHex: __encFromHex__,
  base64Encode: __encBase64Encode__,
  base64Decode: __encBase64Decode__,
  urlBase64Encode: __encURLBase64Encode__,
  urlBase64Decode: __encURLBase64Decode__,
  newHash: __newHash__,
  hmac: __hmac__,
});

var SEEK_START = 0;
var SEEK_CURRENT = 1;
var SEEK_END = 2;
