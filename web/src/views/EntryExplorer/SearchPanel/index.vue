<template>
  <div
    ref="thisEl"
    class="search-panel glass-surface"
    data-ui="search-panel"
    data-surface="glass"
    :class="{ active: showing }"
    role="search"
    @focusout="onFocusOut"
  >
    <div class="search-panel__search">
      <input
        ref="qEl"
        v-model="queryInput"
        type="text"
        class="search-panel__search-input"
        :aria-label="$t('app.search.placeholder')"
        :placeholder="$t('app.search.placeholder')"
        @input="onInput"
        @keydown.enter="triggerSearch"
        @keydown.stop
        @focus="onInputFocus"
      />
      <span
        class="visually-hidden"
        role="status"
        aria-live="polite"
        aria-atomic="true"
      >{{ searchStatus }}</span>
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
const searchStatus = computed(() => {
  if (searching.value) return t('app.search.searching')
  if (searchError.value) return searchError.value
  if (result.value.length > 0) {
    return t('app.search.results', { n: result.value.length })
  }
  return ''
})

let searchGeneration = 0
const pendingRequests = new Set<string>()

const triggerSearch = () => {
  const generation = ++searchGeneration
  result.value = []
  searchError.value = ''
  next.value = 0
  if (!q.value) {
    searching.value = false
    return
  }
  void doSearch(generation)
}

const loadNextPage = debounce(() => {
  if (next.value === -1 || searching.value) return
  void doSearch(searchGeneration)
}, 100)

const doSearch = async (generation: number) => {
  const query = q.value
  const cursor = next.value
  const requestKey = `${generation}:${cursor}`
  if (pendingRequests.has(requestKey)) return
  pendingRequests.add(requestKey)
  searching.value = true
  searchError.value = ''
  try {
    const res = await searchEntries(props.path, query, cursor)
    if (generation !== searchGeneration || query !== q.value) return
    result.value.push(...res.items)
    searchError.value =
      res.items.length === 0 && result.value.length === 0
        ? t('app.search.no_result')
        : ''
    next.value = res.next
  } catch (e: any) {
    if (generation === searchGeneration && query === q.value) {
      searchError.value = e.message
    }
  } finally {
    pendingRequests.delete(requestKey)
    if (generation === searchGeneration) searching.value = false
  }
}

const reset = () => {
  searchGeneration++
  queryInput.value = ''
  result.value = []
  next.value = 0
  searchError.value = ''
  searching.value = false
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

const onFocusOut = (e: FocusEvent) => {
  const next = e.relatedTarget as Node | null
  if (next && thisEl.value?.contains(next)) return
  if (result.value.length === 0) setActive(false)
}

const onResultScroll = (e: Event) => {
  const target = e.target as HTMLElement
  if (
    !searching.value &&
    target.scrollHeight - target.scrollTop - target.clientHeight < 100
  ) {
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
    document.addEventListener('pointerdown', onDocumentPointerDown)
  }
  if (!active && eventAttached) {
    eventAttached = false
    document.removeEventListener('pointerdown', onDocumentPointerDown)
  }
}

useHotKey((e) => {
  e.preventDefault()
  setActive(true)
}, 'k', { primary: true })

useHotKey(
  () => {
    setActive(false)
  },
  'Escape',
  { el: () => qEl.value! }
)

const onDocumentPointerDown = (e: PointerEvent) => {
  let target = e.target as HTMLElement | null
  do {
    if (target === thisEl.value) break
  } while ((target = target!.parentElement))
  if (target) return
  if (showing.value) e.stopPropagation()
  setActive(false)
}

onUnmounted(() => {
  searchGeneration++
  setActive(false)
})

defineExpose({ setActive })
</script>
<style lang="scss">
.search-panel {
  box-sizing: border-box;
  border: 1px solid var(--color-glass-border);
  border-radius: var(--radius-glass);
  outline: none;
  transition: box-shadow var(--motion-duration-normal)
    var(--motion-easing-standard);
  background-color: var(--color-bg-glass);
  color: var(--color-text);
  overflow: hidden;

  &.active {
    box-shadow: var(--shadow-elevated);
  }
}

.search-panel__result {
  max-height: min(500px, 70vh);
  overflow: hidden auto;
}

.search-panel__search {
  padding: 0 16px;
}

.search-panel__search-input {
  box-sizing: border-box;
  width: 100%;
  border: none;
  background-color: transparent;
  outline: none;
  font-size: 16px;
  color: var(--color-text);
  padding: 16px 0;

  &::placeholder {
    color: var(--color-text-muted);
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
  padding: 16px;
  color: var(--color-text-muted);
  text-align: center;

  p {
    margin: 0;
    line-height: 48px;
  }

  em {
    padding: 0 2px;
    font-style: normal;
    border: solid 1px var(--color-field-border);
    border-radius: 4px;
    margin: 0 6px 6px;
  }
}

.search-panel__items {
  margin: 0;
  padding: 0;
}
</style>
