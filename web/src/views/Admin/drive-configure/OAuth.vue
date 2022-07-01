<template>
  <div class="oauth-configure">
    <SimpleButton @click="doOauth">{{ data.text }}</SimpleButton>
    <div v-if="data.principal" class="oauth-principal">
      {{ $t('p.admin.oauth_connected', { p: data.principal }) }}
    </div>
  </div>
</template>
<script setup lang="ts">
import { initDrive } from '@/api/admin'
import { DriveInitOAuth } from '@/types'
import { alert, loading } from '@/utils/ui-utils'
import { onBeforeMount, onBeforeUnmount } from 'vue'

const props = defineProps({
  configured: {
    type: Boolean,
    required: true,
  },
  data: {
    type: Object as PropType<DriveInitOAuth>,
    required: true,
  },
  drive: {
    type: Object,
    required: true,
  },
})

const emit = defineEmits<{ (e: 'refresh'): void }>()

let w: Window | null = null

const doOauth = () => {
  w?.close()

  const win = window.open(
    props.data.url,
    undefined,
    'width=400,height=600,menubar=0,toolbar=0'
  )
  w = win
}

const authorized = async ({ data }: any) => {
  if (!data.oauth) return
  w?.close()

  if (data.error) {
    console.error('OAuth error: ', data)
    alert(data.error + ': ' + data.data.error_description)
    return
  }

  loading(true)
  try {
    await initDrive(props.drive.name, data.data)
  } catch (e: any) {
    alert(e.message)
    return
  } finally {
    loading()
  }
  emit('refresh')
}

onBeforeMount(() => {
  window.addEventListener('message', authorized)
})

onBeforeUnmount(() => {
  window.removeEventListener('message', authorized)
  w?.close()
})
</script>
<style lang="scss">
.oauth-configure {
  .oauth-principal {
    margin-top: 0.5em;
    font-size: 14px;
    color: var(--secondary-text-color);
  }
}
</style>
