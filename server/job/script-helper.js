/// <reference path="../../docs/scripts/env/jobs.d.ts"/>

(function (global) {
  const copyOrMove = (isMove, from, to, override) => {
    for (const fromEntry of findEntries(drive, from)) {
      const toPath = pathUtils.join(to, fromEntry.name);
      if (isMove) {
        drive.move(fromEntry, toPath, !!override, progress);
      } else {
        drive.copy(fromEntry, toPath, !!override, progress);
      }
    }
  };

  defineFrozenGlobal("cp", (from, to, override) => {
    copyOrMove(false, from, to, override);
  });

  defineFrozenGlobal("mv", (from, to, override) => {
    copyOrMove(true, from, to, override);
  });

  defineFrozenGlobal("rm", (path) => {
    const entries = findEntries(drive, path);
    for (const entry of [...entries].reverse()) {
      try {
        drive.delete(entry.path, progress);
      } catch (e) {
        if (!(e instanceof NotFoundError)) throw e;
      }
    }
  });

  defineFrozenGlobal("ls", (path) => drive.list(path));

  defineFrozenGlobal("mkdir", (path) => drive.makeDir(path));
})(this);
