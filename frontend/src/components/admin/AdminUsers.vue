<template>
  <div class="page-stack">
    <section class="section-card">
      <div class="section-head">
        <div>
          <div class="section-title">用户列表</div>
          <div class="section-desc">共 {{ total }} 个账号，支持按用户名、邮箱、角色与状态筛选</div>
        </div>
        <button class="btn btn-primary btn-sm" type="button" @click="openCreate">
          <AdminIcon name="plus" :size="15" /> 新建用户
        </button>
      </div>

      <div class="section-body" style="padding-bottom: 0.6rem">
        <div class="toolbar">
          <input
            v-model="filters.q"
            class="input"
            style="max-width: 16rem"
            placeholder="搜索用户名或邮箱"
            @keydown.enter="load(1)"
          />
          <select v-model="filters.role" class="select" style="max-width: 8rem" @change="load(1)">
            <option value="">全部角色</option>
            <option value="admin">管理员</option>
            <option value="user">普通用户</option>
          </select>
          <select v-model="filters.status" class="select" style="max-width: 8rem" @change="load(1)">
            <option value="">全部状态</option>
            <option value="active">正常</option>
            <option value="disabled">禁用</option>
          </select>
          <button class="btn btn-sm" type="button" :disabled="loading" @click="load(1)">
            <AdminIcon name="search" :size="14" /> {{ loading ? '查询中' : '查询' }}
          </button>
        </div>
      </div>

      <div v-if="loading && !users.length" class="section-body">
        <div class="skeleton" style="width: 100%; margin-bottom: 0.6rem" />
        <div class="skeleton" style="width: 92%; margin-bottom: 0.6rem" />
        <div class="skeleton" style="width: 96%" />
      </div>
      <div v-else-if="!users.length" class="empty-state">
        <AdminIcon name="users" :size="24" />
        <strong>没有匹配的用户</strong>
        <span>调整筛选条件，或创建一个新账号</span>
      </div>
      <div v-else class="table-wrap">
        <table class="data-table">
          <thead>
            <tr>
              <th>用户</th>
              <th>角色</th>
              <th>状态</th>
              <th>今日（搜索/API）</th>
              <th>每日额度（搜索/API）</th>
              <th class="hide-sm">最近登录</th>
              <th style="text-align: right">操作</th>
            </tr>
          </thead>
          <tbody>
            <tr v-for="u in users" :key="u.id">
              <td>
                <div class="user-cell">
                  <span class="mini-avatar" :class="{ admin: u.role === 'admin' }">{{ (u.username || '?').slice(0, 1).toUpperCase() }}</span>
                  <div>
                    <div class="cell-strong">{{ u.username }} <span v-if="u.must_change_password" class="badge badge-warning">待改密</span></div>
                    <div class="cell-sub">{{ u.email || '未绑定邮箱' }} · ID {{ u.id }}</div>
                  </div>
                </div>
              </td>
              <td><span class="badge" :class="u.role === 'admin' ? 'badge-primary' : ''">{{ u.role === 'admin' ? '管理员' : '普通用户' }}</span></td>
              <td><span class="badge" :class="u.status === 'active' ? 'badge-success' : 'badge-danger'">{{ u.status === 'active' ? '正常' : '禁用' }}</span></td>
              <td class="mono">{{ u.search_count_today || 0 }} / {{ u.api_search_count_today || 0 }}</td>
              <td>
                <span class="quota-pill">{{ u.daily_search_limit || '不限' }}</span>
                <span class="quota-sep">/</span>
                <span class="quota-pill">{{ u.api_daily_search_limit || '跟随' }}</span>
              </td>
              <td class="hide-sm cell-sub">{{ u.last_login_at ? shortTime(u.last_login_at) : '从未登录' }}</td>
              <td>
                <div class="cell-actions" style="justify-content: flex-end">
                  <button class="btn btn-ghost btn-sm" type="button" @click="openEdit(u)">编辑</button>
                  <button class="btn btn-ghost btn-sm" type="button" @click="resetPassword(u)">重置密码</button>
                  <button class="btn btn-ghost btn-sm" type="button" style="color: #b91c1c" :disabled="u.id === authState.user?.id" @click="removeUser(u)">删除</button>
                </div>
              </td>
            </tr>
          </tbody>
        </table>
      </div>

      <div class="pager">
        <span>第 {{ page }} 页 · 共 {{ total }} 个</span>
        <button class="btn btn-sm" type="button" :disabled="page <= 1" @click="load(page - 1)">上一页</button>
        <button class="btn btn-sm" type="button" :disabled="page * pageSize >= total" @click="load(page + 1)">下一页</button>
      </div>
    </section>

    <Teleport to="body">
      <div v-if="modal.open" class="modal-backdrop" @click.self="modal.open = false">
        <div class="modal">
          <div class="modal-head">
            <div>
              <h3>{{ modal.mode === 'create' ? '新建用户' : `编辑用户 · ${modal.original?.username || ''}` }}</h3>
              <p>{{ modal.mode === 'create' ? '创建后用户即可使用账号密码登录' : '修改角色、状态、配额或密码' }}</p>
            </div>
            <button class="icon-btn" type="button" aria-label="关闭" @click="modal.open = false">
              <AdminIcon name="close" :size="16" />
            </button>
          </div>
          <div class="modal-body">
            <div class="form-grid">
              <div class="field">
                <label>用户名</label>
                <input v-model="form.username" class="input" placeholder="2-32 位字母、数字、下划线" />
              </div>
              <div class="field">
                <label>邮箱</label>
                <input v-model="form.email" class="input" type="email" placeholder="可选" />
              </div>
              <div class="field">
                <label>角色</label>
                <div class="role-readonly">
                  <span class="badge" :class="modal.original?.role === 'admin' ? 'badge-primary' : ''">
                    {{ modal.original?.role === 'admin' ? '管理员' : '普通用户' }}
                  </span>
                  <span class="hint">管理员全局唯一，角色不可通过用户列表调整</span>
                </div>
              </div>
              <div class="field" v-if="modal.mode === 'edit'">
                <label>状态</label>
                <select v-model="form.status" class="select">
                  <option value="active">正常</option>
                  <option value="disabled">禁用</option>
                </select>
              </div>
              <div class="field">
                <label>每日搜索额度（网页 + API）</label>
                <input v-model.number="form.daily_search_limit" class="input" type="number" min="0" />
                <span class="hint">0 表示跟随全局默认，不单独限制</span>
              </div>
              <div class="field">
                <label>每日 API 额度</label>
                <input v-model.number="form.api_daily_search_limit" class="input" type="number" min="0" />
                <span class="hint">仅统计 /api/search；0 表示与每日搜索额度共享</span>
              </div>
              <div class="field">
                <label>{{ modal.mode === 'create' ? '初始密码' : '新密码' }}</label>
                <input v-model="form.password" class="input" type="password" :placeholder="modal.mode === 'edit' ? '留空则不修改' : '至少 8 位'" />
              </div>
            </div>
          </div>
          <div class="modal-foot">
            <button class="btn" type="button" @click="modal.open = false">取消</button>
            <button class="btn btn-primary" type="button" :disabled="saving" @click="save">
              {{ saving ? '保存中…' : '保存' }}
            </button>
          </div>
        </div>
      </div>
    </Teleport>
  </div>
</template>

<script setup lang="ts">
import { onMounted, reactive, ref } from 'vue';
import {
  adminCreateUser,
  adminDeleteUser,
  adminListUsers,
  adminUpdateUser,
  type AuthUser,
} from '@/api/auth';
import { authState } from '@/stores/auth';
import AdminIcon from './AdminIcon.vue';
import { toast } from '@/lib/toast';

const users = ref<AuthUser[]>([]);
const total = ref(0);
const page = ref(1);
const pageSize = 12;
const loading = ref(false);
const saving = ref(false);

const filters = reactive({ q: '', role: '', status: '' });

const modal = reactive<{ open: boolean; mode: 'create' | 'edit'; original: AuthUser | null }>({
  open: false,
  mode: 'create',
  original: null,
});

const form = reactive({
  username: '',
  email: '',
  role: 'user',
  status: 'active',
  daily_search_limit: 0,
  api_daily_search_limit: 0,
  password: '',
});

function shortTime(value: string) {
  return value ? new Date(value).toLocaleString() : '';
}

function resetForm() {
  form.username = '';
  form.email = '';
  form.role = 'user';
  form.status = 'active';
  form.daily_search_limit = 0;
  form.api_daily_search_limit = 0;
  form.password = '';
}

const load = async (target = page.value) => {
  loading.value = true;
  try {
    const res = await adminListUsers(filters.q.trim(), target, pageSize, filters.role, filters.status);
    users.value = res.items || [];
    total.value = res.total || 0;
    page.value = res.page || target;
  } catch (err: any) {
    toast.error('加载用户失败', err.response?.data?.error || '');
  } finally {
    loading.value = false;
  }
};

function openCreate() {
  resetForm();
  modal.mode = 'create';
  modal.original = null;
  modal.open = true;
}

function openEdit(user: AuthUser) {
  resetForm();
  modal.mode = 'edit';
  modal.original = user;
  form.username = user.username;
  form.email = user.email || '';
  form.role = user.role;
  form.status = user.status;
  form.daily_search_limit = user.daily_search_limit || 0;
  form.api_daily_search_limit = user.api_daily_search_limit || 0;
  modal.open = true;
}

async function save() {
  if (!form.username.trim()) {
    toast.error('请填写用户名');
    return;
  }
  if (modal.mode === 'create' && form.password.length < 8) {
    toast.error('初始密码至少 8 位');
    return;
  }
  saving.value = true;
  try {
    if (modal.mode === 'create') {
      await adminCreateUser({
        username: form.username.trim(),
        email: form.email.trim(),
        password: form.password,
        role: 'user',
        daily_search_limit: form.daily_search_limit || 0,
        api_daily_search_limit: form.api_daily_search_limit || 0,
      });
      toast.success('用户已创建');
    } else if (modal.original) {
      const payload: Record<string, unknown> = {
        username: form.username.trim(),
        email: form.email.trim(),
        status: form.status,
        daily_search_limit: form.daily_search_limit || 0,
        api_daily_search_limit: form.api_daily_search_limit || 0,
      };
      if (form.password) payload.password = form.password;
      await adminUpdateUser(modal.original.id, payload);
      toast.success('用户已更新');
    }
    modal.open = false;
    await load(modal.mode === 'create' ? 1 : page.value);
  } catch (err: any) {
    toast.error('保存失败', err.response?.data?.error || '');
  } finally {
    saving.value = false;
  }
}

async function resetPassword(user: AuthUser) {
  const next = window.prompt(`为「${user.username}」设置新密码（至少 8 位）`);
  if (!next) return;
  if (next.length < 8) {
    toast.error('密码至少 8 位');
    return;
  }
  try {
    await adminUpdateUser(user.id, { password: next });
    toast.success('密码已重置', `用户 ${user.username} 需要重新登录`);
    await load();
  } catch (err: any) {
    toast.error('重置密码失败', err.response?.data?.error || '');
  }
}

async function removeUser(user: AuthUser) {
  if (!window.confirm(`确定删除用户「${user.username}」？该操作不可恢复。`)) return;
  try {
    await adminDeleteUser(user.id);
    toast.success('用户已删除');
    await load();
  } catch (err: any) {
    toast.error('删除失败', err.response?.data?.error || '');
  }
}

onMounted(() => load(1));
</script>

<style scoped>
.user-cell { display: flex; align-items: center; gap: 0.55rem; }
.mini-avatar {
  display: grid;
  place-items: center;
  width: 1.9rem;
  height: 1.9rem;
  border-radius: 999px;
  background: hsl(var(--muted));
  color: hsl(var(--muted-foreground));
  font-size: 0.7rem;
  font-weight: 700;
  flex: 0 0 auto;
}
.mini-avatar.admin { background: hsl(var(--primary) / 0.12); color: hsl(var(--primary)); }
.quota-pill {
  display: inline-block;
  padding: 0.08rem 0.35rem;
  border-radius: 0.35rem;
  background: hsl(var(--muted));
  font-size: 0.68rem;
  font-variant-numeric: tabular-nums;
}
.quota-sep { margin: 0 0.15rem; color: hsl(var(--muted-foreground)); font-size: 0.66rem; }
.role-readonly { display: flex; align-items: center; gap: 0.5rem; min-height: 2.5rem; flex-wrap: wrap; }
</style>
