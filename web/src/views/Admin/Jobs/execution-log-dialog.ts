import showBaseDialog, {
  BaseDialogOptions,
  createDialog,
} from '@/utils/ui-utils/base-dialog'
import { T } from '@/i18n'
import ExecutionLogDialogInner from './ExecutionLogDialogInner.vue'
import type { RequestTask } from '@/utils/http'
import type { StreamHttpResponse } from '@/api/http'
import type { Task } from '@/types'

const ExecutionLogDialog = createDialog(
  'ExecutionLogDialog',
  ExecutionLogDialogInner
)

export interface ExecutionLogDialogOptions extends BaseDialogOptions {
  execute: () => RequestTask<StreamHttpResponse<Task>>
}

export const showExecutionDialog = (opts: ExecutionLogDialogOptions) => {
  return showBaseDialog<void>(ExecutionLogDialog, {
    ...opts,
    closeable: false,
    cancelText: T('p.admin.jobs.close'),
  })
}
