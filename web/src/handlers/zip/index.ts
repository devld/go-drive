import { deleteTask } from '@/api'
import {
  ARTIFACT_ZIP,
  ArtifactInfo,
  artifactRefUrl,
  prepareArtifact,
  zipSelectionArgs,
} from '@/api/artifact'
import { Task } from '@/types'
import { formatBytes, TASK_CANCELLED, taskDone } from '@/utils'
import { T } from '@go-drive/i18n'
import { EntryHandler } from '../types'

export default {
  name: 'zip',
  display: {
    name: T('handler.zip.name'),
    description: T('handler.zip.desc'),
    icon: 'archive',
  },
  supports: () => true,
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
        ARTIFACT_ZIP,
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
      link.href = artifactRefUrl(path, meta, ARTIFACT_ZIP, info.ref)
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
