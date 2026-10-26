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
import { consoleNav, consolePages } from '@/console/config';

const emit = defineEmits<{ back: []; 'user-center': []; logout: [] }>();

const onSwitchMode = (mode: 'admin' | 'user') => {
  if (mode === 'user') emit('user-center');
};

const nav = consoleNav.admin;
const active = ref('overview');
const current = computed(() => consolePages.admin[active.value] || consolePages.admin.overview);
</script>
