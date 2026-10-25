import { computed, reactive } from 'vue';
import {
  getMe,
  getPublicConfig,
  login as apiLogin,
  logout as apiLogout,
  register as apiRegister,
  sendRegisterCode,
  type APIToken,
  type AuthUser,
  type LoginResponse,
  type MeResponse,
  type PublicConfig,
} from '@/api/auth';

interface AuthState {
  ready: boolean;
  user: AuthUser | null;
  config: PublicConfig | null;
  tokenMeta: APIToken | null;
  oauthAccounts: MeResponse['oauth_accounts'];
  token: string;
}

export const authState = reactive<AuthState>({
  ready: false,
  user: null,
  config: null,
  tokenMeta: null,
  oauthAccounts: [],
  token: localStorage.getItem('auth_token') || '',
});

export const isLoggedIn = computed(() => !!authState.user);
export const isAdmin = computed(() => authState.user?.role === 'admin');
export const mustChangePassword = computed(() => !!authState.user?.must_change_password);

export function storeToken(token: string) {
  authState.token = token;
  localStorage.setItem('auth_token', token);
}

export function clearToken() {
  authState.token = '';
  authState.user = null;
  authState.tokenMeta = null;
  authState.oauthAccounts = [];
  localStorage.removeItem('auth_token');
  localStorage.removeItem('auth_username');
}

export async function loadPublicConfig(): Promise<PublicConfig | null> {
  try {
    authState.config = await getPublicConfig();
  } catch (err) {
    console.warn('读取站点认证配置失败', err);
  }
  return authState.config;
}

export async function refreshMe(): Promise<AuthUser | null> {
  if (!authState.token) {
    authState.ready = true;
    return null;
  }
  try {
    const me = await getMe();
    authState.user = me.user;
    authState.tokenMeta = me.token;
    authState.oauthAccounts = me.oauth_accounts || [];
    if (me.auth) {
      authState.config = authState.config
        ? { ...authState.config, auth: { ...authState.config.auth, ...me.auth } }
        : authState.config;
    }
    localStorage.setItem('auth_username', me.user.username);
    return me.user;
  } catch {
    clearToken();
    return null;
  } finally {
    authState.ready = true;
  }
}

export async function initAuth() {
  await Promise.all([loadPublicConfig(), refreshMe()]);
  authState.ready = true;
}

export async function login(account: string, password: string): Promise<LoginResponse> {
  const res = await apiLogin(account, password);
  storeToken(res.token);
  authState.user = res.user;
  await loadPublicConfig();
  return res;
}

export async function register(username: string, email: string, password: string, code: string): Promise<LoginResponse> {
  const res = await apiRegister(username, email, password, code);
  storeToken(res.token);
  authState.user = res.user;
  await loadPublicConfig();
  return res;
}

export async function sendCode(email: string) {
  return sendRegisterCode(email);
}

export async function logout() {
  try {
    await apiLogout();
  } finally {
    clearToken();
  }
}
