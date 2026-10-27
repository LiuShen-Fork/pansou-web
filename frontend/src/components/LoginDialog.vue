<template>
  <div v-if="visible" class="login-overlay" @click.self="close">
    <div class="login-dialog">
      <button class="dialog-close" type="button" aria-label="关闭" @click="close">×</button>
      <div class="login-header">
        <h2>🔐 {{ mode === 'register' ? '注册账号' : '登录' }}</h2>
        <p>{{ siteName }}</p>
      </div>

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
          <label>账号或邮箱</label>
          <input v-model="loginForm.account" type="text" placeholder="用户名 / 邮箱" required autocomplete="username" />
        </div>
        <div class="form-group">
          <label>密码</label>
          <input v-model="loginForm.password" type="password" placeholder="请输入密码" required autocomplete="current-password" />
        </div>
        <div v-if="error" class="error-message">{{ error }}</div>
        <div v-if="notice" class="notice-message">{{ notice }}</div>
        <button type="submit" :disabled="loading" class="login-button">
          {{ loading ? '登录中...' : '登录' }}
        </button>
        <p v-if="!config?.auth.email_login_enabled" class="hint">
          普通用户的密码登录已关闭，管理员仍可使用密码登录。
        </p>
      </form>

      <form v-else class="login-form" @submit.prevent="handleRegister">
        <div class="form-group">
          <label>用户名</label>
          <input v-model="registerForm.username" type="text" placeholder="2-32 位字母、数字、下划线" required autocomplete="username" />
        </div>
        <div class="form-group">
          <label>邮箱</label>
          <input v-model="registerForm.email" type="email" placeholder="用于接收验证码" required autocomplete="email" />
        </div>
        <div class="form-group">
          <label>邮箱验证码</label>
          <div class="code-row">
            <input v-model="registerForm.code" type="text" inputmode="numeric" maxlength="6" placeholder="6 位验证码" required />
            <button type="button" class="code-button" :disabled="sendingCode || countdown > 0" @click="handleSendCode">
              {{ countdown > 0 ? `${countdown}s` : (sendingCode ? '发送中...' : '发送验证码') }}
            </button>
          </div>
        </div>
        <div class="form-group">
          <label>密码</label>
          <input v-model="registerForm.password" type="password" placeholder="至少 8 位" required autocomplete="new-password" />
        </div>
        <div v-if="error" class="error-message">{{ error }}</div>
        <div v-if="notice" class="notice-message">{{ notice }}</div>
        <button type="submit" :disabled="loading" class="login-button">
          {{ loading ? '注册中...' : '注册并登录' }}
        </button>
      </form>
    </div>
  </div>
</template>

<script setup lang="ts">
import { computed, reactive, ref, watch } from 'vue';
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
let timer: number | null = null;

const loginForm = reactive({ account: '', password: '' });
const registerForm = reactive({ username: '', email: '', password: '', code: '' });

const config = computed(() => authState.config);
const siteName = computed(() => authState.config?.site.name || '清羽盘搜');
const oauthProvider = computed(() => authState.config?.oauth_providers?.[0] || null);
const canRegister = computed(() => !!authState.config?.auth.email_register_enabled);

watch(
  () => props.visible,
  (value) => {
    if (value) {
      error.value = '';
      notice.value = '';
      loading.value = false;
      loginForm.password = '';
      registerForm.password = '';
      registerForm.code = '';
      if (mode.value === 'register' && !canRegister.value) mode.value = 'login';
    } else if (timer) {
      window.clearInterval(timer);
      timer = null;
      countdown.value = 0;
    }
  }
);

watch(canRegister, (value) => {
  if (!value && mode.value === 'register') mode.value = 'login';
});

const close = () => emit('update:visible', false);

const switchMode = (next: 'login' | 'register') => {
  mode.value = next;
  error.value = '';
  notice.value = '';
};

const startOAuth = (slug: string) => {
  window.location.href = oauthStartUrl(slug, 'login');
};

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
  background: rgba(15, 23, 42, 0.45);
  backdrop-filter: blur(6px);
  display: flex;
  align-items: center;
  justify-content: center;
  z-index: 9999;
  padding: 1rem;
  animation: fadeIn 0.2s ease-out;
}

@keyframes fadeIn {
  from { opacity: 0; }
  to { opacity: 1; }
}

.login-dialog {
  position: relative;
  background: hsl(var(--background-light, 0 0% 100%));
  border-radius: 16px;
  padding: 2rem;
  width: 100%;
  max-width: 420px;
  max-height: 92vh;
  overflow-y: auto;
  box-shadow: 0 24px 70px rgba(15, 23, 42, 0.28);
  border: 1px solid hsl(var(--border));
  animation: slideUp 0.25s ease-out;
}

@keyframes slideUp {
  from { opacity: 0; transform: translateY(16px); }
  to { opacity: 1; transform: translateY(0); }
}

.dialog-close {
  position: absolute;
  top: 0.85rem;
  right: 0.85rem;
  width: 2rem;
  height: 2rem;
  border: 0;
  border-radius: 999px;
  background: transparent;
  color: hsl(var(--muted-foreground));
  font-size: 1.35rem;
  cursor: pointer;
}
.dialog-close:hover { background: hsl(var(--muted)); }

.login-header { text-align: center; margin-bottom: 1.25rem; }
.login-header h2 { font-size: 1.4rem; font-weight: 650; margin-bottom: 0.35rem; }
.login-header p { color: hsl(var(--muted-foreground)); font-size: 0.85rem; }

.auth-tabs {
  display: grid;
  grid-template-columns: 1fr 1fr;
  gap: 0.35rem;
  padding: 0.3rem;
  background: hsl(var(--muted));
  border-radius: 0.75rem;
  margin-bottom: 1.1rem;
}
.auth-tabs button {
  border: 0;
  background: transparent;
  padding: 0.55rem;
  border-radius: 0.55rem;
  color: hsl(var(--muted-foreground));
  font-weight: 600;
  cursor: pointer;
}
.auth-tabs button.active {
  background: hsl(var(--background-light, 0 0% 100%));
  color: hsl(var(--foreground));
  box-shadow: 0 2px 8px rgba(15, 23, 42, 0.08);
}

.oauth-row { display: flex; flex-direction: column; gap: 0.6rem; margin-bottom: 1rem; }
.oauth-button {
  display: flex;
  align-items: center;
  justify-content: center;
  gap: 0.55rem;
  width: 100%;
  padding: 0.7rem 1rem;
  border: 1px solid hsl(var(--border));
  border-radius: 0.65rem;
  background: hsl(var(--background-light, 0 0% 100%));
  color: hsl(var(--foreground));
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
  font-size: 0.75rem;
}
.divider::before, .divider::after {
  content: '';
  flex: 1;
  height: 1px;
  background: hsl(var(--border));
}

.login-form { display: flex; flex-direction: column; gap: 0.9rem; }
.form-group { display: flex; flex-direction: column; gap: 0.4rem; }
.form-group label { font-size: 0.82rem; font-weight: 550; color: hsl(var(--foreground)); }
.form-group input {
  padding: 0.7rem 0.85rem;
  border: 1px solid hsl(var(--border));
  border-radius: 0.55rem;
  background: hsl(var(--background));
  color: hsl(var(--foreground));
  font-size: 0.9rem;
}
.form-group input:focus {
  outline: none;
  border-color: hsl(var(--primary));
  box-shadow: 0 0 0 3px hsl(var(--primary) / 0.12);
}
.code-row { display: flex; gap: 0.5rem; }
.code-row input { flex: 1; min-width: 0; }
.code-button {
  flex: 0 0 auto;
  padding: 0 0.85rem;
  border: 1px solid hsl(var(--border));
  border-radius: 0.55rem;
  background: hsl(var(--muted));
  color: hsl(var(--foreground));
  font-size: 0.8rem;
  cursor: pointer;
  white-space: nowrap;
}
.code-button:disabled { opacity: 0.6; cursor: not-allowed; }

.error-message, .notice-message {
  padding: 0.6rem 0.75rem;
  border-radius: 0.55rem;
  font-size: 0.82rem;
  line-height: 1.45;
}
.error-message { background: #fef2f2; border: 1px solid #fecaca; color: #b91c1c; }
.notice-message { background: #eff6ff; border: 1px solid #bfdbfe; color: #1d4ed8; }

.login-button {
  padding: 0.75rem 1.5rem;
  background: hsl(var(--primary));
  color: hsl(var(--primary-foreground));
  border: none;
  border-radius: 0.65rem;
  font-size: 0.92rem;
  font-weight: 600;
  cursor: pointer;
}
.login-button:disabled { opacity: 0.65; cursor: not-allowed; }
.hint { font-size: 0.75rem; color: hsl(var(--muted-foreground)); text-align: center; line-height: 1.5; }
</style>
