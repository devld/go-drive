<template>
  <div class="office-preview-view">
    <handler-title-bar :title="filename" @close="emit('close')">
      <template #actions>
        <select v-model="serviceIndex">
          <option v-for="(s, i) in services" :key="s.name" :value="i">
            {{ s.name }}
          </option>
        </select>
      </template>
    </handler-title-bar>

    <iframe
      ref="iframe"
      :key="previewURL"
      class="office-preview-iframe"
      :src="previewURL"
      frameborder="0"
    ></iframe>
  </div>
</template>
<script setup>
import { fileUrl } from '@/api'
import HandlerTitleBar from '@/components/HandlerTitleBar.vue'
import { buildURL, filename as filenameFn } from '@/utils'
import { computed, ref, watch } from 'vue'

const props = defineProps({
  entry: {
    type: Object,
    required: true,
  },
  entries: { type: Array },
})

const emit = defineEmits(['close'])

const path = computed(() => props.entry.path)
const filename = computed(() => filenameFn(path.value))
const fileURL = computed(() => fileUrl(path.value, props.entry.meta))

const services = [
  {
    name: 'Microsoft',
    url: (u) =>
      buildURL('https://view.officeapps.live.com/op/embed.aspx', { src: u }),
  },
  {
    name: 'Google',
    url: (u) =>
      buildURL('https://docs.google.com/gview?embedded=true', { url: u }),
  },
]

const STORAGE_KEY = 'office-preview-service'
const serviceIndex = ref(+localStorage.getItem(STORAGE_KEY) || 0)
if (!services[serviceIndex.value]) serviceIndex.value = 0
watch(
  () => serviceIndex.value,
  (v) => {
    localStorage.setItem(STORAGE_KEY, v)
  }
)

const service = computed(() => services[serviceIndex.value])

const previewURL = computed(() => service.value.url(fileURL.value))
</script>
<style lang="scss">
.office-preview-view {
  position: relative;
  overflow: hidden;
  width: 100vw;
  height: 100vh;
  padding-top: 48px;
  background-color: var(--secondary-bg-color);
  box-shadow: 0 0 10px rgba(0, 0, 0, 0.1);
  box-sizing: border-box;

  .handler-title-bar {
    position: absolute;
    top: 0;
    left: 0;
    right: 0;
  }

  .office-preview-iframe {
    width: 100%;
    height: 100%;
  }
}
</style>
