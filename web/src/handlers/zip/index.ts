import {
  ArtifactInfo,
  artifactURL,
  deleteTask,
  prepareArtifact,
} from '@/api'
import { Task } from '@/types'
import { formatBytes, TASK_CANCELLED, taskDone } from '@/utils'
import { T } from '@go-drive/i18n'
import { EntryHandler } from '../types'

const ZIP_HANDLER = 'zip'

function zipSelectionArgs(dir: string, paths: string[]) {
  const prefix = dir ? `${dir}/` : ''
  const relative = paths
    .map((path) =>
      prefix && (path === dir || path.startsWith(prefix))
        ? path.slice(prefix.length)
        : path
    )
    .filter((path) => path !== '')
  return JSON.stringify(relative)
}

export default {
  name: 'zip',
  display: {
    name: T('handler.zip.name'),
    description: T('handler.zip.desc'),
    icon: 'archive',
  },
  supports: [],
  multiple: true,
  handler: async ({ entry: entries, parent }, { alert, loading }) => {
    let task: Task<ArtifactInfo> | undefined
    let canceled = false
    const onCancel = () => {
      canceled = true
      return task && deleteTask(task.id)
    }
    loading({ text: T('handler.zip.packaging'), onCancel })
    try {
      const path = parent?.path ?? ''
      const meta = parent?.meta ?? entries[0].meta
      const prepared = await prepareArtifact(
        path,
        meta,
        ZIP_HANDLER,
        zipSelectionArgs(
          path,
          entries.map((entry) => entry.path)
        )
      )
      const info =
        'info' in prepared
          ? prepared.info
          : await taskDone(prepared.task, (running) => {
              if (canceled) return false
              task = running
              loading({
                text: T('handler.zip.packaging_progress', {
                  p: running.progress
                    ? `${formatBytes(running.progress.loaded)}/${formatBytes(
                        running.progress.total
                      )}`
                    : '',
                }),
                onCancel,
              })
            })
      if (!info || !info.ref) {
        throw new Error(String(T('handler.zip.pack_expired')))
      }
      const link = document.createElement('a')
      link.href = artifactURL(path, meta, ZIP_HANDLER, { ref: info.ref })
      link.target = '_blank'
      link.rel = 'noreferrer noopener nofollow'
      document.body.appendChild(link)
      link.click()
      document.body.removeChild(link)
    } catch (e: any) {
      if (e !== TASK_CANCELLED) alert(e.message)
    } finally {
      loading()
    }
  },
  order: 2000,
} as EntryHandler
