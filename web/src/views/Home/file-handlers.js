/* eslint-disable quote-props */
import { filenameExt } from '@/utils'

const codeEditor = (entry, path) => ({ name: 'editor' })

export const fileList = (entry, path) => {
  return { name: 'files' }
}

const fileViewers = {
  'md': codeEditor,
  'js': codeEditor,
  'html': codeEditor,
  'css': codeEditor,
  'java': codeEditor,
  'kt': codeEditor,
  'gradle': codeEditor,
  'xml': codeEditor
}

export function resolveEntryHandlerPage (entry, entryPath) {
  if (entry.type === 'drive' || entry.type === 'dir') {
    return fileList(entry, entryPath)
  }

  const ext = filenameExt(entry.name)

  if (entry.type === 'file') {
    const fileViewer = fileViewers[ext]
    if (fileViewer) {
      return fileViewer(entry, entryPath)
    }
  }
}
