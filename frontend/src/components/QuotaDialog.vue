<template>
  <div v-if="visible" class="quota-overlay" @click.self="emit('close')">
    <div class="quota-dialog" role="dialog" aria-modal="true" aria-labelledby="quota-title">
      <div class="quota-accent" aria-hidden="true" />
      <button class="quota-close" type="button" aria-label="关闭" @click="emit('close')">
        <svg viewBox="0 0 24 24" aria-hidden="true"><path d="M6 6l12 12M18 6 6 18" /></svg>
      </button>

      <div class="quota-icon" :class="{ danger: !anonymous }">
        <svg viewBox="0 0 24 24" aria-hidden="true">
          <path d="M12 3l7 3v6c0 4.2-2.9 7.4-7 9-4.1-1.6-7-4.8-7-9V6l7-3z" />
          <path d="M12 8v5M12 16h.01" />
        </svg>
      </div>

      <span class="quota-eyebrow">{{ anonymous ? 'ANONYMOUS LIMIT' : 'QUOTA REACHED' }}</span>
      <h2 id="quota-title">{{ title }}</h2>
      <p class="quota-message">{{ message }}</p>

      <div class="quota-meter">
        <div class="quota-meter-head">
          <span>已使用 {{ used }} 次</span>
          <strong>上限 {{ limit }} 次</strong>
        </div>
        <div class="quota-meter-track"><i :style="{ width: percentage }" /></div>
      </div>

      <div v-if="anonymous" class="quota-login-tip">
        登录后可享受更高的每日网页搜索与 API 额度，并保留个人搜索记录。
      </div>
      <div v-else class="quota-login-tip">
        如需更高额度，请联系管理员在「用户管理」中调整你的账号配额。
      </div>

      <div class="quota-actions">
        <button v-if="anonymous" class="btn btn-primary" type="button" @click="emit('login')">登录 / 注册</button>
        <button class="btn" type="button" @click="emit('close')">{{ anonymous ? '稍后再说' : '知道了' }}</button>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { computed } from 'vue';

const props = defineProps<{
  visible: boolean;
  anonymous: boolean;
  title: string;
  message: string;
  limit: number;
  used: number;
}>();

const emit = defineEmits<{ close: []; login: [] }>();

const percentage = computed(() => {
  if (props.limit <= 0) return '100%';
  return `${Math.min(100, Math.round((props.used / props.limit) * 100))}%`;
});
</script>

<style scoped>
.quota-overlay {
  position: fixed;
  inset: 0;
  z-index: 10000;
  display: flex;
  align-items: center;
  justify-content: center;
  padding: 1rem;
  background: rgba(15, 23, 42, 0.5);
  backdrop-filter: blur(10px);
  animation: fadeIn 0.18s ease-out;
}
@keyframes fadeIn { from { opacity: 0; } to { opacity: 1; } }
.quota-dialog {
  position: relative;
  width: 100%;
  max-width: 420px;
  padding: 1.7rem 1.6rem 1.4rem;
  border: 1px solid hsl(var(--border));
  border-radius: 1.1rem;
  background: hsl(var(--background-light, 0 0% 100%));
  box-shadow: 0 30px 80px rgba(15, 23, 42, 0.3);
  text-align: center;
  animation: dialogIn 0.2s cubic-bezier(0.22, 1, 0.36, 1);
}
@keyframes dialogIn { from { transform: scale(0.985); } to { transform: scale(1); } }
.quota-accent {
  position: absolute;
  top: 0;
  left: 1.6rem;
  right: 1.6rem;
  height: 2px;
  border-radius: 0 0 999px 999px;
  background: linear-gradient(90deg, transparent, hsl(var(--primary) / 0.7), transparent);
}
.quota-close {
  position: absolute;
  top: 0.8rem;
  right: 0.8rem;
  display: grid;
  place-items: center;
  width: 1.9rem;
  height: 1.9rem;
  border: 0;
  border-radius: 999px;
  background: transparent;
  color: hsl(var(--muted-foreground));
  cursor: pointer;
}
.quota-close:hover { background: hsl(var(--muted)); color: hsl(var(--foreground)); }
.quota-close svg,
.quota-icon svg {
  width: 1rem;
  height: 1rem;
  fill: none;
  stroke: currentColor;
  stroke-width: 1.8;
  stroke-linecap: round;
  stroke-linejoin: round;
}
.quota-icon {
  display: grid;
  place-items: center;
  width: 3.2rem;
  height: 3.2rem;
  margin: 0 auto 0.8rem;
  border-radius: 1rem;
  background: hsl(var(--primary) / 0.12);
  color: hsl(var(--primary));
}
.quota-icon.danger { background: hsl(var(--destructive) / 0.12); color: hsl(var(--destructive)); }
.quota-icon svg { width: 1.5rem; height: 1.5rem; }
.quota-eyebrow {
  font-size: 0.62rem;
  font-weight: 700;
  letter-spacing: 0.16em;
  color: hsl(var(--primary));
}
.quota-dialog h2 { margin: 0.35rem 0 0.4rem; font-size: 1.28rem; color: hsl(var(--foreground)); }
.quota-message { margin: 0; font-size: 0.8rem; line-height: 1.6; color: hsl(var(--muted-foreground)); }
.quota-meter {
  margin: 1rem 0 0.9rem;
  padding: 0.7rem 0.8rem;
  border: 1px solid hsl(var(--border));
  border-radius: 0.7rem;
  background: hsl(var(--muted) / 0.28);
}
.quota-meter-head { display: flex; justify-content: space-between; gap: 0.8rem; font-size: 0.72rem; color: hsl(var(--muted-foreground)); }
.quota-meter-head strong { color: hsl(var(--foreground)); font-weight: 600; }
.quota-meter-track { height: 5px; margin-top: 0.5rem; overflow: hidden; border-radius: 999px; background: hsl(var(--muted)); }
.quota-meter-track i { display: block; height: 100%; border-radius: inherit; background: hsl(var(--destructive)); }
.quota-login-tip {
  padding: 0.65rem 0.75rem;
  border-radius: 0.6rem;
  background: hsl(var(--primary) / 0.08);
  font-size: 0.72rem;
  line-height: 1.6;
  color: hsl(var(--muted-foreground));
  text-align: left;
}
.quota-actions { display: flex; justify-content: center; gap: 0.6rem; margin-top: 1rem; }
@media (max-width: 480px) {
  .quota-dialog { padding: 1.4rem 1.1rem 1.1rem; }
  .quota-actions { flex-direction: column-reverse; }
  .quota-actions .btn { width: 100%; }
}
</style>
