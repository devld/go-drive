<template>
  <div class="login-view">
    <form action="" @submit="onSubmit">
      <span class="form-item username">
        <input
          v-model="username"
          v-focus
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
        <SimpleButton native-type="submit" class="submit" :loading="loading">
          {{ $t('p.login.login') }}
        </SimpleButton>
      </span>
    </form>
  </div>
</template>
<script setup lang="ts">
import { login } from '@/api'
import { useAppStore } from '@/store'
import { User } from '@/types'
import { alert } from '@/utils/ui-utils'
import { ref } from 'vue'

const emit = defineEmits<{ (e: 'success', v?: User): void }>()

const store = useAppStore()

const username = ref('')
const password = ref('')
const loading = ref(false)

const onSubmit = async (e: Event) => {
  e.preventDefault()
  if (loading.value) return
  loading.value = true
  try {
    await login(username.value, password.value)
    const user = await store.getUser()
    emit('success', user)
  } catch (e: any) {
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
