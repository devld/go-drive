<template>
  <div class="user-area">
    <button class="plain-button small login-button" v-if="!isLoggedIn" @click="login">Login</button>
    <span v-else>
      <span
        class="username"
        :title="`Username: ${user.username}\nGroups: ${user.groups.map(g => g.name).join(', ')}`"
      >{{ user.username }}</span>
      <button class="plain-button small logout-button" @click="logout">Logout</button>
    </span>
  </div>
</template>
<script>
import { logout } from '@/api'
import { mapState } from 'vuex'

export default {
  name: 'UserArea',
  computed: {
    ...mapState(['user']),
    isLoggedIn () {
      return !!this.user
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
      } catch (e) {
        this.$alert(e.message)
      } finally {
        this.$loading(false)
      }
    }
  }
}
</script>
<style lang="scss">
.user-area {
  display: inline-block;

  .username {
    font-weight: bold;
  }

  .logout-button {
    margin-left: 24px;
  }
}
</style>
