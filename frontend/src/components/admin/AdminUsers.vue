<template>
  <div class="admin-content">
    <div v-if="error" class="admin-alert error">{{ error }}</div>
    <div v-if="message" class="admin-alert ok">{{ message }}</div>

    <div class="admin-card">
      <div class="admin-actions" style="justify-content: space-between; margin-bottom: 0.8rem">
        <h3 style="margin: 0">用户管理</h3>
        <div class="admin-actions">
          <input v-model="query" class="search-input" placeholder="搜索用户名 / 邮箱" @keydown.enter="load(1)" />
          <button class="admin-btn" type="button" @click="load(1)">搜索</button>
          <button class="admin-btn primary" type="button" @click="showCreate = !showCreate">
            {{ showCreate ? '取消' : '新建用户' }}
          </button>
        </div>
      </div>

      <div v-if="showCreate" class="create-panel">
        <div class="admin-grid">
          <div class="admin-field"><label>用户名</label><input v-model="createForm.username" placeholder="2-32 位" /></div>
          <div class="admin-field"><label>邮箱</label><input v-model="createForm.email" type="email" placeholder="可选" /></div>
          <div class="admin-field"><label>初始密码</label><input v-model="createForm.password" type="password" placeholder="至少 8 位" /></div>
          <div class="admin-field">
            <label>角色</label>
            <select v-model="createForm.role">
              <option value="user">普通用户</option>
              <option value="admin">管理员</option>
            </select>
          </div>
          <div class="admin-field">
            <label>每日搜索上限（0 = 跟随全局）</label>
            <input v-model.number="createForm.daily_search_limit" type="number" min="0" />
          </div>
        </div>
        <button class="admin-btn primary" type="button" :disabled="creating" @click="createUser">
          {{ creating ? '创建中...' : '创建' }}
        </button>
      </div>

      <div class="admin-table-wrap">
        <table class="admin-table">
          <thead>
            <tr>
              <th>ID</th><th>用户名 / 邮箱</th><th>角色</th><th>状态</th><th>今日搜索</th>
              <th>每日上限</th><th>最近登录</th><th>操作</th>
            </tr>
          </thead>
          <tbody>
            <tr v-for="u in users" :key="u.id">
              <td>{{ u.id }}</td>
              <td>
                <strong>{{ u.username }}</strong>
                <div class="admin-muted">{{ u.email || '—' }}</div>
                <span v-if="u.must_change_password" class="admin-badge warn">待改密</span>
              </td>
              <td><span class="admin-badge" :class="{ admin: u.role === 'admin' }">{{ u.role }}</span></td>
              <td><span class="admin-badge" :class="u.status === 'active' ? 'ok' : 'bad'">{{ u.status }}</span></td>
              <td>{{ u.search_count_today || 0 }}</td>
              <td>{{ u.daily_search_limit || '跟随全局' }}</td>
              <td>{{ u.last_login_at ? shortTime(u.last_login_at) : '—' }}</td>
              <td>
                <div class="admin-actions">
                  <button class="admin-btn" type="button" @click="startEdit(u)">编辑</button>
                  <button class="admin-btn danger" type="button" @click="removeUser(u)">删除</button>
                </div>
              </td>
            </tr>
            <tr v-if="!users.length">
              <td colspan="8" class="admin-muted">暂无用户。</td>
            </tr>
          </tbody>
        </table>
      </div>

      <div class="admin-actions pager">
        <button class="admin-btn" type="button" :disabled="page <= 1" @click="load(page - 1)">上一页</button>
        <span class="admin-muted">第 {{ page }} 页 / 共 {{ total }} 个用户</span>
        <button class="admin-btn" type="button" :disabled="page * pageSize >= total" @click="load(page + 1)">下一页</button>
      </div>
    </div>

    <div v-if="editing" class="admin-card">
      <h3>编辑用户：{{ editing.username }}</h3>
      <div class="admin-grid">
        <div class="admin-field"><label>用户名</label><input v-model="editForm.username" /></div>
        <div class="admin-field"><label>邮箱</label><input v-model="editForm.email" type="email" /></div>
        <div class="admin-field">
          <label>角色</label>
          <select v-model="editForm.role">
            <option value="user">普通用户</option>
            <option value="admin">管理员</option>
          </select>
        </div>
        <div class="admin-field">
          <label>状态</label>
          <select v-model="editForm.status">
            <option value="active">正常</option>
            <option value="disabled">禁用</option>
          </select>
        </div>
        <div class="admin-field">
          <label>每日搜索上限（0 = 跟随全局）</label>
          <input v-model.number="editForm.daily_search_limit" type="number" min="0" />
        </div>
        <div class="admin-field">
          <label>重置密码（留空则不修改）</label>
          <input v-model="editForm.password" type="password" placeholder="至少 8 位" />
        </div>
      </div>
      <div class="admin-actions">
        <button class="admin-btn primary" type="button" :disabled="savingEdit" @click="saveEdit">
          {{ savingEdit ? '保存中...' : '保存' }}
        </button>
        <button class="admin-btn" type="button" @click="editing = null">取消</button>
      </div>
    </div>
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

const users = ref<AuthUser[]>([]);
const total = ref(0);
const page = ref(1);
const pageSize = 20;
const query = ref('');
const loading = ref(false);
const error = ref('');
const message = ref('');

const showCreate = ref(false);
const creating = ref(false);
const createForm = reactive({
  username: '',
  email: '',
  password: '',
  role: 'user',
  daily_search_limit: 0,
});

const editing = ref<AuthUser | null>(null);
const savingEdit = ref(false);
const editForm = reactive({
  username: '',
  email: '',
  role: 'user',
  status: 'active',
  daily_search_limit: 0,
  password: '',
});

const shortTime = (s: string) => (s ? new Date(s).toLocaleString() : '');

const load = async (targetPage = page.value) => {
  loading.value = true;
  error.value = '';
  try {
    const res = await adminListUsers(query.value.trim(), targetPage, pageSize);
    users.value = res.items || [];
    total.value = res.total || 0;
    page.value = res.page || targetPage;
  } catch (err: any) {
    error.value = err.response?.data?.error || '读取用户失败';
  } finally {
    loading.value = false;
  }
};

const createUser = async () => {
  error.value = '';
  message.value = '';
  if (!createForm.username.trim() || createForm.password.length < 8) {
    error.value = '请填写用户名和至少 8 位密码';
    return;
  }
  creating.value = true;
  try {
    await adminCreateUser({
      username: createForm.username.trim(),
      email: createForm.email.trim(),
      password: createForm.password,
      role: createForm.role,
      daily_search_limit: createForm.daily_search_limit || 0,
    });
    message.value = '用户已创建';
    createForm.username = '';
    createForm.email = '';
    createForm.password = '';
    createForm.role = 'user';
    createForm.daily_search_limit = 0;
    showCreate.value = false;
    await load(1);
  } catch (err: any) {
    error.value = err.response?.data?.error || '创建失败';
  } finally {
    creating.value = false;
  }
};

const startEdit = (u: AuthUser) => {
  editing.value = u;
  editForm.username = u.username;
  editForm.email = u.email || '';
  editForm.role = u.role;
  editForm.status = u.status;
  editForm.daily_search_limit = u.daily_search_limit || 0;
  editForm.password = '';
};

const saveEdit = async () => {
  if (!editing.value) return;
  error.value = '';
  message.value = '';
  savingEdit.value = true;
  try {
    const payload: Record<string, unknown> = {
      username: editForm.username.trim(),
      email: editForm.email.trim(),
      role: editForm.role,
      status: editForm.status,
      daily_search_limit: editForm.daily_search_limit || 0,
    };
    if (editForm.password) payload.password = editForm.password;
    await adminUpdateUser(editing.value.id, payload);
    message.value = '用户已更新';
    editing.value = null;
    await load();
  } catch (err: any) {
    error.value = err.response?.data?.error || '保存失败';
  } finally {
    savingEdit.value = false;
  }
};

const removeUser = async (u: AuthUser) => {
  error.value = '';
  message.value = '';
  if (!confirm(`确定删除用户「${u.username}」？该操作不可恢复。`)) return;
  try {
    await adminDeleteUser(u.id);
    message.value = '用户已删除';
    await load();
  } catch (err: any) {
    error.value = err.response?.data?.error || '删除失败';
  }
};

onMounted(() => load(1));
</script>

<style scoped>
.search-input {
  padding: 0.48rem 0.65rem;
  border: 1px solid hsl(var(--border));
  border-radius: 0.5rem;
  background: hsl(var(--background));
  color: hsl(var(--foreground));
  font-size: 0.82rem;
  min-width: 180px;
}
.create-panel {
  border: 1px dashed hsl(var(--border));
  border-radius: 0.8rem;
  padding: 0.9rem;
  margin-bottom: 1rem;
  background: hsl(var(--muted) / 0.35);
}
.pager {
  margin-top: 0.9rem;
  justify-content: flex-end;
}
</style>
