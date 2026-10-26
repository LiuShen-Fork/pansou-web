<template>
  <AdminShell
    :title="current.title"
    :subtitle="current.subtitle"
    :active="active"
    mode="user"
    :nav="nav"
    @select="active = $event"
    @switch-mode="onSwitchMode"
    @back="$emit('back')"
    @logout="$emit('logout')"
  >
    <div class="page-stack">
      <div v-if="user?.must_change_password" class="notice notice-warning">
        <AdminIcon name="alert" :size="16" />
        <div>当前仍在使用默认密码。请先在「安全设置」中修改密码；改密后即可进入管理后台并使用 API 令牌等完整功能。</div>
      </div>

      <!-- 概览 -->
      <template v-if="active === 'overview'">
        <section class="user-hero">
          <div class="hero-main">
            <div class="hero-avatar">{{ (user?.username || '?').slice(0, 1).toUpperCase() }}</div>
            <div class="hero-copy">
              <div class="hero-title">
                <h2>{{ user?.username || '未登录' }}</h2>
                <span class="badge" :class="user?.role === 'admin' ? 'badge-primary' : ''">{{ user?.role === 'admin' ? '管理员' : '普通用户' }}</span>
              </div>
              <p>{{ user?.email || '未绑定邮箱' }} · 注册于 {{ shortDate(user?.created_at) }}</p>
              <div class="hero-meta">
                <span><AdminIcon name="search" :size="14" /> 今日搜索 {{ user?.search_count_today || 0 }}{{ limitText }}</span>
                <span><AdminIcon name="clock" :size="14" /> 最近登录 {{ user?.last_login_at ? shortTime(user.last_login_at) : '—' }}</span>
              </div>
            </div>
          </div>
          <div class="hero-actions">
            <button class="btn" type="button" @click="active = 'profile'"><AdminIcon name="user" :size="15" /> 编辑资料</button>
            <button class="btn" type="button" @click="active = 'security'"><AdminIcon name="lock" :size="15" /> 修改密码</button>
          </div>
        </section>

        <div class="stat-grid">
          <div class="stat-card">
            <div>
              <div class="stat-label">今日搜索</div>
              <div class="stat-value">{{ user?.search_count_today || 0 }}</div>
              <div class="stat-meta">每日上限 {{ user?.daily_search_limit || '跟随全局' }}</div>
            </div>
            <div class="stat-icon"><AdminIcon name="search" :size="19" /></div>
          </div>
          <div class="stat-card">
            <div>
              <div class="stat-label">账号角色</div>
              <div class="stat-value" style="font-size: 1rem">
                <span class="badge" :class="user?.role === 'admin' ? 'badge-primary' : ''">{{ user?.role === 'admin' ? '管理员' : '普通用户' }}</span>
              </div>
              <div class="stat-meta">状态 {{ user?.status === 'active' ? '正常' : '已禁用' }}</div>
            </div>
            <div class="stat-icon violet"><AdminIcon name="shield" :size="19" /></div>
          </div>
          <div class="stat-card">
            <div>
              <div class="stat-label">API 令牌</div>
              <div class="stat-value" style="font-size: 1rem">
                <span class="badge" :class="tokenMeta ? 'badge-success' : ''">{{ tokenMeta ? '已生成' : '未生成' }}</span>
              </div>
              <div class="stat-meta">用于脚本或第三方客户端</div>
            </div>
            <div class="stat-icon emerald"><AdminIcon name="key" :size="19" /></div>
          </div>
          <div class="stat-card">
            <div>
              <div class="stat-label">OAuth 绑定</div>
              <div class="stat-value">{{ bindings.length }}</div>
              <div class="stat-meta">第三方登录方式</div>
            </div>
            <div class="stat-icon amber"><AdminIcon name="external" :size="19" /></div>
          </div>
        </div>

        <div class="user-grid">
          <section class="section-card">
            <div class="section-head">
              <div>
                <div class="section-title"><AdminIcon name="lock" :size="16" /> 账号安全</div>
                <div class="section-desc">密码与登录方式状态</div>
              </div>
            </div>
            <div class="section-body">
              <div class="detail-list">
                <div class="detail-row"><span>登录密码</span><strong>
                  <span class="badge" :class="user?.must_change_password ? 'badge-warning' : 'badge-success'">
                    {{ user?.must_change_password ? '默认密码，待修改' : '已设置' }}
                  </span>
                </strong></div>
                <div class="detail-row"><span>邮箱</span><strong>{{ user?.email || '未绑定' }}</strong></div>
                <div class="detail-row"><span>OAuth 绑定</span><strong>{{ bindings.length ? bindings.map((b) => b.provider_name).join('、') : '无' }}</strong></div>
              </div>
              <button class="btn" style="margin-top: 0.75rem" type="button" @click="active = 'security'">
                <AdminIcon name="shield" :size="15" /> 管理安全设置
              </button>
            </div>
          </section>

          <section class="section-card">
            <div class="section-head">
              <div>
                <div class="section-title">快捷入口</div>
                <div class="section-desc">常用账户操作</div>
              </div>
            </div>
            <div class="section-body quick-grid">
              <button type="button" @click="active = 'profile'"><AdminIcon name="user" :size="16" /> 个人资料</button>
              <button type="button" @click="active = 'security'"><AdminIcon name="lock" :size="16" /> 修改密码</button>
              <button type="button" @click="active = 'token'"><AdminIcon name="key" :size="16" /> API 令牌</button>
              <button type="button" @click="active = 'oauth'"><AdminIcon name="external" :size="16" /> 第三方绑定</button>
            </div>
          </section>
        </div>
      </template>

      <!-- 个人资料 -->
      <section v-else-if="active === 'profile'" class="section-card">
        <div class="section-head">
          <div>
            <div class="section-title"><AdminIcon name="user" :size="16" /> 个人资料</div>
            <div class="section-desc">用户名用于登录；邮箱可用于登录与接收通知</div>
          </div>
        </div>
        <div class="section-body" style="max-width: 34rem">
          <div class="field">
            <label>用户名</label>
            <input v-model="profile.username" class="input" maxlength="32" />
          </div>
          <div class="field" style="margin-top: 0.75rem">
            <label>邮箱</label>
            <input v-model="profile.email" class="input" type="email" placeholder="可选" />
          </div>
          <button class="btn btn-primary" style="margin-top: 0.85rem" type="button" :disabled="savingProfile" @click="saveProfile">
            <AdminIcon name="check" :size="15" /> {{ savingProfile ? '保存中…' : '保存资料' }}
          </button>
        </div>
      </section>

      <!-- 安全设置 -->
      <section v-else-if="active === 'security'" class="section-card">
        <div class="section-head">
          <div>
            <div class="section-title"><AdminIcon name="lock" :size="16" /> 修改密码</div>
            <div class="section-desc">修改后其他设备上的会话会退出</div>
          </div>
        </div>
        <div class="section-body" style="max-width: 34rem">
          <div class="field">
            <label>当前密码</label>
            <input v-model="password.current" class="input" type="password" placeholder="OAuth 注册的账号可留空" autocomplete="current-password" />
          </div>
          <div class="field" style="margin-top: 0.75rem">
            <label>新密码</label>
            <input v-model="password.next" class="input" type="password" placeholder="至少 8 位" autocomplete="new-password" />
          </div>
          <button class="btn btn-primary" style="margin-top: 0.85rem" type="button" :disabled="savingPassword" @click="savePassword">
            <AdminIcon name="shield" :size="15" /> {{ savingPassword ? '提交中…' : '更新密码' }}
          </button>
        </div>
      </section>

      <!-- API 令牌 -->
      <section v-else-if="active === 'token'" class="section-card">
        <div class="section-head">
          <div>
            <div class="section-title"><AdminIcon name="key" :size="16" /> API 令牌</div>
            <div class="section-desc">用于脚本或第三方客户端调用受保护的搜索 API</div>
          </div>
          <span class="badge" :class="tokenMeta ? 'badge-success' : ''">{{ tokenMeta ? '已生成' : '未生成' }}</span>
        </div>
        <div class="section-body" style="max-width: 46rem">
          <p class="api-hint">
            请求头示例：<code class="mono">Authorization: Bearer ps_...</code><br />
            接口：<code class="mono">GET /api/search?kw=关键词</code>
          </p>
          <div v-if="tokenMeta" class="token-box">
            <div>
              <div class="field-label">当前令牌</div>
              <div class="mono">{{ tokenMeta.prefix }}••••••••</div>
              <div class="hint">创建于 {{ shortTime(tokenMeta.created_at) }}<template v-if="tokenMeta.last_used_at"> · 最近使用 {{ shortTime(tokenMeta.last_used_at) }}</template></div>
            </div>
            <button class="btn btn-sm" type="button" @click="resetToken">
              <AdminIcon name="refresh" :size="14" /> 重置
            </button>
          </div>
          <div v-else class="empty-state" style="padding: 1.2rem 0">
            <AdminIcon name="key" :size="22" />
            <strong>尚未生成 API 令牌</strong>
            <span>生成后会完整显示一次，请立即保存</span>
          </div>
          <div v-if="newToken" class="new-token">
            <div class="field-label">新令牌（仅显示一次）</div>
            <div class="new-token-row">
              <input class="input mono" :value="newToken" readonly @focus="($event.target as HTMLInputElement).select()" />
              <button class="btn btn-sm" type="button" @click="copyToken"><AdminIcon name="copy" :size="14" /> 复制</button>
            </div>
          </div>
          <button v-if="!tokenMeta" class="btn btn-primary" style="margin-top: 0.75rem" type="button" :disabled="resetting" @click="resetToken">
            <AdminIcon name="plus" :size="15" /> {{ resetting ? '生成中…' : '生成 API 令牌' }}
          </button>
        </div>
      </section>

      <!-- 第三方绑定 -->
      <section v-else-if="active === 'oauth'" class="section-card">
        <div class="section-head">
          <div>
            <div class="section-title"><AdminIcon name="external" :size="16" /> 第三方账号绑定</div>
            <div class="section-desc">绑定后可以使用 OAuth2 登录；至少保留一种登录方式</div>
          </div>
        </div>
        <div class="section-body" style="max-width: 40rem">
          <div v-if="bindings.length" class="binding-list">
            <div v-for="item in bindings" :key="item.provider_id" class="binding-row">
              <div class="binding-icon">{{ item.provider_name.slice(0, 1).toUpperCase() }}</div>
              <div class="binding-copy">
                <strong>{{ item.provider_name }}</strong>
                <span>{{ item.email || '已绑定' }}</span>
              </div>
              <button class="btn btn-sm btn-danger" type="button" @click="removeBinding(item.provider_id)">解绑</button>
            </div>
          </div>
          <p v-else class="hint" style="margin-bottom: 0.6rem">尚未绑定第三方账号。</p>

          <div v-if="bindableProviders.length" class="bind-actions">
            <span class="field-label">可绑定</span>
            <a v-for="provider in bindableProviders" :key="provider.slug" class="btn btn-sm" :href="oauthStartUrl(provider.slug, 'bind')">
              <AdminIcon name="external" :size="14" /> {{ provider.name }}
            </a>
          </div>
          <p v-else-if="!bindings.length" class="hint">管理员尚未开启 OAuth 登录/注册，或所有服务商都已绑定。</p>
        </div>
      </section>
    </div>
  </AdminShell>
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
import AdminShell from './admin/AdminShell.vue';
import AdminIcon from './admin/AdminIcon.vue';
import { toast } from '@/lib/toast';

const emit = defineEmits<{ back: []; logout: []; admin: [] }>();

const nav = [
  { key: 'overview', label: '我的概览', icon: 'dashboard' },
  { key: 'profile', label: '个人资料', icon: 'user' },
  { key: 'security', label: '安全设置', icon: 'lock' },
  { key: 'token', label: 'API 令牌', icon: 'key' },
  { key: 'oauth', label: '第三方绑定', icon: 'external' },
];

const pages: Record<string, { title: string; subtitle: string }> = {
  overview: { title: '我的概览', subtitle: '账号状态、搜索配额与快捷入口' },
  profile: { title: '个人资料', subtitle: '用户名与邮箱设置' },
  security: { title: '安全设置', subtitle: '修改登录密码' },
  token: { title: 'API 令牌', subtitle: '个人访问令牌管理' },
  oauth: { title: '第三方绑定', subtitle: 'OAuth2 登录方式' },
};

const active = ref('overview');
const current = computed(() => pages[active.value] || pages.overview);
const user = computed(() => authState.user);
const limitText = computed(() => {
  const limit = user.value?.daily_search_limit || 0;
  return limit > 0 ? ` / ${limit}` : '（不限）';
});

const bindings = ref<OAuthAccount[]>([]);
const bindableProviders = computed(() => {
  const bound = new Set(bindings.value.map((item) => item.provider_slug));
  return (authState.config?.oauth_providers || []).filter((provider) => !bound.has(provider.slug));
});

const profile = reactive({ username: authState.user?.username || '', email: authState.user?.email || '' });
const password = reactive({ current: '', next: '' });
const tokenMeta = ref<APIToken | null>(null);
const newToken = ref('');
const savingProfile = ref(false);
const savingPassword = ref(false);
const resetting = ref(false);

function shortTime(value?: string) {
  return value ? new Date(value).toLocaleString() : '';
}
function shortDate(value?: string) {
  return value ? new Date(value).toLocaleDateString() : '—';
}

const onSwitchMode = (mode: 'admin' | 'user') => {
  if (mode === 'admin') emit('admin');
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
  savingProfile.value = true;
  try {
    await updateProfile(profile.username.trim(), profile.email.trim());
    await refreshMe();
    toast.success('资料已保存');
  } catch (err: any) {
    toast.error('保存失败', err.response?.data?.error || '');
  } finally {
    savingProfile.value = false;
  }
};

const savePassword = async () => {
  if (password.next.length < 8) {
    toast.error('新密码至少 8 位');
    return;
  }
  savingPassword.value = true;
  try {
    await changePassword(password.current, password.next);
    password.current = '';
    password.next = '';
    await refreshMe();
    await Promise.all([loadToken(), loadBindings()]);
    toast.success('密码已更新', '其他设备的会话已退出');
  } catch (err: any) {
    toast.error('修改密码失败', err.response?.data?.error || '');
  } finally {
    savingPassword.value = false;
  }
};

const resetToken = async () => {
  if (tokenMeta.value && !window.confirm('重置后旧令牌立即失效，确定继续？')) return;
  resetting.value = true;
  try {
    const res = await resetAPIToken();
    newToken.value = res.token;
    await loadToken();
    toast.success('API 令牌已生成', '请立即保存，离开页面后将无法再次查看完整令牌');
  } catch (err: any) {
    toast.error('生成失败', err.response?.data?.error || '');
  } finally {
    resetting.value = false;
  }
};

const copyToken = async () => {
  try {
    await navigator.clipboard.writeText(newToken.value);
    toast.success('已复制到剪贴板');
  } catch {
    toast.error('复制失败，请手动选择令牌');
  }
};

const removeBinding = async (providerId: number) => {
  if (!window.confirm('确定解绑该第三方账号？')) return;
  try {
    await unbindOAuth(providerId);
    await loadBindings();
    await refreshMe();
    toast.success('已解绑');
  } catch (err: any) {
    toast.error('解绑失败', err.response?.data?.error || '');
  }
};
</script>

<style scoped>
.user-hero {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 1rem;
  padding: 1.15rem 1.25rem;
  border: 1px solid hsl(var(--border));
  border-radius: var(--console-radius);
  background:
    radial-gradient(circle at 90% -20%, rgba(99, 102, 241, 0.12), transparent 16rem),
    hsl(var(--background-light, 0 0% 100%) / 0.94);
  box-shadow: var(--console-shadow);
}
.hero-main { display: flex; align-items: center; gap: 0.9rem; min-width: 0; }
.hero-avatar {
  display: grid;
  place-items: center;
  width: 3.2rem;
  height: 3.2rem;
  border-radius: 1rem;
  background: linear-gradient(135deg, hsl(var(--primary)), #818cf8);
  color: #fff;
  font-size: 1.15rem;
  font-weight: 750;
  box-shadow: 0 10px 22px rgba(79, 70, 229, 0.25);
}
.hero-copy { min-width: 0; }
.hero-title { display: flex; align-items: center; gap: 0.5rem; flex-wrap: wrap; }
.hero-title h2 { font-size: 1.2rem; font-weight: 750; letter-spacing: -0.02em; }
.hero-copy p { margin-top: 0.15rem; font-size: 0.75rem; color: hsl(var(--muted-foreground)); }
.hero-meta { display: flex; flex-wrap: wrap; gap: 0.8rem; margin-top: 0.4rem; font-size: 0.7rem; color: hsl(var(--muted-foreground)); }
.hero-meta span { display: inline-flex; align-items: center; gap: 0.3rem; }
.hero-actions { display: flex; align-items: center; gap: 0.4rem; flex-wrap: wrap; }
.user-grid { display: grid; grid-template-columns: 1fr 1fr; gap: 1rem; align-items: start; }
.quick-grid { display: grid; grid-template-columns: 1fr 1fr; gap: 0.5rem; }
.quick-grid button {
  display: flex; align-items: center; gap: 0.45rem; padding: 0.65rem;
  border: 1px solid hsl(var(--border)); border-radius: 0.6rem;
  background: hsl(var(--muted) / 0.3); color: hsl(var(--foreground));
  font-size: 0.75rem; font-weight: 550; cursor: pointer; text-align: left;
}
.quick-grid button:hover { border-color: hsl(var(--primary) / 0.5); color: hsl(var(--primary)); background: hsl(var(--primary) / 0.06); }
.api-hint { font-size: 0.72rem; line-height: 1.6; color: hsl(var(--muted-foreground)); margin-bottom: 0.75rem; }
.api-hint code { background: hsl(var(--muted)); padding: 0.1rem 0.3rem; border-radius: 0.3rem; }
.token-box {
  display: flex; align-items: center; justify-content: space-between; gap: 0.75rem;
  padding: 0.7rem 0.8rem; border: 1px dashed hsl(var(--border)); border-radius: 0.65rem;
  background: hsl(var(--muted) / 0.3);
}
.new-token { margin-top: 0.75rem; padding: 0.7rem 0.8rem; border: 1px solid #a7f3d0; border-radius: 0.65rem; background: #ecfdf5; }
.new-token-row { display: flex; gap: 0.4rem; margin-top: 0.35rem; }
.new-token-row .input { flex: 1; }
.binding-list { display: flex; flex-direction: column; gap: 0.5rem; }
.binding-row { display: flex; align-items: center; gap: 0.6rem; padding: 0.55rem 0; border-bottom: 1px dashed hsl(var(--border)); }
.binding-row:last-child { border-bottom: 0; }
.binding-icon {
  display: grid; place-items: center; width: 2rem; height: 2rem; border-radius: 0.6rem;
  background: #eef2ff; color: #4338ca; font-size: 0.75rem; font-weight: 750;
}
.binding-copy { flex: 1; min-width: 0; }
.binding-copy strong { display: block; font-size: 0.8rem; }
.binding-copy span { display: block; font-size: 0.68rem; color: hsl(var(--muted-foreground)); overflow: hidden; text-overflow: ellipsis; white-space: nowrap; }
.bind-actions { display: flex; align-items: center; gap: 0.4rem; flex-wrap: wrap; margin-top: 0.6rem; }
.hint { font-size: 0.7rem; color: hsl(var(--muted-foreground)); }

@media (max-width: 860px) {
  .user-grid { grid-template-columns: 1fr; }
  .user-hero { flex-direction: column; align-items: flex-start; }
  .hero-actions { width: 100%; }
  .hero-actions .btn { flex: 1; }
}
</style>
