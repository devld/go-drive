<template>
  <div class="login-view">
    <SimpleForm
      v-if="loginProvider"
      :key="loginProvider.provider"
      v-model="formData"
      :form="loginForm"
      @submit="onSubmit"
    >
      <template #submit>
        <SimpleButton native-type="submit" :loading="loading">
          {{ $t('p.login.login') }}
        </SimpleButton>
      </template>
    </SimpleForm>
  </div>
</template>
<script setup lang="ts">
import { login } from '@/api'
import { useAppStore } from '@/store'
import { AuthProvider, FormItem, User } from '@/types'
import { alert } from '@/utils/ui-utils'
import { computed, ref } from 'vue'

const emit = defineEmits<{ (e: 'success', v?: User): void }>()

const store = useAppStore()

const loginProvider = computed<AuthProvider | undefined>(() =>
  store.config?.auth.providers.find((provider) => provider.type === 'form')
)
const formData = ref<Record<string, string>>()
const loading = ref(false)

const loginForm = computed<FormItem[]>(() => {
  const fields = (loginProvider.value?.form ?? []).map((item) => ({
    ...item,
    label: undefined,
    ariaLabel: item.label,
    autocomplete:
      item.field === 'username'
        ? 'username'
        : item.field === 'password'
          ? 'current-password'
          : undefined,
    placeholder: item.placeholder || item.label,
    class: 'login-field',
  }))
  return [...fields, { slot: 'submit', class: 'submit' }]
})

const onSubmit = async () => {
  const provider = loginProvider.value
  if (!provider || loading.value) return
  loading.value = true
  try {
    await login(provider.provider, formData.value ?? {})
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
  box-sizing: border-box;
  width: 300px;
  padding: 16px;

  .simple-form {
    display: block;
  }

  .login-field,
  .submit {
    width: 100%;
    padding-right: 0;
  }

  .login-field {
    margin-bottom: 8px;
  }

  .submit {
    text-align: right;
    margin: 16px 0 0;
  }
}
</style>
