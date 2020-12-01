export default {
  app: {
    login: 'Login',
    logout: 'Logout',
    home: 'Home',
    admin: 'Admin',
    username: 'Username',
    groups: 'Groups',
    file: 'File',
    folder: 'Folder',
    empty_list: 'Nothing here',
    go_back: 'Go back',
    root_path: 'Root'
  },
  error: {
    not_allowed: 'Operation not allowed',
    not_found: 'Resource not found',
    server_error: 'Server Error'
  },
  form: {
    required_msg: '{f} is required'
  },
  routes: {
    title: {
      users: 'Users',
      groups: 'Groups',
      drives: 'Drives',
      misc: 'Misc'
    }
  },
  md: {
    error: 'An error occurred while rendering markdown'
  },
  dialog: {
    base: {
      ok: 'OK'
    },
    open: {
      max_items: 'Select at most {n} items.',
      n_selected: '{n} items selected.',
      clear: 'clear'
    },
    text: {
      yes: 'Yes',
      no: 'No'
    },
    loading: {
      cancel: 'Cancel'
    }
  },
  p: {
    admin: {
      oauth_connected: 'Already connected to {p}',
      t_users: 'Users',
      t_groups: 'Groups',
      t_drives: 'Drives',
      t_misc: 'Misc',
      drive: {
        reload_drives: 'Reload drives',
        reload_tip: 'Reload drives to take effect',
        name: 'Name',
        type: 'Type',
        operation: 'Operation',
        edit: 'Edit',
        delete: 'Delete',
        add_drive: 'Add drive',
        edit_drive: 'Edit drive: {n}',
        save: 'Save',
        cancel: 'Cancel',
        configure: 'Configure',
        configured: 'Configured',
        not_configured: 'Not configured',
        add: 'Add',
        or_edit: ' or edit drive',
        f_name: 'Name',
        f_enabled: 'Enabled',
        f_type: 'Type',
        delete_drive: 'Delete drive',
        confirm_delete: 'Are you sure to delete drive {n}?'
      },
      user: {
        username: 'Username',
        operation: 'Operation',
        add_user: 'Add user',
        edit: 'Edit',
        delete: 'Delete',
        edit_user: 'Edit user {n}',
        groups: 'Groups',
        save: 'Save',
        cancel: 'Cancel',
        add: 'Add',
        or_edit: ' or edit user',
        f_username: 'Username',
        f_password: 'Password',
        delete_user: 'Delete user',
        confirm_delete: 'Are you sure to delete user {n}?'
      },
      group: {
        name: 'Name',
        operation: 'Operation',
        add_group: 'Add group',
        edit: 'Edit',
        delete: 'Delete',
        edit_group: 'Edit group {n}',
        users: 'Users',
        save: 'Save',
        cancel: 'Cancel',
        add: 'Add',
        or_edit: ' or edit group',
        f_name: 'Name',
        delete_group: 'Delete group',
        confirm_delete: 'Are you sure to delete group {n}?'
      },
      misc: {
        permission_of_root: 'Permission of root',
        save: 'Save',
        clean: 'Clean',
        clean_invalid: 'Clean invalid permissions and mounts',
        clean_cache: 'Clean cache',
        statistics: 'Statistics',
        refresh_in: 'Refresh in {n}s'
      },
      p_edit: {
        subject: 'Subject',
        rw: '(R/W)',
        policy: 'Policy',
        any: 'ANY',
        reject: 'Reject',
        accept: 'Accept'
      }
    },
    task: {
      empty: 'Nothing here',
      start: 'Start',
      pause: 'Pause',
      stop: 'Stop',
      remove: 'Remove',

      s_created: 'Created',
      s_starting: 'Starting',
      s_paused: 'Paused',
      s_stopped: 'Stopped',
      s_error: 'Error',
      s_complete: 'Completed'
    },
    home: {
      file_exists: '\'{n}\' already exists, override or skip?',
      apply_all: 'Apply for all',
      readme_loading: 'Loading README...',
      readme_failed: 'Failed to load README.',
      unsaved_confirm: 'You have some unsaved changes, are you sure to leave?'
    },
    new_entry: {
      new_item: 'New item',
      upload_file: 'Upload file',
      create_folder: 'Create folder',
      upload_tasks: 'Upload Tasks',
      tasks_status: 'Tasks {p}',
      drop_tip: 'Drop files here to upload',
      invalid_folder_name: 'Invalid folder name',
      confirm_stop_task: 'Stop this task?',
      confirm_remove_task: 'Remove this task, cannot be undone?',
      file_exists: 'File exists',
      file_exists_confirm: '\'{n}\' already exists, override or skip?',
      skip: 'Skip',
      override: 'Override'
    },
    login: {
      username: 'Username',
      password: 'Password',
      login: 'Login'
    }
  },
  hv: {
    download: {
      download: 'Download'
    },
    permission: {
      save: 'Save'
    },
    text_edit: {
      save: 'Save'
    }
  },
  handler: {
    copy_move: {
      copy: 'Copy',
      move: 'Move',
      copy_to: 'Copy to',
      move_to: 'Move to',
      copy_desc: 'Copy files',
      move_desc: 'Move files',
      copying: 'Copying {n} {p}',
      moving: 'Moving {n} {p}',
      copy_open_title: 'Select copy to',
      move_open_title: 'Select move to',
      override_or_skip: 'Override or skip for duplicates?',
      override: 'Override',
      skip: 'Skip'
    },
    delete: {
      name: 'Delete',
      desc: 'Delete files',
      confirm_n: 'Delete these {n} files?',
      confirm: 'Delete this file?',
      deleting: 'Deleting {n} {p}'
    },
    download: {
      name: 'Download',
      desc: 'Download file'
    },
    image: {
      name: 'Gallery',
      desc: 'View images'
    },
    media: {
      name: 'Play',
      desc: 'Play media'
    },
    mount: {
      name: 'Mount to',
      desc: 'Mount entries to another location',
      open_title: 'Select mount to'
    },
    permission: {
      name: 'Permissions',
      desc: 'Set permissions for this item'
    },
    rename: {
      name: 'Rename',
      desc: 'Rename this file',
      input_title: 'Rename',
      invalid_filename: 'Invalid filename'
    },
    text_edit: {
      edit_name: 'Edit',
      view_name: 'View',
      edit_desc: 'Edit this file',
      view_desc: 'View this file'
    }
  }
}
