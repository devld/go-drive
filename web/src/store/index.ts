import { getConfig, getUser } from '@/api'
import { Config, User } from '@/types'
import { isAdmin } from '@/utils'
import { createPinia, defineStore } from 'pinia'

const configOptions = ['web.officePreviewEnabled']

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
      const config = Object.freeze(await getConfig(configOptions))
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
