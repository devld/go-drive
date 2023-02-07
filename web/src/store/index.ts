import { getConfig, getUser } from '@/api'
import { Config, User } from '@/types'
import { isAdmin } from '@/utils'
import { createPinia, defineStore } from 'pinia'
import { ConfigOptions, ConfigOptionsMap } from './options'

interface TypedConfig extends Config {
  options: ConfigOptionsMap
}

interface AppState {
  inited: boolean

  user?: User
  config?: Readonly<TypedConfig>

  showLogin: boolean

  progressBar: number | boolean
}

export const useAppStore = defineStore('app', {
  state: (): AppState => ({
    inited: false,
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
      const config = Object.freeze(await getConfig(Object.keys(ConfigOptions)))
      Object.keys(config.options).forEach((key) => {
        const co = ConfigOptions[key]
        if (!co) return
        if (!config.options[key] && co.defaultValue) {
          config.options[key] = co.defaultValue
        }
        config.options[key] = co.process(config.options[key])
      })
      Object.freeze(config.options)
      this.config = config as TypedConfig
      return config as TypedConfig
    },
    async init() {
      await this.getConfig()
      await this.getUser()
      this.inited = true
    },
    destroy() {
      this.user = undefined
      this.inited = false
    },
  },
})

const pinia = createPinia()

export default pinia
