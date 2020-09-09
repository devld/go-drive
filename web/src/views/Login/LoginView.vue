<template>
  <div class="login-view">
    <form action @submit="onSubmit">
      <span class="form-item username">
        <input ref="username" type="text" required placeholder="Username" v-model="username" />
      </span>
      <span class="form-item password">
        <input ref="password" type="password" required placeholder="Password" v-model="password" />
      </span>
      <span class="form-item submit">
        <simple-button native-type="submit" class="submit" :loading="loading">Login</simple-button>
      </span>
    </form>
  </div>
</template>
<script>
import { login } from '@/api'

export default {
  name: 'LoginView',
  data () {
    return {
      username: '',
      password: '',

      loading: false
    }
  },
  methods: {
    async onSubmit (e) {
      e.preventDefault()
      if (this.loading) return
      this.loading = true
      try {
        await login(this.username, this.password)
        const user = await this.$store.dispatch('getUser')
        this.$emit('success', user)
      } catch (e) {
        this.$alert(e.message)
      } finally {
        this.loading = false
      }
    }
  }
}
</script>
<style lang="scss">
.login-view {
  width: 300px;
  display: flex;
  flex-direction: column;
  align-items: center;
  padding: 16px 0;

  .form-item {
    display: block;
  }

  input {
    border: solid 1px rgba(0, 0, 0, 0.1);
    font-size: 16px;
    outline: none;
    padding: 6px;
  }

  .username {
    margin-bottom: 0 !important;
  }

  .username input {
    border-bottom: none;
  }

  .password {
    margin-bottom: 16px;
  }

  .submit {
    text-align: right;
  }
}
</style>
