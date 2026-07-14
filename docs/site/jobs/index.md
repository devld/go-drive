---
title: Automated Jobs
description: Automate go-drive copy, move, delete, and JavaScript actions with cron schedules or file-event triggers and execution history.
lang: en
translation_key: jobs
---

# Automated Jobs

Create background jobs under **Admin → Jobs**. A job has an enabled state, one or more triggers, and one action.

## Triggers

### Standard cron

The current implementation uses standard five-field cron syntax. It has no seconds field and does not support Quartz `?`:

```text
┌──────── minute 0-59
│ ┌────── hour 0-23
│ │ ┌──── day of month 1-31
│ │ │ ┌── month 1-12
│ │ │ │ ┌ day of week 0-6
│ │ │ │ │
0 2 * * *
```

This example runs every day at 02:00 local time. The scheduler uses the local timezone of the go-drive process; explicitly verify the timezone setting in containers.

### File events

A file-event trigger contains:

- A path pattern, such as `incoming/**/*.jpg`.
- One or more event types: `updated` and `deleted`.

An update event may also represent creation or overwrite. Event data is passed to script actions:

```js
log(JSON.stringify($event))
// {
//   type: "entry",
//   data: { path: "...", eventType: "updated", includeDescendants: "false" }
// }
```

Files changed by a job may match its event trigger again. Design path rules to prevent self-triggering loops—for example, separate input and output directories and exclude the output path.

## Actions

Four action types are currently supported.

### Copy/move

- Source paths accept one entry per line and may use wildcards.
- The destination must be an existing directory.
- You can select move and overwrite behavior.
- Cross-Drive operations, directories, and storage without native copy support transfer data through the server.

### Delete

Enter one path pattern per line. Matches are deleted in reverse order so children are removed first. Before using a broad `**`, verify the pattern against a low-privilege test path or with the script `ls` function.

### Flow

A flow combines multiple operations (copy, delete, script, etc.) into a single job. Steps execute in order. Each step can optionally enable **Ignore errors** so that a failure does not stop subsequent steps.

### JavaScript

Common functions:

```ts
cp(from, to, override)
mv(from, to, override)
rm(path)
ls(path)
mkdir(path)
log(message)
```

The shared runtime also provides `http`, `newContext`, `newContextWithTimeout`, `sleep`, `pathUtils`, `encUtils`, error constructors, and the Drive API. Complete type definitions are in the source repository:

- [`docs/scripts/global.d.ts`](https://github.com/devld/go-drive/blob/master/docs/scripts/global.d.ts)
- [`docs/scripts/env/jobs.d.ts`](https://github.com/devld/go-drive/blob/master/docs/scripts/env/jobs.d.ts)
- [`docs/scripts/libs`](https://github.com/devld/go-drive/tree/master/docs/scripts/libs)

Example:

```js
log('trigger: ' + JSON.stringify($event))

// Copy all jpg files under incoming to archive, allowing overwrite
cp('incoming/**/*.jpg', 'archive', true)

// Call an external webhook
var ctx = newContextWithTimeout(newContext(), ms(10000))
try {
  var resp = http(ctx, 'POST', 'https://example.com/hook', {
    'content-type': 'application/json'
  }, JSON.stringify($event))
  try {
    log('webhook: ' + resp.Status)
  } finally {
    resp.Dispose()
  }
} finally {
  ctx.Cancel()
}
```

Job scripts are trusted administrator code with access to the root Drive and the network. Do not run scripts from unknown sources or write secrets into logs visible to other administrators.

## Execution, logs, and cancellation

- The run button in the list triggers a job immediately.
- While editing a script, you can run it interactively and inspect its logs.
- Execution history records start/completion times, status, logs, and errors.
- A running job can be aborted; immediate cancellation depends on whether the underlying remote request responds to context cancellation.
- Execution history can be cleared.

During debugging, run the job manually before enabling cron or event triggers. Use a dedicated test directory for move, delete, and recursive patterns.
