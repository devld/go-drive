import { zipUrl } from '@/api'
import { AUTH_PARAM, getToken } from '@/api/http'
import { T } from '@/i18n'
import { EntryHandler } from '../types'

export default {
  name: 'zip',
  display: {
    name: T('handler.zip.name'),
    description: T('handler.zip.desc'),
    icon: '#icon-zip-grey',
  },
  supports: () => true,
  multiple: true,
  handler: async ({ entry: entries, parent }) => {
    const form = document.createElement('form')
    form.style.display = 'none'
    form.method = 'post'
    form.action = zipUrl()
    form.target = '_blank'

    if (parent) {
      const prefix = document.createElement('input')
      prefix.name = 'prefix'
      prefix.value = parent.path
      form.appendChild(prefix)
    }

    const token = document.createElement('input')
    token.name = AUTH_PARAM
    token.value = getToken()!
    form.appendChild(token)

    const files = document.createElement('textarea')
    files.name = 'files'
    files.value = entries.map((e) => e.path).join('\n')
    form.appendChild(files)

    document.body.appendChild(form)
    form.submit()
    document.body.removeChild(form)
  },
  order: 2000,
} as EntryHandler
