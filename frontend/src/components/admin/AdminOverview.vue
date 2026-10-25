<template>
  <div class="admin-content">
    <div v-if="error" class="admin-alert error">{{ error }}</div>
    <div class="admin-card">
      <div class="admin-actions" style="justify-content: space-between; margin-bottom: 0.8rem">
        <h3 style="margin: 0">运行概览</h3>
        <button class="admin-btn" type="button" :disabled="loading" @click="load">
          {{ loading ? '刷新中...' : '刷新' }}
        </button>
      </div>
      <div class="admin-stats">
        <div class="admin-stat"><span>用户总数</span><strong>{{ overview?.users_total ?? '—' }}</strong></div>
        <div class="admin-stat"><span>活跃用户</span><strong>{{ overview?.users_active ?? '—' }}</strong></div>
        <div class="admin-stat"><span>管理员</span><strong>{{ overview?.admins_total ?? '—' }}</strong></div>
        <div class="admin-stat"><span>今日搜索</span><strong>{{ overview?.searches_today ?? '—' }}</strong></div>
        <div class="admin-stat"><span>累计搜索</span><strong>{{ overview?.searches_total ?? '—' }}</strong></div>
        <div class="admin-stat"><span>匿名搜索（今日）</span><strong>{{ overview?.anonymous_today ?? '—' }}</strong></div>
        <div class="admin-stat"><span>有效会话</span><strong>{{ overview?.sessions_active ?? '—' }}</strong></div>
      </div>
      <div v-if="overview?.default_admin_alive" class="admin-alert warn" style="margin-top: 0.9rem; margin-bottom: 0">
        默认管理员仍在使用初始密码。请前往「用户中心」修改密码，或删除该测试账号。
      </div>
    </div>

    <div class="admin-card">
      <h3>最近搜索</h3>
      <div v-if="!overview?.recent_searches?.length" class="admin-muted">暂无搜索记录。</div>
      <div v-else class="admin-table-wrap">
        <table class="admin-table">
          <thead>
            <tr><th>时间</th><th>用户</th><th>来源</th><th>关键词</th><th>IP</th></tr>
          </thead>
          <tbody>
            <tr v-for="log in overview.recent_searches" :key="log.id">
              <td>{{ shortTime(log.created_at) }}</td>
              <td>{{ log.username || '匿名' }}</td>
              <td><span class="admin-badge">{{ log.source }}</span></td>
              <td>{{ log.keyword || '—' }}</td>
              <td><code>{{ log.ip }}</code></td>
            </tr>
          </tbody>
        </table>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { onMounted, ref } from 'vue';
import { adminGetOverview, type Overview } from '@/api/auth';

const overview = ref<Overview | null>(null);
const loading = ref(false);
const error = ref('');

const shortTime = (s: string) => (s ? new Date(s).toLocaleString() : '');

const load = async () => {
  loading.value = true;
  error.value = '';
  try {
    overview.value = await adminGetOverview();
  } catch (err: any) {
    error.value = err.response?.data?.error || '读取概览失败';
  } finally {
    loading.value = false;
  }
};

onMounted(load);
</script>
