// My Drive
// > Here and below is the drive's description
// > It supports `markdown`
// > It will be shown above the configuration form
// 
// Please fill the required `Some Field` below, and .......
// 
// > You must leave an empty line below indicate the description is ended


/// <reference path="./scripts/env/drive.d.ts"/>

/**
 * @returns {FormItem[]}
 */
function initForm() {
    return [
        {
            Label: "Some Field",
            Description: 'This is a required field, you can get it from......',
            Type: "text",
            Field: "some_field",
            Required: true,
        },
    ];
}

defineCreate(function (ctx, config, utils) {
    // use saved configuration data to create a drive instance
    var data = utils.Data.Load("some_field", "_id");
    return new DriveHolder(data, utils.CreateCache());
});

defineInitConfig(function (ctx, config, utils) {
    // load saved configuration data
    var data = utils.Data.Load("some_field");

    // create init form
    var form = initForm();

    if (!data.some_field) {
        // if some required fields not provided
        return { Configured: false, Form: form, Value: data };
    }

    return { Configured: true, Form: form, Value: data };
});

defineInit(function (ctx, data, config, utils) {
    var data = utils.Data.Load("some_field", "_id");
    if (!data.some_field) {
        // if the configuration data not saved, just save it
        utils.Data.Save(data);
        return;
    }

    // we generate an unique drive id here
    // the id is to identify the Entry is ours in Copy/Move
    if (!data._id) {
        var id = (Math.random() * 1000000).toFixed(0);
        utils.Data.Save({ _id: id });
    }
});

/**
 * @typedef DriveDataHolder
 * @property {DriveCache} _cache
 * @property {Duration} _cacheTTL
 * @property {string} _rid
 * @property {string} someField TODO
 */

/**
 * @typedef {Drive & DriveDataHolder} MyDrive
 */

/**
 * @type {MyDrive}
 */
var DriveImpl = {
    meta: function (ctx) { },
    get: function (ctx, path) {
        if (DEBUG) console.log("get", path);
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

        // TODO request

        this._cache.PutEntry(entry, this._cacheTTL);
        return entry;
    },
    save: function (ctx, path, size, override, reader) {
        // TODO upload
        this._cache.Evict(path, false);
        this._cache.Evict(pathUtils.parent(path), false);
        return this.get(ctx, path);
    },
    makeDir: function (ctx, path) {
        // TODO request
        this._cache.Evict(pathUtils.parent(path), false);
        return this.get(ctx, path);
    },
    copy: function (ctx, from, to, override) {
        var dat = from.Data();
        if (!dat || dat.d !== this._rid) {
            throw ErrUnsupported();
        }

        // TODO request

        this._cache.Evict(to, true);
        this._cache.Evict(pathUtils.pa(to), false);
        return this.get(ctx, to);
    },
    move: function (ctx, from, to, override) {
        from = from.Unwrap();
        var dat = from.Data();
        if (!dat || dat.d !== this._rid) {
            throw ErrUnsupported();
        }

        // TODO request

        this._cache.Evict(to, true);
        this._cache.Evict(pathUtils.parent(to), false);
        this._cache.Evict(from.Path, true);
        this._cache.Evict(pathUtils.parent(from.Path()), false);
        return this.get(ctx, to);
    },
    list: function (ctx, path) {
        if (DEBUG) console.log("list", path);
        var _this = this;
        var cachedItems = this._cache.GetChildren(path);
        if (cachedItems) {
            return cachedItems.map(cacheItemToEntry);
        }

        // TODO request

        this._cache.PutChildren(path, result, this._cacheTTL);
        return result;
    },
    delete: function (ctx, path) {
        if (DEBUG) console.log("delete", path);

        // TODO request

        this._cache.Evict(pathUtils.parent(path), false);
        this._cache.Evict(path, true);
    },
    upload: function (ctx, path, size, override, config) {
        return useLocalProvider(size);
    },
    getReader: function (ctx, entry, start, size) {
        throw ErrUnsupported();
    },
    getURL: function (ctx, entry) {
        if (DEBUG) console.log("getURL", entry.Path);
        // TODO
    },
    hasThumbnail: function (entry) {
        // TODO
        return false;
    },
    getThumbnail: function (ctx, entry) {
        // TODO
    },
};

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
 * @constructor
 * @param {any} data
 * @param {DriveCache} cache
 */
function DriveHolder(data, cache) {
    // Notice: instance variables that start with $ are shared variables.
    // Shared variables can be modified at runtime, while the rest of the variables cannot.
    // In other words, variables that do not start with $ are constants and can only be initialized in defineCreate
    this.$sharedVar = 123;

    this._rid = data._id;
    // save some instance data

    // TODO
    this.someField = data.some_field;

    this._cache = cache;
    this._cacheTTL = ms(2 * 60 * 60 * 1000);
}

Object.keys(DriveImpl).forEach(function (fn) {
    DriveHolder.prototype[fn] = DriveImpl[fn];
});
