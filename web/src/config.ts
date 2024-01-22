export const EXPLORER_PATH_BASE = '/_'

export const TEXT_EDITOR_MAX_FILE_SIZE = 128 * 1024 // 128kb

export const DEFAULT_TEXT_FILE_EXTS =
  'txt,md,xml,html,css,scss,js,json,jsx,ts,properties,yml,yaml,ini,c,h,cpp,go,java,kt,gradle,ps1'

export const DEFAULT_IMAGE_FILE_EXTS = 'jpg,jpeg,png,gif'

export const DEFAULT_AUDIO_FILE_EXTS = 'mp3,m4a,flac'

export const DEFAULT_VIDEO_FILE_EXTS = 'mp4,ogg'

export const DEFAULT_EXTERNAL_FILE_PREVIEWERS = `
# lines starting with # are comments and will be ignored
# <extensions list> <url template> <name>
pdf pdf.js/web/viewer.html?file={URL} PDF Viewer

# uncomment the next two lines to enable the Office files preview
#docx,doc,xlsx,xls,pptx,ppt https://view.officeapps.live.com/op/embed.aspx?src={URL} Microsoft
#docx,doc,xlsx,xls,pptx,ppt https://docs.google.com/gview?embedded=true&url={URL} Google
`.trim()

export const FILE_BUCKET_SECRET_KEY = 't'
