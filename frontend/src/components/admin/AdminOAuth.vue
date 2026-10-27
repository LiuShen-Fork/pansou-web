<template>
  <div class="page-stack">
    <section class="section-card">
      <div class="section-head oauth-head">
        <div>
          <div class="section-title"><AdminIcon name="key" :size="16" /> OAuth2 登录</div>
          <div class="section-desc">只保留一个第三方登录入口；配置后可在「认证与站点」开启登录或注册</div>
        </div>
        <div class="oauth-head-actions">
          <span class="badge" :class="statusClass">{{ statusLabel }}</span>
          <button class="btn btn-primary btn-sm" type="button" :disabled="saving || loading" @click="save">
            <AdminIcon name="check" :size="14" /> {{ saving ? '保存中…' : '保存配置' }}
          </button>
        </div>
      </div>

      <div v-if="loading" class="section-body">
        <div class="skeleton" style="width: 100%; margin-bottom: 0.6rem" />
        <div class="skeleton" style="width: 84%; margin-bottom: 0.6rem" />
        <div class="skeleton" style="width: 92%" />
      </div>

      <template v-else>
        <div class="section-body oauth-body">
          <div class="callback-box">
            <div>
              <div class="field-label">统一回调地址</div>
              <div class="callback-url mono">{{ redirectUri }}</div>
              <div class="hint">在 GitHub / Google / Linux DO 等平台创建 OAuth 应用时填写这个地址</div>
            </div>
            <button class="btn btn-sm" type="button" @click="copyCallback">
              <AdminIcon name="copy" :size="14" /> 复制
            </button>
          </div>

          <div class="oauth-grid">
            <aside class="oauth-preview">
              <div class="provider-logo" :class="logoTone">{{ providerInitial }}</div>
              <strong>{{ form.name || '未命名服务商' }}</strong>
              <span class="badge" :class="statusClass">{{ statusLabel }}</span>
              <div class="preview-meta">
                <span class="mono">{{ providerHost || '填写 UserInfo URL 后显示服务地址' }}</span>
              </div>
              <p>保存并启用后，登录弹窗只显示一个「使用 {{ form.name || '第三方账号' }} 登录」按钮。</p>
            </aside>

            <div class="oauth-form">
              <div class="field">
                <label>快速预设</label>
                <div class="preset-row">
                  <button
                    v-for="item in presets"
                    :key="item.key"
                    type="button"
                    :class="{ active: presetKey === item.key }"
                    @click="applyPreset(item.key)"
                  >
                    <span class="preset-mark" :class="item.key">{{ item.label.slice(0, 1) }}</span>
                    {{ item.label }}
                  </button>
                  <button type="button" :class="{ active: presetKey === 'custom' }" @click="presetKey = 'custom'">
                    <span class="preset-mark custom">自</span>自定义
                  </button>
                </div>
              </div>

              <div class="form-grid" style="margin-top: 0.85rem">
                <div class="field">
                  <label>显示名称</label>
                  <input v-model="form.name" class="input" maxlength="32" placeholder="例如 GitHub" />
                </div>
                <div class="field">
                  <label>Client ID</label>
                  <input v-model="form.client_id" class="input" autocomplete="off" placeholder="OAuth 应用的 Client ID" />
                </div>
                <div class="field">
                  <label>Client Secret</label>
                  <input
                    v-model="form.client_secret"
                    class="input"
                    type="password"
                    autocomplete="new-password"
                    :placeholder="secretSet ? '已设置，留空保持不变' : 'OAuth 应用的 Client Secret'"
                  />
                </div>
                <div class="field">
                  <label>Scopes</label>
                  <input v-model="form.scopes" class="input" placeholder="openid profile email" />
                </div>
                <div class="field field-full">
                  <label>Authorization URL</label>
                  <input v-model="form.auth_url" class="input" placeholder="https://…/oauth/authorize" />
                </div>
                <div class="field field-full">
                  <label>Token URL</label>
                  <input v-model="form.token_url" class="input" placeholder="https://…/oauth/token" />
                </div>
                <div class="field field-full">
                  <label>UserInfo URL</label>
                  <input v-model="form.userinfo_url" class="input" placeholder="https://…/api/user" />
                </div>
              </div>

              <div class="switch-row oauth-switch">
                <div>
                  <strong>启用这个 OAuth2 服务商</strong>
                  <small>启用前需要填写完整的 Client ID、Secret 与三个接口地址</small>
                </div>
                <label class="switch"><input v-model="form.enabled" type="checkbox" /><span /></label>
              </div>
            </div>
          </div>
        </div>
      </template>
    </section>

    <section class="section-card tips-card">
      <div class="section-head">
        <div>
          <div class="section-title"><AdminIcon name="shield" :size="16" /> 接入步骤</div>
          <div class="section-desc">一个服务商即可完成第三方登录与注册</div>
        </div>
      </div>
      <div class="section-body tips-grid">
        <div class="tip-item">
          <span class="tip-index">01</span>
          <div><strong>创建 OAuth 应用</strong><small>在服务商后台登记上面的回调地址。</small></div>
        </div>
        <div class="tip-item">
          <span class="tip-index">02</span>
          <div><strong>填写凭证与接口</strong><small>可使用快速预设，再替换 Client ID / Secret。</small></div>
        </div>
        <div class="tip-item">
          <span class="tip-index">03</span>
          <div><strong>开启登录或注册</strong><small>前往「认证与站点」打开 OAuth2 登录 / 注册。</small></div>
        </div>
      </div>
    </section>
  </div>
</template>

<script setup lang="ts">
import { computed, onMounted, reactive, ref } from 'vue';
import {
  adminGetOAuthProvider,
  adminSaveOAuthProviderSingle,
  type OAuthProvider,
} from '@/api/auth';
import AdminIcon from './AdminIcon.vue';
import { toast } from '@/lib/toast';

const loading = ref(true);
const saving = ref(false);
const secretSet = ref(false);
const hasProvider = ref(false);
const redirectUri = ref('');
const presetKey = ref('');
const form = reactive<OAuthProvider>({
  id: 0,
  slug: '',
  name: '',
  enabled: false,
  client_id: '',
  client_secret: '',
  auth_url: '',
  token_url: '',
  userinfo_url: '',
  scopes: 'openid profile email',
});

const presets = [
  {
    key: 'github',
    label: 'GitHub',
    name: 'GitHub',
    auth_url: 'https://github.com/login/oauth/authorize',
    token_url: 'https://github.com/login/oauth/access_token',
    userinfo_url: 'https://api.github.com/user',
    scopes: 'read:user user:email',
  },
  {
    key: 'google',
    label: 'Google',
    name: 'Google',
    auth_url: 'https://accounts.google.com/o/oauth2/v2/auth',
    token_url: 'https://oauth2.googleapis.com/token',
    userinfo_url: 'https://openidconnect.googleapis.com/v1/userinfo',
    scopes: 'openid profile email',
  },
  {
    key: 'linuxdo',
    label: 'Linux DO',
    name: 'Linux DO',
    auth_url: 'https://connect.linux.do/oauth2/authorize',
    token_url: 'https://connect.linux.do/oauth2/token',
    userinfo_url: 'https://connect.linux.do/api/user',
    scopes: 'openid profile email',
  },
];

const statusLabel = computed(() => {
  if (!hasProvider.value) return '未配置';
  if (!form.enabled) return '未启用';
  return '已启用';
});
const statusClass = computed(() => {
  if (statusLabel.value === '已启用') return 'badge-success';
  if (statusLabel.value === '未配置') return '';
  return 'badge-warning';
});
const providerInitial = computed(() => (form.name || 'O').slice(0, 1).toUpperCase());
const logoTone = computed(() => presetKey.value || 'custom');
const providerHost = computed(() => {
  try {
    return form.userinfo_url ? new URL(form.userinfo_url).host : '';
  } catch {
    return '';
  }
});

function applyPreset(key: string) {
  const preset = presets.find((item) => item.key === key);
  if (!preset) return;
  presetKey.value = key;
  form.name = preset.name;
  form.auth_url = preset.auth_url;
  form.token_url = preset.token_url;
  form.userinfo_url = preset.userinfo_url;
  form.scopes = preset.scopes;
}

async function load() {
  loading.value = true;
  try {
    const state = await adminGetOAuthProvider();
    redirectUri.value = state.redirect_uri || `${window.location.origin}/api/auth/oauth/callback`;
    secretSet.value = state.secret_set;
    hasProvider.value = !!state.provider;
    if (state.provider) {
      Object.assign(form, state.provider);
    }
    form.client_secret = '';
    if (state.provider) {
      const matched = presets.find((item) => item.userinfo_url === form.userinfo_url);
      presetKey.value = matched?.key || 'custom';
    }
  } catch (err: any) {
    toast.error('读取 OAuth 配置失败', err.response?.data?.error || '');
  } finally {
    loading.value = false;
  }
}

async function save() {
  if (!form.name.trim()) {
    toast.error('请填写服务商显示名称');
    return;
  }
  if (form.enabled) {
    const missing = !form.client_id.trim()
      ? 'Client ID'
      : (!form.client_secret.trim() && !secretSet.value)
        ? 'Client Secret'
        : !form.auth_url.trim()
          ? 'Authorization URL'
          : !form.token_url.trim()
            ? 'Token URL'
            : !form.userinfo_url.trim()
              ? 'UserInfo URL'
              : '';
    if (missing) {
      toast.error('配置不完整', `启用前请填写 ${missing}`);
      return;
    }
  }
  saving.value = true;
  try {
    await adminSaveOAuthProviderSingle({ ...form });
    toast.success('OAuth2 配置已保存');
    await load();
  } catch (err: any) {
    toast.error('保存失败', err.response?.data?.error || '');
  } finally {
    saving.value = false;
  }
}

async function copyCallback() {
  try {
    await navigator.clipboard.writeText(redirectUri.value);
    toast.success('回调地址已复制');
  } catch {
    toast.error('复制失败，请手动选择地址');
  }
}

onMounted(load);
</script>

<style scoped>
.oauth-head { align-items: center; }
.oauth-head-actions { display: flex; align-items: center; gap: 0.5rem; flex-wrap: wrap; justify-content: flex-end; }
.oauth-body { display: flex; flex-direction: column; gap: 1rem; }

.callback-box {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 1rem;
  padding: 0.75rem 0.85rem;
  border: 1px dashed hsl(var(--border));
  border-radius: 0.7rem;
  background: hsl(var(--muted) / 0.3);
}
.callback-url { margin-top: 0.2rem; word-break: break-all; font-size: 0.76rem; }
.hint { margin-top: 0.15rem; font-size: 0.68rem; color: hsl(var(--muted-foreground)); }

.oauth-grid {
  display: grid;
  grid-template-columns: 15rem minmax(0, 1fr);
  gap: 1rem;
  align-items: stretch;
}
.oauth-preview {
  display: flex;
  flex-direction: column;
  align-items: flex-start;
  gap: 0.5rem;
  padding: 1rem;
  border: 1px solid hsl(var(--border));
  border-radius: 0.8rem;
  background:
    radial-gradient(circle at 100% 0%, hsl(var(--primary) / 0.1), transparent 12rem),
    hsl(var(--muted) / 0.22);
}
.provider-logo {
  display: grid;
  place-items: center;
  width: 2.9rem;
  height: 2.9rem;
  margin-bottom: 0.2rem;
  border-radius: 0.8rem;
  background: hsl(var(--primary) / 0.12);
  color: hsl(var(--primary));
  font-size: 1.05rem;
  font-weight: 800;
}
.provider-logo.github { background: #111827; color: #fff; }
.provider-logo.google { background: #fef2f2; color: #dc2626; }
.provider-logo.linuxdo { background: #eff6ff; color: #2563eb; }
.oauth-preview strong { font-size: 0.92rem; }
.preview-meta { width: 100%; font-size: 0.66rem; color: hsl(var(--muted-foreground)); word-break: break-all; }
.oauth-preview p { margin-top: auto; padding-top: 0.5rem; font-size: 0.68rem; line-height: 1.6; color: hsl(var(--muted-foreground)); }

.oauth-form { min-width: 0; }
.preset-row { display: flex; gap: 0.4rem; flex-wrap: wrap; }
.preset-row button {
  display: inline-flex;
  align-items: center;
  gap: 0.35rem;
  padding: 0.35rem 0.55rem;
  border: 1px solid hsl(var(--border));
  border-radius: 999px;
  background: transparent;
  color: hsl(var(--muted-foreground));
  font-size: 0.7rem;
  font-weight: 550;
  cursor: pointer;
  transition: all 0.15s ease;
}
.preset-row button:hover { border-color: hsl(var(--primary) / 0.45); color: hsl(var(--primary)); }
.preset-row button.active { border-color: hsl(var(--primary)); background: hsl(var(--primary) / 0.08); color: hsl(var(--primary)); }
.preset-mark {
  display: grid;
  place-items: center;
  width: 1.1rem;
  height: 1.1rem;
  border-radius: 999px;
  background: hsl(var(--muted));
  font-size: 0.58rem;
  font-weight: 750;
}
.preset-mark.github { background: #111827; color: #fff; }
.preset-mark.google { background: #dc2626; color: #fff; }
.preset-mark.linuxdo { background: #2563eb; color: #fff; }
.preset-mark.custom { background: hsl(var(--primary) / 0.14); color: hsl(var(--primary)); }
.field-full { grid-column: 1 / -1; }
.oauth-switch { margin-top: 0.9rem; padding-top: 0.75rem; border-top: 1px solid hsl(var(--border) / 0.7); }

.tips-card .section-head { border-bottom: 0; }
.tips-grid { display: grid; grid-template-columns: repeat(3, 1fr); gap: 0.8rem; }
.tip-item { display: flex; gap: 0.6rem; padding: 0.75rem; border: 1px solid hsl(var(--border) / 0.8); border-radius: 0.7rem; background: hsl(var(--muted) / 0.18); }
.tip-index { font-size: 0.68rem; font-weight: 800; color: hsl(var(--primary)); letter-spacing: 0.04em; }
.tip-item strong { display: block; font-size: 0.76rem; }
.tip-item small { display: block; margin-top: 0.2rem; font-size: 0.68rem; line-height: 1.55; color: hsl(var(--muted-foreground)); }

@media (max-width: 900px) {
  .oauth-grid { grid-template-columns: 1fr; }
  .oauth-preview { flex-direction: row; align-items: center; flex-wrap: wrap; }
  .oauth-preview p { width: 100%; margin-top: 0.3rem; padding-top: 0; }
  .tips-grid { grid-template-columns: 1fr; }
}
@media (max-width: 560px) {
  .oauth-head { flex-direction: column; align-items: flex-start; }
  .oauth-head-actions { width: 100%; justify-content: space-between; }
  .callback-box { flex-direction: column; align-items: flex-start; }
  .callback-box .btn { width: 100%; }
}
</style>
