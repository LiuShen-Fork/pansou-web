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
            <div class="site-brand-preview">
              <div class="brand-preview-icon">
                <img v-if="site.icon" :src="site.icon" alt="站点图标" @error="iconBroken = true" @load="iconBroken = false" />
                <AdminIcon v-else name="home" :size="22" />
                <span v-if="iconBroken" class="brand-preview-fallback">!</span>
              </div>
              <div class="brand-preview-copy">
                <strong>{{ site.name || '站点名称' }}</strong>
                <span>{{ site.slogan || '站点 Slogan' }}</span>
              </div>
            </div>
            <div class="form-grid">
              <div class="field">
                <label>站点名称</label>
                <input v-model="site.name" class="input" placeholder="清羽盘搜" maxlength="40" />
              </div>
              <div class="field">
                <label>站点 Slogan</label>
                <input v-model="site.slogan" class="input" placeholder="清羽飞扬自建网盘搜索，众多资源，一键直达" maxlength="80" />
              </div>
              <div class="field">
                <label>站点描述</label>
                <input v-model="site.description" class="input" placeholder="网盘资源搜索" maxlength="120" />
              </div>
              <div class="field">
                <label>站点图标</label>
                <div class="icon-input-row">
                  <input v-model="site.icon" class="input" placeholder="/favicon.ico 或 https://…" />
                  <label class="btn btn-sm icon-upload">
                    上传
                    <input type="file" accept="image/png,image/jpeg,image/svg+xml,image/webp,image/x-icon" hidden @change="onIconFile" />
                  </label>
                </div>
                <span class="hint">支持外链、站内路径或上传图片（≤ 512KB），会同步更新浏览器标签页图标。</span>
              </div>
            </div>
            <div class="field" style="margin-top: 0.8rem">
              <label>站点公告</label>
              <textarea v-model="site.announcement" class="textarea announcement-input" placeholder="支持 Markdown：**加粗**、[链接](https://…)、- 列表、## 标题" />
              <span class="hint">留空则不显示。公告会在访问首页时以弹窗展示，5 分钟内不重复打扰，支持 Markdown 语法。</span>
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
            <div class="section-desc">匿名、登录用户、网页与 API 的配额分开控制；用户级设置会覆盖这里的默认值</div>
          </div>
          <span class="badge badge-success">配额已拆分</span>
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

          <div class="quota-grid">
            <div class="quota-group">
              <div class="quota-group-head">
                <div>
                  <strong>匿名用户（未登录）</strong>
                  <span>按客户端 IP 统计；触发后前端会提示登录</span>
                </div>
                <span class="badge">IP 维度</span>
              </div>
              <div class="form-grid">
                <div class="field">
                  <label>单 IP 每日额度</label>
                  <input v-model.number="auth.anonymous_daily_limit" class="input" type="number" min="1" />
                  <span class="hint">默认 20 次/天</span>
                </div>
                <div class="field">
                  <label>每分钟限频</label>
                  <input v-model.number="auth.anonymous_rate_limit" class="input" type="number" min="1" />
                  <span class="hint">默认 4 次/分钟</span>
                </div>
                <div class="field">
                  <label>全站每日匿名总额度</label>
                  <input v-model.number="auth.anonymous_global_daily_limit" class="input" type="number" min="1" />
                  <span class="hint">默认 200 次/天，用完只影响匿名访问</span>
                </div>
              </div>
            </div>

            <div class="quota-group">
              <div class="quota-group-head">
                <div>
                  <strong>登录用户默认额度</strong>
                  <span>网页与 API 独立计算；用户管理里可逐个覆盖</span>
                </div>
                <span class="badge badge-primary">用户维度</span>
              </div>
              <div class="form-grid">
                <div class="field">
                  <label>网页搜索每日额度</label>
                  <input v-model.number="auth.default_web_daily_limit" class="input" type="number" min="1" />
                  <span class="hint">默认 100 次/天</span>
                </div>
                <div class="field">
                  <label>网页搜索每分钟限频</label>
                  <input v-model.number="auth.default_web_rate_limit" class="input" type="number" min="1" />
                  <span class="hint">默认 10 次/分钟</span>
                </div>
                <div class="field">
                  <label>API 每日额度</label>
                  <input v-model.number="auth.default_api_daily_limit" class="input" type="number" min="1" />
                  <span class="hint">默认 100 次/天，仅统计 /api/search</span>
                </div>
                <div class="field">
                  <label>API 每分钟限频</label>
                  <input v-model.number="auth.default_api_rate_limit" class="input" type="number" min="1" />
                  <span class="hint">默认 10 次/分钟</span>
                </div>
              </div>
            </div>
          </div>

          <div class="field" style="margin-top: 0.9rem; max-width: 20rem">
            <label>会话有效期（小时）</label>
            <input v-model.number="auth.session_ttl_hours" class="input" type="number" min="1" max="8760" />
          </div>
        </div>
      </section>

      <section class="section-card">
        <div class="section-head">
          <div>
            <div class="section-title"><AdminIcon name="pin" :size="16" /> IP 归属地</div>
            <div class="section-desc">可选功能：使用 ip2region xdb 文件把搜索与审计日志中的 IP 解析为国家 / 地区 / 城市</div>
          </div>
          <span class="badge" :class="auth.ip_geo_enabled ? 'badge-success' : ''">{{ auth.ip_geo_enabled ? '已启用' : '未启用' }}</span>
        </div>
        <div class="section-body">
          <div class="switch-row">
            <div><strong>启用 IP 归属地</strong><small>启用后必须配置 IPv4 xdb；查询失败只记录空归属地，不影响搜索。</small></div>
            <label class="switch"><input v-model="auth.ip_geo_enabled" type="checkbox" /><span /></label>
          </div>
          <div class="form-grid" style="margin-top: 0.8rem">
            <div class="field">
              <label>IPv4 xdb 路径（必填）</label>
              <input v-model.trim="auth.ip_geo_ipv4_path" class="input" placeholder="/app/data/geoip/ip2region.xdb" spellcheck="false" />
            </div>
            <div class="field">
              <label>IPv6 xdb 路径（可选）</label>
              <input v-model.trim="auth.ip_geo_ipv6_path" class="input" placeholder="留空则不解析 IPv6，例如 ip2region_v6.xdb" spellcheck="false" />
            </div>
          </div>
          <span class="hint">把 ip2region xdb 文件挂载到容器内（例如 <code>./geoip:/app/data/geoip:ro</code>）后填写容器内路径。开启时后端会校验文件版本与可读性。</span>
          <div v-if="geoStatus" class="geo-status">
            当前：IPv4 {{ geoStatus.ipv4_loaded ? '已加载' : '未加载' }} · IPv6 {{ geoStatus.ipv6_loaded ? '已加载' : '未加载' }}
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
  type GeoIPStatus,
} from '@/api/auth';
import { loadPublicConfig } from '@/stores/auth';
import AdminIcon from './AdminIcon.vue';
import { toast } from '@/lib/toast';

const site = reactive({ name: '', description: '', slogan: '', icon: '/favicon.ico', announcement: '' });
const iconBroken = ref(false);
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
  default_web_daily_limit: 100,
  default_web_rate_limit: 10,
  default_api_daily_limit: 100,
  default_api_rate_limit: 10,
  anonymous_rate_limit: 4,
  anonymous_daily_limit: 20,
  anonymous_global_daily_limit: 200,
  ip_geo_enabled: false,
  ip_geo_ipv4_path: '',
  ip_geo_ipv6_path: '',
  quota_defaults_version: 1,
  session_ttl_hours: 168,
  site_slogan: '',
  site_icon: '/favicon.ico',
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
const geoStatus = ref<GeoIPStatus | null>(null);
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
    site.slogan = data.site.slogan || '';
    site.icon = data.site.icon || '/favicon.ico';
    site.announcement = data.site.announcement;
    iconBroken.value = false;
    Object.assign(auth, data.auth);
    Object.assign(email, data.email);
    email.password = '';
    passwordSet.value = !!data.email_password_set;
    geoStatus.value = data.geo_status || null;
  } catch (err: any) {
    toast.error('读取设置失败', err.response?.data?.error || '');
  } finally {
    loading.value = false;
  }
};

const onIconFile = (event: Event) => {
  const input = event.target as HTMLInputElement;
  const file = input.files?.[0];
  input.value = '';
  if (!file) return;
  if (file.size > 512 * 1024) {
    toast.error('图片过大', '请选择不超过 512KB 的图标');
    return;
  }
  const reader = new FileReader();
  reader.onload = () => {
    site.icon = String(reader.result || '');
    iconBroken.value = false;
  };
  reader.onerror = () => toast.error('读取图片失败');
  reader.readAsDataURL(file);
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
      auth: {
        ...auth,
        site_name: site.name,
        site_description: site.description,
        site_slogan: site.slogan,
        site_icon: site.icon,
        announcement: site.announcement,
      },
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
  top: calc(var(--console-topbar-h, 6.2rem) + 0.65rem);
  z-index: 20;
}
.settings-bar-copy strong { display: block; font-size: 0.82rem; }
.settings-bar-copy span { display: block; margin-top: 0.15rem; font-size: 0.7rem; color: hsl(var(--muted-foreground)); }
.settings-grid { display: grid; grid-template-columns: 1fr 1fr; gap: 1rem; }
.quota-grid { display: grid; grid-template-columns: 1fr 1fr; gap: 0.9rem; margin-top: 1rem; }
.quota-group {
  padding: 0.85rem;
  border: 1px solid hsl(var(--border));
  border-radius: 0.75rem;
  background: hsl(var(--muted) / 0.22);
}
.quota-group-head { display: flex; align-items: flex-start; justify-content: space-between; gap: 0.6rem; margin-bottom: 0.75rem; }
.quota-group-head strong { display: block; font-size: 0.82rem; }
.quota-group-head span:not(.badge) { display: block; margin-top: 0.15rem; font-size: 0.7rem; color: hsl(var(--muted-foreground)); }
.quota-group .form-grid { grid-template-columns: 1fr 1fr; }
.geo-status {
  margin-top: 0.7rem;
  padding: 0.55rem 0.7rem;
  border-radius: 0.55rem;
  background: hsl(var(--muted) / 0.35);
  font-size: 0.72rem;
  color: hsl(var(--muted-foreground));
}
.smtp-test { display: flex; align-items: flex-end; gap: 0.6rem; margin-top: 0.9rem; }
.site-brand-preview {
  display: flex;
  align-items: center;
  gap: 0.7rem;
  margin-bottom: 0.9rem;
  padding: 0.7rem 0.8rem;
  border: 1px dashed hsl(var(--border));
  border-radius: 0.7rem;
  background: hsl(var(--muted) / 0.3);
}
.brand-preview-icon {
  position: relative;
  display: grid;
  place-items: center;
  width: 2.9rem;
  height: 2.9rem;
  flex: 0 0 2.9rem;
  border: 1px solid hsl(var(--border));
  border-radius: 0.8rem;
  background: #fff;
  color: hsl(var(--muted-foreground));
  overflow: hidden;
}
.brand-preview-icon img { width: 100%; height: 100%; object-fit: contain; padding: 0.25rem; }
.brand-preview-fallback {
  position: absolute;
  inset: 0;
  display: grid;
  place-items: center;
  background: hsl(var(--destructive) / 0.1);
  color: hsl(var(--destructive));
  font-weight: 700;
}
.brand-preview-copy { min-width: 0; }
.brand-preview-copy strong { display: block; font-size: 0.85rem; }
.brand-preview-copy span { display: block; margin-top: 0.15rem; font-size: 0.7rem; color: hsl(var(--muted-foreground)); overflow: hidden; text-overflow: ellipsis; white-space: nowrap; }
.icon-input-row { display: flex; gap: 0.4rem; }
.icon-input-row .input { flex: 1; min-width: 0; }
.icon-upload { position: relative; flex: 0 0 auto; cursor: pointer; }
.announcement-input { min-height: 8rem; font-family: ui-monospace, SFMono-Regular, Menlo, monospace; font-size: 0.75rem; line-height: 1.6; }
@media (max-width: 900px) {
  .settings-grid,
  .quota-grid,
  .quota-group .form-grid { grid-template-columns: 1fr; }
  .settings-bar { position: static; flex-direction: column; align-items: flex-start; }
  .smtp-test { flex-direction: column; align-items: stretch; }
}
</style>
