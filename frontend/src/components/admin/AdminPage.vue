<template>
  <div class="admin-page">
    <div class="admin-head">
      <div>
        <p class="eyebrow">管理员控制台</p>
        <h2>管理后台</h2>
        <p class="sub">账户、搜索开关、SMTP、OAuth 与上游数据同步</p>
      </div>
      <div class="admin-actions">
        <button class="admin-btn" type="button" @click="$emit('user-center')">用户中心</button>
        <button class="admin-btn" type="button" @click="$emit('back')">← 返回搜索</button>
      </div>
    </div>

    <div class="admin-shell">
      <aside class="admin-sidebar">
        <button
          v-for="tab in tabs"
          :key="tab.key"
          type="button"
          :class="{ active: active === tab.key }"
          @click="active = tab.key"
        >
          {{ tab.label }}
        </button>
      </aside>

      <div class="admin-content">
        <AdminOverview v-if="active === 'overview'" />
        <AdminUsers v-else-if="active === 'users'" />
        <AdminSettings v-else-if="active === 'settings'" />
        <AdminOAuth v-else-if="active === 'oauth'" />
        <AdminUpstream v-else-if="active === 'upstream'" />
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { ref } from 'vue';
import AdminOverview from './AdminOverview.vue';
import AdminUsers from './AdminUsers.vue';
import AdminSettings from './AdminSettings.vue';
import AdminOAuth from './AdminOAuth.vue';
import AdminUpstream from './AdminUpstream.vue';

defineEmits<{ back: []; 'user-center': [] }>();

const tabs = [
  { key: 'overview', label: '概览' },
  { key: 'users', label: '用户管理' },
  { key: 'settings', label: '认证与站点' },
  { key: 'oauth', label: 'OAuth 服务商' },
  { key: 'upstream', label: '上游自动同步' },
];

const active = ref('overview');
</script>
