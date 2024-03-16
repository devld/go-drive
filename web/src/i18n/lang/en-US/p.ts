export default {
  admin: {
    admin_group_required: "Permission of group 'admin' required",
    oauth_connected: 'Already connected to {p}',
    t_site: 'Site',
    t_users: 'Users',
    t_groups: 'Groups',
    t_drives: 'Drives',
    t_extra_drives: 'Extra Drives',
    t_jobs: 'Jobs',
    t_path_meta: 'Path Attrs',
    t_file_buckets: 'File Buckets',
    t_misc: 'Misc',
    t_statistics: 'Statistics',
    save: 'Save',
    site: {
      site_settings: 'Site',
      app_name: 'Site Title',
      global_styles: 'Global CSS',
      inject_scripts: 'Inject Script',
      anonymous_root_path: 'Anonymous Root Path',
      anonymous_root_path_desc:
        'Restrict non-logged-in users to access resources in this directory only, with paths that do not start with /',
      file_preview_config: 'File Preview config',
      external_file_viewers: 'External File Viewers',
      external_file_viewers_desc:
        'Use third-party services preview file (note: if the service is provided by a third-party, then the third-party service must have access to this application). The configuration format is one per line: [list of file extensions(separated by commas)]<space>[URL template]<space>[service name]. Lines starting with # are comments and will be ignored.',
      text_file_exts: 'Text file extensions',
      text_file_exts_desc:
        "List of text file extensions(or match the full filename starting with '/', e.g. '/.gitignore') supported for viewing and editing, separated by comma",
      image_file_exts: 'Image file extensions',
      image_file_exts_desc:
        'List of supported image file extensions, separated by comma',
      audio_file_exts: 'Audio file extensions',
      audio_file_exts_desc:
        'List of supported audio file extensions, separated by comma',
      video_file_exts: 'Video file extensions',
      video_file_exts_desc:
        'List of supported video file extensions, separated by comma',
      monaco_editor_exts: 'Use the Monaco editor',
      monaco_editor_exts_desc:
        "Comma-separated file extensions(or match the full filename starting with '/', e.g. '/.gitignore'), which will be opened using the Monaco editor",
      thumbnail_config: 'Thumbnail config',
      thumbnail_mapping: 'Thumbnail Generator Mapping',
      thumbnail_mapping_tips:
        'Configure the tag used to match thumbnails generator in a path, one rule per line. The format is: tag1,tag2:path-pattern\nThe mounted path and chrooted path will be resolved to an absolute path for matching\n\n** matches zero or more directories;\n* matches any sequence of non-path-separators;\n? matches any single non-path-separator character.',
      thumbnail_mapping_placeholder: 'Example: a,b:Pictures/**/*.jpg',
      thumbnail_mapping_invalid: 'Invalid mapping pattern',
      download_options: 'Download Options',
      proxy_max: 'Max Proxy Size',
      proxy_max_desc:
        'Maximum allowed file size for downloading via proxy. Units: b, k, m, g, t',
      zip_max_size: 'Maximum allowed size for zip downloads',
      zip_max_size_desc:
        'Maximum total size of files allowed to be packaged and downloaded, Units: b, k, m, g, t',
    },
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
      start_configure: 'Configure',
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
      confirm_delete: 'Are you sure to delete drive {n}?',
      reload_tips: 'You need to reload drives to take effect',
    },
    extra_drive: {
      name: 'Name',
      scripts: 'Scripts',
      ops: 'Operations',
      install: 'Install',
      uninstall: 'Remove',
      edit: 'Edit',
      uninstall_confirm: 'Confirm deletion？',
      save: 'Save',
      refresh_repository: 'Re-pull from Repository',
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
      f_rootPath: 'Root Path',
      f_rootPath_desc:
        'Restrict the user to only access resources in this directory (users in the admin group will ignore this), with paths that do not start with /',
      delete_user: 'Delete user',
      confirm_delete: 'Are you sure to delete user {n}?',
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
      confirm_delete: 'Are you sure to delete group {n}?',
    },
    path_meta: {
      add: 'Add',
      path: 'Path',
      password: 'Password',
      def_sort: 'Default Sorting',
      def_mode: 'Default Listing',
      hidden_pattern: 'Hidden Pattern',
      operation: 'Operation',
      edit: 'Edit',
      delete: 'Delete',
      save: 'Save',
      cancel: 'Cancel',
      delete_item: 'Delete',
      confirm_delete: 'Are you sure to delete?',
      f_path: 'Path',
      f_password: 'Directory Password',
      f_password_desc:
        'After setting a directory password, anonymous users will need to enter the password to view the directory contents; WebDAV anonymous users will not be able to access password-protected directories',
      f_password_r: 'Apply to subpath',
      f_def_sort: 'Default Sort',
      f_def_sort_r: 'Apply to subpath',
      f_def_mode: 'Default List Mode',
      f_def_mode_r: 'Apply to subpath',
      f_hidden_pattern: 'Hidden Files Pattern',
      f_hidden_pattern_desc:
        'Set the regular expression of the hidden file (folder) name in this directory (only hidden during display), such as .*\\.mp4$',
      f_hidden_pattern_r: 'Apply to subpath',
      fo_mode_list: 'List',
      fo_mode_thumbnail: 'Thumbnail',
    },
    file_bucket: {
      edit: 'Edit',
      add: 'Add',
      delete: 'Delete',
      save: 'Save',
      cancel: 'Cancel',
      name: 'Name',
      target_path: 'Target path',
      operation: 'Operation',
      f_name: 'Name',
      f_name_desc:
        'Name is the unique identifier of the file bucket, also the part of the URL when uploading or accessing files',
      f_target_path: 'Target path',
      f_target_path_desc:
        'The target path of the file bucket, not starting with /',
      f_key_template: 'File path template',
      f_key_template_desc: (
        'The file path template when uploading files to this bucket. Supports the following variables:\n' +
        '{year}: Year\n{month}: Month\n{date}: Date\n{hour}: Hour\n{minute}: Minute\n{second}: Second\n{millisecond}: Millisecond\n' +
        '{timestamp}: Millisecond timestamp\n{rand}: Random text\n{name}: File name (without extension)\n{ext}: File extension(e.g. .jpg)\n\n' +
        "Example: '{year}/{month}/{date}/{hour}/{minute}/{second}/{name}{ext}' will generate '2024/01/28/12/34/56/test.jpg'\n\n" +
        'If leave blank will use default: {year}{month}{date}/{name}-{rand}{ext}'
      ).replace(/([{}])/g, "{'$1'}"),
      f_secret_token: 'Upload secret key',
      f_secret_token_desc: 'Key required when uploading files to this bucket',
      f_url_template: 'Download URL Template',
      f_url_template_desc: (
        'URL template used when returning the file download link during upload. Supports the following variables:\n' +
        '{origin}: Current server api prefix, such as https://example.com/api\n{bucket}: File bucket name\n{key}: File path\n\n' +
        'If leave blank will use default: {origin}/f/{bucket}/{key}'
      ).replace(/([{}])/g, "{'$1'}"),
      f_custom_key: 'Allow custom file path on upload',
      f_custom_key_desc:
        'When enabled, supports custom file paths when uploading files',
      f_allowed_types: 'Allowed file types for upload',
      f_allowed_types_desc:
        'Supports mime-type or file extensions, multiple types separated by commas, e.g. image/*,video/mp4,.pdf',
      f_max_size: 'Maximum upload file size',
      f_max_size_desc:
        'Limits the file size for uploads, can use units such as b, k, m, g, t',
      f_allowed_referrers: 'Allowed Referer list',
      f_allowed_referrers_desc:
        'Allowed Referer list, multiple separated by commas, leave blank to turn off hotlink protection by default. * can be used to match subdomain names. \nFor example: example.com,*.example.com',
      f_cache_max_age: 'Download cache TTL',
      f_cache_max_age_desc:
        'File access cache time (Cache-Control), valid units are ms, s, m, h, d, the default is one day.',
      delete_item: 'Delete',
      confirm_delete: 'Confirm deletion?',
      upload_api_p_path: '<UPLOAD PATH>',
      upload_api_p_secret_token: '<SECRET TOKEN>',
      upload_help_doc_md: `Upload file using the following API：
\`\`\`
POST {api}
\`\`\`

If **Allow custom file path on upload** is enabled, you can also upload file using the following API:

\`\`\`
POST {api_with_path}
\`\`\`

> Supports two types of upload formats: direct file stream or form upload. When using form upload, the file's \`key\` is \`file\`.

<details>
<summary>Upload file using cURL</summary>

\`\`\`bash
curl -F 'file={'@'}FILE_PATH' '{api}' # Form upload
curl -X POST --data-binary {'@'}FILE_PATH '{api}' # Direct file stream upload
\`\`\`
</details>
`,
    },
    jobs: {
      job: 'Jobs',
      enabled: 'Enabled',
      schedule: 'Schedule',
      schedule_desc: 'Cron Expression. see https://crontab.cronhub.io/',
      next_run: 'Next RunTime',
      desc: 'Description',
      add_job: 'Add job',
      edit_job: 'Edit job',
      view_log: 'View executions log',
      job_executions: 'Executions Log: {n}',
      operation: 'Operation',
      edit: 'Edit',
      execute: 'Execute',
      delete: 'Delete',
      save: 'Save',
      cancel: 'Cancel',
      delete_job: 'Delete job',
      abort_execution: 'Abort Execution',
      confirm_abort_execution: 'Are you confirm to abort this execution?',
      confirm_delete: 'Are you confirm to delete?',
      execute_of: 'Execute: {name}',
      abort: 'Abort',
      close: 'Close',
      eval_code: 'Execute code',
      eval_code_log: 'Log',
      status: 'Status',
      started_at: 'Started At',
      completed_at: 'Completed At',
      execution_duration: 'Duration',
      logs: 'Logs',
      error_msg: 'Error Message',
      success: 'Success',
      failed: 'Failed',
      running: 'Running',
    },
    misc: {
      permission_of_root: 'Permission of root',
      clean: 'Clean',
      clean_invalid: 'Clean Invalid Permissions and Mounts',
      clean_cache: 'Clean Cache',
      refresh_in: 'Refresh in {n}s',
      invalid_path_cleaned: '{n} invalid paths cleaned',
      search_index: 'Files Index',
      search_disabled: 'Search is not enabled',
      search_form_filter: 'Filters',
      search_form_filter_desc:
        'Filters line by line, line starts with + for including, line starts with - for excluding. Or leave blank to include all files.\n** matches zero or more directories;\n* matches any sequence of non-path-separators;\n? matches any single non-path-separator character.',
      search_form_filter_placeholder:
        'Examples:\n-**/node_modules/**\n+**/*.jpg\n+**/*.png',
      search_form_filter_invalid: 'Invalid filters',
      search_form_path: 'Path',
      search_form_path_desc: 'Leave blank to index all files',
      search_submit_index: 'Index now',
      search_th_path: 'Path',
      search_th_status: 'Status',
      search_th_created_at: 'Started At',
      search_th_updated_at: 'Updated At',
      search_th_ops: 'Operations',
      search_index_stop: 'Stop',
      search_op_index: 'Index',
      search_op_delete: 'Delete',
    },
    p_edit: {
      subject: 'Subject',
      rw: '(R/W)',
      policy: 'Policy',
      any: 'ANY',
      reject: 'Reject',
      accept: 'Accept',
    },
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
    s_completed: 'Completed',
  },
  home: {
    readme_loading: 'Loading README...',
    readme_failed: 'Failed to load README.',
    unsaved_confirm: 'You have some unsaved changes, are you sure to leave?',
  },
  new_entry: {
    new_item: 'New item',
    create_file: 'Create empty file',
    upload_file: 'Upload file',
    create_folder: 'Create folder',
    upload_tasks: 'Upload Tasks',
    tasks_status: 'Tasks {p}',
    drop_tip: 'Drop files here to upload',
    invalid_filename: 'Invalid filename',
    invalid_folder_name: 'Invalid folder name',
    confirm_stop_task: 'Stop this task?',
    confirm_remove_task: 'Remove this task, cannot be undone?',
    resolve_file: '{n} files/directories...',
    upload_clipboard: 'Uploading files from the clipboard?',
  },
  login: {
    username: 'Username',
    password: 'Password',
    login: 'Login',
  },
}
