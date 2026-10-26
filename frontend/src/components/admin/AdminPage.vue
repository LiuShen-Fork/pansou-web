<template>
  <AdminShell
    :title="current.title"
    :subtitle="current.subtitle"
    :active="active"
    mode="admin"
    :nav="nav"
    @select="active = $event"
    @back="$emit('back')"
    @switch-mode="onSwitchMode"
    @logout="$emit('logout')"
  >
    <AdminOverview v-if="active === 'overview'" @navigate="active = $event" />
    <AdminUsers v-else-if="active === 'users'" />
    <AdminLogs v-else-if="active === 'logs'" />
    <AdminSettings v-else-if="active === 'settings'" />
    <AdminOAuth v-else-if="active === 'oauth'" />
    <AdminUpstream v-else-if="active === 'upstream'" />
    <AdminSystem v-else-if="active === 'system'" />
  </AdminShell>
</template>

<script setup lang="ts">
import { computed, ref } from 'vue';
import AdminShell from './AdminShell.vue';
import AdminOverview from './AdminOverview.vue';
import AdminUsers from './AdminUsers.vue';
import AdminLogs from './AdminLogs.vue';
import AdminSettings from './AdminSettings.vue';
import AdminOAuth from './AdminOAuth.vue';
import AdminUpstream from './AdminUpstream.vue';
import AdminSystem from './AdminSystem.vue';

const emit = defineEmits<{ back: []; 'user-center': []; logout: [] }>();

const onSwitchMode = (mode: 'admin' | 'user') => {
  if (mode === 'user') emit('user-center');
};

const nav = [
  { key: 'overview', label: '运营概览', icon: 'dashboard' },
  { key: 'users', label: '用户管理', icon: 'users' },
  { key: 'logs', label: '搜索日志', icon: 'logs' },
  { key: 'settings', label: '认证与站点', icon: 'settings' },
  { key: 'oauth', label: 'OAuth 服务商', icon: 'key' },
  { key: 'upstream', label: '上游自动同步', icon: 'upstream' },
  { key: 'system', label: '系统状态', icon: 'system' },
];

const pages: Record<string, { title: string; subtitle: string }> = {
  overview: { title: '运营概览', subtitle: '用户增长、搜索趋势与服务健康度' },
  users: { title: '用户管理', subtitle: '账号、角色、状态与每日搜索配额' },
  logs: { title: '搜索日志', subtitle: '按用户与关键词追踪搜索行为' },
  settings: { title: '认证与站点', subtitle: '登录方式、注册策略、API 权限与 SMTP' },
  oauth: { title: 'OAuth 服务商', subtitle: '配置第三方登录与注册入口' },
  upstream: { title: '上游自动同步', subtitle: '定时刷新 TG 频道与插件默认值' },
  system: { title: '系统状态', subtitle: '插件、频道、TG 可达性与存活观测' },
};

const active = ref('overview');
const current = computed(() => pages[active.value] || pages.overview);
</script>
