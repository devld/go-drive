export const languages = {
  cpp: () => import('@codemirror/lang-cpp').then((m) => m.cpp()),
  css: () => import('@codemirror/lang-css').then((m) => m.css()),
  html: () => import('@codemirror/lang-html').then((m) => m.html()),
  java: () => import('@codemirror/lang-java').then((m) => m.java()),
  javascript: () =>
    import('@codemirror/lang-javascript').then((m) =>
      m.javascript({ jsx: true, typescript: false })
    ),
  typescript: () =>
    import('@codemirror/lang-javascript').then((m) =>
      m.javascript({ jsx: true, typescript: true })
    ),
  json: () => import('@codemirror/lang-json').then((m) => m.json()),
  markdown: () => import('@codemirror/lang-markdown').then((m) => m.markdown()),
  php: () => import('@codemirror/lang-php').then((m) => m.php()),
  python: () => import('@codemirror/lang-python').then((m) => m.python()),
  sql: () => import('@codemirror/lang-sql').then((m) => m.sql()),
  xml: () => import('@codemirror/lang-xml').then((m) => m.xml()),
}

const mapping: { [k in keyof typeof languages]: string[] } = {
  cpp: [
    'cpp',
    'c++',
    'cc',
    'cp',
    'cxx',
    'h',
    'h++',
    'hh',
    'hpp',
    'hxx',
    'inc',
    'inl',
    'ipp',
    'tcc',
    'tpp',
    'c',
  ],
  css: ['scss', 'css', 'less'],
  html: ['html', 'htm', 'xhtml'],
  java: ['java'],
  javascript: ['js', 'jsx'],
  typescript: ['ts', 'tsx'],
  json: ['json'],
  markdown: ['md', 'markdown'],
  php: ['php'],
  python: ['py'],
  sql: ['sql'],
  xml: ['xml', 'ant', 'plist', 'xsd'],
}

const extMapping: Record<string, keyof typeof languages> = {}
Object.entries(mapping).forEach(([language, extensions]) => {
  extensions.forEach((extension) => {
    extMapping[extension] = language as keyof typeof languages
  })
})

export const getLang = async (lang: string) => {
  const l = languages[lang as keyof typeof languages]
  if (!l) return
  return l()
}

export const getLangByFilename = (filename: string) => {
  const name = filename.toLowerCase()
  const extension = name.includes('.') ? name.split('.').pop() : undefined
  return extension ? extMapping[extension] : undefined
}
