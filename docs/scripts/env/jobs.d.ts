/// <reference path="../global.d.ts"/>

/** Root Drive of this go-drive instance. */
declare const drive: DriveInstance;

/** Write a log line for this job run. */
declare function log(...msg: any[]): void;

/** Copy `from` to `to`. Paths may include wildcards. */
declare function cp(from: string, to: string, override: boolean): DriveEntry;
/** Move `from` to `to`. Paths may include wildcards. */
declare function mv(from: string, to: string, override: boolean): DriveEntry;
/** Delete a path. Supports wildcards. */
declare function rm(path: string): void;
/** List a directory. */
declare function ls(path: string): DriveEntry[];
/** Create a directory. */
declare function mkdir(path: string): DriveEntry;

/**
 * Trigger that started this run.
 * Manual runs are `undefined`. `entry` is a file-event trigger; `cron` is a schedule.
 */
declare const $event:
  | {
      type: "entry";
      data?: {
        eventType: "updated" | "deleted";
        includeDescendants: boolean;
        path: string;
      };
    }
  | {
      type: "cron";
    }
  | undefined;
