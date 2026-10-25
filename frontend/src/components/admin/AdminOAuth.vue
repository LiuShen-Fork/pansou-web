<template>
  <div class="admin-content">
    <div v-if="error" class="admin-alert error">{{ error }}</div>
    <div v-if="message" class="admin-alert ok">{{ message }}</div>

    <div class="admin-card">
      <div class="admin-actions" style="justify-content: space-between; margin-bottom: 0.8rem">
        <h3 style="margin: 0">OAuth2 服务商</h3>
        <button class="admin-btn primary" type="button" @click="resetForm">新增服务商</button>
      </div>
      <p class="admin-muted">
        回调地址（需在服务商后台登记）：<code class="admin-code">{{ redirectUri }}</code>
      </p>
      <div class="admin-table-wrap">
        <table class="admin-table">
          <thead><tr><th>名称</th><th>标识</th><th>状态</th><th>UserInfo</th><th>操作</th></tr></thead>
          <tbody>
            <tr v-for="p in providers" :key="p.id">
              <td>{{ p.name }}</td>
              <td><code>{{ p.slug }}</code></td>
              <td><span class="admin-badge" :class="p.enabled ? 'ok' : ''">{{ p.enabled ? '启用' : '停用' }}</span></td>
              <td><span class="admin-muted">{{ p.userinfo_url || '—' }}</span></td>
              <td>
                <div class="admin-actions">
                  <button class="admin-btn" type="button" @click="edit(p)">编辑</button>
                  <button class="admin-btn danger" type="button" @click="remove(p)">删除</button>
                </div>
              </td>
            </tr>
            <tr v-if="!providers.length"><td colspan="5" class="admin-muted">尚未配置 OAuth 服务商。</td></tr>
          </tbody>
        </table>
      </div>
    </div>

    <div class="admin-card">
      <h3>{{ form.id ? '编辑服务商' : '新增服务商' }}</h3>
      <div class="admin-grid">
        <div class="admin-field">
          <label>快速预设</label>
          <select :value="presetKey" @change="applyPreset(($event.target as HTMLSelectElement).value)">
            <option value="">自定义 / 不修改</option>
            <option value="github">GitHub</option>
            <option value="google">Google</option>
            <option value="linuxdo">Linux DO</option>
            <option value="generic">通用 OIDC</option>
          </select>
        </div>
        <div class="admin-field"><label>显示名称</label><input v-model="form.name" placeholder="GitHub" /></div>
        <div class="admin-field"><label>标识（URL 中使用）</label><input v-model="form.slug" placeholder="github" /></div>
        <div class="admin-field"><label>Client ID</label><input v-model="form.client_id" /></div>
        <div class="admin-field">
          <label>Client Secret</label>
          <input v-model="form.client_secret" type="password" :placeholder="form.id ? '留空保持不变' : ''" />
        </div>
        <div class="admin-field">
          <label>Scopes</label>
          <input v-model="form.scopes" placeholder="openid profile email" />
        </div>
      </div>
      <div class="admin-field"><label>Auth URL</label><input v-model="form.auth_url" /></div>
      <div class="admin-field"><label>Token URL</label><input v-model="form.token_url" /></div>
      <div class="admin-field"><label>UserInfo URL</label><input v-model="form.userinfo_url" /></div>
      <div class="admin-field">
        <label class="inline-check"><input v-model="form.enabled" type="checkbox" /> 启用该服务商</label>
      </div>
      <div class="admin-actions">
        <button class="admin-btn primary" type="button" :disabled="saving" @click="save">
          {{ saving ? '保存中...' : '保存服务商' }}
        </button>
        <button class="admin-btn" type="button" @click="resetForm">清空</button>
      </div>
    </div>
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

const providers = ref<OAuthProvider[]>([]);
const saving = ref(false);
const error = ref('');
const message = ref('');
const redirectUri = computed(() => `${window.location.origin}/api/auth/oauth/callback`);
const presetKey = ref('');

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

const load = async () => {
  error.value = '';
  try {
    providers.value = await adminListOAuthProviders();
  } catch (err: any) {
    error.value = err.response?.data?.error || '读取 OAuth 服务商失败';
  }
};

const resetForm = () => {
  Object.assign(form, blank());
  presetKey.value = '';
  message.value = '';
};

const applyPreset = (key: string) => {
  presetKey.value = key;
  const preset = presets[key];
  if (preset) Object.assign(form, { ...blank(), ...preset });
};

const edit = (p: OAuthProvider) => {
  Object.assign(form, {
    ...p,
    client_secret: '',
  });
  presetKey.value = '';
  message.value = '';
};

const save = async () => {
  error.value = '';
  message.value = '';
  if (!form.slug.trim() || !form.name.trim()) {
    error.value = '请填写名称与标识';
    return;
  }
  saving.value = true;
  try {
    await adminSaveOAuthProvider({ ...form });
    message.value = '服务商已保存';
    await load();
    resetForm();
  } catch (err: any) {
    error.value = err.response?.data?.error || '保存失败';
  } finally {
    saving.value = false;
  }
};

const remove = async (p: OAuthProvider) => {
  if (!confirm(`确定删除 OAuth 服务商「${p.name}」？已绑定的账号将无法再用它登录。`)) return;
  error.value = '';
  message.value = '';
  try {
    await adminDeleteOAuthProvider(p.id);
    message.value = '已删除';
    await load();
  } catch (err: any) {
    error.value = err.response?.data?.error || '删除失败';
  }
};

onMounted(load);
</script>

<style scoped>
.inline-check {
  display: flex;
  align-items: center;
  gap: 0.45rem;
  font-size: 0.82rem;
}
</style>
