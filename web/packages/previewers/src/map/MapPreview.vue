<template>
  <div ref="container" class="map-preview" />
</template>
<script setup lang="ts">
import { useI18n } from '@go-drive/i18n'
import { onBeforeUnmount, onMounted, ref, watch } from 'vue'
import * as L from 'leaflet'
import { gpx, kml } from '@tmcw/togeojson'
import 'leaflet/dist/leaflet.css'

const { t } = useI18n({ useScope: 'global' })

const props = withDefaults(
  defineProps<{
    data?: string
    format?: 'geojson' | 'gpx' | 'kml'
    tileUrl?: string
    tileAttribution?: string
  }>(),
  {
    data: '',
    format: 'geojson',
    tileUrl: 'https://{s}.tile.openstreetmap.org/{z}/{x}/{y}.png',
    tileAttribution: '&copy; OpenStreetMap contributors',
  }
)

const container = ref<HTMLDivElement | null>(null)
let map: L.Map | undefined
let dataLayer: L.GeoJSON | undefined

const parseData = () => {
  if (!props.data) return
  if (props.format === 'geojson') return JSON.parse(props.data)
  const document = new DOMParser().parseFromString(props.data, 'text/xml')
  if (document.querySelector('parsererror')) {
    throw new Error(t('preview.map.invalidXml'))
  }
  return props.format === 'gpx' ? gpx(document) : kml(document)
}

const renderData = () => {
  if (!map) return
  dataLayer?.remove()
  dataLayer = undefined
  try {
    const data = parseData()
    if (!data) return
    dataLayer = L.geoJSON(data as GeoJSON.GeoJsonObject).addTo(map)
    const bounds = dataLayer.getBounds()
    if (bounds.isValid()) map.fitBounds(bounds.pad(0.1))
  } catch (error) {
    const message = error instanceof Error ? error.message : t('preview.map.error')
    const notice = new L.Control({ position: 'topright' })
    notice.onAdd = () => {
      const element = L.DomUtil.create('div', 'map-preview__error')
      element.textContent = message
      return element
    }
    notice.addTo(map)
  }
}

onMounted(() => {
  if (!container.value) return
  map = L.map(container.value).setView([0, 0], 2)
  L.tileLayer(props.tileUrl, {
    attribution: props.tileAttribution,
    maxZoom: 19,
  }).addTo(map)
  renderData()
})

watch(
  () => [props.data, props.format, props.tileUrl, props.tileAttribution],
  renderData
)

onBeforeUnmount(() => {
  map?.remove()
  map = undefined
  dataLayer = undefined
})
</script>
<style lang="scss">
.map-preview {
  width: 100%;
  height: 100%;
  min-height: 320px;
  background: #e5e3df;
}

.map-preview__error {
  max-width: 280px;
  padding: 8px 10px;
  border: 1px solid rgba(0, 0, 0, 0.2);
  border-radius: 4px;
  color: #842029;
  background: #f8d7da;
  box-shadow: 0 1px 5px rgba(0, 0, 0, 0.3);
  font-size: 12px;
}
</style>
