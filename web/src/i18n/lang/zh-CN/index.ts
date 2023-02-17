import app from './app'
import p from './p'
import handlers from './handlers'

export default {
  form: {
    required_msg: '{f}是必填的',
  },
  routes: {
    title: {
      site: '站点',
      users: '用户',
      groups: '用户组',
      drives: '盘',
      extra_drives: '其他盘',
      jobs: '任务',
      path_meta: '路径属性',
      misc: '其他',
      statistics: '状态',
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
