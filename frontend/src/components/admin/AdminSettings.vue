<template>
  <div class="admin-content">
    <div v-if="error" class="admin-alert error">{{ error }}</div>
    <div v-if="message" class="admin-alert ok">{{ message }}</div>

    <div class="admin-card">
      <h3>站点信息</h3>
      <div class="admin-grid">
        <div class="admin-field"><label>站点名称</label><input v-model="site.name" /></div>
        <div class="admin-field"><label>站点描述</label><input v-model="site.description" /></div>
      </div>
      <div class="admin-field">
        <label>公告（显示在搜索页顶部，留空关闭）</label>
        <textarea v-model="site.announcement" placeholder="例如：服务维护通知"></textarea>
      </div>
    </div>

    <div class="admin-card">
      <h3>登录与注册</h3>
      <div class="admin-switch">
        <div><strong>密码登录（用户名 / 邮箱）</strong><small>关闭后普通用户只能使用 OAuth；管理员始终可用密码登录，避免锁死。</small></div>
        <input v-model="auth.email_login_enabled" type="checkbox" />
      </div>
      <div class="admin-switch">
        <div><strong>邮箱注册</strong><small>开启前必须启用 SMTP。注册时强制邮箱验证码验证。</small></div>
        <input v-model="auth.email_register_enabled" type="checkbox" />
      </div>
      <div class="admin-switch">
        <div><strong>OAuth2 登录</strong><small>允许已绑定的第三方账号登录。</small></div>
        <input v-model="auth.oauth_login_enabled" type="checkbox" />
      </div>
      <div class="admin-switch">
        <div><strong>OAuth2 注册</strong><small>允许第三方账号自动创建新用户。</small></div>
        <input v-model="auth.oauth_register_enabled" type="checkbox" />
      </div>
    </div>

    <div class="admin-card">
      <h3>搜索与 API</h3>
      <div class="admin-switch">
        <div><strong>API 必须登录</strong><small>开启后 /api/search、链接检测与插件 Web 接口都需要会话或 API 令牌。</small></div>
        <input v-model="auth.api_require_login" type="checkbox" />
      </div>
      <div class="admin-switch">
        <div><strong>前端直接搜索</strong><small>允许未登录用户在网页上搜索（走 /api/web/search，受匿名限流）。关闭后必须登录。</small></div>
        <input v-model="auth.web_search_enabled" type="checkbox" />
      </div>
      <div class="admin-switch">
        <div><strong>允许用户生成 API 令牌</strong><small>用户可在用户中心查看/重置个人令牌，用于按用户统计与配额。</small></div>
        <input v-model="auth.allow_user_api_tokens" type="checkbox" />
      </div>
      <div class="admin-grid">
        <div class="admin-field">
          <label>默认每日搜索上限（0 = 不限）</label>
          <input v-model.number="auth.default_daily_search_limit" type="number" min="0" />
        </div>
        <div class="admin-field">
          <label>匿名网页搜索限流（次/分钟）</label>
          <input v-model.number="auth.anonymous_web_search_limit" type="number" min="0" />
        </div>
        <div class="admin-field">
          <label>会话有效期（小时）</label>
          <input v-model.number="auth.session_ttl_hours" type="number" min="1" max="8760" />
        </div>
      </div>
    </div>

    <div class="admin-card">
      <h3>SMTP 邮件</h3>
      <div class="admin-switch">
        <div><strong>启用 SMTP</strong><small>邮箱注册与验证码依赖 SMTP；465 端口使用隐式 TLS，587 使用 STARTTLS。</small></div>
        <input v-model="email.enabled" type="checkbox" />
      </div>
      <div class="admin-grid">
        <div class="admin-field"><label>SMTP 服务器</label><input v-model="email.host" placeholder="smtp.example.com" /></div>
        <div class="admin-field"><label>端口</label><input v-model.number="email.port" type="number" /></div>
        <div class="admin-field"><label>用户名</label><input v-model="email.username" /></div>
        <div class="admin-field">
          <label>密码 / 授权码</label>
          <input v-model="email.password" type="password" :placeholder="passwordSet ? '已设置，留空保持不变' : '未设置'" />
        </div>
        <div class="admin-field"><label>发件人地址</label><input v-model="email.from" placeholder="noreply@example.com" /></div>
        <div class="admin-field"><label>发件人名称</label><input v-model="email.from_name" /></div>
      </div>
      <div class="admin-field">
        <label class="inline-check"><input v-model="email.use_tls" type="checkbox" /> 使用 TLS（465 端口自动隐式 TLS）</label>
      </div>
      <div class="admin-actions">
        <input v-model="testTo" class="test-input" type="email" placeholder="测试收件邮箱" />
        <button class="admin-btn" type="button" :disabled="testing" @click="sendTest">
          {{ testing ? '发送中...' : '发送测试邮件' }}
        </button>
      </div>
    </div>

    <div class="admin-actions">
      <button class="admin-btn primary" type="button" :disabled="saving" @click="save">
        {{ saving ? '保存中...' : '保存全部设置' }}
      </button>
      <span class="admin-muted">修改后立即生效；正在进行的搜索不受影响。</span>
    </div>
  </div>
</template>

<script setup lang="ts">
import { onMounted, reactive, ref } from 'vue';
import {
  adminGetSettings,
  adminSaveSettings,
  adminTestEmail,
  type AuthSettings,
  type EmailSettings,
} from '@/api/auth';
import { loadPublicConfig } from '@/stores/auth';

const site = reactive({ name: '', description: '', announcement: '' });
const auth = reactive<AuthSettings>({
  site_name: '',
  site_description: '',
  announcement: '',
  email_login_enabled: true,
  email_register_enabled: false,
  oauth_login_enabled: false,
  oauth_register_enabled: false,
  api_require_login: true,
  web_search_enabled: true,
  allow_user_api_tokens: true,
  default_daily_search_limit: 0,
  anonymous_web_search_limit: 20,
  session_ttl_hours: 168,
});
const email = reactive<EmailSettings>({
  enabled: false,
  host: '',
  port: 587,
  username: '',
  password: '',
  from: '',
  from_name: '',
  use_tls: true,
});
const passwordSet = ref(false);
const loading = ref(false);
const saving = ref(false);
const testing = ref(false);
const testTo = ref('');
const error = ref('');
const message = ref('');

const load = async () => {
  loading.value = true;
  error.value = '';
  try {
    const data = await adminGetSettings();
    site.name = data.site.name;
    site.description = data.site.description;
    site.announcement = data.site.announcement;
    Object.assign(auth, data.auth);
    Object.assign(email, data.email);
    email.password = '';
    passwordSet.value = !!data.email_password_set;
  } catch (err: any) {
    error.value = err.response?.data?.error || '读取设置失败';
  } finally {
    loading.value = false;
  }
};

const save = async () => {
  error.value = '';
  message.value = '';
  if (auth.email_register_enabled && (!email.enabled || !email.host)) {
    error.value = '开启邮箱注册前必须启用并配置 SMTP';
    return;
  }
  saving.value = true;
  try {
    const authPayload: AuthSettings = {
      ...auth,
      site_name: site.name,
      site_description: site.description,
      announcement: site.announcement,
    };
    await adminSaveSettings({ site: { ...site }, auth: authPayload, email: { ...email } });
    message.value = '设置已保存';
    await loadPublicConfig();
    await load();
  } catch (err: any) {
    error.value = err.response?.data?.error || '保存失败';
  } finally {
    saving.value = false;
  }
};

const sendTest = async () => {
  error.value = '';
  message.value = '';
  if (!testTo.value.trim()) {
    error.value = '请填写测试收件邮箱';
    return;
  }
  testing.value = true;
  try {
    await adminTestEmail(testTo.value.trim());
    message.value = '测试邮件已发送，请检查收件箱';
  } catch (err: any) {
    error.value = err.response?.data?.error || '测试邮件发送失败';
  } finally {
    testing.value = false;
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
.test-input {
  padding: 0.5rem 0.65rem;
  border: 1px solid hsl(var(--border));
  border-radius: 0.5rem;
  background: hsl(var(--background));
  color: hsl(var(--foreground));
  font-size: 0.82rem;
  min-width: 220px;
}
</style>
