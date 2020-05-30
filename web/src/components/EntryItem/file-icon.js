/* eslint-disable quote-props */

import { filenameExt } from '@/utils'

const fileExts = {
  'com': ['com', 'sh', 'bat'],
  'conf': ['conf', 'config', 'ini', 'yml', 'yaml', 'properties'],
  'csv': ['csv'],
  'dat': ['dat', 'data'],
  'deb': ['deb', 'rpm', 'msi'],
  'epub': ['epub'],
  'mp': ['mp3', 'm4a', 'ogg', 'flac'],
  'exe': ['exe'],
  'gitignore': ['gitignore'],
  'gitattributes': ['gitattributes'],
  'ico': ['icon'],
  'iso': ['iso', 'img'],
  'jpeg': ['jpg', 'jpeg', 'png', 'gif', 'bmp', 'webp'],
  'json': ['json'],
  'log': ['log'],
  'md': ['md'],
  'mp1': ['mp4', 'mov', 'flv', 'rmvb', 'mkv'],
  'pdf': ['pdf'],
  'doc': ['doc', 'docx'],
  'pptx': ['ppt', 'pptx'],
  'xlsx': ['xls', 'xlsx'],
  'xml': ['html', 'htm', 'css', 'scss', 'sass', 'js', 'jsx', 'xml', 'pom', 'java', 'go', 'ts', 'kt', 'kts', 'py', 'lua', 'c', 'cpp', 'h', 'vue', 'gradle'],
  'zip': ['zip', 'tar', 'gz', 'rar', '7z'],
  'apk': ['apk']
}

const extMapping = {}
Object.keys(fileExts).forEach(icon => {
  fileExts[icon].forEach(ext => { extMapping[ext] = icon })
})

const dirIcon = 'folder'
const parentDirIcon = 'iconfanhuishangyiji'
const fileFallbackIcon = 'file'

export function getIconSVG (entry) {
  let icon
  if (entry.type === 'dir') icon = dirIcon
  if (entry.type === 'file') icon = extMapping[filenameExt(entry.name)] || fileFallbackIcon
  if (entry.name === '..') icon = parentDirIcon
  return '#icon-' + icon
}
