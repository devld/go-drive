import app from './app'
import p from './p'
import handlers from './handlers'

export default {
  form: {
    required_msg: '{f} is required',
    select_path: 'Select',
  },
  routes: {
    title: {
      site: 'Site',
      users: 'Users',
      groups: 'Groups',
      drives: 'Drives',
      extra_drives: 'Extra Drives',
      jobs: 'Jobs',
      path_meta: 'Path Attrs',
      file_buckets: 'File Buckets',
      misc: 'Misc',
      statistics: 'Statistics',
    },
  },
  md: {
    error: 'An error occurred while rendering markdown',
  },
  dialog: {
    base: {
      ok: 'OK',
    },
    open: {
      max_items: 'Select at most {n} items.',
      n_selected: '{n} items selected.',
      clear: 'clear',
    },
    text: {
      yes: 'Yes',
      no: 'No',
    },
    loading: {
      cancel: 'Cancel',
    },
  },
  app,
  p,
  ...handlers,
}
