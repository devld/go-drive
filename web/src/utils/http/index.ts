import Axios, { AxiosResponse } from 'axios'
import { wrapAxios } from './utils'

export * from './types'
export * from './utils'

const axios = Axios.create()

export default wrapAxios<AxiosResponse<any>>(axios)
