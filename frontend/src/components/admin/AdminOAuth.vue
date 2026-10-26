<template>
  <div class="page-stack">
    <section class="section-card">
      <div class="section-head">
        <div>
          <div class="section-title"><AdminIcon name="key" :size="16" /> OAuth2 服务商</div>
          <div class="section-desc">开启登录/注册前，至少需要一个启用且配置完整的服务商</div>
        </div>
        <button class="btn btn-primary btn-sm" type="button" @click="openCreate">
          <AdminIcon name="plus" :size="15" /> 新增服务商
        </button>
      </div>

      <div class="section-body" style="padding-bottom: 0.7rem">
        <div class="callback-box">
          <div>
            <div class="field-label">统一回调地址</div>
            <div class="callback-url mono">{{ redirectUri }}</div>
            <div class="hint">在 GitHub / Google / Linux DO 等平台登记这个地址</div>
          </div>
          <button class="btn btn-sm" type="button" @click="copyCallback">
            <AdminIcon name="copy" :size="14" /> 复制
          </button>
        </div>
      </div>

      <div v-if="loading" class="section-body">
        <div class="skeleton" style="width: 100%; margin-bottom: 0.6rem" />
        <div class="skeleton" style="width: 82%" />
      </div>
      <div v-else-if="!providers.length" class="empty-state">
        <AdminIcon name="key" :size="24" />
        <strong>尚未配置 OAuth 服务商</strong>
        <span>建议先添加 GitHub 或 Linux DO，再开启 OAuth 登录/注册</span>
      </div>
      <div v-else class="provider-list">
        <article v-for="p in providers" :key="p.id" class="provider-card">
          <div class="provider-icon" :class="providerTone(p.slug)">{{ providerInitial(p) }}</div>
          <div class="provider-main">
            <div class="provider-title">
              <strong>{{ p.name }}</strong>
              <span class="badge" :class="p.enabled ? 'badge-success' : ''">{{ p.enabled ? '启用' : '停用' }}</span>
            </div>
            <div class="provider-slug mono">{{ p.slug }}</div>
            <div class="provider-meta">
              <span class="mono">{{ p.userinfo_url || '未填写 UserInfo URL' }}</span>
            </div>
          </div>
          <div class="provider-actions">
            <button class="btn btn-sm" type="button" @click="openEdit(p)"><AdminIcon name="edit" :size="14" /> 编辑</button>
            <button class="btn btn-sm btn-danger" type="button" @click="removeProvider(p)"><AdminIcon name="trash" :size="14" /> 删除</button>
          </div>
        </article>
      </div>
    </section>

    <Teleport to="body">
      <div v-if="modal.open" class="modal-backdrop" @click.self="modal.open = false">
        <div class="modal wide">
          <div class="modal-head">
            <div>
              <h3>{{ form.id ? '编辑 OAuth 服务商' : '新增 OAuth 服务商' }}</h3>
              <p>可以从预设快速填入，也可以接入任意标准 OAuth2 / OIDC 服务</p>
            </div>
            <button class="icon-btn" type="button" aria-label="关闭" @click="modal.open = false">
              <AdminIcon name="close" :size="16" />
            </button>
          </div>
          <div class="modal-body">
            <div class="field" style="margin-bottom: 0.8rem">
              <label>快速预设</label>
              <select :value="presetKey" class="select" @change="applyPreset(($event.target as HTMLSelectElement).value)">
                <option value="">自定义 / 不修改</option>
                <option value="github">GitHub</option>
                <option value="google">Google</option>
                <option value="linuxdo">Linux DO</option>
                <option value="generic">通用 OIDC</option>
              </select>
            </div>
            <div class="form-grid">
              <div class="field">
                <label>显示名称</label>
                <input v-model="form.name" class="input" placeholder="GitHub" />
              </div>
              <div class="field">
                <label>服务商标识</label>
                <input v-model="form.slug" class="input" placeholder="github" />
                <span class="hint">仅小写字母、数字、下划线与连字符</span>
              </div>
              <div class="field">
                <label>Client ID</label>
                <input v-model="form.client_id" class="input" autocomplete="off" />
              </div>
              <div class="field">
                <label>Client Secret</label>
                <input v-model="form.client_secret" class="input" type="password" autocomplete="new-password" :placeholder="form.id ? '留空保持不变' : ''" />
              </div>
              <div class="field" style="grid-column: 1 / -1">
                <label>Scopes</label>
                <input v-model="form.scopes" class="input" placeholder="openid profile email" />
              </div>
              <div class="field" style="grid-column: 1 / -1">
                <label>Authorization URL</label>
                <input v-model="form.auth_url" class="input" />
              </div>
              <div class="field" style="grid-column: 1 / -1">
                <label>Token URL</label>
                <input v-model="form.token_url" class="input" />
              </div>
              <div class="field" style="grid-column: 1 / -1">
                <label>UserInfo URL</label>
                <input v-model="form.userinfo_url" class="input" />
              </div>
            </div>
            <div class="switch-row" style="margin-top: 0.5rem">
              <div><strong>启用该服务商</strong><small>启用前需填写 Client ID / Secret 与三个 URL</small></div>
              <label class="switch"><input v-model="form.enabled" type="checkbox" /><span /></label>
            </div>
          </div>
          <div class="modal-foot">
            <button class="btn" type="button" @click="modal.open = false">取消</button>
            <button class="btn btn-primary" type="button" :disabled="saving" @click="save">
              {{ saving ? '保存中…' : '保存服务商' }}
            </button>
          </div>
        </div>
      </div>
    </Teleport>
  </div>
</template>

<script setup lang="ts">
import { computed, onMounted, reactive, ref } from 'vue';
import {
  adminDeleteOAuthProvider,
  adminListOAuthProviders,
  adminSaveOAuthProvider,
  type OAuthProvider,
} from '@/api/auth';
import AdminIcon from './AdminIcon.vue';
import { toast } from '@/lib/toast';

const providers = ref<OAuthProvider[]>([]);
const loading = ref(true);
const saving = ref(false);
const presetKey = ref('');
const modal = reactive({ open: false });
const redirectUri = computed(() => `${window.location.origin}/api/auth/oauth/callback`);

const blank = (): OAuthProvider => ({
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
const form = reactive<OAuthProvider>(blank());

const presets: Record<string, Partial<OAuthProvider>> = {
  github: {
    name: 'GitHub',
    slug: 'github',
    auth_url: 'https://github.com/login/oauth/authorize',
    token_url: 'https://github.com/login/oauth/access_token',
    userinfo_url: 'https://api.github.com/user',
    scopes: 'read:user user:email',
  },
  google: {
    name: 'Google',
    slug: 'google',
    auth_url: 'https://accounts.google.com/o/oauth2/v2/auth',
    token_url: 'https://oauth2.googleapis.com/token',
    userinfo_url: 'https://openidconnect.googleapis.com/v1/userinfo',
    scopes: 'openid profile email',
  },
  linuxdo: {
    name: 'Linux DO',
    slug: 'linuxdo',
    auth_url: 'https://connect.linux.do/oauth2/authorize',
    token_url: 'https://connect.linux.do/oauth2/token',
    userinfo_url: 'https://connect.linux.do/api/user',
    scopes: 'openid profile email',
  },
  generic: {
    name: '',
    slug: '',
    auth_url: '',
    token_url: '',
    userinfo_url: '',
    scopes: 'openid profile email',
  },
};

function providerInitial(p: OAuthProvider) {
  return (p.name || p.slug || '?').slice(0, 1).toUpperCase();
}
function providerTone(slug: string) {
  if (slug === 'github') return 'dark';
  if (slug === 'google') return 'red';
  if (slug === 'linuxdo') return 'blue';
  return 'violet';
}

const load = async () => {
  loading.value = true;
  try {
    providers.value = await adminListOAuthProviders();
  } catch (err: any) {
    toast.error('读取 OAuth 服务商失败', err.response?.data?.error || '');
  } finally {
    loading.value = false;
  }
};

function openCreate() {
  Object.assign(form, blank());
  presetKey.value = '';
  modal.open = true;
}

function openEdit(p: OAuthProvider) {
  Object.assign(form, { ...p, client_secret: '' });
  presetKey.value = '';
  modal.open = true;
}

function applyPreset(key: string) {
  presetKey.value = key;
  const preset = presets[key];
  if (preset) Object.assign(form, { ...blank(), ...preset });
}

async function save() {
  if (!form.name.trim() || !form.slug.trim()) {
    toast.error('请填写名称与服务商标识');
    return;
  }
  saving.value = true;
  try {
    await adminSaveOAuthProvider({ ...form });
    toast.success('服务商已保存');
    modal.open = false;
    await load();
  } catch (err: any) {
    toast.error('保存失败', err.response?.data?.error || '');
  } finally {
    saving.value = false;
  }
}

async function removeProvider(p: OAuthProvider) {
  if (!window.confirm(`确定删除「${p.name}」？已绑定的用户将无法再用它登录。`)) return;
  try {
    await adminDeleteOAuthProvider(p.id);
    toast.success('已删除');
    await load();
  } catch (err: any) {
    toast.error('删除失败', err.response?.data?.error || '');
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

.provider-list { display: flex; flex-direction: column; }
.provider-card {
  display: flex;
  align-items: center;
  gap: 0.85rem;
  padding: 0.9rem 1.1rem;
  border-top: 1px solid hsl(var(--border) / 0.7);
}
.provider-icon {
  display: grid;
  place-items: center;
  width: 2.4rem;
  height: 2.4rem;
  border-radius: 0.7rem;
  font-size: 0.9rem;
  font-weight: 800;
  flex: 0 0 auto;
  background: #f5f3ff;
  color: #6d28d9;
}
.provider-icon.dark { background: #111827; color: #fff; }
.provider-icon.red { background: #fef2f2; color: #dc2626; }
.provider-icon.blue { background: #eff6ff; color: #2563eb; }
.provider-main { min-width: 0; flex: 1; }
.provider-title { display: flex; align-items: center; gap: 0.45rem; }
.provider-title strong { font-size: 0.86rem; }
.provider-slug { font-size: 0.68rem; color: hsl(var(--muted-foreground)); }
.provider-meta { margin-top: 0.2rem; font-size: 0.68rem; color: hsl(var(--muted-foreground)); overflow: hidden; text-overflow: ellipsis; white-space: nowrap; }
.provider-actions { display: flex; gap: 0.35rem; flex-wrap: wrap; justify-content: flex-end; }
@media (max-width: 680px) {
  .provider-card { flex-wrap: wrap; }
  .provider-actions { width: 100%; justify-content: flex-start; margin-left: 3.25rem; }
}
</style>
