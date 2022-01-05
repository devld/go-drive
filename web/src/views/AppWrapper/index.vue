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

        <router-link
          v-for="m in navMenus"
          :key="m.to"
          class="plain-button small nav-button"
          :to="m.to"
        >
          {{ m.name }}
        </router-link>

        <span v-if="isLoggedIn" class="user-info">
          <span
            class="username"
            :title="
              `
              ${$t('app.username')}: ${user.username}\n` +
              `${$t('app.groups')}: ${user.groups.map((g) => g.name).join(', ')}
            `
            "
            >{{ user.username }}</span
          >
          <button class="plain-button small logout-button" @click="logout">
            {{ $t('app.logout') }}
          </button>
        </span>
      </div>
    </header>

    <router-view />

    <!-- login dialog -->
    <dialog-view
      v-model:show="loginDialogShowing"
      overlay-close
      esc-close
      transition="flip-fade"
      :title="$t('app.login')"
    >
      <login-view @success="afterLogin" />
    </dialog-view>
    <!-- login dialog -->

    <progress-bar :show="progressBarValue" />
  </div>
</template>
<script>
export default { name: 'AppWrapper' }
</script>
<script setup>
import LoginView from '@/views/Login/LoginView.vue'

import { logout as logoutApi } from '@/api'
import { useStore } from 'vuex'
import { alert, loading } from '@/utils/ui-utils'
import { computed } from 'vue'
import { useI18n } from 'vue-i18n'

const store = useStore()
const { t } = useI18n()

const loginDialogShowing = computed({
  get: () => store.state.showLogin,
  set: (v) => store.commit('showLogin', v),
})
const user = computed(() => store.state.user)
const progressBarValue = computed(() => store.state.progressBar)

const isLoggedIn = computed(() => !!user.value)
const isAdmin = computed(() => store.getters.isAdmin)

const navMenus = computed(() => {
  const menus = [{ name: t('app.home'), to: '/' }]
  if (isAdmin.value) {
    menus.push({ name: t('app.admin'), to: '/admin' })
  }
  return menus
})

const login = () => {
  store.commit('showLogin', true)
}

const logout = async () => {
  loading(true)
  try {
    await logoutApi()
    await store.dispatch('getUser')
    location.reload()
  } catch (e) {
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
