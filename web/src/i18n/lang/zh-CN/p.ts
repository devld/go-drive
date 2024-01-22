export default {
  admin: {
    admin_group_required: "需要 'admin' 用户组权限",
    oauth_connected: '已连接到 {p}',
    t_site: '站点',
    t_users: '用户',
    t_groups: '用户组',
    t_drives: '盘',
    t_extra_drives: '其他盘',
    t_jobs: '任务',
    t_path_meta: '路径属性',
    t_file_buckets: '文件桶',
    t_misc: '其他',
    t_statistics: '状态',
    save: '保存',
    site: {
      site_settings: '站点',
      app_name: '站点标题',
      global_styles: '全局 CSS 样式',
      inject_scripts: '插入脚本',
      anonymous_root_path: '匿名用户根路径',
      anonymous_root_path_desc:
        '限制未登录的用户只能访问这个目录下的资源，路径不以 / 开头',
      file_preview_config: '文件预览配置',
      external_file_viewers: '外部文件预览器',
      external_file_viewers_desc:
        '使用第三方的服务预览文件（注意：如果服务由第三方提供，那么第三方服务必须可以访问到此应用）。配置格式为每行一条：文件后缀名列表（使用英文逗号隔开）<空格>URL模板<空格>服务名称。以 # 开头的行为注释，将被忽略。',
      text_file_exts: '文本文件后缀名',
      text_file_exts_desc:
        "支持查看和编辑的文本文件后缀名列表（或以 '/' 开头匹配全文件名，如 '/.gitignore'），用英文逗号隔开",
      image_file_exts: '图片文件后缀名',
      image_file_exts_desc: '支持查看的图片文件后缀名列表，用英文逗号隔开',
      audio_file_exts: '音频文件后缀名',
      audio_file_exts_desc: '支持查看的音频文件后缀名列表，用英文逗号隔开',
      video_file_exts: '视频文件后缀名',
      video_file_exts_desc: '支持查看的视频文件后缀名列表，用英文逗号隔开',
      monaco_editor_exts: '使用 Monaco 编辑器',
      monaco_editor_exts_desc:
        "用英文逗号隔开的文件后缀名（或以 '/' 开头匹配全文件名，如 '/.gitignore'），这些文件将会使用 Monaco 编辑器打开",
      thumbnail_config: '缩略图配置',
      thumbnail_mapping: '缩略图生成器映射',
      thumbnail_mapping_tips:
        '配置某个路径中生成缩略图所使用的生成器 tag，每行一个规则。\n格式为: tag1,tag2:路径匹配规则\n其中，挂载路径和 chroot 路径将会被解析为绝对路径来进行匹配\n\n** 匹配零个或多个目录；\n* 匹配任意个数的非目录分隔符字符;\n? 匹配单个非目录分隔符字符。',
      thumbnail_mapping_placeholder: '例如: a,b:Pictures/**/*.jpg',
      thumbnail_mapping_invalid: '无效的映射规则',
      download_options: '下载配置',
      proxy_max: '最大代理大小',
      proxy_max_desc:
        '最大允许通过代理下载的文件大小，可使用 b, k, m, g, t 单位',
      zip_max_size: '打包下载最大允许大小',
      zip_max_size_desc:
        '最大允许打包下载的文件总大小，可使用 b, k, m, g, t 单位',
    },
    drive: {
      reload_drives: '重新加载盘',
      reload_tip: '编辑配置后，重新加载才可生效',
      name: '名称',
      type: '类型',
      operation: '操作',
      edit: '编辑',
      delete: '删除',
      add_drive: '添加盘',
      edit_drive: '编辑 {n}',
      save: '保存',
      cancel: '取消',
      configure: '配置',
      start_configure: '配置',
      configured: '已配置',
      not_configured: '尚未配置',
      add: '添加',
      or_edit: ' 或编辑盘',
      f_name: '名称',
      f_enabled: '已启用',
      f_type: '类型',
      delete_drive: '删除盘',
      confirm_delete: '确认删除 {n}？',
      reload_tips: '你所做的更改只有在重新加载盘后才会生效',
    },
    extra_drive: {
      name: '名称',
      scripts: '脚本',
      ops: '操作',
      install: '安装',
      uninstall: '删除',
      edit: '编辑',
      uninstall_confirm: '确认删除？',
      save: '保存',
      refresh_repository: '重新从仓库拉取',
    },
    user: {
      username: '用户名',
      operation: '操作',
      add_user: '添加用户',
      edit: '编辑',
      delete: '删除',
      edit_user: '编辑 {n}',
      groups: '所属组',
      save: '保存',
      cancel: '取消',
      add: '添加',
      or_edit: ' 或编辑用户',
      f_username: '用户名',
      f_password: '密码',
      f_rootPath: '根目录',
      f_rootPath_desc:
        '限制用户只能访问这个目录下的资源（admin 组的用户将忽略此参数），路径不以 / 开头',
      delete_user: '删除用户',
      confirm_delete: '确认删除 {n}？',
    },
    group: {
      name: '名称',
      operation: '操作',
      add_group: '添加组',
      edit: '编辑',
      delete: '删除',
      edit_group: '编辑 {n}',
      users: '包含用户',
      save: '保存',
      cancel: '取消',
      add: '添加',
      or_edit: ' 或编辑组',
      f_name: '名称',
      delete_group: '删除组',
      confirm_delete: '确认删除 {n}？',
    },
    path_meta: {
      add: '添加',
      path: '路径',
      password: '密码保护',
      def_sort: '默认排序',
      def_mode: '默认展示',
      hidden_pattern: '隐藏文件',
      operation: '操作',
      edit: '编辑',
      delete: '删除',
      save: '保存',
      cancel: '取消',
      delete_item: '删除',
      confirm_delete: '确认删除？',
      f_path: '路径',
      f_password: '目录密码',
      f_password_desc:
        '设置后，未登录的用户需要输入密码才能查看目录内容；WebDAV 未登录状态下将无法访问受密码保护的目录',
      f_password_r: '应用到子路径',
      f_def_sort: '默认排序模式',
      f_def_sort_r: '应用到子路径',
      f_def_mode: '默认展示模式',
      f_def_mode_r: '应用到子路径',
      f_hidden_pattern: '隐藏文件规则',
      f_hidden_pattern_desc:
        '设置该目录下隐藏的文件(夹)名的正则表达式（仅在展示时隐藏），如 .*\\.mp4$',
      f_hidden_pattern_r: '应用到子路径',
      fo_mode_list: '列表',
      fo_mode_thumbnail: '缩略图',
    },
    file_bucket: {
      edit: '编辑',
      add: '添加',
      delete: '删除',
      save: '保存',
      cancel: '取消',
      name: '名称',
      target_path: '目标路径',
      operation: '操作',
      f_name: '名称',
      f_name_desc:
        '名称是这个文件桶的唯一标识，也时上传或访问文件时 URL 中的一部分',
      f_target_path: '目标路径',
      f_target_path_desc: '文件将上传至这个目录下，目标路径不以 / 开头',
      f_key_template: '文件路径模板',
      f_key_template_desc: (
        '文件上传时的路径模板。支持以下变量:\n' +
        '{year}: 年\n{month}: 月\n{date}: 日\n{hour}: 时\n{minute}: 分\n{second}: 秒\n{millisecond}: 毫秒\n' +
        '{timestamp}: 毫秒时间戳\n{rand}: 随机文本\n{name}: 文件名（不包括后缀名）\n{ext}: 文件后缀名（如 .jpg）\n\n' +
        '例如：{year}/{month}/{date}/{hour}/{minute}/{second}/{name}.{ext} 将生成 2024/01/28/12/34/56/test.jpg\n\n' +
        '留空默认为：{year}{month}{date}/{name}-{rand}{ext}'
      ).replace(/([{}])/g, "{'$1'}"),
      f_secret_token: '上传密钥',
      f_secret_token_desc: '向此文件桶中上传文件时需附带的密钥',
      f_url_template: '下载URL模板',
      f_url_template_desc: (
        '上传时返回文件下载链接时使用的 URL 模板。支持以下变量:\n' +
        '{origin}: 当前服务器前缀，如 https://example.com/api\n{bucket}: 文件桶名称\n{key}: 文件路径\n\n' +
        '留空默认为：{origin}/f/{bucket}/{key}'
      ).replace(/([{}])/g, "{'$1'}"),
      f_custom_key: '允许上传时自定义文件路径',
      f_custom_key_desc: '当开启后，上传文件时支持自定义文件路径',
      f_allowed_types: '允许上传的文件类型',
      f_allowed_types_desc:
        '支持 mime-type 或文件后缀名，多个使用英文逗号分隔，如 image/*,video/mp4,.pdf',
      f_max_size: '最大上传文件大小',
      f_max_size_desc: '限制上传时的文件大小，可使用 b, k, m, g, t 单位',
      delete_item: '删除',
      confirm_delete: '确认删除？',
      upload_api_p_path: '<上传路径>',
      upload_api_p_secret_token: '<上传密钥>',
      upload_help_doc_md: `通过如下的 API 上传文件：
\`\`\`
POST {api}
\`\`\`

如果开启了 **允许上传时自定义文件路径**，同时可通过如下的 API 上传文件：

\`\`\`
POST {api_with_path}
\`\`\`

> 支持两种上传格式：文件流直传或 Form 上传。当使用 Form 上传时，文件的 \`key\` 为 \`file\`

<details>
<summary>通过 cURL 上传</summary>

\`\`\`bash
curl -F 'file={'@'}文件路径' {api} # Form 上传方式
curl -X POST --data-binary {'@'}文件路径 {api} # 文件流上传方式
\`\`\`
</details>
`,
    },
    jobs: {
      job: '操作',
      enabled: '启用',
      schedule: '执行计划',
      schedule_desc: 'Cron 表达式。请参考 https://crontab.cronhub.io/',
      next_run: '下次运行时间',
      desc: '描述',
      add_job: '新建任务',
      edit_job: '编辑任务',
      view_log: '查看执行记录',
      job_executions: '执行记录：{n}',
      operation: '操作',
      edit: '编辑',
      execute: '执行',
      delete: '删除',
      save: '保存',
      cancel: '取消',
      delete_job: '删除任务',
      abort_execution: '终止运行',
      confirm_abort_execution: '确认终止运行？',
      confirm_delete: '确认删除？',
      execute_of: '执行：{name}',
      abort: '终止',
      close: '关闭',
      eval_code: '运行代码',
      eval_code_log: '日志',
      status: '状态',
      started_at: '开始于',
      completed_at: '结束于',
      execution_duration: '耗时',
      logs: '日志',
      error_msg: '错误信息',
      success: '成功',
      failed: '失败',
      running: '运行中',
    },
    misc: {
      permission_of_root: '根路径权限',
      clean: '清除',
      clean_invalid: '清理无效的权限项/挂载项',
      clean_cache: '清除缓存',
      refresh_in: '{n} 秒后刷新',
      invalid_path_cleaned: '已清理 {n} 个无效的路径',
      search_index: '文件索引',
      search_disabled: '搜索功能未开启',
      search_form_filter: '过滤器',
      search_form_filter_desc:
        '每行一个过滤器，以 + 开始的行表示包含，已 - 开始的行表示排除。 或者留空将包含所有文件。\n\n** 匹配零个或多个目录；\n* 匹配任意个数的非目录分隔符字符;\n? 匹配单个非目录分隔符字符。',
      search_form_filter_placeholder:
        '例如：\n-**/node_modules/**\n+**/*.jpg\n+**/*.png',
      search_form_filter_invalid: '无效的过滤规则',
      search_form_path: '路径',
      search_form_path_desc: '留空将索引所有文件',
      search_submit_index: '开始索引',
      search_th_path: '路径',
      search_th_status: '状态',
      search_th_created_at: '开始于',
      search_th_updated_at: '更新于',
      search_th_ops: '操作',
      search_index_stop: '停止',
      search_op_index: '索引',
      search_op_delete: '删除',
    },
    p_edit: {
      subject: '主体',
      rw: '(读/写)',
      policy: '策略',
      any: '任何',
      reject: '拒绝',
      accept: '接受',
    },
  },
  task: {
    empty: '现在没有任务',
    start: '开始',
    pause: '暂停',
    stop: '停止',
    remove: '移除',

    s_created: '已创建',
    s_starting: '开始',
    s_paused: '已暂停',
    s_stopped: '已停止',
    s_error: '错误',
    s_completed: '已完成',
  },
  home: {
    readme_loading: '加载 README...',
    readme_failed: '加载 README 失败',
    unsaved_confirm: '尚未保存，确认离开？',
  },
  new_entry: {
    new_item: '新建',
    create_file: '创建空文件',
    upload_file: '上传文件',
    create_folder: '创建文件夹',
    upload_tasks: '上传任务',
    tasks_status: '上传 {p}',
    drop_tip: '拖放到这里以上传',
    invalid_filename: '无效的文件名',
    invalid_folder_name: '无效的文件夹名称',
    confirm_stop_task: '确认停止该任务？',
    confirm_remove_task: '确认移除该任务，不可恢复？',
    resolve_file: '{n} 个文件/文件夹...',
    upload_clipboard: '上传来自剪贴板的文件？',
  },
  login: {
    username: '用户名',
    password: '密码',
    login: '登录',
  },
}
