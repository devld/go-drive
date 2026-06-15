<template>
  <div class="audio-view-page">
    <HandlerTitleBar :title="entry.name" @close="emit('close')" />

    <div class="audio-player">
      <div class="audio-player__cover">
        <img
          v-if="currentCover && !coverErrored"
          :src="currentCover"
          :alt="currentTrack?.name"
          class="audio-player__cover-img"
          @error="coverErrored = true"
        />
        <svg v-else class="audio-player__cover-icon" viewBox="0 0 24 24">
          <path
            fill="currentColor"
            d="M12 3a1 1 0 0 1 1.3-.95l5 1.67A1 1 0 0 1 19 4.67V7a1 1 0 0 1-1.32.95L14 6.72v8.78a3.5 3.5 0 1 1-2-3.16V3zm-1.5 11a1.5 1.5 0 1 0 0 3 1.5 1.5 0 0 0 0-3z"
          />
        </svg>
      </div>

      <div class="audio-player__progress">
        <span class="audio-player__time">{{ formatTime(currentTime) }}</span>
        <div
          ref="progressEl"
          class="audio-player__bar"
          @pointerdown="onProgressDown"
        >
          <div class="audio-player__bar-bg">
            <div
              class="audio-player__bar-loaded"
              :style="{ width: loadedRatio * 100 + '%' }"
            />
            <div
              class="audio-player__bar-played"
              :style="{ width: playedRatio * 100 + '%' }"
            />
            <div
              class="audio-player__bar-thumb"
              :style="{ left: playedRatio * 100 + '%' }"
            />
          </div>
        </div>
        <span class="audio-player__time">{{ formatTime(duration) }}</span>
      </div>

      <div class="audio-player__controls">
        <div class="audio-player__controls-side audio-player__controls-left">
          <button
            class="audio-player__btn plain-button"
            :class="{ 'is-active': shuffle }"
            :title="$t('handler.audio.shuffle')"
            @click="shuffle = !shuffle"
          >
            <svg viewBox="0 0 24 24" class="audio-player__icon">
              <path
                fill="none"
                stroke="currentColor"
                stroke-width="2"
                stroke-linecap="round"
                stroke-linejoin="round"
                d="M3 7h3.5a4 4 0 0 1 3.3 1.8l4.4 6.4a4 4 0 0 0 3.3 1.8H21M3 17h3.5a4 4 0 0 0 3.3-1.8l.7-1M14.2 8.3l.7-1A4 4 0 0 1 18.5 7H21"
              />
              <path
                fill="currentColor"
                d="M18.5 4.5 22 7l-3.5 2.5zM18.5 14.5 22 17l-3.5 2.5z"
              />
            </svg>
          </button>
        </div>

        <div class="audio-player__controls-center">
          <button
            class="audio-player__btn plain-button"
            :title="$t('handler.audio.prev')"
            @click="prev"
          >
            <svg viewBox="0 0 24 24" class="audio-player__icon">
              <path
                fill="currentColor"
                d="M6 6h2v12H6V6zm3.5 6l8.5 6V6l-8.5 6z"
              />
            </svg>
          </button>

          <button
            class="audio-player__btn audio-player__btn--play plain-button"
            :title="
              playing ? $t('handler.audio.pause') : $t('handler.audio.play')
            "
            @click="togglePlay"
          >
            <svg v-if="playing" viewBox="0 0 24 24" class="audio-player__icon">
              <path fill="currentColor" d="M6 5h4v14H6V5zm8 0h4v14h-4V5z" />
            </svg>
            <svg v-else viewBox="0 0 24 24" class="audio-player__icon">
              <path fill="currentColor" d="M8 5v14l11-7L8 5z" />
            </svg>
          </button>

          <button
            class="audio-player__btn plain-button"
            :title="$t('handler.audio.next')"
            @click="next"
          >
            <svg viewBox="0 0 24 24" class="audio-player__icon">
              <path fill="currentColor" d="M16 6h2v12h-2V6zM6 6l8.5 6L6 18V6z" />
            </svg>
          </button>
        </div>

        <div class="audio-player__volume audio-player__controls-side">
          <button
            class="audio-player__btn plain-button"
            :title="muted ? $t('handler.audio.unmute') : $t('handler.audio.mute')"
            @click="toggleMute"
          >
            <svg
              v-if="muted || volume === 0"
              viewBox="0 0 24 24"
              class="audio-player__icon"
            >
              <path
                fill="currentColor"
                d="M3 9v6h4l5 5V4L7 9H3zm13.6 3l2.7-2.7-1.4-1.4L15.2 11l-2.7-2.7-1.4 1.4 2.7 2.7-2.7 2.7 1.4 1.4 2.7-2.7 2.7 2.7 1.4-1.4-2.7-2.7z"
              />
            </svg>
            <svg v-else viewBox="0 0 24 24" class="audio-player__icon">
              <path
                fill="currentColor"
                d="M3 9v6h4l5 5V4L7 9H3zm13.5 3a4.5 4.5 0 00-2.5-4v8a4.5 4.5 0 002.5-4zM14 3.2v2.1a7 7 0 010 13.4v2.1a9 9 0 000-17.6z"
              />
            </svg>
          </button>
          <div
            ref="volumeEl"
            class="audio-player__volume-bar"
            @pointerdown="onVolumeDown"
          >
            <div class="audio-player__volume-bg">
              <div
                class="audio-player__volume-value"
                :style="{ width: (muted ? 0 : volume) * 100 + '%' }"
              />
              <div
                class="audio-player__volume-thumb"
                :style="{ left: (muted ? 0 : volume) * 100 + '%' }"
              />
            </div>
          </div>
        </div>
      </div>
    </div>

    <ul class="audio-player__list">
      <li
        v-for="(track, index) in tracks"
        :key="track.path"
        class="audio-player__list-item"
        :class="{ 'is-current': index === currentIndex }"
        @click="switchTo(index, true)"
      >
        <span class="audio-player__list-index">
          <svg
            v-if="index === currentIndex && playing"
            viewBox="0 0 24 24"
            class="audio-player__list-playing"
          >
            <path fill="currentColor" d="M8 5v14l11-7L8 5z" />
          </svg>
          <template v-else>{{ index + 1 }}</template>
        </span>
        <span class="audio-player__list-name" :title="track.name">{{
          track.name
        }}</span>
      </li>
    </ul>

    <audio
      ref="audioEl"
      @timeupdate="onTimeUpdate"
      @loadedmetadata="onLoadedMetadata"
      @progress="onProgress"
      @ended="onEnded"
      @play="playing = true"
      @pause="playing = false"
    />
  </div>
</template>
<script setup lang="ts">
import { fileThumbnail, fileUrl } from '@/api'
import HandlerTitleBar from '@/components/HandlerTitleBar.vue'
import { useAppStore } from '@/store'
import { Entry } from '@/types'
import { filenameBase, filenameExt } from '@/utils'
import { computed, onMounted, onUnmounted, ref } from 'vue'
import { EntryHandlerContext } from '../types'

interface Track {
  name: string
  url: string
  path: string
  entry: Entry
}

const props = defineProps({
  entry: {
    type: Object as PropType<Entry>,
    required: true,
  },
  entries: {
    type: Array as PropType<Entry[]>,
    required: true,
  },
  ctx: {
    type: Object as PropType<EntryHandlerContext>,
    required: true,
  },
})

const emit = defineEmits<{
  (e: 'close'): void
  (e: 'entry-change', v: string): void
}>()

const store = useAppStore()

const audioEl = ref<HTMLAudioElement>()
const progressEl = ref<HTMLElement>()
const volumeEl = ref<HTMLElement>()

const tracks = ref<Track[]>([])
const currentIndex = ref(0)
const playing = ref(false)
const currentTime = ref(0)
const duration = ref(0)
const loaded = ref(0)
const volume = ref(1)
const muted = ref(false)
const shuffle = ref(false)
const coverErrored = ref(false)

const audioExts = computed<readonly string[]>(
  () => props.ctx.options['web.audioFileExts'] ?? []
)

const currentTrack = computed(() => tracks.value[currentIndex.value])

const playedRatio = computed(() =>
  duration.value > 0 ? currentTime.value / duration.value : 0
)
const loadedRatio = computed(() =>
  duration.value > 0 ? loaded.value / duration.value : 0
)

const supportThumbnail = (entry: Entry) => {
  const t = entry.meta.thumbnail
  if (typeof t === 'string') return true
  if (t === true) return true
  const ext = filenameExt(entry.name)
  return !!store.config?.thumbnail.extensions?.[ext]
}

const currentCover = computed(() => {
  const track = currentTrack.value
  if (!track) return undefined
  const t = track.entry.meta.thumbnail
  if (typeof t === 'string') return t
  if (supportThumbnail(track.entry)) {
    return fileThumbnail(track.entry.path, track.entry.meta)
  }
  return undefined
})

const formatTime = (seconds: number) => {
  if (!Number.isFinite(seconds) || seconds < 0) seconds = 0
  const total = Math.floor(seconds)
  const m = Math.floor(total / 60)
  const s = total % 60
  return `${m}:${s.toString().padStart(2, '0')}`
}

const loadTrack = (index: number, autoPlay: boolean) => {
  const track = tracks.value[index]
  if (!track || !audioEl.value) return
  currentIndex.value = index
  coverErrored.value = false
  currentTime.value = 0
  duration.value = 0
  loaded.value = 0
  audioEl.value.src = track.url
  audioEl.value.load()
  emit('entry-change', track.path)
  if (autoPlay) {
    audioEl.value.play().catch(() => undefined)
  }
}

const switchTo = (index: number, autoPlay = true) => {
  if (index === currentIndex.value && audioEl.value?.src) {
    togglePlay()
    return
  }
  loadTrack(index, autoPlay)
}

const togglePlay = () => {
  const el = audioEl.value
  if (!el) return
  if (el.paused) el.play().catch(() => undefined)
  else el.pause()
}

const nextIndex = () => {
  const len = tracks.value.length
  if (len <= 1) return currentIndex.value
  if (shuffle.value) {
    let i = currentIndex.value
    while (i === currentIndex.value) i = Math.floor(Math.random() * len)
    return i
  }
  return (currentIndex.value + 1) % len
}

const prevIndex = () => {
  const len = tracks.value.length
  if (len <= 1) return currentIndex.value
  if (shuffle.value) {
    let i = currentIndex.value
    while (i === currentIndex.value) i = Math.floor(Math.random() * len)
    return i
  }
  return (currentIndex.value - 1 + len) % len
}

const next = () => loadTrack(nextIndex(), true)
const prev = () => loadTrack(prevIndex(), true)

const onTimeUpdate = () => {
  currentTime.value = audioEl.value?.currentTime ?? 0
}
const onLoadedMetadata = () => {
  duration.value = audioEl.value?.duration ?? 0
}
const onProgress = () => {
  const el = audioEl.value
  if (!el || el.buffered.length === 0) return
  loaded.value = el.buffered.end(el.buffered.length - 1)
}
const onEnded = () => loadTrack(nextIndex(), true)

const applyVolume = () => {
  if (!audioEl.value) return
  audioEl.value.volume = volume.value
  audioEl.value.muted = muted.value
}

const toggleMute = () => {
  muted.value = !muted.value
  applyVolume()
}

const seekByRatio = (ratio: number) => {
  if (!audioEl.value || duration.value <= 0) return
  audioEl.value.currentTime = ratio * duration.value
  currentTime.value = audioEl.value.currentTime
}

const setVolumeByRatio = (ratio: number) => {
  volume.value = Math.min(1, Math.max(0, ratio))
  muted.value = volume.value === 0
  applyVolume()
}

const ratioFromEvent = (el: HTMLElement, e: PointerEvent) => {
  const rect = el.getBoundingClientRect()
  return Math.min(1, Math.max(0, (e.clientX - rect.left) / rect.width))
}

const createDrag = (
  elRef: typeof progressEl,
  onChange: (ratio: number) => void
) => {
  const onMove = (e: PointerEvent) => {
    if (elRef.value) onChange(ratioFromEvent(elRef.value, e))
  }
  const onUp = () => {
    window.removeEventListener('pointermove', onMove)
    window.removeEventListener('pointerup', onUp)
  }
  return (e: PointerEvent) => {
    if (!elRef.value) return
    onChange(ratioFromEvent(elRef.value, e))
    window.addEventListener('pointermove', onMove)
    window.addEventListener('pointerup', onUp)
  }
}

const onProgressDown = createDrag(progressEl, seekByRatio)
const onVolumeDown = createDrag(volumeEl, setVolumeByRatio)

const init = () => {
  const audioFiles = props.entries.filter(
    (f) => f.type === 'file' && audioExts.value.includes(filenameExt(f.name))
  )
  tracks.value = audioFiles.map((f) => ({
    name: filenameBase(f.name),
    url: fileUrl(f.path, f.meta, { useProxy: 'referrer' }),
    path: f.path,
    entry: f,
  }))

  const startIndex = Math.max(
    0,
    tracks.value.findIndex((t) => t.path === props.entry.path)
  )
  applyVolume()
  loadTrack(startIndex, true)
}

onMounted(init)

onUnmounted(() => {
  audioEl.value?.pause()
})
</script>
<style lang="scss">
.audio-view-page {
  position: relative;
  width: 500px;
  max-width: 96vw;
  overflow: hidden;
  display: flex;
  flex-direction: column;
  padding-top: 48px;
  background-color: var(--secondary-bg-color);
  color: var(--primary-text-color);

  .handler-title-bar {
    position: absolute;
    top: 0;
    left: 0;
    right: 0;
  }
}

.audio-player {
  padding: 16px 20px 12px;
  display: flex;
  flex-direction: column;
  align-items: center;

  &__cover {
    width: 160px;
    height: 160px;
    border-radius: 12px;
    overflow: hidden;
    background-color: var(--hover-bg-color);
    display: flex;
    align-items: center;
    justify-content: center;
    box-shadow: 0 6px 18px rgba(0, 0, 0, 0.12);
  }

  &__cover-img {
    width: 100%;
    height: 100%;
    object-fit: cover;
  }

  &__cover-icon {
    width: 76px;
    height: 76px;
    color: var(--secondary-text-color);
    opacity: 0.35;
  }

  &__progress {
    width: 100%;
    margin-top: 18px;
    display: flex;
    align-items: center;
    gap: 8px;
  }

  &__time {
    flex-shrink: 0;
    width: 40px;
    font-size: 12px;
    color: var(--secondary-text-color);
    font-variant-numeric: tabular-nums;

    &:first-child {
      text-align: right;
    }
  }

  &__bar {
    flex: 1;
    padding: 8px 0;
    cursor: pointer;
    touch-action: none;
  }

  &__bar-bg {
    position: relative;
    height: 4px;
    border-radius: 2px;
    background-color: var(--border-color);
  }

  &__bar-loaded {
    position: absolute;
    top: 0;
    left: 0;
    height: 100%;
    border-radius: 2px;
    background-color: var(--hover-bg-color);
  }

  &__bar-played {
    position: absolute;
    top: 0;
    left: 0;
    height: 100%;
    border-radius: 2px;
    background-color: var(--link-color);
  }

  &__bar-thumb {
    position: absolute;
    top: 50%;
    width: 12px;
    height: 12px;
    border-radius: 50%;
    background-color: var(--link-color);
    transform: translate(-50%, -50%);
    box-shadow: 0 1px 3px rgba(0, 0, 0, 0.25);
    opacity: 0;
    transition: opacity 0.15s;
  }

  &__bar:hover &__bar-thumb {
    opacity: 1;
  }

  &__controls {
    width: 100%;
    margin-top: 10px;
    display: flex;
    align-items: center;
    gap: 6px;
  }

  &__controls-side {
    flex: 1;
    display: flex;
    align-items: center;
  }

  &__controls-left {
    justify-content: flex-start;
  }

  &__controls-center {
    flex-shrink: 0;
    display: flex;
    align-items: center;
    gap: 6px;
  }

  &__btn {
    display: flex;
    align-items: center;
    justify-content: center;
    width: 38px;
    height: 38px;
    border-radius: 50%;
    color: var(--primary-text-color);
    cursor: pointer;
    transition: background-color 0.15s, color 0.15s;

    &:hover {
      background-color: var(--hover-bg-color);
    }

    &.is-active {
      color: var(--link-color);
    }
  }

  &__icon {
    width: 22px;
    height: 22px;
  }

  &__btn--play {
    width: 52px;
    height: 52px;
    background-color: var(--link-color);
    color: #fff;

    .audio-player__icon {
      width: 28px;
      height: 28px;
    }

    &:hover {
      background-color: var(--link-color);
      filter: brightness(1.08);
    }
  }

  &__volume {
    justify-content: flex-end;
    gap: 2px;
  }

  &__volume-bar {
    width: 72px;
    padding: 8px 0;
    cursor: pointer;
    touch-action: none;
  }

  &__volume-bg {
    position: relative;
    height: 4px;
    border-radius: 2px;
    background-color: var(--border-color);
  }

  &__volume-value {
    position: absolute;
    top: 0;
    left: 0;
    height: 100%;
    border-radius: 2px;
    background-color: var(--link-color);
  }

  &__volume-thumb {
    position: absolute;
    top: 50%;
    width: 10px;
    height: 10px;
    border-radius: 50%;
    background-color: var(--link-color);
    transform: translate(-50%, -50%);
  }

  &__list {
    list-style: none;
    margin: 0;
    padding: 0;
    max-height: 220px;
    overflow-y: auto;
    border-top: solid 1px var(--border-color);
    background-color: var(--primary-bg-color);
  }

  &__list-item {
    display: flex;
    align-items: center;
    gap: 10px;
    padding: 10px 20px;
    cursor: pointer;
    border-bottom: solid 1px var(--border-color);
    transition: background-color 0.15s;

    &:last-child {
      border-bottom: none;
    }

    &:hover {
      background-color: var(--hover-bg-color);
    }

    &.is-current {
      background-color: var(--select-bg-color);

      .audio-player__list-name,
      .audio-player__list-index {
        color: var(--link-color);
        font-weight: 500;
      }
    }
  }

  &__list-index {
    flex-shrink: 0;
    width: 20px;
    text-align: center;
    font-size: 13px;
    color: var(--secondary-text-color);
    font-variant-numeric: tabular-nums;
  }

  &__list-playing {
    width: 14px;
    height: 14px;
    color: var(--link-color);
  }

  &__list-name {
    flex: 1;
    overflow: hidden;
    text-overflow: ellipsis;
    white-space: nowrap;
    font-size: 14px;
  }
}
</style>
