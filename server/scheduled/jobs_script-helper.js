/// <reference path="../../docs/scripts/env/jobs.d.ts"/>

function __copyOrMove__(isMove, from, to, override) {
  var drive = rootDrive.Get();
  var ctx = newContext();

  var fromEntry = drive.Get(ctx, from);

  try {
    var toEntry = drive.Get(ctx, to);
    if (toEntry.Type() === "dir") {
      to = pathUtils.join(to, pathUtils.base(fromEntry.Path()));
    }
  } catch (e) {
    if (!isNotFoundErr(e)) throw e;
  }

  var taskCtx = newTaskCtx(ctx);
  if (isMove) {
    return drive.Move(taskCtx, fromEntry, to, !!override);
  } else {
    return drive.Copy(taskCtx, fromEntry, to, !!override);
  }
}

function cp(from, to, override) {
  return __copyOrMove__(false, from, to, override);
}

function mv(from, to, override) {
  return __copyOrMove__(true, from, to, override);
}

function rm(path) {
  var drive = rootDrive.Get();
  return drive.Delete(newTaskCtx(newContext()), path);
}

function ls(path) {
  var drive = rootDrive.Get();
  return drive.List(newContext(), path);
}

function mkdir(path) {
  var drive = rootDrive.Get();
  return drive.MakeDir(newContext(), path);
}
