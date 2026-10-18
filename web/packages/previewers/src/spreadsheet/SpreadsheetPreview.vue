<template>
  <div class="spreadsheet-preview">
    <div v-if="error" class="spreadsheet-preview__error">{{ error }}</div>
    <table v-else-if="rows.length" class="spreadsheet-preview__table">
      <thead>
        <tr>
          <th v-for="(cell, index) in rows[0]" :key="index">
            {{ displayCell(cell) }}
          </th>
        </tr>
      </thead>
      <tbody>
        <tr v-for="(row, rowIndex) in rows.slice(1)" :key="rowIndex">
          <td v-for="(cell, index) in row" :key="index">
            {{ displayCell(cell) }}
          </td>
        </tr>
      </tbody>
    </table>
    <div v-else class="spreadsheet-preview__empty">
      {{ t('preview.spreadsheet.empty') }}
    </div>
  </div>
</template>
<script setup lang="ts">
import { useI18n } from '@go-drive/i18n'
import { ref, watch } from 'vue'
import * as XLSX from 'xlsx'

const { t } = useI18n({ useScope: 'global' })

const props = withDefaults(
  defineProps<{
    content?: string
    delimiter?: ',' | '\t'
  }>(),
  {
    content: '',
    delimiter: ',',
  }
)

const error = ref('')
const rows = ref<unknown[][]>([])

const parseContent = () => {
  error.value = ''
  if (!props.content) {
    rows.value = []
    return
  }

  try {
    const workbook = XLSX.read(props.content, {
      type: 'string',
      raw: false,
      FS: props.delimiter,
    })
    const sheet = workbook.Sheets[workbook.SheetNames[0]]
    if (!sheet) {
      rows.value = []
      return
    }
    rows.value = XLSX.utils.sheet_to_json(sheet, {
      header: 1,
      raw: false,
      defval: '',
      blankrows: false,
    }) as unknown[][]
  } catch (e) {
    error.value = e instanceof Error ? e.message : t('preview.spreadsheet.error')
    rows.value = []
  }
}

watch(() => [props.content, props.delimiter], parseContent, { immediate: true })

const displayCell = (cell: unknown) =>
  cell === null || typeof cell === 'undefined' ? '' : String(cell)
</script>
<style lang="scss">
.spreadsheet-preview {
  box-sizing: border-box;
  width: 100%;
  height: 100%;
  overflow: auto;
  padding: 16px;
  background: var(--color-bg-elevated, #fff);
}

.spreadsheet-preview__table {
  width: max-content;
  min-width: 100%;
  border-collapse: collapse;
  color: var(--color-text, #222);
  font-size: 14px;

  th,
  td {
    min-width: 120px;
    max-width: 480px;
    padding: 8px 10px;
    border: 1px solid var(--color-border, #ddd);
    text-align: left;
    white-space: pre-wrap;
    overflow-wrap: anywhere;
    vertical-align: top;
  }

  th {
    position: sticky;
    top: 0;
    z-index: 1;
    background: var(--color-bg-hover, #f2f2f2);
    font-weight: 600;
  }
}

.spreadsheet-preview__empty,
.spreadsheet-preview__error {
  padding: 32px;
  color: var(--color-text-muted, #777);
  text-align: center;
}

.spreadsheet-preview__error {
  color: var(--color-danger, #c00);
}
</style>
