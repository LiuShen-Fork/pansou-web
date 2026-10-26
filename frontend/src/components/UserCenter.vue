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
                <span class="badge" :class="tokens.length ? 'badge-success' : ''">{{ tokens.length ? tokens.length + ' 个' : '未创建' }}</span>
              </div>
              <div class="stat-meta">共享每日搜索额度</div>
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
        <div class="section-head token-head">
          <div>
            <div class="section-title"><AdminIcon name="key" :size="16" /> API 令牌</div>
            <div class="section-desc">用于脚本或第三方客户端；每个令牌只完整显示一次，所有令牌共享账号额度</div>
          </div>
          <div class="token-head-actions">
            <span class="badge" :class="tokens.length ? 'badge-success' : ''">{{ tokens.length }} / {{ tokenMax }}</span>
            <button class="btn btn-primary btn-sm" type="button" :disabled="tokens.length >= tokenMax" @click="openCreateToken">
              <AdminIcon name="plus" :size="14" /> 添加令牌
            </button>
          </div>
        </div>

        <div class="section-body token-body">
          <div class="api-guide">
            <div class="api-guide-head">
              <div>
                <strong>接口调用</strong>
                <span>兼容上游 PanSou 搜索接口，可直接接入视频源、机器人等应用</span>
              </div>
              <span class="badge badge-primary">Bearer</span>
            </div>
            <div class="api-row">
              <span class="api-label">接口</span>
              <code class="mono">GET /api/search?kw=关键词</code>
              <button class="api-copy" type="button" title="复制接口地址" @click="copyText('GET /api/search?kw=关键词')">
                <AdminIcon name="copy" :size="13" />
              </button>
            </div>
            <div class="api-row">
              <span class="api-label">请求头</span>
              <code class="mono">Authorization: Bearer &lt;你的令牌&gt;</code>
              <button class="api-copy" type="button" title="复制请求头" @click="copyText('Authorization: Bearer <你的令牌>')">
                <AdminIcon name="copy" :size="13" />
              </button>
            </div>
          </div>

          <div class="token-quota">
            <div>
              <span>今日 API 调用</span>
              <strong>{{ user?.api_search_count_today || 0 }}<template v-if="apiLimit"> / {{ apiLimit }}</template></strong>
            </div>
            <div>
              <span>每日搜索总额度</span>
              <strong>{{ user?.daily_search_limit || '跟随全站' }}</strong>
            </div>
            <p>所有令牌共享同一账号额度；API 独立额度由管理员设置。</p>
          </div>

          <div class="token-list-head">
            <span>已创建的令牌</span>
            <small v-if="tokens.length">共 {{ tokens.length }} 个</small>
          </div>

          <div v-if="tokens.length" class="token-list">
            <div v-for="item in tokens" :key="item.id" class="token-row">
              <div class="token-icon"><AdminIcon name="key" :size="16" /></div>
              <div class="token-copy">
                <div class="token-title">
                  <strong>{{ item.name }}</strong>
                  <span class="token-mask mono">{{ item.prefix }}••••••••</span>
                </div>
                <div class="token-times">
                  <span><AdminIcon name="clock" :size="12" /> 创建 {{ shortTime(item.created_at) }}</span>
                  <span v-if="item.last_used_at"><AdminIcon name="refresh" :size="12" /> 最近使用 {{ shortTime(item.last_used_at) }}</span>
                  <span v-else class="token-unused">尚未使用</span>
                </div>
              </div>
              <button class="token-delete" type="button" title="删除令牌" @click="removeToken(item)">
                <AdminIcon name="trash" :size="14" />
              </button>
            </div>
          </div>
          <div v-else class="empty-state token-empty">
            <div class="token-empty-icon"><AdminIcon name="key" :size="22" /></div>
            <strong>还没有 API 令牌</strong>
            <span>点击右上角「添加令牌」，创建后可在此查看创建与使用时间</span>
            <button class="btn btn-primary btn-sm" type="button" style="margin-top: 0.6rem" @click="openCreateToken">
              <AdminIcon name="plus" :size="14" /> 添加第一个令牌
            </button>
          </div>
          <p v-if="tokens.length >= tokenMax" class="hint" style="text-align: center; margin-top: 0.8rem">
            已达到 {{ tokenMax }} 个令牌上限，请先删除不再使用的令牌。
          </p>
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

      <!-- 令牌创建 / 一次性展示弹窗 -->
      <Teleport to="body">
        <div v-if="tokenModal" class="modal-backdrop" @click.self="closeTokenModal">
          <div class="modal token-modal" role="dialog" aria-modal="true">
            <template v-if="tokenModal === 'create'">
              <div class="modal-head">
                <div>
                  <h3>添加 API 令牌</h3>
                  <p>给令牌起一个便于识别的名称，例如「视频源脚本」</p>
                </div>
                <button class="icon-btn" type="button" aria-label="关闭" @click="closeTokenModal">
                  <AdminIcon name="close" :size="16" />
                </button>
              </div>
              <div class="modal-body">
                <div class="field">
                  <label>令牌名称</label>
                  <input v-model="newTokenName" class="input" maxlength="32" placeholder="例如：视频源脚本（可选）" @keyup.enter="generateToken" />
                  <span class="hint">留空将使用「默认令牌」；名称不超过 32 个字符。</span>
                </div>
              </div>
              <div class="modal-foot">
                <button class="btn" type="button" @click="closeTokenModal">取消</button>
                <button class="btn btn-primary" type="button" :disabled="creatingToken" @click="generateToken">
                  <AdminIcon name="plus" :size="14" /> {{ creatingToken ? '创建中…' : '创建令牌' }}
                </button>
              </div>
            </template>

            <template v-else>
              <div class="modal-head">
                <div>
                  <h3>令牌已创建</h3>
                  <p>请立即复制保存，关闭后将无法再次查看完整令牌</p>
                </div>
                <span class="badge badge-warning">仅显示一次</span>
              </div>
              <div class="modal-body">
                <div class="token-reveal">
                  <div class="token-reveal-label">
                    <span>{{ createdTokenName || '默认令牌' }}</span>
                    <span class="token-reveal-warn">离开此弹窗后不可见</span>
                  </div>
                  <div class="token-reveal-box">
                    <code class="mono">{{ createdTokenRaw }}</code>
                    <button class="btn btn-sm" type="button" @click="copyToken">
                      <AdminIcon name="copy" :size="14" /> 复制
                    </button>
                  </div>
                </div>
                <p class="hint token-reveal-hint">
                  请求时在 Header 中携带 <code>Authorization: Bearer &lt;令牌&gt;</code>。令牌遵循账号的每日 API 额度。
                </p>
              </div>
              <div class="modal-foot">
                <button class="btn btn-primary" type="button" @click="closeTokenModal">我已保存，完成</button>
              </div>
            </template>
          </div>
        </div>
      </Teleport>
    </div>
  </AdminShell>
</template>

<script setup lang="ts">
import { computed, onMounted, reactive, ref } from 'vue';
import {
  changePassword,
  createAPIToken,
  deleteAPIToken,
  getOAuthBindings,
  listAPITokens,
  oauthStartUrl,
  unbindOAuth,
  updateProfile,
  type APIToken,
  type OAuthAccount,
} from '@/api/auth';
import { authState, refreshMe } from '@/stores/auth';
import AdminShell from './admin/AdminShell.vue';
import AdminIcon from './admin/AdminIcon.vue';
import { consoleNav, consolePages } from '@/console/config';
import { toast } from '@/lib/toast';

const emit = defineEmits<{ back: []; logout: []; admin: [] }>();

const nav = consoleNav.user;
const active = ref('overview');
const current = computed(() => consolePages.user[active.value] || consolePages.user.overview);
const user = computed(() => authState.user);
const limitText = computed(() => {
  const limit = user.value?.daily_search_limit || 0;
  return limit > 0 ? ` / ${limit}` : '（不限）';
});
const apiLimit = computed(() => user.value?.api_daily_search_limit || 0);

const bindings = ref<OAuthAccount[]>([]);
const bindableProviders = computed(() => {
  const bound = new Set(bindings.value.map((item) => item.provider_slug));
  return (authState.config?.oauth_providers || []).filter((provider) => !bound.has(provider.slug));
});

const profile = reactive({ username: authState.user?.username || '', email: authState.user?.email || '' });
const password = reactive({ current: '', next: '' });
const tokens = ref<APIToken[]>([]);
const tokenMax = ref(10);
const tokenModal = ref<'create' | 'reveal' | null>(null);
const newTokenName = ref('');
const createdTokenRaw = ref('');
const createdTokenName = ref('');
const savingProfile = ref(false);
const savingPassword = ref(false);
const creatingToken = ref(false);

function shortTime(value?: string) {
  return value ? new Date(value).toLocaleString() : '';
}
function shortDate(value?: string) {
  return value ? new Date(value).toLocaleDateString() : '—';
}

const onSwitchMode = (mode: 'admin' | 'user') => {
  if (mode === 'admin') emit('admin');
};

const loadTokens = async () => {
  if (!authState.config?.auth.allow_user_api_tokens) return;
  try {
    const data = await listAPITokens();
    tokens.value = data.items;
    tokenMax.value = data.max;
  } catch {
    tokens.value = [];
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
  await Promise.all([loadTokens(), loadBindings()]);
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
    await Promise.all([loadTokens(), loadBindings()]);
    toast.success('密码已更新', '其他设备的会话已退出');
  } catch (err: any) {
    toast.error('修改密码失败', err.response?.data?.error || '');
  } finally {
    savingPassword.value = false;
  }
};

const openCreateToken = () => {
  if (tokens.value.length >= tokenMax.value) {
    toast.error('已达到令牌上限', `每个账号最多创建 ${tokenMax.value} 个令牌`);
    return;
  }
  newTokenName.value = '';
  tokenModal.value = 'create';
};

const closeTokenModal = () => {
  tokenModal.value = null;
  createdTokenRaw.value = '';
  createdTokenName.value = '';
};

const generateToken = async () => {
  if (tokens.value.length >= tokenMax.value) return;
  creatingToken.value = true;
  try {
    const res = await createAPIToken(newTokenName.value.trim());
    createdTokenRaw.value = res.token;
    createdTokenName.value = res.item.name;
    newTokenName.value = '';
    tokenModal.value = 'reveal';
    await loadTokens();
  } catch (err: any) {
    toast.error('创建失败', err.response?.data?.error || '');
  } finally {
    creatingToken.value = false;
  }
};

const removeToken = async (item: APIToken) => {
  if (!window.confirm(`删除令牌「${item.name}」后，使用该令牌的脚本将立即失效，确定继续？`)) return;
  try {
    await deleteAPIToken(item.id);
    await loadTokens();
    toast.success('令牌已删除');
  } catch (err: any) {
    toast.error('删除失败', err.response?.data?.error || '');
  }
};

const copyText = async (text: string) => {
  try {
    await navigator.clipboard.writeText(text);
    toast.success('已复制到剪贴板');
  } catch {
    toast.error('复制失败，请手动选择内容');
  }
};

const copyToken = async () => {
  if (!createdTokenRaw.value) {
    toast.error('令牌已隐藏', '完整令牌只在创建时显示一次');
    return;
  }
  await copyText(createdTokenRaw.value);
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
.user-grid { display: grid; grid-template-columns: 1fr 1fr; gap: 1rem; }
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
.token-head { align-items: center; }
.token-head-actions { display: flex; align-items: center; gap: 0.5rem; flex-wrap: wrap; justify-content: flex-end; }
.token-body { width: 100%; max-width: 46rem; margin: 0 auto; }
.api-guide {
  padding: 0.9rem 1rem;
  border: 1px solid hsl(var(--border));
  border-radius: 0.8rem;
  background:
    radial-gradient(circle at 100% 0%, hsl(var(--primary) / 0.09), transparent 14rem),
    hsl(var(--muted) / 0.22);
}
.api-guide-head { display: flex; align-items: flex-start; justify-content: space-between; gap: 0.75rem; margin-bottom: 0.7rem; }
.api-guide-head strong { display: block; font-size: 0.82rem; }
.api-guide-head span { display: block; margin-top: 0.18rem; font-size: 0.7rem; color: hsl(var(--muted-foreground)); line-height: 1.5; }
.api-row { display: flex; align-items: center; gap: 0.55rem; padding: 0.4rem 0; border-top: 1px dashed hsl(var(--border) / 0.8); }
.api-row:first-of-type { border-top: 0; }
.api-label { flex: 0 0 3.4rem; font-size: 0.68rem; color: hsl(var(--muted-foreground)); }
.api-row code { flex: 1; min-width: 0; padding: 0.32rem 0.5rem; border-radius: 0.4rem; background: hsl(var(--background-light, 0 0% 100%) / 0.8); border: 1px solid hsl(var(--border) / 0.75); font-size: 0.7rem; overflow: hidden; text-overflow: ellipsis; white-space: nowrap; }
.api-copy { display: grid; place-items: center; width: 1.75rem; height: 1.75rem; flex: 0 0 1.75rem; border: 1px solid hsl(var(--border)); border-radius: 0.45rem; background: transparent; color: hsl(var(--muted-foreground)); cursor: pointer; }
.api-copy:hover { color: hsl(var(--primary)); border-color: hsl(var(--primary) / 0.4); }
.token-quota {
  display: grid;
  grid-template-columns: 1fr 1fr;
  gap: 0.5rem 0.75rem;
  margin-top: 0.75rem;
  padding: 0.65rem 0.8rem;
  border: 1px solid hsl(var(--border));
  border-radius: 0.65rem;
  background: hsl(var(--muted) / 0.22);
}
.token-quota > div { display: flex; align-items: baseline; justify-content: space-between; gap: 0.5rem; }
.token-quota span { font-size: 0.68rem; color: hsl(var(--muted-foreground)); }
.token-quota strong { font-size: 0.78rem; font-variant-numeric: tabular-nums; }
.token-quota p { grid-column: 1 / -1; font-size: 0.64rem; color: hsl(var(--muted-foreground)); }
.token-list-head { display: flex; align-items: center; justify-content: space-between; margin: 1rem 0 0.5rem; }
.token-list-head span { font-size: 0.76rem; font-weight: 650; }
.token-list-head small { font-size: 0.66rem; color: hsl(var(--muted-foreground)); }
.token-list { display: flex; flex-direction: column; gap: 0.45rem; }
.token-row {
  display: flex; align-items: center; gap: 0.7rem;
  padding: 0.65rem 0.75rem; border: 1px solid hsl(var(--border)); border-radius: 0.7rem;
  background: hsl(var(--background-light, 0 0% 100%) / 0.7);
  transition: border-color 0.15s ease, background 0.15s ease, transform 0.15s ease;
}
.token-row:hover { border-color: hsl(var(--border-hover)); background: hsl(var(--muted) / 0.35); transform: translateY(-1px); }
.token-icon {
  display: grid; place-items: center; width: 2rem; height: 2rem; flex: 0 0 2rem;
  border-radius: 0.55rem; background: hsl(var(--primary) / 0.1); color: hsl(var(--primary));
}
.token-copy { flex: 1; min-width: 0; }
.token-title { display: flex; align-items: center; gap: 0.45rem; flex-wrap: wrap; }
.token-title strong { font-size: 0.8rem; font-weight: 650; }
.token-mask { font-size: 0.68rem; color: hsl(var(--muted-foreground)); letter-spacing: 0.02em; }
.token-times { display: flex; align-items: center; flex-wrap: wrap; gap: 0.3rem 0.8rem; margin-top: 0.22rem; font-size: 0.66rem; color: hsl(var(--muted-foreground)); }
.token-times span { display: inline-flex; align-items: center; gap: 0.22rem; }
.token-unused { color: hsl(var(--muted-foreground) / 0.7); }
.token-delete { display: grid; place-items: center; width: 1.9rem; height: 1.9rem; flex: 0 0 1.9rem; border: 1px solid transparent; border-radius: 0.5rem; background: transparent; color: hsl(var(--muted-foreground)); cursor: pointer; }
.token-delete:hover { color: hsl(var(--destructive)); background: hsl(var(--destructive) / 0.08); border-color: hsl(var(--destructive) / 0.2); }
.token-empty { padding: 2rem 1rem; }
.token-empty-icon { display: grid; place-items: center; width: 3rem; height: 3rem; margin-bottom: 0.6rem; border-radius: 999px; background: hsl(var(--muted)); color: hsl(var(--muted-foreground)); }
.token-empty strong { font-size: 0.85rem; }
.token-empty span { margin-top: 0.2rem; font-size: 0.7rem; color: hsl(var(--muted-foreground)); }
.token-modal { max-width: 30rem; width: 100%; }
.token-reveal { padding: 0.85rem; border: 1px solid hsl(var(--success, 142 76% 36%) / 0.3); border-radius: 0.7rem; background: hsl(var(--success, 142 76% 36%) / 0.06); }
.token-reveal-label { display: flex; align-items: center; justify-content: space-between; gap: 0.5rem; font-size: 0.72rem; font-weight: 650; }
.token-reveal-warn { font-size: 0.64rem; font-weight: 500; color: hsl(var(--destructive)); }
.token-reveal-box { display: flex; align-items: center; gap: 0.45rem; margin-top: 0.55rem; padding: 0.55rem 0.6rem; border: 1px solid hsl(var(--border)); border-radius: 0.55rem; background: hsl(var(--background-light, 0 0% 100%)); }
.token-reveal-box code { flex: 1; min-width: 0; font-size: 0.72rem; word-break: break-all; }
.token-reveal-hint { margin-top: 0.7rem; line-height: 1.6; }
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
@media (max-width: 620px) {
  .token-head { align-items: flex-start; flex-direction: column; }
  .token-head-actions { width: 100%; justify-content: space-between; }
  .api-row { flex-wrap: nowrap; }
  .api-row code { flex: 1 1 auto; font-size: 0.64rem; }
  .api-copy { flex: 0 0 1.65rem; width: 1.65rem; height: 1.65rem; }
  .token-times { gap: 0.2rem 0.55rem; }
  .token-quota { grid-template-columns: 1fr; }
  .token-reveal-box { flex-direction: column; align-items: stretch; }
  .token-reveal-box .btn { width: 100%; }
}
</style>
