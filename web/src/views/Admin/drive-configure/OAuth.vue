<template>
  <div class="oauth-configure">
    <simple-button @click="doOauth">{{ data.text }}</simple-button>
    <div v-if="data.principal" class="oauth-principal">
      {{ $t('p.admin.oauth_connected', { p: data.principal }) }}
    </div>
  </div>
</template>
<script setup>
import { initDrive } from '@/api/admin'
import { alert, loading } from '@/utils/ui-utils'
import { onBeforeMount, onBeforeUnmount } from 'vue'

const props = defineProps({
  configured: {
    type: Boolean,
    required: true,
  },
  data: {
    type: null,
    required: true,
  },
  drive: {
    type: Object,
    required: true,
  },
})

const emit = defineEmits(['refresh'])

let w

const doOauth = () => {
  w?.close()

  const win = window.open(
    props.data.url,
    props.data.title,
    'width=400,height=600,menubar=0,toolbar=0'
  )
  w = win
}

const authorized = async ({ data }) => {
  w.close()
  if (!data.oauth) return

  if (data.error) {
    console.error('OAuth error: ', data)
    alert(data.error + ': ' + data.data.error_description)
    return
  }

  loading(true)
  try {
    await initDrive(props.drive.name, data.data)
  } catch (e) {
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
