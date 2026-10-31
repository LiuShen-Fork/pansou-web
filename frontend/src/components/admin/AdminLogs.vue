<template>
  <div class="page-stack">
    <section class="section-card">
      <div class="section-head">
        <div>
          <div class="section-title">搜索与审计日志</div>
          <div class="section-desc">搜索任务记录关键词、来源、结果数、耗时与 IP 归属地；审计日志记录登录、配额触发与后台操作</div>
        </div>
        <div class="toolbar">
          <button class="btn btn-sm" type="button" :disabled="!hasRows" @click="exportCsv">
            <AdminIcon name="database" :size="14" /> 导出当前页
          </button>
          <button class="btn btn-sm" type="button" :disabled="loading" @click="load(page)">
            <AdminIcon name="refresh" :size="14" /> 刷新
          </button>
        </div>
      </div>

      <div class="section-body" style="padding-bottom: 0.6rem">
        <div class="log-tabs">
          <button type="button" :class="{ active: tab === 'search' }" @click="switchTab('search')">
            <AdminIcon name="search" :size="14" /> 搜索任务
          </button>
          <button type="button" :class="{ active: tab === 'audit' }" @click="switchTab('audit')">
            <AdminIcon name="shield" :size="14" /> 审计日志
          </button>
        </div>

        <div class="toolbar" style="margin-top: 0.7rem">
          <input
            v-model="filters.q"
            class="input"
            style="max-width: 18rem"
            :placeholder="tab === 'search' ? '搜索用户名、关键词、IP' : '搜索用户、对象、详情或 IP'"
            @keydown.enter="load(1)"
          />
          <template v-if="tab === 'search'">
            <select v-model="filters.source" class="select" style="max-width: 9rem" @change="load(1)">
              <option value="">全部来源</option>
              <option value="web">网页搜索</option>
              <option value="api">API 调用</option>
            </select>
            <select v-model="filters.status" class="select" style="max-width: 9rem" @change="load(1)">
              <option value="">全部状态</option>
              <option value="ok">完成</option>
              <option value="pending">进行中</option>
              <option value="error">失败</option>
              <option value="bad_request">参数错误</option>
            </select>
          </template>
          <select v-else v-model="filters.event" class="select" style="max-width: 11rem" @change="load(1)">
            <option value="">全部事件</option>
            <option v-for="item in auditEvents" :key="item.value" :value="item.value">{{ item.label }}</option>
          </select>
          <button class="btn btn-sm" type="button" :disabled="loading" @click="load(1)">
            <AdminIcon name="search" :size="14" /> 查询
          </button>
        </div>
      </div>

      <div v-if="loading && !hasRows" class="section-body">
        <div class="skeleton" style="width: 100%; margin-bottom: 0.6rem" />
        <div class="skeleton" style="width: 94%; margin-bottom: 0.6rem" />
        <div class="skeleton" style="width: 88%" />
      </div>
      <div v-else-if="!hasRows" class="empty-state">
        <AdminIcon name="logs" :size="24" />
        <strong>{{ tab === 'search' ? '暂无搜索日志' : '暂无审计日志' }}</strong>
        <span>{{ tab === 'search' ? '用户搜索后会记录在这里' : '登录、配额触发和后台操作会记录在这里' }}</span>
      </div>

      <div v-else class="table-wrap">
        <table v-if="tab === 'search'" class="data-table log-table">
          <thead>
            <tr>
              <th>时间</th>
              <th>用户</th>
              <th>来源</th>
              <th>关键词</th>
              <th class="hide-sm">IP / 归属地</th>
              <th class="hide-sm">结果 / 耗时</th>
              <th>状态</th>
            </tr>
          </thead>
          <tbody>
            <tr v-for="log in items" :key="log.id">
              <td class="nowrap cell-sub">{{ shortTime(log.created_at) }}</td>
              <td>
                <span v-if="log.username" class="cell-strong">{{ log.username }}</span>
                <span v-else class="badge">匿名</span>
              </td>
              <td><span class="badge" :class="log.source === 'web' ? 'badge-warning' : 'badge-primary'">{{ log.source === 'web' ? '网页' : 'API' }}</span></td>
              <td class="cell-strong">{{ log.keyword || '—' }}</td>
              <td class="hide-sm">
                <div class="mono">{{ log.ip || '—' }}</div>
                <div class="cell-sub">{{ location(log) }}</div>
              </td>
              <td class="hide-sm mono">{{ log.result_count || 0 }} / {{ log.duration_ms ? log.duration_ms + 'ms' : '—' }}</td>
              <td>
                <span class="badge" :class="statusClass(log.status)">{{ statusLabel(log.status) }}</span>
                <div v-if="log.error" class="cell-sub error-line" :title="log.error">{{ log.error }}</div>
              </td>
            </tr>
          </tbody>
        </table>

        <table v-else class="data-table log-table">
          <thead>
            <tr>
              <th>时间</th>
              <th>用户</th>
              <th>事件</th>
              <th>对象</th>
              <th class="hide-sm">详情</th>
              <th class="hide-sm">IP / 归属地</th>
            </tr>
          </thead>
          <tbody>
            <tr v-for="log in auditItems" :key="log.id">
              <td class="nowrap cell-sub">{{ shortTime(log.created_at) }}</td>
              <td>
                <span v-if="log.username" class="cell-strong">{{ log.username }}</span>
                <span v-else class="badge">匿名</span>
              </td>
              <td><span class="badge" :class="eventClass(log.event)">{{ eventLabel(log.event) }}</span></td>
              <td class="cell-strong">{{ log.target || '—' }}</td>
              <td class="hide-sm cell-sub detail-line" :title="log.detail">{{ log.detail || '—' }}</td>
              <td class="hide-sm">
                <div class="mono">{{ log.ip || '—' }}</div>
                <div class="cell-sub">{{ location(log) }}</div>
              </td>
            </tr>
          </tbody>
        </table>
      </div>

      <div class="pager">
        <span>第 {{ page }} 页 · 共 {{ total }} 条</span>
        <button class="btn btn-sm" type="button" :disabled="page <= 1" @click="load(page - 1)">上一页</button>
        <button class="btn btn-sm" type="button" :disabled="page * pageSize >= total" @click="load(page + 1)">下一页</button>
      </div>
    </section>
  </div>
</template>

<script setup lang="ts">
import { computed, onMounted, reactive, ref } from 'vue';
import {
  adminListAuditLogs,
  adminListSearchLogs,
  type AuditLog,
  type SearchLog,
} from '@/api/auth';
import AdminIcon from './AdminIcon.vue';
import { toast } from '@/lib/toast';

type Tab = 'search' | 'audit';

const tab = ref<Tab>('search');
const items = ref<SearchLog[]>([]);
const auditItems = ref<AuditLog[]>([]);
const total = ref(0);
const page = ref(1);
const pageSize = 20;
const loading = ref(false);
const filters = reactive({ q: '', source: '', status: '', event: '' });

const auditEvents = [
  { value: 'login_success', label: '登录成功' },
  { value: 'login_failed', label: '登录失败' },
  { value: 'login_throttled', label: '登录限流' },
  { value: 'logout', label: '退出登录' },
  { value: 'register', label: '邮箱注册' },
  { value: 'password_change', label: '修改密码' },
  { value: 'profile_update', label: '更新资料' },
  { value: 'quota_exceeded', label: '配额触发' },
  { value: 'settings_update', label: '设置变更' },
  { value: 'runtime_update', label: '运行参数' },
  { value: 'user_create', label: '创建用户' },
  { value: 'user_update', label: '更新用户' },
  { value: 'user_delete', label: '删除用户' },
  { value: 'upstream_sync', label: '上游同步' },
  { value: 'api_token_create', label: '创建令牌' },
  { value: 'api_token_delete', label: '删除令牌' },
];

const eventMap = computed(() => Object.fromEntries(auditEvents.map((item) => [item.value, item.label])));
const hasRows = computed(() => (tab.value === 'search' ? items.value.length > 0 : auditItems.value.length > 0));

function shortTime(value: string) {
  return value ? new Date(value).toLocaleString() : '';
}
function location(log: { country?: string; region?: string; city?: string }) {
  const parts = [log.country, log.region, log.city].filter(Boolean);
  return parts.length ? parts.join(' · ') : '归属地未记录';
}
function statusLabel(status: string) {
  switch (status) {
    case 'ok': return '完成';
    case 'pending': return '进行中';
    case 'error': return '失败';
    case 'bad_request': return '参数错误';
    default: return status || '—';
  }
}
function statusClass(status: string) {
  if (status === 'ok') return 'badge-success';
  if (status === 'error' || status === 'bad_request') return 'badge-danger';
  if (status === 'pending') return 'badge-warning';
  return '';
}
function eventLabel(event: string) {
  return eventMap.value[event] || event || '—';
}
function eventClass(event: string) {
  if (event === 'quota_exceeded' || event === 'login_failed' || event === 'login_throttled') return 'badge-danger';
  if (event === 'settings_update' || event === 'runtime_update' || event === 'upstream_sync') return 'badge-primary';
  if (event === 'login_success' || event === 'register') return 'badge-success';
  return '';
}

const switchTab = (next: Tab) => {
  tab.value = next;
  page.value = 1;
  items.value = [];
  auditItems.value = [];
  total.value = 0;
  load(1);
};

const load = async (target = page.value) => {
  loading.value = true;
  try {
    if (tab.value === 'search') {
      const res = await adminListSearchLogs(filters.q.trim(), target, pageSize, filters.source, filters.status);
      items.value = res.items || [];
      total.value = res.total || 0;
      page.value = res.page || target;
    } else {
      const res = await adminListAuditLogs(filters.q.trim(), target, pageSize, filters.event);
      auditItems.value = res.items || [];
      total.value = res.total || 0;
      page.value = res.page || target;
    }
  } catch (err: any) {
    toast.error('加载日志失败', err.response?.data?.error || '');
  } finally {
    loading.value = false;
  }
};

function exportCsv() {
  const rows: string[][] = [];
  if (tab.value === 'search') {
    rows.push(['时间', '用户', '来源', '关键词', 'IP', '归属地', '结果数', '耗时(ms)', '状态', '错误']);
    for (const log of items.value) {
      rows.push([
        shortTime(log.created_at), log.username || '匿名', log.source, log.keyword, log.ip,
        location(log), String(log.result_count), String(log.duration_ms), statusLabel(log.status), log.error || '',
      ]);
    }
  } else {
    rows.push(['时间', '用户', '事件', '对象', '详情', 'IP', '归属地']);
    for (const log of auditItems.value) {
      rows.push([
        shortTime(log.created_at), log.username || '匿名', eventLabel(log.event), log.target || '',
        log.detail || '', log.ip || '', location(log),
      ]);
    }
  }
  const csv = rows.map((row) => row.map((cell) => `"${String(cell).replace(/"/g, '""')}"`).join(',')).join('\n');
  const blob = new Blob([`﻿${csv}`], { type: 'text/csv;charset=utf-8' });
  const url = URL.createObjectURL(blob);
  const link = document.createElement('a');
  link.href = url;
  link.download = `${tab.value === 'search' ? 'search-logs' : 'audit-logs'}.csv`;
  link.click();
  URL.revokeObjectURL(url);
}

onMounted(() => load(1));
</script>

<style scoped>
.log-tabs {
  display: inline-flex;
  gap: 0.25rem;
  padding: 0.28rem;
  border: 1px solid hsl(var(--border));
  border-radius: 0.6rem;
  background: hsl(var(--muted) / 0.35);
}
.log-tabs button {
  display: inline-flex;
  align-items: center;
  gap: 0.35rem;
  padding: 0.38rem 0.7rem;
  border: 0;
  border-radius: 0.45rem;
  background: transparent;
  color: hsl(var(--muted-foreground));
  font-size: 0.74rem;
  font-weight: 600;
  cursor: pointer;
}
.log-tabs button.active {
  background: hsl(var(--background-light, 0 0% 100%));
  color: hsl(var(--primary));
  box-shadow: 0 2px 8px rgba(15, 23, 42, 0.08);
}
.log-table td { vertical-align: top; }
.error-line { max-width: 12rem; margin-top: 0.15rem; color: hsl(var(--destructive)); overflow: hidden; text-overflow: ellipsis; white-space: nowrap; }
.detail-line { max-width: 16rem; overflow: hidden; text-overflow: ellipsis; white-space: nowrap; }
@media (max-width: 700px) {
  .log-tabs { width: 100%; }
  .log-tabs button { flex: 1; justify-content: center; }
}
</style>
