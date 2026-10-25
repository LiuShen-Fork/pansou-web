export const DEFAULT_GYING_BASE_URL = 'https://www.gying.net'
export const GYING_BASE_URL_STORAGE_KEY = 'gying_base_url'

const normalizeStoredBaseURL = (baseURL: string | null | undefined) => {
  const value = baseURL?.trim()
  return value || DEFAULT_GYING_BASE_URL
}

export const getStoredGyingBaseURL = () => {
  if (typeof window === 'undefined') {
    return DEFAULT_GYING_BASE_URL
  }

  return normalizeStoredBaseURL(window.localStorage.getItem(GYING_BASE_URL_STORAGE_KEY))
}

export const saveStoredGyingBaseURL = (baseURL: string) => {
  if (typeof window === 'undefined') {
    return
  }

  const normalizedBaseURL = normalizeStoredBaseURL(baseURL)
  window.localStorage.setItem(GYING_BASE_URL_STORAGE_KEY, normalizedBaseURL)
  window.dispatchEvent(new StorageEvent('storage', { key: GYING_BASE_URL_STORAGE_KEY }))
}
