import axios, { type AxiosInstance, type AxiosRequestConfig } from 'axios'

import {
  ApiContractError,
  isAuthenticationCode,
  isSuccessCode,
  parseResult,
  toLegacyResult,
  type ApiResponse
} from '@stellar-beacon/api-contract'

export interface ApiClientOptions {
  baseURL?: string
  timeout?: number
  getToken?: () => string | null | undefined
  onUnauthorized?: () => void
  /** Keep the old response shape for the unchanged blog presentation layer. */
  legacyResponse?: boolean
  /** When false, business failures remain resolved Axios responses. */
  rejectBusinessErrors?: boolean
}

export const API_BASE_URL = '/api/v1'

export function createApiClient(options: ApiClientOptions = {}): AxiosInstance {
  const client = axios.create({
    baseURL: options.baseURL ?? API_BASE_URL,
    timeout: options.timeout ?? 15_000,
    headers: { Accept: 'application/json' }
  })
  const rejectBusinessErrors = options.rejectBusinessErrors ?? true

  client.interceptors.request.use((config) => {
    const token = options.getToken?.()
    if (token) {
      config.headers = config.headers || {}
      config.headers.Authorization = `Bearer ${token}`
    }
    return config
  })

  client.interceptors.response.use(
    (response) => {
      const result = parseResult(response.data)
      if (!isSuccessCode(result.code) && isAuthenticationCode(result.code)) options.onUnauthorized?.()
      if (!isSuccessCode(result.code) && rejectBusinessErrors) {
        return Promise.reject(withResponse(new ApiContractError(result.message, result.code), response))
      }
      response.data = (options.legacyResponse ? toLegacyResult(result) : result) as never
      return response
    },
    (error) => {
      if (error.response) {
        const result = parseResult(error.response.data)
        error.response.data = (options.legacyResponse ? toLegacyResult(result) : result) as never
        if (error.response.status === 401 || isAuthenticationCode(result.code)) options.onUnauthorized?.()
      }
      return Promise.reject(error)
    }
  )

  return client
}

function withResponse(error: ApiContractError, response: unknown): ApiContractError & { response: unknown } {
  Object.assign(error, { response })
  return error as ApiContractError & { response: unknown }
}

export function isApiResponse<T = unknown>(value: unknown): value is ApiResponse<T> {
  return parseResult<T>(value).code !== 'MALFORMED_RESPONSE'
}

export type { AxiosInstance, AxiosRequestConfig }
