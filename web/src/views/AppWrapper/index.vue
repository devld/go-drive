<template>
  <div class="app-wrapper">
    <header class="app-header">
      <div class="user-area">
        <button
          class="plain-button small login-button"
          v-if="!isLoggedIn"
          @click="login"
        >
          Login
        </button>

        <router-link
          v-for="m in navMenus"
          :key="m.to"
          class="plain-button small nav-button"
          :to="m.to"
          >{{ m.name }}</router-link
        >

        <span class="user-info" v-if="isLoggedIn">
          <span
            class="username"
            :title="`Username: ${user.username}\nGroups: ${user.groups
              .map((g) => g.name)
              .join(', ')}`"
            >{{ user.username }}</span
          >
          <button class="plain-button small logout-button" @click="logout">
            Logout
          </button>
        </span>
      </div>
    </header>

    <router-view />

    <!-- login dialog -->
    <dialog-view
      v-model="loginDialogShowing"
      overlay-close
      esc-close
      transition="flip-fade"
      title="Login"
    >
      <login-view @success="afterLogin" />
    </dialog-view>
    <!-- login dialog -->

    <progress-bar :value="progressBar" />
  </div>
</template>
<script>
import LoginView from '@/views/Login/LoginView'

import { logout } from '@/api'
import { mapGetters, mapState } from 'vuex'

export default {
  name: 'AppWrapper',
  components: { LoginView },
  data () {
    return {
    }
  },
  computed: {
    loginDialogShowing: {
      get () { return this.$store.state.showLogin },
      set (v) { this.$store.commit('showLogin', v) }
    },
    ...mapState(['user', 'progressBar']),
    ...mapGetters(['isAdmin']),
    isLoggedIn () { return !!this.user },
    navMenus () {
      const menus = [{ name: 'Home', to: '/' }]
      if (this.isAdmin) {
        menus.push({ name: 'Admin', to: '/admin' })
      }
      return menus
    }
  },
  methods: {
    login () {
      this.$store.commit('showLogin', true)
    },
    async logout () {
      this.$loading(true)
      try {
        await logout()
        await this.$store.dispatch('getUser')
        location.reload()
      } catch (e) {
        this.$alert(e.message)
      } finally {
        this.$loading(false)
      }
    },
    afterLogin () {
      this.loginDialogShowing = false
      location.reload()
    }
  }
}
</script>
<style lang="scss">
.app-header {
  padding: 16px;
  overflow: hidden;

  .user-area {
    float: right;

    .username {
      font-weight: bold;
    }

    .user-info {
      & > *:not(:last-child) {
        margin-right: 16px;
      }

      &::before {
        content: "|";
        margin: 0 1em;
      }
    }

    .nav-button:not(:first-child) {
      margin-left: 16px;
    }
  }
}

.app-wrapper {
  & > .progress-bar {
    position: fixed;
    top: 0;
    left: 0;
    right: 0;
  }
}
</style>
