import { getTask } from '@/api'

import focus from './directives/focus'
import markdown from './directives/markdown'
import lazySrc from './directives/lazy-src'
import type { Task, User } from '../types'
import type { Directive, Plugin } from 'vue'
import type { LocationQuery } from 'vue-router'
import { wait } from '@go-drive/utils'

export * from '@go-drive/utils'

export const IS_DEBUG = process.env.NODE_ENV === 'development'

export function setTitle(title?: I18nText) {
  const pageTitle = title?.toString().trim() ?? ''
  const appName = window.___config___.appName?.trim() ?? ''
  document.title =
    pageTitle && appName
      ? `${pageTitle} - ${appName}`
      : pageTitle || appName || 'go-drive'
}

export function getRouteQuery(q: LocationQuery, key: string) {
  const value = q[key]
  return Array.isArray(value) ? value[0] : value
}

export function isAdmin(user?: User) {
  return !!(
    user &&
    user.groups &&
    user.groups.findIndex((group) => group.name === 'admin') !== -1
  )
}

export const TASK_CANCELLED = { message: 'task canceled' }

export async function taskDone<T = any>(
  task_: PromiseValue<Task<T>>,
  cb?: Fn1<Task<T>, PromiseValue<false | undefined | void>>,
  interval = 1000
) {
  let task = await task_
  while (task.status === 'pending' || task.status === 'running') {
    if (cb && (await cb(task)) === false) throw TASK_CANCELLED
    try {
      task = await getTask(task.id)
    } catch (e: any) {
      if (e.status === 404) throw TASK_CANCELLED
      throw e
    }
    await wait(interval)
  }
  if (task.status === 'done') {
    cb && (await cb(task))
    return task.result
  } else if (task.status === 'error') {
    throw task.error
  } else if (task.status === 'canceled') {
    throw TASK_CANCELLED
  } else {
    console.warn('unknown task status', task)
    throw new Error('unknown error')
  }
}

const directives: O<Directive> = {
  markdown,
  focus,
  lazySrc,
}

export default {
  install(app) {
    Object.keys(directives).forEach((key) => {
      app.directive(key, directives[key])
    })
  },
} as Plugin
