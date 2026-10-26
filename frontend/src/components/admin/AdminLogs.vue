<template>
  <div class="page-stack">
    <section class="section-card">
      <div class="section-head">
        <div>
          <div class="section-title">搜索日志</div>
          <div class="section-desc">共 {{ total }} 条记录，仅保留最近 180 天</div>
        </div>
        <div class="toolbar">
          <button class="btn btn-sm" type="button" :disabled="!items.length" @click="exportCsv">
            <AdminIcon name="database" :size="14" /> 导出当前页
          </button>
          <button class="btn btn-sm" type="button" :disabled="loading" @click="load(page)">
            <AdminIcon name="refresh" :size="14" /> 刷新
          </button>
        </div>
      </div>

      <div class="section-body" style="padding-bottom: 0.6rem">
        <div class="toolbar">
          <input
            v-model="filters.q"
            class="input"
            style="max-width: 18rem"
            placeholder="搜索用户名或关键词"
            @keydown.enter="load(1)"
          />
          <select v-model="filters.source" class="select" style="max-width: 10rem" @change="load(1)">
            <option value="">全部来源</option>
            <option value="api">API / 登录用户</option>
            <option value="web">网页匿名</option>
          </select>
          <button class="btn btn-sm" type="button" :disabled="loading" @click="load(1)">
            <AdminIcon name="search" :size="14" /> 查询
          </button>
        </div>
      </div>

      <div v-if="loading && !items.length" class="section-body">
        <div class="skeleton" style="width: 100%; margin-bottom: 0.6rem" />
        <div class="skeleton" style="width: 94%; margin-bottom: 0.6rem" />
        <div class="skeleton" style="width: 88%" />
      </div>
      <div v-else-if="!items.length" class="empty-state">
        <AdminIcon name="logs" :size="24" />
        <strong>暂无搜索日志</strong>
        <span>用户搜索后会记录在这里</span>
      </div>
      <div v-else class="table-wrap">
        <table class="data-table">
          <thead>
            <tr>
              <th>时间</th>
              <th>用户</th>
              <th>来源</th>
              <th>关键词</th>
              <th class="hide-sm">IP</th>
            </tr>
          </thead>
          <tbody>
            <tr v-for="log in items" :key="log.id">
              <td class="hide-sm nowrap">{{ shortTime(log.created_at) }}</td>
              <td>
                <span v-if="log.username" class="cell-strong">{{ log.username }}</span>
                <span v-else class="badge">匿名</span>
              </td>
              <td><span class="badge" :class="log.source === 'web' ? 'badge-warning' : 'badge-primary'">{{ log.source === 'web' ? '网页' : 'API' }}</span></td>
              <td class="cell-strong">{{ log.keyword || '—' }}</td>
              <td class="hide-sm mono">{{ log.ip }}</td>
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
import { onMounted, reactive, ref } from 'vue';
import { adminListSearchLogs, type SearchLog } from '@/api/auth';
import AdminIcon from './AdminIcon.vue';
import { toast } from '@/lib/toast';

const items = ref<SearchLog[]>([]);
const total = ref(0);
const page = ref(1);
const pageSize = 20;
const loading = ref(false);
const filters = reactive({ q: '', source: '' });

function shortTime(value: string) {
  return value ? new Date(value).toLocaleString() : '';
}

const load = async (target = page.value) => {
  loading.value = true;
  try {
    const res = await adminListSearchLogs(filters.q.trim(), target, pageSize, filters.source);
    items.value = res.items || [];
    total.value = res.total || 0;
    page.value = res.page || target;
  } catch (err: any) {
    toast.error('加载搜索日志失败', err.response?.data?.error || '');
  } finally {
    loading.value = false;
  }
};

function exportCsv() {
  const header = ['时间', '用户', '来源', '关键词', 'IP'];
  const rows = items.value.map((log) => [
    shortTime(log.created_at),
    log.username || '匿名',
    log.source,
    log.keyword,
    log.ip,
  ]);
  const csv = [header, ...rows]
    .map((row) => row.map((cell) => `"${String(cell ?? '').replace(/"/g, '""')}"`).join(','))
    .join('\r\n');
  const blob = new Blob(['﻿' + csv], { type: 'text/csv;charset=utf-8' });
  const url = URL.createObjectURL(blob);
  const a = document.createElement('a');
  a.href = url;
  a.download = `pansou-search-logs-${new Date().toISOString().slice(0, 10)}.csv`;
  a.click();
  URL.revokeObjectURL(url);
}

onMounted(() => load(1));
</script>

<style scoped>
.nowrap { white-space: nowrap; }
</style>
