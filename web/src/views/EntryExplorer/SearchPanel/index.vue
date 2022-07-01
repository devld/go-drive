<template>
  <div ref="thisEl" class="search-panel" :class="{ active: showing }">
    <div class="search-panel__search">
      <input
        ref="qEl"
        v-model="queryInput"
        type="text"
        class="search-panel__search-input"
        :placeholder="$t('app.search.placeholder')"
        @input="onInput"
        @keydown.enter="triggerSearch"
        @keydown.stop
        @focus="onInputFocus"
      />
      <span class="search-panel__search-input-key">F</span>
    </div>
    <div v-if="showing" class="search-panel__result" @scroll="onResultScroll">
      <div v-if="result.length === 0" class="search-panel__tip">
        <template v-if="searching">{{ $t('app.search.searching') }}</template>
        <template v-else-if="result.length === 0">
          <p>{{ searchError }}</p>
          <div v-if="searchExamples?.length" class="search-panel__help">
            <span>{{ $t('app.search.search_help') }}</span>
            <em v-for="(e, i) in searchExamples" :key="i">{{ e }}</em>
          </div>
        </template>
      </div>

      <ul class="search-panel__items">
        <SearchItem
          v-for="item in result"
          :key="item.entry.path"
          :item="item"
          @click="itemClicked"
        />
      </ul>
    </div>
  </div>
</template>
<script lang="ts">
import { searchEntries } from '@/api'
import { EntryEventData } from '@/components/entry'
import { useAppStore } from '@/store'
import { Entry, SearchHitItem } from '@/types'
import { debounce } from '@/utils'
import { useHotKey } from '@/utils/hooks/hotkey'
import { computed, onUnmounted, ref } from 'vue'
import { useI18n } from 'vue-i18n'
import SearchItem from './SearchItem.vue'

export default { name: 'SearchPanel' }
</script>
<script setup lang="ts">
const { t } = useI18n()
const store = useAppStore()

const props = defineProps({
  path: {
    type: String,
    required: true,
  },
})

const emit = defineEmits<{ (e: 'navigate', v: Entry): void }>()

const thisEl = ref<HTMLElement | null>(null)
const qEl = ref<HTMLInputElement | null>(null)
const queryInput = ref('')
const q = computed(() => queryInput.value.trim())
const next = ref(0)
const searching = ref(false)
const result = ref<SearchHitItem[]>([])
const searchError = ref('')
const showing = ref(false)

const searchExamples = computed(() => store.config?.search?.examples)

const triggerSearch = () => {
  result.value = []
  if (!q.value) {
    return
  }
  next.value = 0
  doSearch()
}

const loadNextPage = debounce(() => {
  if (next.value === -1) return
  doSearch()
}, 100)

const doSearch = async () => {
  searching.value = true
  searchError.value = ''
  let res
  try {
    res = await searchEntries(props.path, q.value, next.value)
  } catch (e: any) {
    searchError.value = e.message
    return
  } finally {
    searching.value = false
  }

  result.value.push(...res.items)
  searchError.value = res.items.length === 0 ? t('app.search.no_result') : ''
  next.value = res.next
}

const reset = () => {
  queryInput.value = ''
  result.value = []
  next.value = 0
  searchError.value = ''
}

const itemClicked = (e: EntryEventData) => {
  emit('navigate', e.entry!)
  setActive(false)
}

const onInput = () => {
  if (!queryInput.value) reset()
}

const onInputFocus = () => {
  setActive(true)
}

const onResultScroll = (e: UIEvent) => {
  const target = e.target as HTMLElement
  if (target.scrollHeight - target.scrollTop - target.clientHeight < 100) {
    loadNextPage()
  }
}

let eventAttached = false
const setActive = (active: boolean) => {
  showing.value = !!active
  if (active) qEl.value?.focus()
  else qEl.value?.blur()
  if (active && !eventAttached) {
    eventAttached = true
    document.addEventListener('mousedown', onDocumentTouched)
  }
  if (!active && eventAttached) {
    eventAttached = false
    document.removeEventListener('mousedown', onDocumentTouched)
  }
}

useHotKey((e) => {
  e.preventDefault()
  setActive(true)
}, 'f')

useHotKey(
  () => {
    setActive(false)
  },
  'Escape',
  { el: () => qEl.value! }
)

const onDocumentTouched = (e: MouseEvent) => {
  let target = e.target as HTMLElement | null
  do {
    if (target === thisEl.value) break
  } while ((target = target!.parentElement))
  if (target) return
  if (showing.value) e.stopPropagation()
  setActive(false)
}

onUnmounted(() => {
  setActive(false)
})

defineExpose({ setActive })
</script>
<style lang="scss">
.search-panel {
  border-radius: 16px;
  transition: 0.4s;
  background-color: var(--primary-bg-color);
  color: var(--primary-text-color);
  overflow: hidden;

  &.active {
    box-shadow: 0 8px 32px rgba(0, 0, 0, 0.2);

    .search-panel__search {
      padding-right: 16px;
    }

    .search-panel__search-input-key {
      display: none;
    }
  }
}

.search-panel__result {
  max-height: 70vh;
  overflow: hidden auto;
}

.search-panel__search {
  position: relative;
  padding: 0 36px 0 16px;
}

.search-panel__search-input-key {
  position: absolute;
  display: block;
  top: 50%;
  right: 16px;
  transform: translateY(-50%);
  width: 16px;
  height: 16px;
  line-height: 16px;
  text-align: center;
  font-size: 14px;
  color: var(--secondary-text-color);
  user-select: none;
  border-radius: 2px;
  border: solid 1px var(--secondary-text-color);
}

.search-panel__search-input {
  box-sizing: border-box;
  width: 100%;
  border: none;
  background-color: transparent;
  outline: none;
  font-size: 16px;
  color: var(--primary-text-color);
  padding: 16px 0;

  &::placeholder {
    color: var(--secondary-text-color);
  }
}

.search-panel__help {
  display: flex;
  flex-wrap: wrap;
  justify-content: center;
}

.search-panel__tip {
  display: flex;
  flex-direction: column;
  justify-content: center;
  flex-wrap: wrap;
  align-items: center;
  font-size: 14px;
  padding: 16px 0;
  color: var(--secondary-text-color);
  text-align: center;

  p {
    margin: 0;
    line-height: 48px;
  }

  em {
    padding: 0 2px;
    font-style: normal;
    border: solid 1px var(--secondary-text-color);
    border-radius: 4px;
    margin: 0 6px 6px;
  }
}

.search-panel__items {
  margin: 0;
  padding: 0;
}
</style>
