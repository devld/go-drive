<template>
  <div
    ref="containerEl"
    class="video-view-page"
    :class="{ 'is-idle': controlsHidden }"
    @mousemove="showControls"
    @mouseleave="scheduleHide"
  >
    <HandlerTitleBar
      :title="entry.name"
      class="video-view-page__title"
      @close="emit('close')"
    />

    <div class="video-player" @click="togglePlay" @dblclick="toggleFullscreen">
      <video
        ref="videoEl"
        class="video-player__video"
        :src="fileUrl(entry.path, entry.meta, { useProxy: 'referrer' })"
        @timeupdate="onTimeUpdate"
        @loadedmetadata="onLoadedMetadata"
        @progress="onProgress"
        @ended="playing = false"
        @play="playing = true"
        @pause="playing = false"
      />
    </div>

    <div class="video-controls" @click.stop>
      <div
        ref="progressEl"
        class="video-controls__bar"
        @pointerdown="onProgressDown"
      >
        <div class="video-controls__bar-bg">
          <div
            class="video-controls__bar-loaded"
            :style="{ width: loadedRatio * 100 + '%' }"
          />
          <div
            class="video-controls__bar-played"
            :style="{ width: playedRatio * 100 + '%' }"
          />
          <div
            class="video-controls__bar-thumb"
            :style="{ left: playedRatio * 100 + '%' }"
          />
        </div>
      </div>

      <div class="video-controls__row">
        <button
          class="video-controls__btn plain-button"
          :title="
            playing ? $t('handler.video.pause') : $t('handler.video.play')
          "
          @click="togglePlay"
        >
          <svg v-if="playing" viewBox="0 0 24 24" class="video-controls__icon">
            <path fill="currentColor" d="M6 5h4v14H6V5zm8 0h4v14h-4V5z" />
          </svg>
          <svg v-else viewBox="0 0 24 24" class="video-controls__icon">
            <path fill="currentColor" d="M8 5v14l11-7L8 5z" />
          </svg>
        </button>

        <span class="video-controls__time">
          {{ formatTime(currentTime) }} / {{ formatTime(duration) }}
        </span>

        <span class="video-controls__spacer" />

        <div class="video-controls__volume">
          <button
            class="video-controls__btn plain-button"
            :title="
              muted
                ? $t('handler.video.unmute')
                : $t('handler.video.mute')
            "
            @click="toggleMute"
          >
            <svg
              v-if="muted || volume === 0"
              viewBox="0 0 24 24"
              class="video-controls__icon"
            >
              <path
                fill="currentColor"
                d="M3 9v6h4l5 5V4L7 9H3zm13.6 3l2.7-2.7-1.4-1.4L15.2 11l-2.7-2.7-1.4 1.4 2.7 2.7-2.7 2.7 1.4 1.4 2.7-2.7 2.7 2.7 1.4-1.4-2.7-2.7z"
              />
            </svg>
            <svg v-else viewBox="0 0 24 24" class="video-controls__icon">
              <path
                fill="currentColor"
                d="M3 9v6h4l5 5V4L7 9H3zm13.5 3a4.5 4.5 0 00-2.5-4v8a4.5 4.5 0 002.5-4zM14 3.2v2.1a7 7 0 010 13.4v2.1a9 9 0 000-17.6z"
              />
            </svg>
          </button>
          <div
            ref="volumeEl"
            class="video-controls__volume-bar"
            @pointerdown="onVolumeDown"
          >
            <div class="video-controls__volume-bg">
              <div
                class="video-controls__volume-value"
                :style="{ width: (muted ? 0 : volume) * 100 + '%' }"
              />
              <div
                class="video-controls__volume-thumb"
                :style="{ left: (muted ? 0 : volume) * 100 + '%' }"
              />
            </div>
          </div>
        </div>

        <button
          v-if="supportsPip"
          class="video-controls__btn plain-button"
          :title="$t('handler.video.pip')"
          @click="togglePip"
        >
          <svg viewBox="0 0 24 24" class="video-controls__icon">
            <path
              fill="currentColor"
              d="M19 11h-8v6h8v-6zm4 8V4.98C23 3.88 22.1 3 21 3H3c-1.1 0-2 .88-2 1.98V19c0 1.1.9 2 2 2h18c1.1 0 2-.9 2-2zm-2 .02H3V4.97h18v14.05z"
            />
          </svg>
        </button>

        <button
          class="video-controls__btn plain-button"
          :title="
            isFullscreen
              ? $t('handler.video.exit_fullscreen')
              : $t('handler.video.fullscreen')
          "
          @click="toggleFullscreen"
        >
          <svg
            v-if="isFullscreen"
            viewBox="0 0 24 24"
            class="video-controls__icon"
          >
            <path
              fill="currentColor"
              d="M5 16h3v3h2v-5H5v2zm3-8H5v2h5V5H8v3zm6 11h2v-3h3v-2h-5v5zm2-11V5h-2v5h5V8h-3z"
            />
          </svg>
          <svg v-else viewBox="0 0 24 24" class="video-controls__icon">
            <path
              fill="currentColor"
              d="M7 14H5v5h5v-2H7v-3zm-2-4h2V7h3V5H5v5zm12 7h-3v2h5v-5h-2v3zM14 5v2h3v3h2V5h-5z"
            />
          </svg>
        </button>
      </div>
    </div>
  </div>
</template>
<script setup lang="ts">
import { fileUrl } from '@/api'
import HandlerTitleBar from '@/components/HandlerTitleBar.vue'
import { Entry } from '@/types'
import { createDrag } from '@/utils/dom'
import { computed, onMounted, onUnmounted, ref } from 'vue'

defineProps({
  entry: {
    type: Object as PropType<Entry>,
    required: true,
  },
  entries: { type: Array as PropType<Entry[]> },
})

const emit = defineEmits<{ (e: 'close'): void }>()

const containerEl = ref<HTMLElement>()
const videoEl = ref<HTMLVideoElement>()
const progressEl = ref<HTMLElement>()
const volumeEl = ref<HTMLElement>()

const playing = ref(false)
const currentTime = ref(0)
const duration = ref(0)
const loaded = ref(0)
const volume = ref(1)
const muted = ref(false)
const isFullscreen = ref(false)
const controlsHidden = ref(false)
const supportsPip = ref(false)

let hideTimer = 0

const playedRatio = computed(() =>
  duration.value > 0 ? currentTime.value / duration.value : 0
)
const loadedRatio = computed(() =>
  duration.value > 0 ? loaded.value / duration.value : 0
)

const formatTime = (seconds: number) => {
  if (!Number.isFinite(seconds) || seconds < 0) seconds = 0
  const total = Math.floor(seconds)
  const h = Math.floor(total / 3600)
  const m = Math.floor((total % 3600) / 60)
  const s = total % 60
  if (h > 0) {
    return `${h}:${m.toString().padStart(2, '0')}:${s.toString().padStart(2, '0')}`
  }
  return `${m}:${s.toString().padStart(2, '0')}`
}

const togglePlay = () => {
  const el = videoEl.value
  if (!el) return
  if (el.paused) el.play().catch(() => undefined)
  else el.pause()
}

const onTimeUpdate = () => {
  currentTime.value = videoEl.value?.currentTime ?? 0
}
const onLoadedMetadata = () => {
  duration.value = videoEl.value?.duration ?? 0
}
const onProgress = () => {
  const el = videoEl.value
  if (!el || el.buffered.length === 0) return
  loaded.value = el.buffered.end(el.buffered.length - 1)
}

const applyVolume = () => {
  if (!videoEl.value) return
  videoEl.value.volume = volume.value
  videoEl.value.muted = muted.value
}

const toggleMute = () => {
  muted.value = !muted.value
  applyVolume()
}

const seekByRatio = (ratio: number) => {
  if (!videoEl.value || duration.value <= 0) return
  videoEl.value.currentTime = ratio * duration.value
  currentTime.value = videoEl.value.currentTime
}

const setVolumeByRatio = (ratio: number) => {
  volume.value = Math.min(1, Math.max(0, ratio))
  muted.value = volume.value === 0
  applyVolume()
}

const onProgressDown = createDrag(progressEl, seekByRatio)
const onVolumeDown = createDrag(volumeEl, setVolumeByRatio)

const toggleFullscreen = () => {
  if (!containerEl.value) return
  if (document.fullscreenElement) {
    document.exitFullscreen()
  } else {
    containerEl.value.requestFullscreen().catch(() => undefined)
  }
}

const onFullscreenChange = () => {
  isFullscreen.value = !!document.fullscreenElement
}

const togglePip = async () => {
  const el = videoEl.value
  if (!el) return
  try {
    if (document.pictureInPictureElement) {
      await document.exitPictureInPicture()
    } else {
      await el.requestPictureInPicture()
    }
  } catch {
    // PiP not available
  }
}

const showControls = () => {
  controlsHidden.value = false
  scheduleHide()
}

const scheduleHide = () => {
  clearTimeout(hideTimer)
  hideTimer = window.setTimeout(() => {
    if (playing.value) controlsHidden.value = true
  }, 3000)
}

const seekStep = computed(() => {
  const d = duration.value
  if (d <= 0) return 5
  return Math.min(30, Math.max(1, d * 0.02))
})

const onKeyDown = (e: KeyboardEvent) => {
  if (!videoEl.value) return
  switch (e.key) {
    case ' ':
    case 'k':
      e.preventDefault()
      togglePlay()
      break
    case 'ArrowLeft':
      e.preventDefault()
      videoEl.value.currentTime = Math.max(
        0,
        videoEl.value.currentTime - seekStep.value
      )
      break
    case 'ArrowRight':
      e.preventDefault()
      videoEl.value.currentTime = Math.min(
        duration.value,
        videoEl.value.currentTime + seekStep.value
      )
      break
    case 'ArrowUp':
      e.preventDefault()
      setVolumeByRatio(Math.min(1, volume.value + 0.1))
      break
    case 'ArrowDown':
      e.preventDefault()
      setVolumeByRatio(Math.max(0, volume.value - 0.1))
      break
    case 'f':
      e.preventDefault()
      toggleFullscreen()
      break
    case 'm':
      e.preventDefault()
      toggleMute()
      break
  }
}

onMounted(() => {
  applyVolume()
  supportsPip.value =
    'pictureInPictureEnabled' in document && document.pictureInPictureEnabled
  document.addEventListener('fullscreenchange', onFullscreenChange)
  window.addEventListener('keydown', onKeyDown)
})

onUnmounted(() => {
  videoEl.value?.pause()
  clearTimeout(hideTimer)
  document.removeEventListener('fullscreenchange', onFullscreenChange)
  window.removeEventListener('keydown', onKeyDown)
})
</script>
<style lang="scss">
.video-view-page {
  position: relative;
  width: 100%;
  display: flex;
  flex-direction: column;
  align-items: center;
  background-color: #000;
  color: #fff;

  &__title {
    position: absolute;
    top: 0;
    left: 0;
    right: 0;
    z-index: 3;
    background: linear-gradient(to bottom, rgba(0, 0, 0, 0.6), transparent);
    transition: opacity 0.3s;

    .handler-title-bar-text {
      color: #fff;
    }
    .handler-title-bar-close {
      color: #fff;
    }
  }

  &.is-idle {
    cursor: none;

    .video-view-page__title,
    .video-controls {
      opacity: 0;
      pointer-events: none;
    }
  }
}

.video-player {
  flex: 1;
  display: flex;
  align-items: center;
  justify-content: center;
  width: 100%;
  min-height: 0;
  cursor: pointer;

  &__video {
    max-width: 100%;
    max-height: 80vh;
    outline: none;
  }

  .video-view-page:fullscreen & {
    &__video {
      max-height: 100vh;
    }
  }
}

.video-controls {
  position: absolute;
  bottom: 0;
  left: 0;
  right: 0;
  z-index: 3;
  background: linear-gradient(to top, rgba(0, 0, 0, 0.7), transparent);
  padding: 24px 12px 10px;
  transition: opacity 0.3s;

  &__bar {
    width: 100%;
    padding: 6px 0;
    cursor: pointer;
    touch-action: none;
  }

  &__bar-bg {
    position: relative;
    height: 4px;
    border-radius: 2px;
    background-color: rgba(255, 255, 255, 0.25);
    transition: height 0.12s;
  }

  &__bar:hover &__bar-bg {
    height: 6px;
  }

  &__bar-loaded {
    position: absolute;
    top: 0;
    left: 0;
    height: 100%;
    border-radius: 2px;
    background-color: rgba(255, 255, 255, 0.35);
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
    width: 14px;
    height: 14px;
    border-radius: 50%;
    background-color: var(--link-color);
    transform: translate(-50%, -50%);
    box-shadow: 0 1px 4px rgba(0, 0, 0, 0.4);
    opacity: 0;
    transition: opacity 0.15s;
  }

  &__bar:hover &__bar-thumb {
    opacity: 1;
  }

  &__row {
    display: flex;
    align-items: center;
    gap: 4px;
    margin-top: 2px;
  }

  &__btn {
    display: flex;
    align-items: center;
    justify-content: center;
    width: 36px;
    height: 36px;
    border-radius: 50%;
    color: #fff;
    cursor: pointer;
    transition: background-color 0.15s;

    &:hover {
      background-color: rgba(255, 255, 255, 0.15);
    }
  }

  &__icon {
    width: 22px;
    height: 22px;
  }

  &__time {
    font-size: 13px;
    color: rgba(255, 255, 255, 0.85);
    font-variant-numeric: tabular-nums;
    margin: 0 4px;
    white-space: nowrap;
  }

  &__spacer {
    flex: 1;
  }

  &__volume {
    display: flex;
    align-items: center;
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
    background-color: rgba(255, 255, 255, 0.25);
  }

  &__volume-value {
    position: absolute;
    top: 0;
    left: 0;
    height: 100%;
    border-radius: 2px;
    background-color: #fff;
  }

  &__volume-thumb {
    position: absolute;
    top: 50%;
    width: 10px;
    height: 10px;
    border-radius: 50%;
    background-color: #fff;
    transform: translate(-50%, -50%);
  }
}
</style>
