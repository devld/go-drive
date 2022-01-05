<template>
  <div class="file-exists-confirm-inner">
    <p>{{ $t('p.home.file_exists', { n: opts.filename }) }}</p>
    <p>
      <input v-model="all" type="checkbox" />
      {{ $t('p.home.apply_all') }}
    </p>
  </div>
</template>
<script setup>
import { ref } from 'vue'

defineProps({
  opts: {
    type: Object,
    required: true,
  },
})

const all = ref(false)

const beforeConfirm = () => ({ all: all.value })
const beforeCancel = () => ({ all: all.value })

defineExpose({
  beforeConfirm,
  beforeCancel,
})
</script>
