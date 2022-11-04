import app from './app'
import p from './p'
import handlers from './handlers'

export default {
  error: {
    not_allowed: 'Operation not allowed',
    not_found: 'Resource not found',
    server_error: 'Server Error',
  },
  form: {
    required_msg: '{f} is required',
  },
  routes: {
    title: {
      site: 'Site',
      users: 'Users',
      groups: 'Groups',
      drives: 'Drives',
      jobs: 'Jobs',
      misc: 'Misc',
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
