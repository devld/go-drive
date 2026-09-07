/// <reference path="../global.d.ts"/>

/** Root host `Drive` of this go-drive instance (not `defineDrive` `this`). */
declare const drive: Drive;

/** Write a log line for this job run. */
declare function log(...msg: any[]): void;

/** Copy `from` to `to`. Paths may include wildcards. */
declare function cp(from: string, to: string, override: boolean): Entry;
/** Move `from` to `to`. Paths may include wildcards. */
declare function mv(from: string, to: string, override: boolean): Entry;
/** Delete a path. Supports wildcards. */
declare function rm(path: string): void;
/** List a directory. */
declare function ls(path: string): readonly Entry[];
/** Create a directory. */
declare function mkdir(path: string): Entry;

/**
 * Trigger that started this run.
 * Manual runs are `undefined`. `entry` is a file-event trigger; `cron` is a schedule.
 */
declare const $event:
  | {
      readonly type: "entry";
      readonly data?: {
        readonly eventType: "updated" | "deleted";
        readonly includeDescendants: boolean;
        readonly path: string;
      };
    }
  | {
      readonly type: "cron";
    }
  | undefined;
