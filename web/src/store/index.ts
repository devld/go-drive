import { getConfig, getUser } from '@/api'
import { Config, User } from '@/types'
import { isAdmin, mapOf } from '@/utils'
import { createPinia, defineStore } from 'pinia'

const stringList = (v?: string) => {
  v = v?.trim()
  if (!v) return
  const l = Object.freeze((v || '').split(',').map((e) => e.trim()))
  return l.length === 0 ? undefined : l
}

const configOptions = mapOf(
  [
    { key: 'web.officePreviewEnabled', process: (v?: string) => !!v },
    {
      key: 'web.textFileExts',
      process: stringList,
    },
    {
      key: 'web.imageFileExts',
      process: stringList,
    },
    {
      key: 'web.mediaFileExts',
      process: stringList,
    },
  ],
  (e) => e.key
)

interface AppState {
  user?: User
  config?: Readonly<Config>

  showLogin: boolean

  progressBar: number | boolean
}

export const useAppStore = defineStore('app', {
  state: (): AppState => ({
    user: undefined,
    config: undefined,
    showLogin: false,
    progressBar: false,
  }),
  getters: {
    isAdmin: (s) => isAdmin(s.user),
  },
  actions: {
    setUser(user?: User) {
      this.user = user
    },
    setConfig(config: Config) {
      this.config = config
    },
    toggleLogin(show: boolean) {
      this.showLogin = show
    },
    setProgressBar(val?: number | boolean) {
      if (typeof val === 'boolean' || typeof val === 'number') {
        this.progressBar = val
      } else {
        this.progressBar = false
      }
    },

    async getUser() {
      const user = await getUser()
      this.setUser(user)
      return user
    },
    async getConfig() {
      const config = Object.freeze(await getConfig(Object.keys(configOptions)))
      Object.keys(config.options).forEach((key) => {
        if (configOptions[key]) {
          config.options[key] = configOptions[key].process(config.options[key])
        }
      })
      Object.freeze(config.options)
      this.config = config
      return config
    },
    async init() {
      await this.getConfig()
      await this.getUser()
    },
  },
})

const pinia = createPinia()

export default pinia
