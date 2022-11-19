import app from './app'
import p from './p'
import handlers from './handlers'

export default {
  error: {
    not_allowed: '不允许的操作',
    not_found: '资源不存在',
    server_error: '服务器错误',
  },
  form: {
    required_msg: '{f}是必填的',
  },
  routes: {
    title: {
      site: '站点',
      users: '用户',
      groups: '用户组',
      drives: '盘',
      jobs: '任务',
      misc: '其他',
      statistics: '统计信息',
    },
  },
  md: {
    error: '渲染 Markdown 时出现错误',
  },
  dialog: {
    base: {
      ok: '确定',
    },
    open: {
      max_items: '最多可选择 {n} 个',
      n_selected: '已选 {n} 个',
      clear: '清除',
    },
    text: {
      yes: '是',
      no: '否',
    },
    loading: {
      cancel: '取消',
    },
  },
  app,
  p,
  ...handlers,
}
