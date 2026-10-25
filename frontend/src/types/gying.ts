// ============================================================
// Gying插件类型定义
// ============================================================

// 基础响应接口
export interface GyingBaseResponse {
  success: boolean
  message: string
  data?: any
}

// Gying用户状态
export interface GyingStatus {
  hash: string
  logged_in: boolean
  status: 'pending' | 'active' | 'expired'
  username: string
  login_time: string
  expire_time: string
  expires_in_days: number
}

// 状态响应
export interface GyingStatusResponse extends GyingBaseResponse {
  data: GyingStatus
}

// 站点配置响应
export interface GyingConfigResponse extends GyingBaseResponse {
  data: {
    base_url: string
  }
}

// 登录响应
export interface GyingLoginResponse extends GyingBaseResponse {
  data: {
    status: string
    username: string
  }
}

// 测试搜索结果项
export interface GyingSearchResult {
  unique_id: string
  title: string
  links: {
    type: string
    url: string
    password: string
  }[]
}

// 测试搜索响应
export interface GyingSearchResponse extends GyingBaseResponse {
  data: {
    keyword: string
    total_results: number
    results: GyingSearchResult[]
  }
}

// 退出登录响应
export interface GyingLogoutResponse extends GyingBaseResponse {
  data: {
    status: string
  }
}
