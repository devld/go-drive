import { defineConfig } from 'vitepress'

const siteUrl = 'https://go-drive.top'
const socialImageUrl = `${siteUrl}/social-card.png`

function pageUrl(relativePath: string) {
  const pathname = relativePath
    .replace(/(^|\/)index\.md$/, '$1')
    .replace(/\.md$/, '.html')
  return new URL(pathname, `${siteUrl}/`).href
}

const breadcrumbLabels: Record<string, Record<string, string>> = {
  en: {
    administration: 'Administration',
    configuration: 'Configuration',
    drives: 'Drives',
    extensions: 'Extensions',
    features: 'Features',
    'getting-started': 'Getting Started',
    jobs: 'Jobs',
    reference: 'Reference',
    troubleshooting: 'Troubleshooting'
  },
  'zh-CN': {
    administration: '管理',
    configuration: '配置',
    drives: 'Drive',
    extensions: '扩展',
    features: '功能',
    'getting-started': '快速开始',
    jobs: '自动任务',
    reference: '参考',
    troubleshooting: '故障排查'
  }
}

function breadcrumbs(relativePath: string, title: string, lang: string) {
  const withoutLocale = relativePath.replace(/^zh-CN\//, '')
  const segments = withoutLocale.split('/')
  const localePrefix = lang === 'zh-CN' ? 'zh-CN/' : ''
  const items = [
    {
      '@type': 'ListItem',
      position: 1,
      name: lang === 'zh-CN' ? '首页' : 'Home',
      item: `${siteUrl}/${localePrefix}`
    }
  ]

  const isSectionIndex = segments.at(-1) === 'index.md'
  if (segments.length > 1 && !isSectionIndex) {
    const section = segments[0]
    items.push({
      '@type': 'ListItem',
      position: items.length + 1,
      name: breadcrumbLabels[lang]?.[section] ?? section,
      item: `${siteUrl}/${localePrefix}${section}/`
    })
  }

  items.push({
    '@type': 'ListItem',
    position: items.length + 1,
    name: title,
    item: pageUrl(relativePath)
  })
  return items
}

const enSidebar = [
  {
    text: 'Getting Started',
    items: [
      { text: 'Installation and Startup', link: '/getting-started/' },
      { text: 'Upgrade, Backup, and Restore', link: '/getting-started/upgrade-backup' }
    ]
  },
  {
    text: 'Configuration',
    items: [
      { text: 'Configuration Reference', link: '/configuration/' },
      { text: 'Reverse Proxy', link: '/configuration/reverse-proxy' },
      { text: 'Security', link: '/configuration/security' }
    ]
  },
  {
    text: 'Drives',
    items: [
      { text: 'Overview', link: '/drives/' },
      { text: 'Local', link: '/drives/local' },
      { text: 'FTP', link: '/drives/ftp' },
      { text: 'SFTP', link: '/drives/sftp' },
      { text: 'WebDAV', link: '/drives/webdav' },
      { text: 'S3', link: '/drives/s3' },
      { text: 'OneDrive', link: '/drives/onedrive' },
      { text: 'Google Drive', link: '/drives/google-drive' }
    ]
  },
  {
    text: 'Administration',
    items: [
      { text: 'Access Control', link: '/administration/access-control' },
      { text: 'Path Attributes and Mounts', link: '/administration/path-attrs-mounts' },
      { text: 'Maintenance', link: '/administration/maintenance' }
    ]
  },
  {
    text: 'Features',
    items: [
      { text: 'Search', link: '/features/search' },
      { text: 'WebDAV Access', link: '/features/webdav' },
      { text: 'File Buckets', link: '/features/file-buckets' },
      { text: 'Site Settings', link: '/features/site-settings' },
      { text: 'Preview and Thumbnails', link: '/features/preview-thumbnail' },
      { text: 'Automated Jobs', link: '/jobs/' }
    ]
  },
  {
    text: 'Extensions and Reference',
    items: [
      { text: 'Script Drives', link: '/extensions/script-drives' },
      { text: 'Command Line', link: '/reference/cli' },
      { text: 'Path Patterns', link: '/reference/path-patterns' },
      { text: 'Troubleshooting', link: '/troubleshooting/' },
      { text: 'Privacy', link: '/privacy' }
    ]
  }
]

const zhSidebar = [
  {
    text: '快速开始',
    items: [
      { text: '安装与启动', link: '/zh-CN/getting-started/' },
      { text: '升级、备份与恢复', link: '/zh-CN/getting-started/upgrade-backup' }
    ]
  },
  {
    text: '配置',
    items: [
      { text: '配置文件参考', link: '/zh-CN/configuration/' },
      { text: '反向代理', link: '/zh-CN/configuration/reverse-proxy' },
      { text: '安全指南', link: '/zh-CN/configuration/security' }
    ]
  },
  {
    text: 'Drive',
    items: [
      { text: '总览', link: '/zh-CN/drives/' },
      { text: '本地文件', link: '/zh-CN/drives/local' },
      { text: 'FTP', link: '/zh-CN/drives/ftp' },
      { text: 'SFTP', link: '/zh-CN/drives/sftp' },
      { text: 'WebDAV', link: '/zh-CN/drives/webdav' },
      { text: 'S3', link: '/zh-CN/drives/s3' },
      { text: 'OneDrive', link: '/zh-CN/drives/onedrive' },
      { text: 'Google Drive', link: '/zh-CN/drives/google-drive' }
    ]
  },
  {
    text: '管理',
    items: [
      { text: '用户、组和权限', link: '/zh-CN/administration/access-control' },
      { text: '路径属性与挂载', link: '/zh-CN/administration/path-attrs-mounts' },
      { text: '维护和运行状态', link: '/zh-CN/administration/maintenance' }
    ]
  },
  {
    text: '功能',
    items: [
      { text: '搜索与索引', link: '/zh-CN/features/search' },
      { text: 'WebDAV 访问', link: '/zh-CN/features/webdav' },
      { text: '文件桶', link: '/zh-CN/features/file-buckets' },
      { text: '站点设置', link: '/zh-CN/features/site-settings' },
      { text: '预览与缩略图', link: '/zh-CN/features/preview-thumbnail' },
      { text: '自动任务', link: '/zh-CN/jobs/' }
    ]
  },
  {
    text: '扩展和参考',
    items: [
      { text: '脚本 Drive', link: '/zh-CN/extensions/script-drives' },
      { text: '命令行参考', link: '/zh-CN/reference/cli' },
      { text: '路径模式', link: '/zh-CN/reference/path-patterns' },
      { text: '故障排查', link: '/zh-CN/troubleshooting/' },
      { text: '隐私说明', link: '/zh-CN/privacy' }
    ]
  }
]

export default defineConfig({
  title: 'go-drive',
  description: 'Documentation for the go-drive self-hosted file management server',
  lang: 'en',
  lastUpdated: true,
  head: [['link', { rel: 'icon', href: '/favicon.png' }]],
  sitemap: {
    hostname: siteUrl
  },
  transformPageData(pageData) {
    if (pageData.isNotFound) return

    const { relativePath, title, description, frontmatter } = pageData
    const canonicalUrl = pageUrl(relativePath)
    const lang = frontmatter.lang || 'en'
    const isHome = relativePath === 'index.md' || relativePath === 'zh-CN/index.md'
    const socialTitle = isHome ? title : `${title} | go-drive`
    const structuredData = isHome
      ? {
          '@context': 'https://schema.org',
          '@type': 'WebSite',
          '@id': `${siteUrl}/#website`,
          url: canonicalUrl,
          name: 'go-drive',
          alternateName: lang === 'zh-CN' ? 'go-drive 文档' : 'go-drive Documentation',
          description,
          inLanguage: lang
        }
      : [
          {
            '@context': 'https://schema.org',
            '@type': 'TechArticle',
            headline: title,
            description,
            inLanguage: lang,
            image: socialImageUrl,
            dateModified: pageData.lastUpdated
              ? new Date(pageData.lastUpdated).toISOString()
              : undefined,
            url: canonicalUrl,
            mainEntityOfPage: canonicalUrl,
            isPartOf: {
              '@type': 'WebSite',
              '@id': `${siteUrl}/#website`,
              name: 'go-drive Documentation',
              url: `${siteUrl}/`
            }
          },
          {
            '@context': 'https://schema.org',
            '@type': 'BreadcrumbList',
            itemListElement: breadcrumbs(relativePath, title, lang)
          }
        ]

    frontmatter.head ??= []
    frontmatter.head.push(
      ['link', { rel: 'canonical', href: canonicalUrl }],
      ['meta', { property: 'og:type', content: isHome ? 'website' : 'article' }],
      ['meta', { property: 'og:site_name', content: 'go-drive' }],
      ['meta', { property: 'og:title', content: socialTitle }],
      ['meta', { property: 'og:description', content: description }],
      ['meta', { property: 'og:url', content: canonicalUrl }],
      ['meta', { property: 'og:image', content: socialImageUrl }],
      ['meta', { property: 'og:image:width', content: '1731' }],
      ['meta', { property: 'og:image:height', content: '909' }],
      ['meta', { property: 'og:image:alt', content: 'go-drive' }],
      ['meta', { property: 'og:locale', content: lang === 'zh-CN' ? 'zh_CN' : 'en_US' }],
      ['meta', { property: 'og:locale:alternate', content: lang === 'zh-CN' ? 'en_US' : 'zh_CN' }],
      ['meta', { name: 'twitter:card', content: 'summary_large_image' }],
      ['meta', { name: 'twitter:title', content: socialTitle }],
      ['meta', { name: 'twitter:description', content: description }],
      ['meta', { name: 'twitter:image', content: socialImageUrl }],
      ['meta', { name: 'twitter:image:alt', content: 'go-drive' }],
      ['script', { type: 'application/ld+json' }, JSON.stringify(structuredData)]
    )
  },
  locales: {
    root: {
      label: 'English',
      lang: 'en',
      title: 'go-drive',
      description: 'Installation, configuration, administration, and extension documentation for go-drive',
      themeConfig: {
        nav: [
          { text: 'Home', link: '/' },
          { text: 'Install', link: '/getting-started/' },
          { text: 'Configuration', link: '/configuration/' },
          { text: 'Supported Drives', link: '/drives/' },
          { text: 'Demo', link: 'https://demo.go-drive.top' },
          { text: 'Download', link: 'https://github.com/devld/go-drive/releases' }
        ],
        sidebar: enSidebar,
        outline: { label: 'On this page' },
        editLink: {
          pattern: 'https://github.com/devld/go-drive/edit/master/docs/site/:path',
          text: 'Edit this page on GitHub'
        },
        lastUpdated: { text: 'Last updated' },
        docFooter: { prev: 'Previous', next: 'Next' },
        footer: {
          message: 'Released under the MIT License.',
          copyright: 'Copyright © 2020-present devld'
        }
      }
    },
    'zh-CN': {
      label: '简体中文',
      lang: 'zh-CN',
      link: '/zh-CN/',
      title: 'go-drive',
      description: 'go-drive 自托管文件管理服务器的安装、配置、管理与扩展文档',
      themeConfig: {
        nav: [
          { text: '首页', link: '/zh-CN/' },
          { text: '安装', link: '/zh-CN/getting-started/' },
          { text: '配置', link: '/zh-CN/configuration/' },
          { text: '支持的存储', link: '/zh-CN/drives/' },
          { text: '在线演示', link: 'https://demo.go-drive.top' },
          { text: '下载', link: 'https://github.com/devld/go-drive/releases' }
        ],
        sidebar: zhSidebar,
        outline: { label: '本页目录' },
        editLink: {
          pattern: 'https://github.com/devld/go-drive/edit/master/docs/site/:path',
          text: '在 GitHub 上编辑此页'
        },
        lastUpdated: { text: '最后更新' },
        docFooter: { prev: '上一页', next: '下一页' },
        footer: {
          message: '基于 MIT 许可证发布。',
          copyright: 'Copyright © 2020-present devld'
        }
      }
    }
  },
  themeConfig: {
    logo: '/favicon.png',
    search: {
      provider: 'local'
    },
    socialLinks: [
      { icon: 'github', link: 'https://github.com/devld/go-drive' }
    ]
  }
})
