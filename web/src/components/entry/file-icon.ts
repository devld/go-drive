import { Entry } from '@/types'
import { createEntryExtMatcher } from '@/utils'
import type { IconName } from '@/components/icons'

const exts = (value: string) => value.split(',')

const fileExts: O<string[]> = {
  log: exts(
    'conf,config,cfg,ini,yml,yaml,properties,log,txt,env,editorconfig,gitignore,gitattributes,gitmodules,/.gitignore,/.gitconfig,/.editorconfig,/.npmrc,/.yarnrc,/dockerfile,/.vimrc'
  ),
  mp3: exts(
    'mp3,m4a,flac,mid,midi,wav,aac,aif,aiff,alac,ape,au,cda,m4b,oga,ogg,opus,wma,weba'
  ),
  exe: exts(
    'exe,deb,rpm,com,jar,msi,appimage,bin,dmg,ipa,mpkg,pkg,run'
  ),
  jpeg: exts(
    'jpg,jpeg,jfif,png,gif,bmp,webp,avif,heic,heif,tif,tiff,ico,jxl,svg,dng,cr2,cr3,nef,arw,orf,rw2'
  ),
  md: exts('md,markdown'),
  mp4: exts(
    'mp4,mov,wmv,avi,flv,webm,rmvb,mkv,3g2,3gp,asf,divx,f4v,m2ts,m4v,mpe,mpeg,mpg,ogv,vob'
  ),
  pdf: exts('pdf'),
  doc: exts(
    'doc,docx,docm,odt,rtf,pages,wps,epub,mobi,azw,azw3,fb2'
  ),
  pptx: exts('ppt,pptx,pptm,pps,ppsx,odp,key'),
  xlsx: exts('xls,xlsx,xlsm,xlsb,csv,tsv,ods,numbers'),
  xml: exts(
    'adoc,ahk,ahkl,applescript,as,asc,asciidoc,asp,aspx,astro,awk,bash,bat,c,c++,cc,cjs,cmake,cmd,coffee,cpp,cs,css,cts,cxx,dart,diff,dockerfile,erl,es,escript,ex,exs,fs,fsx,glsl,go,gradle,graphql,groovy,h,h++,haml,hpp,htm,html,hs,hxx,ino,ipynb,java,jl,js,json,jsp,jsx,kt,ktm,kts,less,lisp,lua,m4,mjs,mts,ninja,patch,php,pl,pm,pom,proto,ps1,py,r,rb,rs,sass,scala,scaml,scpt,scss,sh,smali,sol,sql,svelte,swift,tex,tf,tfvars,toml,ts,tsx,vb,vba,vbs,vim,vue,xhtml,xml,zig,zsh,/makefile,/.bashrc,/.zshrc'
  ),
  zip: exts(
    'zip,tar,gz,rar,7z,xz,bz,bz2,cab,iso,lz,lz4,lzh,tbz,tbz2,tgz,txz,zipx,zst'
  ),
  apk: exts('apk,aab,apks,xapk'),
}

const fileIconMatcher = createEntryExtMatcher(fileExts)

const dirIcon: IconName = 'folder'
const parentDirIcon: IconName = 'level-up'
const fileFallbackIcon: IconName = 'file'

export function getEntryIcon(entry: Entry): IconName {
  let icon: IconName = fileFallbackIcon
  if (entry.type === 'dir') icon = dirIcon
  if (entry.type === 'file') {
    icon = (fileIconMatcher(entry) as IconName) || fileFallbackIcon
  }
  if (entry.name === '..') icon = parentDirIcon
  return icon
}
