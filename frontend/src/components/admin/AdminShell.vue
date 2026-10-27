<template>
  <div class="console">
    <div v-if="open" class="console-backdrop" @click="open = false" />

    <aside class="console-sidebar" :class="{ open }">
      <div class="console-brand">
        <div class="brand-mark">
          <img :src="siteIcon" :alt="siteName" @error="onIconError" />
        </div>
        <div class="brand-copy">
          <strong>{{ siteName }}</strong>
          <span>{{ mode === 'admin' ? '管理控制台' : '用户控制台' }}</span>
        </div>
        <button class="icon-button mobile-only" type="button" aria-label="关闭菜单" @click="open = false">
          <AdminIcon name="close" :size="18" />
        </button>
      </div>

      <!-- 左上角控制台切换：只有管理员可见，普通用户不需要切换 -->
      <div v-if="isAdminUser" class="console-switch" aria-label="切换控制台">
        <button type="button" :class="{ active: mode === 'admin' }" @click="$emit('switch-mode', 'admin')">
          <AdminIcon name="shield" :size="15" />
          管理
        </button>
        <button type="button" :class="{ active: mode === 'user' }" @click="$emit('switch-mode', 'user')">
          <AdminIcon name="user" :size="15" />
          用户
        </button>
      </div>

      <nav class="console-nav">
        <p class="nav-label">{{ mode === 'admin' ? '管理' : '我的账户' }}</p>
        <button
          v-for="item in nav"
          :key="item.key"
          type="button"
          :class="{ active: item.key === active }"
          @click="$emit('select', item.key)"
        >
          <AdminIcon :name="item.icon" :size="17" />
          <span class="nav-text">{{ item.label }}</span>
          <span v-if="item.badge" class="nav-badge">{{ item.badge }}</span>
          <AdminIcon v-if="item.key === active" name="chevronRight" :size="14" class="nav-arrow" />
        </button>
      </nav>

      <div class="console-sidebar-footer">
        <div class="console-user">
          <div class="avatar">{{ initial }}</div>
          <div class="user-copy">
            <strong>{{ user?.username || '未登录' }}</strong>
            <span>{{ user?.role === 'admin' ? '管理员' : '普通用户' }}</span>
          </div>
        </div>
        <div class="footer-actions">
          <button type="button" @click="$emit('back')">
            <AdminIcon name="home" :size="15" /> 返回站点
          </button>
          <button type="button" class="danger" @click="$emit('logout')">
            <AdminIcon name="logout" :size="15" /> 退出登录
          </button>
        </div>
      </div>
    </aside>

    <div class="console-main">
      <header class="console-topbar">
        <button class="icon-button mobile-only" type="button" aria-label="打开菜单" @click="open = true">
          <AdminIcon name="menu" :size="19" />
        </button>
        <div class="title-block">
          <div class="breadcrumb">控制台 <span>/</span> {{ title }}</div>
          <h1>{{ title }}</h1>
          <p v-if="subtitle">{{ subtitle }}</p>
        </div>
        <div class="topbar-actions">
          <ThemeToggle />
          <slot name="actions" />
        </div>
      </header>
      <div class="console-body">
        <slot />
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { computed, ref } from 'vue';
import { authState, isAdmin } from '@/stores/auth';
import AdminIcon from './AdminIcon.vue';
import ThemeToggle from '@/components/ThemeToggle.vue';

defineProps<{
  title: string;
  subtitle?: string;
  active: string;
  mode: 'admin' | 'user';
  nav: Array<{ key: string; label: string; icon: string; badge?: string | number }>;
}>();

defineEmits<{
  select: [key: string];
  'switch-mode': [mode: 'admin' | 'user'];
  back: [];
  logout: [];
}>();

const open = ref(false);
const user = computed(() => authState.user);
const isAdminUser = computed(() => isAdmin.value);
const siteName = computed(() => authState.config?.site.name || '清羽盘搜');
const siteIcon = computed(() => authState.config?.site.icon || '/favicon.ico');
const initial = computed(() => (user.value?.username || '?').slice(0, 1).toUpperCase());

function onIconError(event: Event) {
  const img = event.target as HTMLImageElement;
  if (!img.src.endsWith('/favicon.ico')) img.src = '/favicon.ico';
}
</script>

<style scoped>
.console {
  display: flex;
  min-height: 100vh;
  background:
    radial-gradient(circle at 100% 0%, rgba(99, 102, 241, 0.08), transparent 32rem),
    radial-gradient(circle at 0% 100%, rgba(129, 140, 248, 0.07), transparent 28rem),
    hsl(var(--muted) / 0.35);
}

.console-backdrop {
  position: fixed;
  inset: 0;
  z-index: 40;
  background: rgba(15, 23, 42, 0.4);
  backdrop-filter: blur(2px);
}

.console-sidebar {
  position: sticky;
  top: 0;
  z-index: 45;
  display: flex;
  flex-direction: column;
  width: 16rem;
  flex: 0 0 16rem;
  height: 100vh;
  border-right: 1px solid hsl(var(--border) / 0.8);
  background: hsl(var(--background-light, 0 0% 100%) / 0.86);
  backdrop-filter: blur(18px);
}

.console-brand {
  display: flex;
  align-items: center;
  gap: 0.7rem;
  padding: 1.1rem 1rem;
  border-bottom: 1px solid hsl(var(--border) / 0.7);
}

.brand-mark {
  display: grid;
  place-items: center;
  width: 2.35rem;
  height: 2.35rem;
  border-radius: 0.8rem;
  background: #fff;
  border: 1px solid hsl(var(--border));
  box-shadow: 0 4px 12px rgba(15, 23, 42, 0.08);
}

.brand-mark img {
  width: 1.45rem;
  height: 1.45rem;
  object-fit: contain;
}

.brand-copy { min-width: 0; flex: 1; }
.brand-copy strong { display: block; font-size: 0.9rem; font-weight: 700; letter-spacing: -0.01em; }
.brand-copy span { display: block; font-size: 0.68rem; color: hsl(var(--muted-foreground)); margin-top: 0.1rem; }

.console-switch {
  display: grid;
  grid-template-columns: 1fr 1fr;
  gap: 0.2rem;
  margin: 0.8rem;
  padding: 0.22rem;
  border-radius: 0.75rem;
  background: hsl(var(--muted));
}

.console-switch button {
  display: flex;
  align-items: center;
  justify-content: center;
  gap: 0.35rem;
  border: 0;
  border-radius: 0.58rem;
  padding: 0.45rem 0.3rem;
  background: transparent;
  color: hsl(var(--muted-foreground));
  font-size: 0.72rem;
  font-weight: 600;
  cursor: pointer;
}

.console-switch button.active {
  background: hsl(var(--background-light, 0 0% 100%));
  color: hsl(var(--foreground));
  box-shadow: 0 2px 8px rgba(15, 23, 42, 0.08);
}

.console-nav {
  flex: 1;
  overflow-y: auto;
  padding: 0.2rem 0.65rem 1rem;
}

.nav-label {
  margin: 0.9rem 0.55rem 0.45rem;
  font-size: 0.64rem;
  font-weight: 700;
  letter-spacing: 0.14em;
  text-transform: uppercase;
  color: hsl(var(--muted-foreground) / 0.8);
}

.console-nav button {
  display: flex;
  align-items: center;
  gap: 0.6rem;
  width: 100%;
  border: 0;
  border-radius: 0.7rem;
  padding: 0.6rem 0.65rem;
  background: transparent;
  color: hsl(var(--muted-foreground));
  font-size: 0.82rem;
  font-weight: 550;
  text-align: left;
  cursor: pointer;
  transition: all 0.15s ease;
}

.console-nav button:hover {
  background: hsl(var(--muted) / 0.9);
  color: hsl(var(--foreground));
}

.console-nav button.active {
  background: hsl(var(--primary));
  color: hsl(var(--primary-foreground));
  box-shadow: 0 6px 14px rgba(79, 70, 229, 0.22);
}

.nav-text { flex: 1; }
.nav-badge {
  min-width: 1.2rem;
  padding: 0.05rem 0.35rem;
  border-radius: 999px;
  background: hsl(var(--muted));
  color: hsl(var(--muted-foreground));
  font-size: 0.64rem;
  text-align: center;
}
.console-nav button.active .nav-badge { background: rgba(255,255,255,0.22); color: #fff; }
.nav-arrow { opacity: 0.7; }

.console-sidebar-footer {
  border-top: 1px solid hsl(var(--border) / 0.7);
  padding: 0.85rem;
}

.console-user { display: flex; align-items: center; gap: 0.6rem; margin-bottom: 0.65rem; }
.avatar {
  display: grid;
  place-items: center;
  width: 2.1rem;
  height: 2.1rem;
  border-radius: 999px;
  background: hsl(var(--primary) / 0.12);
  color: hsl(var(--primary));
  font-size: 0.8rem;
  font-weight: 700;
}
.user-copy { min-width: 0; }
.user-copy strong { display: block; font-size: 0.78rem; font-weight: 600; overflow: hidden; text-overflow: ellipsis; white-space: nowrap; }
.user-copy span { display: block; font-size: 0.64rem; color: hsl(var(--muted-foreground)); text-transform: uppercase; letter-spacing: 0.08em; }

.footer-actions { display: flex; flex-direction: column; gap: 0.25rem; }
.footer-actions button {
  display: flex;
  align-items: center;
  gap: 0.45rem;
  border: 0;
  border-radius: 0.55rem;
  padding: 0.45rem 0.5rem;
  background: transparent;
  color: hsl(var(--muted-foreground));
  font-size: 0.75rem;
  cursor: pointer;
  text-align: left;
}
.footer-actions button:hover { background: hsl(var(--muted)); color: hsl(var(--foreground)); }
.footer-actions button.danger:hover { color: #b91c1c; background: #fef2f2; }

.console-main { display: flex; flex-direction: column; min-width: 0; flex: 1; }

.console-topbar {
  position: sticky;
  top: 0;
  z-index: 30;
  display: flex;
  align-items: center;
  gap: 0.85rem;
  padding: 1rem 1.5rem;
  border-bottom: 1px solid hsl(var(--border) / 0.75);
  background: hsl(var(--background-light, 0 0% 100%) / 0.72);
  backdrop-filter: blur(18px);
}

.title-block { min-width: 0; flex: 1; }
.breadcrumb { font-size: 0.65rem; color: hsl(var(--muted-foreground)); letter-spacing: 0.04em; }
.breadcrumb span { margin: 0 0.25rem; opacity: 0.6; }
.title-block h1 { margin: 0.1rem 0 0; font-size: 1.15rem; font-weight: 700; letter-spacing: -0.02em; }
.title-block p { margin: 0.15rem 0 0; font-size: 0.75rem; color: hsl(var(--muted-foreground)); }

.topbar-actions { display: flex; align-items: center; gap: 0.45rem; flex-wrap: wrap; justify-content: flex-end; }

.console-body { flex: 1; padding: 1.25rem 1.5rem 2rem; }

.icon-button {
  display: grid;
  place-items: center;
  width: 2rem;
  height: 2rem;
  border: 1px solid hsl(var(--border));
  border-radius: 0.55rem;
  background: hsl(var(--background-light, 0 0% 100%));
  color: hsl(var(--muted-foreground));
  cursor: pointer;
}
.icon-button:hover { color: hsl(var(--foreground)); border-color: hsl(var(--border-hover)); }

.mobile-only { display: none; }

@media (max-width: 980px) {
  .console-sidebar {
    position: fixed;
    inset: 0 auto 0 0;
    height: 100vh;
    transform: translateX(-105%);
    transition: transform 0.22s ease;
  }
  .console-sidebar.open { transform: translateX(0); }
  .mobile-only { display: grid; }
  .console-topbar { padding: 0.85rem 1rem; }
  .console-body { padding: 1rem; }
  .title-block h1 { font-size: 1rem; }
  .title-block p { display: none; }
}

@media (max-width: 560px) {
  .topbar-actions :deep(.admin-btn span) { display: none; }
}
</style>
