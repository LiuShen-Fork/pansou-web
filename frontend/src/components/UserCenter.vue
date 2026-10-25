<template>
  <div class="user-center">
    <div class="page-head">
      <div>
        <p class="eyebrow">账户中心</p>
        <h2>{{ user?.username || '未登录' }}</h2>
        <p class="sub">
          {{ user?.role === 'admin' ? '管理员' : '普通用户' }} ·
          今日搜索 {{ user?.search_count_today || 0 }}{{ limitText }}
        </p>
      </div>
      <button class="ghost-button" type="button" @click="$emit('back')">← 返回搜索</button>
    </div>

    <div v-if="user?.must_change_password" class="alert warn">
      当前仍在使用默认密码，请立即修改后再使用管理功能。
    </div>
    <div v-if="message" class="alert ok">{{ message }}</div>
    <div v-if="error" class="alert error">{{ error }}</div>

    <div class="grid">
      <section class="panel">
        <h3>个人资料</h3>
        <label>用户名<input v-model="profile.username" type="text" maxlength="32" /></label>
        <label>邮箱<input v-model="profile.email" type="email" placeholder="可选" /></label>
        <button class="primary-button" :disabled="savingProfile" @click="saveProfile">
          {{ savingProfile ? '保存中...' : '保存资料' }}
        </button>
      </section>

      <section class="panel">
        <h3>修改密码</h3>
        <label>当前密码<input v-model="password.current" type="password" placeholder="OAuth 注册的账号可留空" /></label>
        <label>新密码<input v-model="password.next" type="password" placeholder="至少 8 位" /></label>
        <button class="primary-button" :disabled="savingPassword" @click="savePassword">
          {{ savingPassword ? '提交中...' : '更新密码' }}
        </button>
      </section>

      <section class="panel">
        <h3>API 令牌</h3>
        <p class="muted">使用 <code>Authorization: Bearer &lt;令牌&gt;</code> 调用 <code>/api/search</code>。</p>
        <div v-if="tokenMeta" class="token-line">
          <code>{{ tokenMeta.prefix }}••••••••</code>
          <span class="muted">创建于 {{ shortTime(tokenMeta.created_at) }}</span>
        </div>
        <div v-if="newToken" class="token-line">
          <input class="token-input" :value="newToken" readonly />
          <button class="ghost-button" type="button" @click="copyToken">复制</button>
        </div>
        <p v-if="newToken" class="muted">新令牌只显示一次，请立即保存。</p>
        <button class="ghost-button" type="button" :disabled="resetting" @click="resetToken">
          {{ resetting ? '重置中...' : '重置令牌' }}
        </button>
      </section>

      <section class="panel">
        <h3>第三方账号绑定</h3>
        <p v-if="!bindings.length" class="muted">尚未绑定 OAuth 账号。</p>
        <div v-for="item in bindings" :key="item.provider_id" class="binding-row">
          <span>
            <strong>{{ item.provider_name }}</strong>
            <small v-if="item.email">{{ item.email }}</small>
          </span>
          <button class="text-button danger" type="button" @click="removeBinding(item.provider_id)">解绑</button>
        </div>
        <div class="binding-row" v-for="provider in bindableProviders" :key="`bind-${provider.slug}`">
          <span>{{ provider.name }}</span>
          <a class="text-button" :href="oauthStartUrl(provider.slug, 'bind')">绑定</a>
        </div>
        <p v-if="!authState.config?.auth.oauth_login_enabled && !authState.config?.auth.oauth_register_enabled" class="muted">
          管理员尚未开启 OAuth 登录/注册。
        </p>
      </section>
    </div>
  </div>
</template>

<script setup lang="ts">
import { computed, onMounted, reactive, ref } from 'vue';
import {
  changePassword,
  getAPIToken,
  getOAuthBindings,
  oauthStartUrl,
  resetAPIToken,
  unbindOAuth,
  updateProfile,
  type APIToken,
  type OAuthAccount,
} from '@/api/auth';
import { authState, refreshMe } from '@/stores/auth';

defineEmits<{ back: [] }>();

const user = computed(() => authState.user);
const limitText = computed(() => {
  const limit = user.value?.daily_search_limit || 0;
  return limit > 0 ? ` / ${limit}` : '（不限）';
});
const bindableProviders = computed(() => {
  const bound = new Set(bindings.value.map((item) => item.provider_slug));
  return (authState.config?.oauth_providers || []).filter((provider) => !bound.has(provider.slug));
});

const profile = reactive({ username: authState.user?.username || '', email: authState.user?.email || '' });
const password = reactive({ current: '', next: '' });
const tokenMeta = ref<APIToken | null>(null);
const newToken = ref('');
const bindings = ref<OAuthAccount[]>([]);
const savingProfile = ref(false);
const savingPassword = ref(false);
const resetting = ref(false);
const message = ref('');
const error = ref('');

const shortTime = (s: string) => (s ? new Date(s).toLocaleString() : '');

const clearMessages = () => {
  message.value = '';
  error.value = '';
};

const loadToken = async () => {
  if (!authState.config?.auth.allow_user_api_tokens) return;
  try {
    tokenMeta.value = await getAPIToken();
  } catch {
    tokenMeta.value = null;
  }
};

const loadBindings = async () => {
  try {
    bindings.value = await getOAuthBindings();
  } catch {
    bindings.value = [];
  }
};

onMounted(async () => {
  await Promise.all([loadToken(), loadBindings()]);
  if (user.value) {
    profile.username = user.value.username;
    profile.email = user.value.email || '';
  }
});

const saveProfile = async () => {
  clearMessages();
  savingProfile.value = true;
  try {
    await updateProfile(profile.username, profile.email);
    await refreshMe();
    message.value = '资料已保存';
  } catch (err: any) {
    error.value = err.response?.data?.error || '保存失败';
  } finally {
    savingProfile.value = false;
  }
};

const savePassword = async () => {
  clearMessages();
  if (password.next.length < 8) {
    error.value = '新密码至少 8 位';
    return;
  }
  savingPassword.value = true;
  try {
    await changePassword(password.current, password.next);
    password.current = '';
    password.next = '';
    await refreshMe();
    await Promise.all([loadToken(), loadBindings()]);
    message.value = '密码已更新，其他设备的会话已退出';
  } catch (err: any) {
    error.value = err.response?.data?.error || '修改密码失败';
  } finally {
    savingPassword.value = false;
  }
};

const resetToken = async () => {
  clearMessages();
  if (!confirm('重置后旧令牌立即失效，确定继续？')) return;
  resetting.value = true;
  try {
    const res = await resetAPIToken();
    newToken.value = res.token;
    await loadToken();
    message.value = '令牌已重置';
  } catch (err: any) {
    error.value = err.response?.data?.error || '重置失败';
  } finally {
    resetting.value = false;
  }
};

const copyToken = async () => {
  try {
    await navigator.clipboard.writeText(newToken.value);
    message.value = '已复制到剪贴板';
  } catch {
    error.value = '复制失败，请手动选择令牌';
  }
};

const removeBinding = async (providerId: number) => {
  clearMessages();
  if (!confirm('确定解绑该第三方账号？')) return;
  try {
    await unbindOAuth(providerId);
    await loadBindings();
    await refreshMe();
    message.value = '已解绑';
  } catch (err: any) {
    error.value = err.response?.data?.error || '解绑失败';
  }
};
</script>

<style scoped>
.user-center { max-width: 1080px; margin: 0 auto; }
.page-head { display: flex; justify-content: space-between; align-items: flex-start; gap: 1rem; margin-bottom: 1.25rem; }
.eyebrow { font-size: 0.75rem; letter-spacing: 0.12em; text-transform: uppercase; color: hsl(var(--muted-foreground)); }
.page-head h2 { font-size: 1.6rem; font-weight: 650; margin: 0.2rem 0; }
.sub { color: hsl(var(--muted-foreground)); font-size: 0.88rem; }
.grid { display: grid; grid-template-columns: repeat(auto-fit, minmax(300px, 1fr)); gap: 1rem; }
.panel {
  background: hsl(var(--background-light, 0 0% 100%) / 0.9);
  border: 1px solid hsl(var(--border));
  border-radius: 1rem;
  padding: 1.15rem;
  box-shadow: 0 8px 24px rgba(15, 23, 42, 0.04);
}
.panel h3 { font-size: 1rem; font-weight: 650; margin-bottom: 0.85rem; }
.panel label { display: flex; flex-direction: column; gap: 0.35rem; font-size: 0.8rem; font-weight: 550; margin-bottom: 0.75rem; }
.panel input {
  padding: 0.6rem 0.7rem;
  border: 1px solid hsl(var(--border));
  border-radius: 0.55rem;
  background: hsl(var(--background));
  color: hsl(var(--foreground));
  font-size: 0.88rem;
}
.panel input:focus { outline: none; border-color: hsl(var(--primary)); }
.primary-button, .ghost-button, .text-button {
  border-radius: 0.55rem;
  cursor: pointer;
  font-size: 0.84rem;
  font-weight: 600;
}
.primary-button { width: 100%; padding: 0.62rem; border: 0; background: hsl(var(--primary)); color: hsl(var(--primary-foreground)); }
.primary-button:disabled { opacity: 0.65; cursor: not-allowed; }
.ghost-button { padding: 0.45rem 0.8rem; border: 1px solid hsl(var(--border)); background: transparent; color: hsl(var(--foreground)); }
.ghost-button:hover { border-color: hsl(var(--primary)); color: hsl(var(--primary)); }
.text-button { border: 0; background: transparent; color: hsl(var(--primary)); padding: 0.25rem; text-decoration: none; }
.text-button.danger { color: #b91c1c; }
.muted { color: hsl(var(--muted-foreground)); font-size: 0.8rem; line-height: 1.5; }
.token-line { display: flex; align-items: center; gap: 0.6rem; flex-wrap: wrap; margin-bottom: 0.6rem; }
.token-line code { background: hsl(var(--muted)); padding: 0.3rem 0.5rem; border-radius: 0.4rem; }
.token-input { flex: 1; min-width: 0; font-family: ui-monospace, SFMono-Regular, Menlo, monospace; }
.binding-row { display: flex; justify-content: space-between; align-items: center; gap: 0.5rem; padding: 0.5rem 0; border-bottom: 1px dashed hsl(var(--border)); }
.binding-row:last-child { border-bottom: 0; }
.binding-row small { display: block; color: hsl(var(--muted-foreground)); font-size: 0.72rem; }
.alert { padding: 0.7rem 0.9rem; border-radius: 0.65rem; font-size: 0.84rem; margin-bottom: 0.8rem; }
.alert.warn { background: #fffbeb; border: 1px solid #fde68a; color: #92400e; }
.alert.ok { background: #ecfdf5; border: 1px solid #a7f3d0; color: #047857; }
.alert.error { background: #fef2f2; border: 1px solid #fecaca; color: #b91c1c; }
@media (max-width: 640px) {
  .page-head { flex-direction: column; }
  .page-head h2 { font-size: 1.35rem; }
}
</style>
