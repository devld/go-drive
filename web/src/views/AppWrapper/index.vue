<template>
  <div class="app-wrapper">
    <header class="app-header">
      <user-area />
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
  </div>
</template>
<script>
import LoginView from '@/views/Login/LoginView'
import UserArea from './UserArea'

export default {
  name: 'AppWrapper',
  components: { LoginView, UserArea },
  data () {
    return {
    }
  },
  computed: {
    loginDialogShowing: {
      get () { return this.$store.state.showLogin },
      set (v) { this.$store.commit('showLogin', v) }
    }
  },
  methods: {
    afterLogin () {
      this.loginDialogShowing = false
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
  }
}
</style>
