export interface WeiboBaseResponse {
  success: boolean
  message: string
  data?: any
}

export interface WeiboStatus {
  hash: string
  logged_in: boolean
  status: 'pending' | 'active' | 'expired'
  uid: string
  login_time: string
  expire_time: string
  expires_in_days: number
  user_ids: string[]
  user_id_count: number
  qrcode_base64?: string
}

export interface WeiboStatusResponse extends WeiboBaseResponse {
  data: WeiboStatus
}

export interface WeiboQRCodeResponse extends WeiboBaseResponse {
  data: {
    qrcode_base64: string
  }
}

export interface WeiboLoginCheckResponse extends WeiboBaseResponse {
  data: {
    login_status: 'waiting' | 'success' | 'expired'
    uid?: string
  }
}

export interface WeiboUserIdsResponse extends WeiboBaseResponse {
  data: {
    user_ids: string[]
    user_id_count: number
    invalid_user_ids: string[]
  }
}

export interface WeiboSearchResult {
  unique_id: string
  title: string
  links: {
    type: string
    url: string
    password: string
  }[]
}

export interface WeiboSearchResponse extends WeiboBaseResponse {
  data: {
    keyword: string
    total_results: number
    user_ids_searched: string[]
    results: WeiboSearchResult[]
  }
}

export interface WeiboLogoutResponse extends WeiboBaseResponse {
  data: {
    status: string
  }
}
