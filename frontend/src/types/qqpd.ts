// ============================================================
// QQPD插件类型定义
// ============================================================

// 基础响应接口
export interface QQPDBaseResponse {
  success: boolean
  message: string
  data?: any
}

// QQPD用户状态
export interface QQPDStatus {
  hash: string
  logged_in: boolean
  status: 'pending' | 'active' | 'expired'
  qq_masked: string
  login_time: string
  expire_time: string
  expires_in_days: number
  channels: string[]
  channel_count: number
  qrcode_base64?: string  // Base64二维码（未登录时有值）
}

// 状态响应
export interface QQPDStatusResponse extends QQPDBaseResponse {
  data: QQPDStatus
}

// 二维码刷新响应
export interface QQPDQRCodeResponse extends QQPDBaseResponse {
  data: {
    qrcode_base64: string
  }
}

// 登录检查响应
export interface QQPDLoginCheckResponse extends QQPDBaseResponse {
  data: {
    login_status: 'waiting' | 'success' | 'expired'
    qq_masked?: string
  }
}

// 频道设置响应
export interface QQPDChannelsResponse extends QQPDBaseResponse {
  data: {
    channels: string[]
    channel_count: number
    invalid_channels: string[]
    guild_ids_cached: number
  }
}

// 测试搜索结果项
export interface QQPDSearchResult {
  unique_id: string
  title: string
  links: {
    type: string
    url: string
    password: string
  }[]
}

// 测试搜索响应
export interface QQPDSearchResponse extends QQPDBaseResponse {
  data: {
    keyword: string
    total_results: number
    channels_searched: string[]
    results: QQPDSearchResult[]
  }
}

// 退出登录响应
export interface QQPDLogoutResponse extends QQPDBaseResponse {
  data: {
    status: string
  }
}

