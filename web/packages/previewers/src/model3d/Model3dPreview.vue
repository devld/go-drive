<template>
  <div class="model3d-preview">
    <div ref="container" class="model3d-preview__canvas" />
    <div v-if="error" class="model3d-preview__error">{{ error }}</div>
  </div>
</template>
<script setup lang="ts">
import { useI18n } from '@go-drive/i18n'
import { onBeforeUnmount, onMounted, ref, watch } from 'vue'
import * as THREE from 'three'
import { OrbitControls } from 'three/addons/controls/OrbitControls.js'
import { GLTFLoader } from 'three/addons/loaders/GLTFLoader.js'
import { loadArrayBuffer } from '../utils/load'

const { t } = useI18n({ useScope: 'global' })

const props = defineProps<{
  src?: string
  data?: ArrayBuffer
  filename?: string
  resolveResource?: (uri: string) => Promise<ArrayBuffer>
}>()

const container = ref<HTMLDivElement | null>(null)
const error = ref('')
let renderer: THREE.WebGLRenderer | undefined
let scene: THREE.Scene | undefined
let camera: THREE.PerspectiveCamera | undefined
let controls: OrbitControls | undefined
let animationFrame = 0
let resizeObserver: ResizeObserver | undefined
let objectUrls: string[] = []
let model: THREE.Object3D | undefined
let mixer: THREE.AnimationMixer | undefined
let clock: THREE.Clock | undefined
let loadToken = 0
let requestController: AbortController | undefined

const disposeObject = (object?: THREE.Object3D) => {
  object?.traverse((node) => {
    const mesh = node as THREE.Mesh
    mesh.geometry?.dispose?.()
    const material = mesh.material
    const materials = Array.isArray(material) ? material : material ? [material] : []
    materials.forEach((item) => {
      Object.values(item).forEach((value) => {
        if (value instanceof THREE.Texture) value.dispose()
      })
      item.dispose()
    })
  })
}

const clearModel = () => {
  if (model && scene) scene.remove(model)
  disposeObject(model)
  model = undefined
  mixer = undefined
  objectUrls.forEach((url) => URL.revokeObjectURL(url))
  objectUrls = []
}

const mimeType = (uri: string, image = false) => {
  const extension = uri.split('?')[0].split('.').pop()?.toLowerCase()
  if (image) {
    if (extension === 'jpg' || extension === 'jpeg') return 'image/jpeg'
    if (extension === 'png') return 'image/png'
    if (extension === 'webp') return 'image/webp'
    if (extension === 'gif') return 'image/gif'
  }
  return extension === 'glsl' ? 'text/plain' : 'application/octet-stream'
}

const prepareResources = async (data: ArrayBuffer) => {
  const resources = new Map<string, string>()
  if (!props.resolveResource || props.filename?.toLowerCase().endsWith('.glb')) {
    return resources
  }

  let document: any
  try {
    document = JSON.parse(new TextDecoder().decode(data))
  } catch {
    return resources
  }

  const entries: { uri: string; image: boolean }[] = [
    ...(document.buffers ?? []).map((item: { uri?: string }) => ({
      uri: item.uri,
      image: false,
    })),
    ...(document.images ?? []).map((item: { uri?: string }) => ({
      uri: item.uri,
      image: true,
    })),
  ]
  for (const entry of entries) {
    if (!entry.uri || entry.uri.startsWith('data:') || resources.has(entry.uri)) {
      continue
    }
    const bytes = await props.resolveResource(entry.uri)
    const url = URL.createObjectURL(
      new Blob([bytes], { type: mimeType(entry.uri, entry.image) })
    )
    objectUrls.push(url)
    resources.set(entry.uri, url)
  }
  return resources
}

const resize = () => {
  if (!renderer || !camera || !container.value) return
  const width = container.value.clientWidth
  const height = container.value.clientHeight
  if (!width || !height) return
  camera.aspect = width / height
  camera.updateProjectionMatrix()
  renderer.setSize(width, height, false)
}

const animate = () => {
  if (!renderer || !scene || !camera) return
  animationFrame = requestAnimationFrame(animate)
  const delta = clock?.getDelta() ?? 0
  mixer?.update(delta)
  controls?.update()
  renderer.render(scene, camera)
}

const parseModel = async () => {
  const token = ++loadToken
  requestController?.abort()
  const controller = new AbortController()
  requestController = controller
  error.value = ''
  clearModel()
  if ((!props.data && !props.src) || !scene) return

  try {
    const data = props.data || (await loadArrayBuffer(props.src!, controller.signal))
    if (token !== loadToken) return
    const resources = await prepareResources(data)
    if (token !== loadToken) return
    const manager = new THREE.LoadingManager()
    manager.setURLModifier((url) => resources.get(url) ?? url)
    const loader = new GLTFLoader(manager)
    const gltf = await new Promise<any>((resolve, reject) => {
      loader.parse(data, '', resolve, reject)
    })
    if (token !== loadToken) return
    const loadedModel = gltf.scene as THREE.Object3D
    model = loadedModel
    scene.add(loadedModel)
    const box = new THREE.Box3().setFromObject(loadedModel)
    const center = box.getCenter(new THREE.Vector3())
    const size = box.getSize(new THREE.Vector3())
    const radius = Math.max(size.x, size.y, size.z, 1)
    loadedModel.position.sub(center)
    camera?.position.set(radius * 1.8, radius * 1.2, radius * 1.8)
    controls?.target.set(0, 0, 0)
    controls?.update()
    if (gltf.animations?.length) {
      mixer = new THREE.AnimationMixer(loadedModel)
      gltf.animations.forEach((clip: THREE.AnimationClip) => {
        mixer?.clipAction(clip).play()
      })
    }
    resize()
  } catch (e) {
    if (token === loadToken) {
      error.value = e instanceof Error ? e.message : t('preview.model3d.error')
    }
  }
}

onMounted(() => {
  if (!container.value) return
  scene = new THREE.Scene()
  scene.background = new THREE.Color(0x202124)
  camera = new THREE.PerspectiveCamera(45, 1, 0.01, 10000)
  camera.position.set(3, 2, 3)
  renderer = new THREE.WebGLRenderer({ antialias: true, alpha: true })
  renderer.setPixelRatio(Math.min(window.devicePixelRatio, 2))
  renderer.outputColorSpace = THREE.SRGBColorSpace
  container.value.appendChild(renderer.domElement)
  scene.add(new THREE.HemisphereLight(0xffffff, 0x444444, 2))
  const light = new THREE.DirectionalLight(0xffffff, 2)
  light.position.set(3, 5, 4)
  scene.add(light)
  controls = new OrbitControls(camera, renderer.domElement)
  controls.enableDamping = true
  clock = new THREE.Clock()
  resizeObserver = new ResizeObserver(resize)
  resizeObserver.observe(container.value)
  resize()
  animate()
  void parseModel()
})

watch(() => [props.src, props.data], parseModel)

onBeforeUnmount(() => {
  loadToken++
  requestController?.abort()
  requestController = undefined
  cancelAnimationFrame(animationFrame)
  resizeObserver?.disconnect()
  controls?.dispose()
  clearModel()
  renderer?.dispose()
  renderer?.domElement.remove()
  scene = undefined
  renderer = undefined
})
</script>
<style lang="scss">
.model3d-preview {
  position: relative;
  width: 100%;
  height: 100%;
  min-height: 320px;
  overflow: hidden;
  background: #202124;
}

.model3d-preview__canvas,
.model3d-preview__canvas canvas {
  display: block;
  width: 100%;
  height: 100%;
}

.model3d-preview__error {
  position: absolute;
  inset: 0;
  display: grid;
  place-items: center;
  padding: 24px;
  color: #ffb4ab;
  background: rgba(32, 33, 36, 0.92);
  text-align: center;
}
</style>
