/// <reference path="../global.d.ts"/>

declare const drive: DriveInstance;

declare function log(s: string): void;

/** copy files */
declare function cp(from: string, to: string, override: boolean): DriveEntry;
/** move files */
declare function mv(from: string, to: string, override: boolean): DriveEntry;
/** delete files */
declare function rm(path: string): void;
/** list directory */
declare function ls(path: string): DriveEntry[];
/** create directory */
declare function mkdir(path: string): DriveEntry;
