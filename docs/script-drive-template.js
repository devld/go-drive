// @name My Drive
// @version 1.0.0
// @uploader my-drive-uploader.js
// @description
// > Here and below is the drive's description
// > It supports `markdown`
// > It will be shown above the configuration form
//
// Please fill the required `Some Field` below, and .......
//
// > You must leave an empty line below indicate the description is ended

/// <reference path="./scripts/env/drive.d.ts"/>

defineDrive(
  {
    configForm: [
      {
        Label: "Some Field",
        Description: "This is a required field, you can get it from......",
        Type: "text",
        Field: "some_field",
        Required: true,
      },
      entryCacheTTLFormItem("2h"),
    ],

    createInstance: function (data) {
      return {
        entryCacheTTL: data.cache_ttl,
        someField: data.some_field,
      };
    },
  },
  {
    get: function (ctx, path) {
      if (DEBUG) console.log("get", path);
      // TODO request
      return {
        Path: path,
        IsDir: false,
        Size: -1,
        ModTime: -1,
      };
    },

    list: function (ctx, path) {
      if (DEBUG) console.log("list", path);
      // TODO request
      return [];
    },

    save: function (ctx, path, size, override, reader) {
      // TODO upload
    },

    makeDir: function (ctx, path) {
      // TODO request
    },

    copy: function (ctx, from, to, override) {
      throw ErrUnsupported();
    },

    move: function (ctx, from, to, override) {
      throw ErrUnsupported();
    },

    delete: function (ctx, path) {
      if (DEBUG) console.log("delete", path);
      // TODO request
    },

    getURL: function (ctx, entry) {
      if (DEBUG) console.log("getURL", entry.Path);
      // TODO
    },
  }
);
