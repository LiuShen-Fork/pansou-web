<template>
  <div v-if="visible" class="login-overlay" @click.self="close">
    <div class="login-dialog" role="dialog" aria-modal="true" aria-labelledby="login-title">
      <div class="dialog-accent" aria-hidden="true" />
      <button class="dialog-close" type="button" aria-label="关闭" @click="close">
        <svg viewBox="0 0 24 24" aria-hidden="true"><path d="M6 6l12 12M18 6 6 18" /></svg>
      </button>

      <header class="login-header">
        <div class="login-brand">
          <img :src="siteIcon" :alt="siteName" @error="onSiteIconError" />
        </div>
        <span class="login-eyebrow">{{ mode === 'register' ? 'CREATE ACCOUNT' : 'SIGN IN' }}</span>
        <h2 id="login-title">{{ mode === 'register' ? '创建账号' : '欢迎回来' }}</h2>
        <p>{{ mode === 'register' ? `注册 ${siteName}，开始使用完整搜索` : `登录 ${siteName}，继续使用完整搜索` }}</p>
      </header>

      <div v-if="canRegister" class="auth-tabs">
        <button type="button" :class="{ active: mode === 'login' }" @click="switchMode('login')">登录</button>
        <button type="button" :class="{ active: mode === 'register' }" @click="switchMode('register')">注册</button>
      </div>

      <div v-if="oauthProvider && (config?.auth.oauth_login_enabled || config?.auth.oauth_register_enabled)" class="oauth-row">
        <button type="button" class="oauth-button" @click="startOAuth(oauthProvider.slug)">
          <span class="oauth-avatar">{{ oauthProvider.name.slice(0, 1).toUpperCase() }}</span>
          <span>使用 {{ oauthProvider.name }} {{ mode === 'register' ? '注册 / 登录' : '登录' }}</span>
        </button>
        <div class="divider"><span>或使用账号密码</span></div>
      </div>

      <form v-if="mode === 'login'" class="login-form" @submit.prevent="handleLogin">
        <div class="form-group">
          <label for="login-account">账号或邮箱</label>
          <input
            id="login-account"
            v-model="loginForm.account"
            type="text"
            placeholder="用户名 / 邮箱"
            required
            autocomplete="username"
            spellcheck="false"
          />
        </div>
        <div class="form-group">
          <label for="login-password">密码</label>
          <div class="password-field">
            <input
              id="login-password"
              v-model="loginForm.password"
              :type="showLoginPassword ? 'text' : 'password'"
              placeholder="请输入密码"
              required
              autocomplete="current-password"
            />
            <button
              class="password-toggle"
              type="button"
              :aria-label="showLoginPassword ? '隐藏密码' : '显示密码'"
              @click="showLoginPassword = !showLoginPassword"
            >
              <svg v-if="!showLoginPassword" viewBox="0 0 24 24" aria-hidden="true">
                <path d="M2 12s3.5-6 10-6 10 6 10 6-3.5 6-10 6S2 12 2 12Z" />
                <circle cx="12" cy="12" r="2.6" />
              </svg>
              <svg v-else viewBox="0 0 24 24" aria-hidden="true">
                <path d="M3 3l18 18M10.6 6.2A10.9 10.9 0 0 1 12 6c6.5 0 10 6 10 6a17 17 0 0 1-3.2 3.7M6.1 6.1A16.7 16.7 0 0 0 2 12s3.5 6 10 6a10.7 10.7 0 0 0 4.3-.9" />
              </svg>
            </button>
          </div>
        </div>
        <div v-if="error" class="form-message error-message">{{ error }}</div>
        <div v-if="notice" class="form-message notice-message">{{ notice }}</div>
        <button type="submit" :disabled="loading" class="login-button">
          <span v-if="loading" class="btn-spinner" aria-hidden="true" />
          {{ loading ? '登录中' : '登录' }}
        </button>
        <p v-if="!config?.auth.email_login_enabled" class="hint">
          普通用户的密码登录已关闭，管理员仍可使用密码登录。
        </p>
      </form>

      <form v-else class="login-form" @submit.prevent="handleRegister">
        <div class="form-group">
          <label for="register-username">用户名</label>
          <input
            id="register-username"
            v-model="registerForm.username"
            type="text"
            placeholder="2-32 位字母、数字、下划线"
            required
            autocomplete="username"
            spellcheck="false"
          />
        </div>
        <div class="form-group">
          <label for="register-email">邮箱</label>
          <input
            id="register-email"
            v-model="registerForm.email"
            type="email"
            placeholder="用于接收注册验证码"
            required
            autocomplete="email"
          />
        </div>
        <div class="form-group">
          <label for="register-code">邮箱验证码</label>
          <div class="code-row">
            <input
              id="register-code"
              v-model="registerForm.code"
              type="text"
              inputmode="numeric"
              maxlength="6"
              placeholder="6 位验证码"
              required
              autocomplete="one-time-code"
            />
            <button type="button" class="code-button" :disabled="sendingCode || countdown > 0" @click="handleSendCode">
              {{ countdown > 0 ? `${countdown}s` : (sendingCode ? '发送中' : '发送验证码') }}
            </button>
          </div>
        </div>
        <div class="form-group">
          <label for="register-password">密码</label>
          <div class="password-field">
            <input
              id="register-password"
              v-model="registerForm.password"
              :type="showRegisterPassword ? 'text' : 'password'"
              placeholder="至少 8 位"
              required
              autocomplete="new-password"
            />
            <button
              class="password-toggle"
              type="button"
              :aria-label="showRegisterPassword ? '隐藏密码' : '显示密码'"
              @click="showRegisterPassword = !showRegisterPassword"
            >
              <svg v-if="!showRegisterPassword" viewBox="0 0 24 24" aria-hidden="true">
                <path d="M2 12s3.5-6 10-6 10 6 10 6-3.5 6-10 6S2 12 2 12Z" />
                <circle cx="12" cy="12" r="2.6" />
              </svg>
              <svg v-else viewBox="0 0 24 24" aria-hidden="true">
                <path d="M3 3l18 18M10.6 6.2A10.9 10.9 0 0 1 12 6c6.5 0 10 6 10 6a17 17 0 0 1-3.2 3.7M6.1 6.1A16.7 16.7 0 0 0 2 12s3.5 6 10 6a10.7 10.7 0 0 0 4.3-.9" />
              </svg>
            </button>
          </div>
        </div>
        <div v-if="error" class="form-message error-message">{{ error }}</div>
        <div v-if="notice" class="form-message notice-message">{{ notice }}</div>
        <button type="submit" :disabled="loading" class="login-button">
          <span v-if="loading" class="btn-spinner" aria-hidden="true" />
          {{ loading ? '注册中' : '注册并登录' }}
        </button>
      </form>

      <footer class="login-footer">
        <span>{{ siteName }}</span>
        <span>安全登录 · 账号信息仅用于本站</span>
      </footer>
    </div>
  </div>
</template>

<script setup lang="ts">
import { computed, onBeforeUnmount, reactive, ref, watch } from 'vue';
import { oauthStartUrl } from '@/api/auth';
import { authState, login, register, sendCode } from '@/stores/auth';

const props = defineProps<{ visible: boolean }>();
const emit = defineEmits<{
  'update:visible': [value: boolean];
  success: [];
}>();

const mode = ref<'login' | 'register'>('login');
const loading = ref(false);
const sendingCode = ref(false);
const countdown = ref(0);
const error = ref('');
const notice = ref('');
const showLoginPassword = ref(false);
const showRegisterPassword = ref(false);
let timer: number | null = null;

const loginForm = reactive({ account: '', password: '' });
const registerForm = reactive({ username: '', email: '', password: '', code: '' });

const config = computed(() => authState.config);
const siteName = computed(() => authState.config?.site.name || '清羽盘搜');
const siteIcon = computed(() => authState.config?.site.icon || '/favicon.ico');
const oauthProvider = computed(() => authState.config?.oauth_providers?.[0] || null);
const canRegister = computed(() => !!authState.config?.auth.email_register_enabled);

// 弹窗打开时锁定页面滚动，避免背景跟着滚轮移动或滚动条在遮罩旁闪烁。
let bodyOverflow = '';
let bodyPaddingRight = '';
let scrollLocked = false;

const lockBodyScroll = () => {
  if (scrollLocked) return;
  const scrollbarWidth = window.innerWidth - document.documentElement.clientWidth;
  bodyOverflow = document.body.style.overflow;
  bodyPaddingRight = document.body.style.paddingRight;
  if (scrollbarWidth > 0) {
    const currentPadding = parseFloat(window.getComputedStyle(document.body).paddingRight) || 0;
    document.body.style.paddingRight = `${currentPadding + scrollbarWidth}px`;
  }
  document.body.style.overflow = 'hidden';
  scrollLocked = true;
};

const unlockBodyScroll = () => {
  if (!scrollLocked) return;
  document.body.style.overflow = bodyOverflow;
  document.body.style.paddingRight = bodyPaddingRight;
  scrollLocked = false;
};

watch(
  () => props.visible,
  (value) => {
    if (value) {
      error.value = '';
      notice.value = '';
      loading.value = false;
      showLoginPassword.value = false;
      showRegisterPassword.value = false;
      loginForm.password = '';
      registerForm.password = '';
      registerForm.code = '';
      if (mode.value === 'register' && !canRegister.value) mode.value = 'login';
      lockBodyScroll();
    } else {
      unlockBodyScroll();
      if (timer) {
        window.clearInterval(timer);
        timer = null;
        countdown.value = 0;
      }
    }
  }
);

onBeforeUnmount(unlockBodyScroll);

watch(canRegister, (value) => {
  if (!value && mode.value === 'register') mode.value = 'login';
});

const close = () => emit('update:visible', false);

const switchMode = (next: 'login' | 'register') => {
  mode.value = next;
  error.value = '';
  notice.value = '';
  showLoginPassword.value = false;
  showRegisterPassword.value = false;
};

const startOAuth = (slug: string) => {
  window.location.href = oauthStartUrl(slug, 'login');
};

function onSiteIconError(event: Event) {
  const img = event.target as HTMLImageElement;
  if (!img.src.endsWith('/favicon.ico')) img.src = '/favicon.ico';
}

const startCountdown = () => {
  countdown.value = 60;
  if (timer) window.clearInterval(timer);
  timer = window.setInterval(() => {
    countdown.value -= 1;
    if (countdown.value <= 0 && timer) {
      window.clearInterval(timer);
      timer = null;
    }
  }, 1000);
};

const handleSendCode = async () => {
  error.value = '';
  notice.value = '';
  if (!registerForm.email.trim()) {
    error.value = '请先填写邮箱';
    return;
  }
  sendingCode.value = true;
  try {
    await sendCode(registerForm.email.trim());
    notice.value = '验证码已发送，请查收邮箱（10 分钟内有效）';
    startCountdown();
  } catch (err: any) {
    error.value = err.response?.data?.error || '验证码发送失败';
  } finally {
    sendingCode.value = false;
  }
};

const handleLogin = async () => {
  error.value = '';
  notice.value = '';
  loading.value = true;
  try {
    await login(loginForm.account.trim(), loginForm.password);
    emit('update:visible', false);
    emit('success');
  } catch (err: any) {
    error.value = err.response?.data?.error || '登录失败，请检查账号密码';
  } finally {
    loading.value = false;
  }
};

const handleRegister = async () => {
  error.value = '';
  notice.value = '';
  loading.value = true;
  try {
    await register(
      registerForm.username.trim(),
      registerForm.email.trim(),
      registerForm.password,
      registerForm.code.trim()
    );
    emit('update:visible', false);
    emit('success');
  } catch (err: any) {
    error.value = err.response?.data?.error || '注册失败';
  } finally {
    loading.value = false;
  }
};
</script>

<style scoped>
.login-overlay {
  position: fixed;
  inset: 0;
  z-index: 9999;
  display: flex;
  align-items: center;
  justify-content: center;
  padding: 1rem;
  overflow: hidden;
  overscroll-behavior: contain;
  background: rgba(15, 23, 42, 0.48);
  backdrop-filter: blur(8px);
  animation: fadeIn 0.18s ease-out;
}

@keyframes fadeIn {
  from { opacity: 0; }
  to { opacity: 1; }
}

.login-dialog {
  position: relative;
  width: 100%;
  max-width: 430px;
  max-height: 92vh;
  overflow-x: hidden;
  overflow-y: auto;
  overscroll-behavior: contain;
  scrollbar-width: thin;
  scrollbar-color: hsl(var(--border-hover)) transparent;
  padding: 1.6rem 1.7rem 1.35rem;
  border: 1px solid hsl(var(--border));
  border-radius: 1.1rem;
  background: hsl(var(--background-light, 0 0% 100%));
  box-shadow: 0 30px 80px rgba(15, 23, 42, 0.3);
  animation: dialogIn 0.2s cubic-bezier(0.22, 1, 0.36, 1);
}

.login-dialog::-webkit-scrollbar {
  width: 6px;
}
.login-dialog::-webkit-scrollbar-track {
  background: transparent;
}
.login-dialog::-webkit-scrollbar-thumb {
  border-radius: 999px;
  background: hsl(var(--border-hover));
}

@keyframes dialogIn {
  from { transform: scale(0.985); }
  to { transform: scale(1); }
}

.dialog-accent {
  position: absolute;
  top: 0;
  left: 1.7rem;
  right: 1.7rem;
  height: 2px;
  border-radius: 0 0 999px 999px;
  background: linear-gradient(90deg, transparent, hsl(var(--primary) / 0.7), transparent);
}

.dialog-close {
  position: absolute;
  top: 0.8rem;
  right: 0.8rem;
  display: grid;
  place-items: center;
  width: 1.9rem;
  height: 1.9rem;
  border: 0;
  border-radius: 999px;
  background: transparent;
  color: hsl(var(--muted-foreground));
  cursor: pointer;
  transition: background 0.15s ease, color 0.15s ease;
}
.dialog-close:hover { background: hsl(var(--muted)); color: hsl(var(--foreground)); }
.dialog-close svg { width: 0.95rem; height: 0.95rem; fill: none; stroke: currentColor; stroke-width: 2; stroke-linecap: round; }

.login-header { text-align: center; margin-bottom: 1.1rem; }
.login-brand {
  display: grid;
  place-items: center;
  width: 3.2rem;
  height: 3.2rem;
  margin: 0 auto 0.65rem;
  border: 1px solid hsl(var(--border));
  border-radius: 0.9rem;
  background: #fff;
  box-shadow: 0 6px 18px rgba(15, 23, 42, 0.1);
  overflow: hidden;
}
.login-brand img { width: 2.1rem; height: 2.1rem; object-fit: contain; }
.login-eyebrow {
  display: block;
  font-size: 0.58rem;
  font-weight: 750;
  letter-spacing: 0.18em;
  color: hsl(var(--primary));
}
.login-header h2 {
  margin: 0.18rem 0 0;
  font-size: 1.32rem;
  font-weight: 750;
  letter-spacing: -0.02em;
}
.login-header p {
  margin-top: 0.25rem;
  font-size: 0.76rem;
  line-height: 1.55;
  color: hsl(var(--muted-foreground));
}

.auth-tabs {
  display: grid;
  grid-template-columns: 1fr 1fr;
  gap: 0.25rem;
  padding: 0.25rem;
  margin-bottom: 1rem;
  border-radius: 0.7rem;
  background: hsl(var(--muted));
}
.auth-tabs button {
  border: 0;
  border-radius: 0.55rem;
  padding: 0.5rem;
  background: transparent;
  color: hsl(var(--muted-foreground));
  font-size: 0.8rem;
  font-weight: 600;
  cursor: pointer;
  transition: background 0.15s ease, color 0.15s ease, box-shadow 0.15s ease;
}
.auth-tabs button.active {
  background: hsl(var(--background-light, 0 0% 100%));
  color: hsl(var(--foreground));
  box-shadow: 0 2px 8px rgba(15, 23, 42, 0.08);
}

.oauth-row { display: flex; flex-direction: column; gap: 0.7rem; margin-bottom: 1rem; }
.oauth-button {
  display: flex;
  align-items: center;
  justify-content: center;
  gap: 0.55rem;
  width: 100%;
  padding: 0.68rem 1rem;
  border: 1px solid hsl(var(--border));
  border-radius: 0.65rem;
  background: hsl(var(--background-light, 0 0% 100%));
  color: hsl(var(--foreground));
  font-size: 0.82rem;
  font-weight: 550;
  cursor: pointer;
  transition: border-color 0.15s ease, color 0.15s ease, background 0.15s ease;
}
.oauth-button:hover { border-color: hsl(var(--primary)); color: hsl(var(--primary)); background: hsl(var(--primary) / 0.05); }
.oauth-avatar {
  display: grid;
  place-items: center;
  width: 1.55rem;
  height: 1.55rem;
  flex: 0 0 1.55rem;
  border-radius: 999px;
  background: hsl(var(--primary) / 0.12);
  color: hsl(var(--primary));
  font-size: 0.68rem;
  font-weight: 750;
}
.divider {
  display: flex;
  align-items: center;
  gap: 0.7rem;
  color: hsl(var(--muted-foreground));
  font-size: 0.7rem;
}
.divider::before, .divider::after {
  content: '';
  flex: 1;
  height: 1px;
  background: hsl(var(--border));
}

.login-form { display: flex; flex-direction: column; gap: 0.8rem; }
.form-group { display: flex; flex-direction: column; gap: 0.35rem; }
.form-group label { font-size: 0.76rem; font-weight: 600; color: hsl(var(--foreground)); }
.form-group input {
  width: 100%;
  padding: 0.65rem 0.75rem;
  border: 1px solid hsl(var(--border));
  border-radius: 0.55rem;
  background: hsl(var(--background));
  color: hsl(var(--foreground));
  font-size: 0.85rem;
  transition: border-color 0.15s ease, box-shadow 0.15s ease, background 0.15s ease;
}
.form-group input::placeholder { color: hsl(var(--muted-foreground) / 0.75); }
.form-group input:focus {
  outline: none;
  border-color: hsl(var(--primary) / 0.65);
  background: hsl(var(--background-light, 0 0% 100%));
  box-shadow: 0 0 0 3px hsl(var(--primary) / 0.12);
}

.password-field { position: relative; }
.password-field input { padding-right: 2.5rem; }
.password-toggle {
  position: absolute;
  top: 50%;
  right: 0.35rem;
  display: grid;
  place-items: center;
  width: 1.9rem;
  height: 1.9rem;
  border: 0;
  border-radius: 0.4rem;
  background: transparent;
  color: hsl(var(--muted-foreground));
  cursor: pointer;
  transform: translateY(-50%);
}
.password-toggle:hover { background: hsl(var(--muted)); color: hsl(var(--foreground)); }
.password-toggle svg { width: 1rem; height: 1rem; fill: none; stroke: currentColor; stroke-width: 1.8; stroke-linecap: round; stroke-linejoin: round; }

.code-row { display: flex; gap: 0.45rem; }
.code-row input { flex: 1; min-width: 0; }
.code-button {
  flex: 0 0 auto;
  padding: 0 0.8rem;
  border: 1px solid hsl(var(--border));
  border-radius: 0.55rem;
  background: hsl(var(--muted));
  color: hsl(var(--foreground));
  font-size: 0.75rem;
  font-weight: 550;
  cursor: pointer;
  white-space: nowrap;
}
.code-button:hover:not(:disabled) { border-color: hsl(var(--primary) / 0.4); color: hsl(var(--primary)); }
.code-button:disabled { opacity: 0.6; cursor: not-allowed; }

.form-message {
  padding: 0.55rem 0.7rem;
  border-radius: 0.5rem;
  font-size: 0.76rem;
  line-height: 1.5;
}
.error-message {
  border: 1px solid hsl(var(--destructive) / 0.3);
  background: hsl(var(--destructive) / 0.08);
  color: hsl(var(--destructive));
}
.notice-message {
  border: 1px solid hsl(var(--primary) / 0.28);
  background: hsl(var(--primary) / 0.08);
  color: hsl(var(--primary));
}

.login-button {
  display: inline-flex;
  align-items: center;
  justify-content: center;
  gap: 0.45rem;
  width: 100%;
  margin-top: 0.15rem;
  padding: 0.72rem 1.5rem;
  border: 0;
  border-radius: 0.6rem;
  background: hsl(var(--primary));
  color: hsl(var(--primary-foreground));
  font-size: 0.86rem;
  font-weight: 650;
  cursor: pointer;
  box-shadow: 0 8px 18px hsl(var(--primary) / 0.22);
  transition: filter 0.15s ease, transform 0.15s ease, box-shadow 0.15s ease;
}
.login-button:hover:not(:disabled) { filter: brightness(1.05); transform: translateY(-1px); box-shadow: 0 10px 22px hsl(var(--primary) / 0.28); }
.login-button:disabled { opacity: 0.68; cursor: not-allowed; box-shadow: none; }
.btn-spinner {
  width: 0.9rem;
  height: 0.9rem;
  border: 2px solid hsl(var(--primary-foreground) / 0.35);
  border-top-color: hsl(var(--primary-foreground));
  border-radius: 50%;
  animation: spin 0.8s linear infinite;
}
@keyframes spin { to { transform: rotate(360deg); } }

.hint { font-size: 0.7rem; color: hsl(var(--muted-foreground)); text-align: center; line-height: 1.5; }

.login-footer {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 0.6rem;
  flex-wrap: wrap;
  margin-top: 1.15rem;
  padding-top: 0.75rem;
  border-top: 1px dashed hsl(var(--border) / 0.8);
  font-size: 0.62rem;
  color: hsl(var(--muted-foreground));
}

@media (max-width: 560px) {
  .login-overlay { padding: 0.7rem; align-items: flex-end; }
  .login-dialog { padding: 1.35rem 1.15rem 1.1rem; border-radius: 1rem; max-height: 90vh; }
  .dialog-accent { left: 1.15rem; right: 1.15rem; }
  .login-brand { width: 2.8rem; height: 2.8rem; }
  .login-brand img { width: 1.85rem; height: 1.85rem; }
  .login-header h2 { font-size: 1.18rem; }
  .login-footer { justify-content: center; text-align: center; }
}
</style>
