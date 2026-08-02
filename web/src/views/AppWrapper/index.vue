<template>
  <div class="app-wrapper">
    <a class="skip-link" href="#main-content" @click.prevent="skipToContent">
      {{ $t('app.skip_to_content') }}
    </a>
    <header class="app-header" data-ui="app-header">
      <div class="user-area">
        <button
          v-if="!isLoggedIn"
          class="plain-button small"
          data-ui="button"
          data-variant="plain"
          @click="login"
        >
          {{ $t('app.login') }}
        </button>

        <RouterLink
          v-for="m in navMenus"
          :key="m.to"
          class="plain-button small nav-button"
          data-ui="button"
          data-variant="plain"
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
          <button
            class="plain-button small"
            data-ui="button"
            data-variant="plain"
            @click="logout"
          >
            {{ $t('app.logout') }}
          </button>
        </span>
      </div>
    </header>

    <main
      id="main-content"
      ref="mainContentEl"
      class="main-content"
      tabindex="-1"
    >
      <h1 class="visually-hidden">{{ pageHeading }}</h1>
      <RouterView />
    </main>

    <!-- login dialog -->
    <DialogView
      v-model:show="loginDialogShowing"
      overlay-close
      esc-close
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
import { computed, ref } from 'vue'
import { useI18n } from 'vue-i18n'
import { useAppStore } from '@/store'
import { useRouter } from 'vue-router'
import { EXPLORER_PATH_BASE } from '@/config'

const router = useRouter()
const store = useAppStore()
const { t } = useI18n()
const mainContentEl = ref<HTMLElement | null>(null)

const loginDialogShowing = computed({
  get: () => store.showLogin,
  set: (v) => store.toggleLogin(v),
})
const user = computed(() => store.user)
const progressBarValue = computed(() => store.progressBar)

const isLoggedIn = computed(() => !!user.value)
const isAdmin = computed(() => store.isAdmin)
const pageHeading = computed(() => {
  const route = router.currentRoute.value
  if (route.meta.title) return route.meta.title.toString()
  if (typeof route.params.path === 'string' && route.params.path) {
    return route.params.path
  }
  return window.___config___.appName || 'go-drive'
})

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

const skipToContent = () => {
  mainContentEl.value?.focus()
}

const toIndexPage = () => {
  store.destroy()
  if (router.currentRoute.value.path.startsWith(EXPLORER_PATH_BASE)) {
    const href = router.resolve(`${EXPLORER_PATH_BASE}/`).href
    location.href = href
  }
  location.reload()
}

const logout = async () => {
  loading(true)
  try {
    await logoutApi()
    toIndexPage()
  } catch (e: any) {
    alert(e.message)
  } finally {
    loading(false)
  }
}

const afterLogin = () => {
  loginDialogShowing.value = false
  toIndexPage()
}
</script>
<style lang="scss">
.app-header {
  padding: 16px 16px 32px;
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

.skip-link {
  position: fixed;
  top: 8px;
  left: 8px;
  z-index: 10000;
  padding: 8px 12px;
  border-radius: var(--radius-control);
  background: var(--color-bg-surface);
  color: var(--color-text);
  transform: translateY(-150%);

  &:focus {
    transform: translateY(0);
  }
}

.main-content {
  display: flow-root;
  width: fit-content;
  max-width: 100%;
  margin: 0 auto 72px;

  &:focus {
    outline: none;
  }

  &:focus-visible {
    border-radius: var(--radius-glass);
    outline: 2px solid transparent;
    outline-offset: -2px;
    box-shadow: 0 0 0 2px var(--color-focus-ring);
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
