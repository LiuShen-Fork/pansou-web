<template>
  <Teleport to="body">
    <div class="toast-host" aria-live="polite">
      <TransitionGroup name="toast">
        <div v-for="item in toasts" :key="item.id" class="toast-item" :class="item.type">
          <div class="toast-icon" aria-hidden="true">
            <svg v-if="item.type === 'success'" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2.2">
              <path d="M20 6L9 17l-5-5" stroke-linecap="round" stroke-linejoin="round" />
            </svg>
            <svg v-else-if="item.type === 'error'" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2.2">
              <path d="M18 6L6 18M6 6l12 12" stroke-linecap="round" />
            </svg>
            <svg v-else viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2.2">
              <circle cx="12" cy="12" r="9" />
              <path d="M12 10v6M12 7h.01" stroke-linecap="round" />
            </svg>
          </div>
          <div class="toast-body">
            <strong>{{ item.title }}</strong>
            <p v-if="item.message">{{ item.message }}</p>
          </div>
          <button class="toast-close" type="button" aria-label="关闭" @click="dismissToast(item.id)">×</button>
        </div>
      </TransitionGroup>
    </div>
  </Teleport>
</template>

<script setup lang="ts">
import { dismissToast, toasts } from '@/lib/toast';
</script>

<style scoped>
.toast-host {
  position: fixed;
  top: 1rem;
  right: 1rem;
  z-index: 10000;
  display: flex;
  flex-direction: column;
  gap: 0.55rem;
  width: min(22rem, calc(100vw - 2rem));
  pointer-events: none;
}

.toast-item {
  pointer-events: auto;
  display: flex;
  align-items: flex-start;
  gap: 0.65rem;
  padding: 0.75rem 0.85rem;
  border-radius: 0.8rem;
  border: 1px solid hsl(var(--border));
  background: hsl(var(--background-light, 0 0% 100%));
  box-shadow: 0 14px 40px rgba(15, 23, 42, 0.16);
}

.toast-icon {
  display: grid;
  place-items: center;
  width: 1.6rem;
  height: 1.6rem;
  border-radius: 999px;
  flex: 0 0 auto;
}

.toast-icon svg {
  width: 0.95rem;
  height: 0.95rem;
}

.toast-item.success .toast-icon { background: #ecfdf5; color: #047857; }
.toast-item.error .toast-icon { background: #fef2f2; color: #b91c1c; }
.toast-item.info .toast-icon { background: #eef2ff; color: #4338ca; }

.toast-body { flex: 1; min-width: 0; }
.toast-body strong { display: block; font-size: 0.84rem; font-weight: 650; color: hsl(var(--foreground)); }
.toast-body p { margin: 0.15rem 0 0; font-size: 0.76rem; line-height: 1.45; color: hsl(var(--muted-foreground)); word-break: break-word; }

.toast-close {
  border: 0;
  background: transparent;
  color: hsl(var(--muted-foreground));
  font-size: 1.1rem;
  line-height: 1;
  cursor: pointer;
  padding: 0.1rem 0.2rem;
}

.toast-enter-active, .toast-leave-active { transition: all 0.22s ease; }
.toast-enter-from, .toast-leave-to { opacity: 0; transform: translateY(-6px) scale(0.98); }
</style>
