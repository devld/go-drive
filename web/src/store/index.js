import { getUser } from '@/api'
import { isAdmin } from '@/utils'
import Vue from 'vue'
import Vuex from 'vuex'

Vue.use(Vuex)

export default new Vuex.Store({
  state: {
    user: null,

    showLogin: false
  },
  getters: {
    isAdmin (state) {
      return isAdmin(state.user)
    }
  },
  mutations: {
    setUser (state, user) {
      state.user = user || null
    },
    showLogin (state, show) {
      state.showLogin = show
    }
  },
  actions: {
    async init (context) {
      await context.dispatch('getUser')
    },
    async getUser (context) {
      const user = await getUser()
      context.commit('setUser', user)
      return user
    }
  },
  modules: {
  }
})
