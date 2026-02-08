/// <reference path="../../docs/scripts/env/jobs.d.ts"/>

function __copyOrMove__(isMove, from, to, override) {
  var ctx = newTaskCtx(newContext());

  var fromEntries = findEntries(ctx, drive, from);

  for (var i = 0; i < fromEntries.length; i++) {
    var fromEntry = fromEntries[i];
    var toPath = pathUtils.join(to, fromEntry.Name());
    if (isMove) {
      drive.Move(ctx, fromEntry, toPath, !!override);
    } else {
      drive.Copy(ctx, fromEntry, toPath, !!override);
    }
  }
}

function cp(from, to, override) {
  return __copyOrMove__(false, from, to, override);
}

function mv(from, to, override) {
  return __copyOrMove__(true, from, to, override);
}

function rm(path) {
  var ctx = newTaskCtx(newContext());
  var entries = findEntries(ctx, drive, path);
  for (var i = entries.length - 1; i >= 0; i--) {
    try {
      drive.Delete(ctx, entries[i].Path());
    } catch (e) {
      if (!isNotFoundErr(e)) throw e;
    }
  }
}

function ls(path) {
  return drive.List(newContext(), path);
}

function mkdir(path) {
  return drive.MakeDir(newContext(), path);
}
