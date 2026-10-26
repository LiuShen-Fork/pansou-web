<template>
  <div class="page-stack">
    <div class="settings-bar">
      <div class="settings-bar-copy">
        <strong>配置修改后立即生效</strong>
        <span>正在进行的搜索不受影响；开启邮箱注册前必须先配置 SMTP。</span>
      </div>
      <button class="btn btn-primary" type="button" :disabled="saving || loading" @click="save">
        <AdminIcon name="check" :size="15" /> {{ saving ? '保存中…' : '保存全部设置' }}
      </button>
    </div>

    <div v-if="loading" class="section-card section-body">
      <div class="skeleton" style="width: 30%; margin-bottom: 0.7rem" />
      <div class="skeleton" style="width: 100%; margin-bottom: 0.5rem" />
      <div class="skeleton" style="width: 86%" />
    </div>

    <template v-else>
      <div class="settings-grid">
        <section class="section-card">
          <div class="section-head">
            <div>
              <div class="section-title"><AdminIcon name="home" :size="16" /> 站点信息</div>
              <div class="section-desc">显示在登录页、页头与页脚</div>
            </div>
          </div>
          <div class="section-body">
            <div class="form-grid">
              <div class="field">
                <label>站点名称</label>
                <input v-model="site.name" class="input" placeholder="Qingyu PanSou" />
              </div>
              <div class="field">
                <label>站点描述</label>
                <input v-model="site.description" class="input" placeholder="网盘资源搜索" />
              </div>
            </div>
            <div class="field" style="margin-top: 0.8rem">
              <label>站点公告</label>
              <textarea v-model="site.announcement" class="textarea" placeholder="留空则不显示公告；支持多行文本" />
              <span class="hint">公告会显示在搜索页顶部，适合维护通知与使用说明。</span>
            </div>
          </div>
        </section>

        <section class="section-card">
          <div class="section-head">
            <div>
              <div class="section-title"><AdminIcon name="lock" :size="16" /> 登录与注册</div>
              <div class="section-desc">邮箱密码登录、邮箱注册与 OAuth2 可自由组合</div>
            </div>
          </div>
          <div class="section-body">
            <div class="switch-row">
              <div><strong>密码登录</strong><small>允许用户名或邮箱 + 密码登录。管理员始终保留密码登录入口，避免锁死。</small></div>
              <label class="switch"><input v-model="auth.email_login_enabled" type="checkbox" /><span /></label>
            </div>
            <div class="switch-row">
              <div><strong>邮箱注册</strong><small>开启后注册必须通过 SMTP 邮箱验证码，未配置 SMTP 将无法保存。</small></div>
              <label class="switch"><input v-model="auth.email_register_enabled" type="checkbox" /><span /></label>
            </div>
            <div class="switch-row">
              <div><strong>OAuth2 登录</strong><small>已绑定第三方账号的用户可以通过 OAuth 登录。</small></div>
              <label class="switch"><input v-model="auth.oauth_login_enabled" type="checkbox" /><span /></label>
            </div>
            <div class="switch-row">
              <div><strong>OAuth2 注册</strong><small>第三方账号首次登录时自动创建本地用户，可单独关闭。</small></div>
              <label class="switch"><input v-model="auth.oauth_register_enabled" type="checkbox" /><span /></label>
            </div>
          </div>
        </section>
      </div>

      <section class="section-card">
        <div class="section-head">
          <div>
            <div class="section-title"><AdminIcon name="shield" :size="16" /> 搜索与 API 权限</div>
            <div class="section-desc">控制 API 是否必须登录，以及网页端匿名搜索的开关与限流</div>
          </div>
        </div>
        <div class="section-body">
          <div class="switch-row">
            <div><strong>API 必须登录</strong><small>开启后 <code>/api/search</code>、链接检测与插件 Web 接口都需要会话或个人 API 令牌。</small></div>
            <label class="switch"><input v-model="auth.api_require_login" type="checkbox" /><span /></label>
          </div>
          <div class="switch-row">
            <div><strong>允许网页直接搜索</strong><small>关闭后未登录用户不能使用搜索页，必须登录。</small></div>
            <label class="switch"><input v-model="auth.web_search_enabled" type="checkbox" /><span /></label>
          </div>
          <div class="switch-row">
            <div><strong>允许用户生成 API 令牌</strong><small>用户在用户中心生成 <code>ps_...</code> 令牌，用于脚本或第三方客户端。</small></div>
            <label class="switch"><input v-model="auth.allow_user_api_tokens" type="checkbox" /><span /></label>
          </div>
          <div class="form-grid" style="margin-top: 0.9rem">
            <div class="field">
              <label>默认每日搜索上限</label>
              <input v-model.number="auth.default_daily_search_limit" class="input" type="number" min="0" />
              <span class="hint">0 表示不限制；用户级设置优先于全局</span>
            </div>
            <div class="field">
              <label>匿名搜索限流（次/分钟）</label>
              <input v-model.number="auth.anonymous_web_search_limit" class="input" type="number" min="1" />
              <span class="hint">按 IP 统计，仅作用于匿名网页搜索</span>
            </div>
            <div class="field">
              <label>会话有效期（小时）</label>
              <input v-model.number="auth.session_ttl_hours" class="input" type="number" min="1" max="8760" />
            </div>
          </div>
        </div>
      </section>

      <section class="section-card">
        <div class="section-head">
          <div>
            <div class="section-title"><AdminIcon name="mail" :size="16" /> SMTP 邮件</div>
            <div class="section-desc">用于注册验证码与测试邮件；465 端口使用隐式 TLS，587 使用 STARTTLS</div>
          </div>
          <span class="badge" :class="email.enabled ? 'badge-success' : ''">{{ email.enabled ? '已启用' : '未启用' }}</span>
        </div>
        <div class="section-body">
          <div class="switch-row">
            <div><strong>启用 SMTP</strong><small>邮箱注册与验证码依赖 SMTP；修改后请发送测试邮件确认。</small></div>
            <label class="switch"><input v-model="email.enabled" type="checkbox" /><span /></label>
          </div>
          <div class="form-grid" style="margin-top: 0.8rem">
            <div class="field">
              <label>SMTP 服务器</label>
              <input v-model="email.host" class="input" placeholder="smtp.example.com" />
            </div>
            <div class="field">
              <label>端口</label>
              <input v-model.number="email.port" class="input" type="number" />
            </div>
            <div class="field">
              <label>用户名</label>
              <input v-model="email.username" class="input" autocomplete="off" />
            </div>
            <div class="field">
              <label>密码 / 授权码</label>
              <input v-model="email.password" class="input" type="password" autocomplete="new-password" :placeholder="passwordSet ? '已设置，留空保持不变' : '未设置'" />
            </div>
            <div class="field">
              <label>发件人地址</label>
              <input v-model="email.from" class="input" placeholder="noreply@example.com" />
            </div>
            <div class="field">
              <label>发件人名称</label>
              <input v-model="email.from_name" class="input" />
            </div>
          </div>
          <div class="smtp-test">
            <div class="field" style="flex: 1">
              <label>测试收件邮箱</label>
              <input v-model="testTo" class="input" type="email" placeholder="you@example.com" />
            </div>
            <button class="btn" type="button" :disabled="testing" @click="sendTest">
              <AdminIcon name="mail" :size="15" /> {{ testing ? '发送中…' : '发送测试邮件' }}
            </button>
          </div>
        </div>
      </section>
    </template>
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
import AdminIcon from './AdminIcon.vue';
import { toast } from '@/lib/toast';

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
const loading = ref(true);
const saving = ref(false);
const testing = ref(false);
const testTo = ref('');

const load = async () => {
  loading.value = true;
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
    toast.error('读取设置失败', err.response?.data?.error || '');
  } finally {
    loading.value = false;
  }
};

const save = async () => {
  if (auth.email_register_enabled && (!email.enabled || !email.host)) {
    toast.error('无法保存', '开启邮箱注册前必须启用并配置 SMTP');
    return;
  }
  saving.value = true;
  try {
    await adminSaveSettings({
      site: { ...site },
      auth: { ...auth, site_name: site.name, site_description: site.description, announcement: site.announcement },
      email: { ...email },
    });
    await loadPublicConfig();
    toast.success('设置已保存');
    await load();
  } catch (err: any) {
    toast.error('保存失败', err.response?.data?.error || '');
  } finally {
    saving.value = false;
  }
};

const sendTest = async () => {
  if (!testTo.value.trim()) {
    toast.error('请填写测试收件邮箱');
    return;
  }
  testing.value = true;
  try {
    await adminTestEmail(testTo.value.trim());
    toast.success('测试邮件已发送', '请检查收件箱与垃圾邮件目录');
  } catch (err: any) {
    toast.error('测试邮件发送失败', err.response?.data?.error || '');
  } finally {
    testing.value = false;
  }
};

onMounted(load);
</script>

<style scoped>
.settings-bar {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 1rem;
  padding: 0.8rem 1rem;
  border: 1px solid hsl(var(--border));
  border-radius: 0.8rem;
  background: hsl(var(--background-light, 0 0% 100%) / 0.92);
  box-shadow: var(--console-shadow);
  position: sticky;
  top: 4.5rem;
  z-index: 20;
}
.settings-bar-copy strong { display: block; font-size: 0.82rem; }
.settings-bar-copy span { display: block; margin-top: 0.15rem; font-size: 0.7rem; color: hsl(var(--muted-foreground)); }
.settings-grid { display: grid; grid-template-columns: 1fr 1fr; gap: 1rem; align-items: start; }
.smtp-test { display: flex; align-items: flex-end; gap: 0.6rem; margin-top: 0.9rem; }
@media (max-width: 900px) {
  .settings-grid { grid-template-columns: 1fr; }
  .settings-bar { position: static; flex-direction: column; align-items: flex-start; }
  .smtp-test { flex-direction: column; align-items: stretch; }
}
</style>
