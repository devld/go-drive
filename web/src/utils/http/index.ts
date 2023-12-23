import { createHttp } from './http'
import {
  transformBlobResponse,
  transformErrorResponse,
  transformJSONRequest,
  transformJSONResponse,
  transformTextResponse,
} from './transformers'

export * from './types'
export * from './utils'

const textContentTypes = ['text/', 'application/json', 'application/xml']

export default createHttp({
  transformRequest: [transformJSONRequest],
  transformResponse: [
    transformBlobResponse(textContentTypes),
    transformTextResponse(textContentTypes),
    transformJSONResponse,
    transformErrorResponse,
  ],
})
