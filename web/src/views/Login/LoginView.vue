<template>
  <div class="login-view">
    <form action @submit="onSubmit">
      <span class="form-item username">
        <input
          v-model="username"
          class="value"
          type="text"
          required
          :placeholder="$t('p.login.username')"
        />
      </span>
      <span class="form-item password">
        <input
          v-model="password"
          class="value"
          type="password"
          required
          :placeholder="$t('p.login.password')"
        />
      </span>
      <span class="form-item submit">
        <simple-button native-type="submit" class="submit" :loading="loading">
          {{ $t('p.login.login') }}
        </simple-button>
      </span>
    </form>
  </div>
</template>
<script setup>
import { login } from '@/api'
import { alert } from '@/utils/ui-utils'
import { ref } from 'vue'
import { useStore } from 'vuex'

const emit = defineEmits(['success'])

const store = useStore()

const username = ref('')
const password = ref('')
const loading = ref(false)

const onSubmit = async (e) => {
  e.preventDefault()
  if (loading.value) return
  loading.value = true
  try {
    await login(username.value, password.value)
    const user = await store.dispatch('getUser')
    emit('success', user)
  } catch (e) {
    alert(e.message)
  } finally {
    loading.value = false
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
