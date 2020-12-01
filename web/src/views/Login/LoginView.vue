<template>
  <div class="login-view">
    <form action @submit="onSubmit">
      <span class="form-item username">
        <input
          ref="username"
          class="value"
          type="text"
          required
          :placeholder="$t('p.login.username')"
          v-model="username"
        />
      </span>
      <span class="form-item password">
        <input
          ref="password"
          class="value"
          type="password"
          required
          :placeholder="$t('p.login.password')"
          v-model="password"
        />
      </span>
      <span class="form-item submit">
        <simple-button native-type="submit" class="submit" :loading="loading">
          {{ $t("p.login.login") }}
        </simple-button>
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

  .username {
    margin-bottom: 0 !important;
  }

  .username input {
    border-bottom: none !important;
  }

  .password {
    margin-bottom: 16px;
  }

  .submit {
    text-align: right;
  }
}
</style>
