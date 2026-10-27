import axios from 'axios';

// 独立实例：启动时探测登录状态不应触发全局 401 弹窗。
export const authApi = axios.create({
  baseURL: '/api',
  timeout: 20000,
});

authApi.interceptors.request.use((config) => {
  const token = localStorage.getItem('auth_token');
  if (token) {
    config.headers.Authorization = `Bearer ${token}`;
  }
  return config;
});

export interface AuthUser {
  id: number;
  username: string;
  email: string;
  role: 'admin' | 'user';
  status: string;
  must_change_password: boolean;
  daily_search_limit: number;
  api_daily_search_limit: number;
  search_count_today?: number;
  api_search_count_today?: number;
  created_at: string;
  updated_at: string;
  last_login_at?: string;
}

export interface PublicAuthConfig {
  email_login_enabled: boolean;
  email_register_enabled: boolean;
  oauth_login_enabled: boolean;
  oauth_register_enabled: boolean;
  api_require_login: boolean;
  web_search_enabled: boolean;
  allow_user_api_tokens: boolean;
  default_daily_search_limit: number;
  smtp_configured?: boolean;
}

export interface PublicProvider {
  slug: string;
  name: string;
}

export interface PublicConfig {
  site: {
    name: string;
    description: string;
    slogan: string;
    icon: string;
    announcement: string;
  };
  auth: PublicAuthConfig;
  oauth_providers: PublicProvider[];
}

export interface AuthSettings extends PublicAuthConfig {
  site_name: string;
  site_description: string;
  site_slogan: string;
  site_icon: string;
  announcement: string;
  anonymous_web_search_limit: number;
  anonymous_daily_limit: number;
  anonymous_global_daily_limit: number;
  session_ttl_hours: number;
}

export interface EmailSettings {
  enabled: boolean;
  host: string;
  port: number;
  username: string;
  password: string;
  from: string;
  from_name: string;
  use_tls: boolean;
}

export interface OAuthProvider {
  id: number;
  slug: string;
  name: string;
  enabled: boolean;
  client_id: string;
  client_secret?: string;
  auth_url: string;
  token_url: string;
  userinfo_url: string;
  scopes: string;
  created_at?: string;
  updated_at?: string;
}

export interface APIToken {
  id: number;
  user_id: number;
  name: string;
  prefix: string;
  created_at: string;
  last_used_at?: string;
}

export interface OAuthAccount {
  provider_id: number;
  provider_slug: string;
  provider_name: string;
  email?: string;
  created_at: string;
}

export interface MeResponse {
  user: AuthUser;
  auth: PublicAuthConfig;
  token: APIToken | null;
  oauth_accounts: OAuthAccount[] | null;
  is_admin: boolean;
}

export interface LoginResponse {
  token: string;
  expires_at: number;
  user: AuthUser;
}

export const getPublicConfig = async (): Promise<PublicConfig> => {
  const { data } = await authApi.get<PublicConfig>('/auth/public-config');
  return data;
};

export const getMe = async (): Promise<MeResponse> => {
  const { data } = await authApi.get<MeResponse>('/auth/me');
  return data;
};

export const login = async (account: string, password: string): Promise<LoginResponse> => {
  const { data } = await authApi.post<LoginResponse>('/auth/login', { account, password });
  return data;
};

export const logout = async (): Promise<void> => {
  await authApi.post('/auth/logout');
};

export const sendRegisterCode = async (email: string) => {
  const { data } = await authApi.post('/auth/register/send-code', { email });
  return data as { ok: boolean; expires_in: number };
};

export const register = async (username: string, email: string, password: string, code: string): Promise<LoginResponse> => {
  const { data } = await authApi.post<LoginResponse>('/auth/register', { username, email, password, code });
  return data;
};

export const changePassword = async (currentPassword: string, newPassword: string) => {
  await authApi.post('/auth/change-password', {
    current_password: currentPassword,
    new_password: newPassword,
  });
};

export const updateProfile = async (username: string, email: string) => {
  const { data } = await authApi.post<{ user: AuthUser }>('/auth/profile', { username, email });
  return data.user;
};

export const listAPITokens = async (): Promise<{ items: APIToken[]; max: number }> => {
  const { data } = await authApi.get<{ items: APIToken[]; max: number }>('/auth/tokens');
  return { items: data.items || [], max: data.max || 10 };
};

export const createAPIToken = async (name: string): Promise<{ token: string; item: APIToken; max: number }> => {
  const { data } = await authApi.post<{ token: string; item: APIToken; max: number }>('/auth/tokens', { name });
  return data;
};

export const deleteAPIToken = async (id: number): Promise<void> => {
  await authApi.delete(`/auth/tokens/${id}`);
};

// 兼容旧调用：返回最早创建的令牌元数据。
export const getAPIToken = async (): Promise<APIToken | null> => {
  const { data } = await authApi.get<{ token: APIToken | null }>('/auth/token');
  return data.token ?? null;
};

export const resetAPIToken = async () => {
  const { data } = await authApi.post<{ token: string; prefix: string; created_at: string }>('/auth/token/reset');
  return data;
};

export const getOAuthBindings = async (): Promise<OAuthAccount[]> => {
  const { data } = await authApi.get<{ items: OAuthAccount[] }>('/auth/oauth/bindings');
  return data.items || [];
};

export const unbindOAuth = async (providerId: number) => {
  await authApi.delete('/auth/oauth/bindings', { params: { provider_id: providerId } });
};

export const oauthStartUrl = (slug: string, mode: 'login' | 'bind' = 'login') =>
  `/api/auth/oauth/${encodeURIComponent(slug)}/start?mode=${mode}`;

// ---------------- 管理员 ----------------

export interface Overview {
  users_total: number;
  users_active: number;
  admins_total: number;
  searches_today: number;
  searches_total: number;
  anonymous_today: number;
  sessions_active: number;
  recent_searches: SearchLog[];
  default_admin_alive: boolean;
}

export interface SearchLog {
  id: number;
  user_id: number;
  username: string;
  source: string;
  keyword: string;
  ip: string;
  created_at: string;
}

export interface AdminSettingsPayload {
  site: { name: string; description: string; slogan: string; icon: string; announcement: string };
  auth: AuthSettings;
  email: EmailSettings;
}

export interface UpstreamSyncSettings {
  enabled: boolean;
  interval_hours: number;
  repo: string;
  ref: string;
}

export interface UpstreamSyncStatus {
  last_check_at: string;
  last_success_at: string;
  last_revision: string;
  last_error: string;
  channel_count: number;
  plugin_count: number;
  channels_changed: boolean;
  plugins_changed: boolean;
  restart_required: boolean;
}

export interface UpstreamSyncInfo {
  settings: UpstreamSyncSettings;
  status: UpstreamSyncStatus;
  running: boolean;
  next_check_at: string;
  channels: string[];
  enabled_plugins: string[];
  defaults_env_path: string;
}

export const adminGetOverview = async (): Promise<Overview> => {
  const { data } = await authApi.get<Overview>('/admin/overview');
  return data;
};

export const adminListUsers = async (q = '', page = 1, pageSize = 20, role = '', status = '') => {
  const { data } = await authApi.get('/admin/users', {
    params: { q, page, page_size: pageSize, role, status },
  });
  return data as { items: AuthUser[]; total: number; page: number; page_size: number };
};

export interface DailyStat {
  day: string;
  searches: number;
  new_users: number;
}

export interface TopUser {
  username: string;
  searches: number;
}

export interface StatsResponse {
  days: number;
  daily: DailyStat[];
  top_users: TopUser[];
}

export const adminGetStats = async (days = 14): Promise<StatsResponse> => {
  const { data } = await authApi.get<StatsResponse>('/admin/stats', { params: { days } });
  return data;
};

export const adminCreateUser = async (payload: {
  username: string;
  email: string;
  password: string;
  role: string;
  daily_search_limit: number;
}) => {
  const { data } = await authApi.post('/admin/users', payload);
  return data.user as AuthUser;
};

export const adminUpdateUser = async (id: number, payload: Record<string, unknown>) => {
  const { data } = await authApi.patch(`/admin/users/${id}`, payload);
  return data.user as AuthUser;
};

export const adminDeleteUser = async (id: number) => {
  await authApi.delete(`/admin/users/${id}`);
};

export const adminGetSettings = async (): Promise<AdminSettingsPayload & {
  email_password_set: boolean;
  oauth_providers: OAuthProvider[];
}> => {
  const { data } = await authApi.get('/admin/settings');
  return data;
};

export const adminSaveSettings = async (payload: AdminSettingsPayload) => {
  await authApi.put('/admin/settings', payload);
};

export const adminTestEmail = async (to: string) => {
  await authApi.post('/admin/settings/email/test', { to });
};

export interface AdminOAuthProviderState {
  provider: OAuthProvider | null;
  redirect_uri: string;
  secret_set: boolean;
}

export const adminGetOAuthProvider = async (): Promise<AdminOAuthProviderState> => {
  const { data } = await authApi.get<AdminOAuthProviderState>('/admin/oauth-provider');
  return {
    provider: data.provider ?? null,
    redirect_uri: data.redirect_uri || '',
    secret_set: !!data.secret_set,
  };
};

export const adminSaveOAuthProviderSingle = async (provider: OAuthProvider): Promise<OAuthProvider> => {
  const { data } = await authApi.put<{ provider: OAuthProvider }>('/admin/oauth-provider', provider);
  return data.provider;
};

export const adminListOAuthProviders = async (): Promise<OAuthProvider[]> => {
  const { data } = await authApi.get<{ items: OAuthProvider[] }>('/admin/oauth-providers');
  return data.items || [];
};

export const adminSaveOAuthProvider = async (provider: OAuthProvider): Promise<OAuthProvider> => {
  const method = provider.id ? 'put' : 'post';
  const { data } = await authApi.request({ method, url: '/admin/oauth-providers', data: provider });
  return data.provider as OAuthProvider;
};

export const adminDeleteOAuthProvider = async (id: number) => {
  await authApi.delete(`/admin/oauth-providers/${id}`);
};

export const adminListSearchLogs = async (q = '', page = 1, pageSize = 20, source = '') => {
  const { data } = await authApi.get('/admin/search-logs', {
    params: { q, page, page_size: pageSize, source },
  });
  return data as { items: SearchLog[]; total: number; page: number; page_size: number };
};

export const adminGetUpstreamSync = async (): Promise<UpstreamSyncInfo> => {
  const { data } = await authApi.get<UpstreamSyncInfo>('/admin/upstream-sync');
  return data;
};

export const adminSaveUpstreamSync = async (settings: UpstreamSyncSettings) => {
  await authApi.put('/admin/upstream-sync', settings);
};

export const adminRunUpstreamSync = async (): Promise<UpstreamSyncStatus> => {
  const { data } = await authApi.post<{ ok: boolean; status: UpstreamSyncStatus }>('/admin/upstream-sync/run');
  return data.status;
};
