<template>
  <div class="drive-code-editor">
    <div class="header">
      <ul class="drive-code-files">
        <li
          v-for="tab in fileTabs"
          :key="tab.filename"
          class="drive-code-file"
          :class="{ active: tab.filename === activeTab }"
          @click="activeTab = tab.filename"
        >
          {{ tab.filename }}
        </li>
      </ul>
      <SimpleButton :loading="saving" :disabled="loading" @click="onSave">{{
        $t('p.admin.extra_drive.save')
      }}</SimpleButton>
      <button class="plain-button close-button" @click="emit('close')">
        <Icon svg="#icon-close" />
      </button>
    </div>
    <div class="drive-code-editors">
      <CodeEditor
        v-for="tab in fileTabs"
        v-show="tab.filename === activeTab"
        :key="tab.filename"
        v-model="tab.content"
        :type-selectable="false"
        :type="tab.type"
        @save="onSave"
      />
    </div>
  </div>
</template>
<script lang="ts" setup>
import { getDriveScriptContent, saveDriveScriptContent } from '@/api/admin'
import { DriveScriptContent } from '@/types'
import { alert } from '@/utils/ui-utils'
import { computed } from 'vue'
import { ref } from 'vue'
import CodeEditor from '@/components/CodeEditor/index.vue'

interface FileTab {
  name: string
  filename: string
  type: string
  content: string

  prop: keyof DriveScriptContent
}

const props = defineProps({
  name: {
    type: String,
    required: true,
  },
})

const emit = defineEmits<{
  (e: 'close'): void
}>()

const loading = ref(false)
const saving = ref(false)
const content = ref<DriveScriptContent>()

const activeTab = ref<string>()
const fileTabs = computed(() => {
  const c = content.value
  if (!c) return []
  const r: FileTab[] = [
    {
      name: props.name,
      filename: `${props.name}.js`,
      type: 'javascript-server-drive',
      content: c.drive,
      prop: 'drive',
    },
  ]
  if (c.uploader) {
    r.push({
      name: props.name,
      filename: `${props.name}-uploader.js`,
      type: 'javascript-uploader',
      content: c.uploader,
      prop: 'uploader',
    })
  }
  return r
})

const loadData = async () => {
  loading.value = true
  try {
    content.value = await getDriveScriptContent(props.name)
    activeTab.value = fileTabs.value[0]?.filename
  } catch (e: any) {
    alert(e.message)
  } finally {
    loading.value = false
  }
}

const onSave = async () => {
  const t = fileTabs.value.find((e) => e.filename === activeTab.value)!
  saving.value = true
  try {
    await saveDriveScriptContent(t.name, { [t.prop]: t.content })
  } catch (e: any) {
    alert(e.message)
  } finally {
    saving.value = false
  }
}

loadData()
</script>
<style lang="scss">
.drive-code-editor {
  width: 100%;
  height: 100%;
  overflow: hidden;
  display: flex;
  flex-direction: column;

  .header {
    display: flex;
    align-items: center;
  }
}

.drive-code-files {
  flex: 1;
  display: flex;
  margin: 0;
  padding: 0;
  overflow: auto hidden;
}

.drive-code-file {
  margin: 0;
  padding: 10px 20px;
  list-style-type: none;
  cursor: pointer;

  &.active {
    color: var(--link-color);
    border-bottom: solid 2px var(--link-color);
  }
}

.drive-code-editors {
  flex: 1;
  overflow: hidden;

  .code-editor {
    height: 100%;
  }
}
</style>
