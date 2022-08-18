<template>
  <div class="music-view-page">
    <HandlerTitleBar :title="entry.name" @close="emit('close')" />
    <div ref="containerEl" class="music-view-player"></div>
  </div>
</template>
<script setup lang="ts">
import { fileUrl } from '@/api'
import HandlerTitleBar from '@/components/HandlerTitleBar.vue'
import { Entry } from '@/types'
import APlayer from 'aplayer'
import 'aplayer/dist/Aplayer.min.css'
import { onMounted, onUnmounted, ref } from 'vue'
import { EntryHandlerContext } from '../types'
import { DEFAULT_AUDIO_FILE_EXTS } from '@/config'
import { filenameBase, filenameExt } from '@/utils'

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

const containerEl = ref<HTMLElement>()

const emit = defineEmits<{
  (e: 'close'): void
  (e: 'entry-change', v: string): void
}>()

let player: any

let audio: { name: string; url: string; cover?: string }[] = []

const init = () => {
  const audioFiles = props.entries.filter(
    (f) =>
      f.type === 'file' &&
      (
        props.ctx.options['web.audioFileExts'] || DEFAULT_AUDIO_FILE_EXTS
      ).includes(filenameExt(f.name))
  )
  audio = audioFiles.map((f) => ({
    name: filenameBase(f.name),
    url: fileUrl(f.path, f.meta, { useProxy: 'referrer' }),
  }))

  player = new APlayer({
    container: containerEl.value!,
    audio,
  })

  player.on('listswitch', ({ index }: { index: number }) => {
    const file = audioFiles[index]
    emit('entry-change', file.path)
  })

  player.list.switch(audioFiles.findIndex((e) => e.path === props.entry.path))
  player.play()
}

onMounted(init)

onUnmounted(() => {
  player.destroy()
})
</script>
<style lang="scss">
.music-view-page {
  position: relative;
  width: 500px;
  overflow: hidden;
  display: flex;
  flex-direction: column;
  justify-content: center;
  align-items: center;
  padding-top: 48px;
  background-color: var(--secondary-bg-color);

  .handler-title-bar {
    position: absolute;
    top: 0;
    left: 0;
    right: 0;
  }

  .music-view-player {
    width: 100%;
    margin: 0;
    box-shadow: none;
    background-color: var(--primary-bg-color);

    .aplayer-list .aplayer-list-light {
      background-color: var(--secondary-bg-color);
    }

    .aplayer-list li {
      border-color: var(--border-color);

      &:hover {
        background-color: var(--hover-bg-color);
      }
    }

    .aplayer-info {
      border-color: var(--border-color);
    }

    .aplayer-author,
    .aplayer-list-author {
      display: none;
    }
  }
}

@media screen and (max-width: 500px) {
  .music-view-page {
    width: 96vw;
  }
}
</style>
