<template>
  <div class="app-wrapper">
    <header class="app-header">
      <div class="user-area">
        <button
          v-if="!isLoggedIn"
          class="plain-button small login-button"
          @click="login"
        >
          {{ $t('app.login') }}
        </button>

        <RouterLink
          v-for="m in navMenus"
          :key="m.to"
          class="plain-button small nav-button"
          :to="m.to"
        >
          {{ m.name }}
        </RouterLink>

        <span v-if="isLoggedIn" class="user-info">
          <span
            class="username"
            :title="
              `
              ${$t('app.username')}: ${user!.username}\n` +
              `${$t('app.groups')}: ${user!.groups.map((g) => g.name).join(', ')}
            `
            "
            >{{ user!.username }}</span
          >
          <button class="plain-button small logout-button" @click="logout">
            {{ $t('app.logout') }}
          </button>
        </span>
      </div>
    </header>

    <RouterView />

    <!-- login dialog -->
    <DialogView
      v-model:show="loginDialogShowing"
      overlay-close
      esc-close
      transition="scale-opacity"
      :title="$t('app.login')"
    >
      <LoginView @success="afterLogin" />
    </DialogView>
    <!-- login dialog -->

    <ProgressBar :show="progressBarValue" />
  </div>
</template>
<script lang="ts">
export default { name: 'AppWrapper' }
</script>
<script setup lang="ts">
import LoginView from '@/views/Login/LoginView.vue'

import { logout as logoutApi } from '@/api'
import { alert, loading } from '@/utils/ui-utils'
import { computed } from 'vue'
import { useI18n } from 'vue-i18n'
import { useAppStore } from '@/store'

const store = useAppStore()
const { t } = useI18n()

const loginDialogShowing = computed({
  get: () => store.showLogin,
  set: (v) => store.toggleLogin(v),
})
const user = computed(() => store.user)
const progressBarValue = computed(() => store.progressBar)

const isLoggedIn = computed(() => !!user.value)
const isAdmin = computed(() => store.isAdmin)

const navMenus = computed(() => {
  const menus = [{ name: t('app.home'), to: '/' }]
  if (isAdmin.value) {
    menus.push({ name: t('app.admin'), to: '/admin' })
  }
  return menus
})

const login = () => {
  store.toggleLogin(true)
}

const logout = async () => {
  loading(true)
  try {
    await logoutApi()
    await store.getUser()
    location.reload()
  } catch (e: any) {
    alert(e.message)
  } finally {
    loading(false)
  }
}

const afterLogin = () => {
  loginDialogShowing.value = false
  location.reload()
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
        content: '|';
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
