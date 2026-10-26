<template>
  <div class="theme-toggle" ref="root">
    <button
      class="theme-button"
      type="button"
      :title="`主题：${currentLabel}（点击切换）`"
      :aria-label="`切换主题，当前${currentLabel}`"
      :aria-expanded="open"
      @click.stop="open = !open"
    >
      <svg v-if="mode === 'light'" viewBox="0 0 24 24" aria-hidden="true">
        <circle cx="12" cy="12" r="4" />
        <path d="M12 2v2M12 20v2M4.93 4.93l1.41 1.41M17.66 17.66l1.41 1.41M2 12h2M20 12h2M4.93 19.07l1.41-1.41M17.66 6.34l1.41-1.41" />
      </svg>
      <svg v-else-if="mode === 'dark'" viewBox="0 0 24 24" aria-hidden="true">
        <path d="M21 12.79A9 9 0 1 1 11.21 3a7 7 0 0 0 9.79 9.79Z" />
      </svg>
      <svg v-else viewBox="0 0 24 24" aria-hidden="true">
        <rect x="2.5" y="4" width="19" height="13" rx="2" />
        <path d="M8 21h8M12 17v4" />
      </svg>
    </button>

    <Transition name="theme-pop">
      <div v-if="open" class="theme-menu" role="menu" @click.stop>
        <button
          v-for="item in options"
          :key="item.value"
          type="button"
          role="menuitemradio"
          :aria-checked="mode === item.value"
          :class="{ active: mode === item.value }"
          @click="choose(item.value)"
        >
          <span class="option-icon" v-html="item.icon" />
          <span class="option-copy">
            <strong>{{ item.label }}</strong>
            <small>{{ item.desc }}</small>
          </span>
          <svg v-if="mode === item.value" class="check" viewBox="0 0 24 24" aria-hidden="true">
            <path d="m5 12 5 5L20 7" />
          </svg>
        </button>
      </div>
    </Transition>
  </div>
</template>

<script setup lang="ts">
import { computed, onBeforeUnmount, onMounted, ref } from 'vue';
import { getStoredTheme, onThemeChange, setTheme, type ThemeMode } from '@/lib/theme';

const open = ref(false);
const mode = ref<ThemeMode>(getStoredTheme());
const root = ref<HTMLElement | null>(null);

const options: Array<{ value: ThemeMode; label: string; desc: string; icon: string }> = [
  {
    value: 'light',
    label: '浅色',
    desc: '清爽明亮',
    icon: '<svg viewBox="0 0 24 24"><circle cx="12" cy="12" r="4"/><path d="M12 2v2M12 20v2M4.93 4.93l1.41 1.41M17.66 17.66l1.41 1.41M2 12h2M20 12h2M4.93 19.07l1.41-1.41M17.66 6.34l1.41-1.41"/></svg>',
  },
  {
    value: 'dark',
    label: '深色',
    desc: '夜间护眼',
    icon: '<svg viewBox="0 0 24 24"><path d="M21 12.79A9 9 0 1 1 11.21 3a7 7 0 0 0 9.79 9.79Z"/></svg>',
  },
  {
    value: 'auto',
    label: '跟随系统',
    desc: '自动切换',
    icon: '<svg viewBox="0 0 24 24"><rect x="2.5" y="4" width="19" height="13" rx="2"/><path d="M8 21h8M12 17v4"/></svg>',
  },
];

const currentLabel = computed(() => options.find((item) => item.value === mode.value)?.label || '跟随系统');

let unsubscribe: (() => void) | null = null;

const choose = (value: ThemeMode) => {
  setTheme(value);
  mode.value = value;
  open.value = false;
};

const onDocumentClick = (event: MouseEvent) => {
  if (!root.value?.contains(event.target as Node)) open.value = false;
};
const onKeydown = (event: KeyboardEvent) => {
  if (event.key === 'Escape') open.value = false;
};

onMounted(() => {
  unsubscribe = onThemeChange((value) => {
    mode.value = value;
  });
  document.addEventListener('click', onDocumentClick);
  document.addEventListener('keydown', onKeydown);
});

onBeforeUnmount(() => {
  unsubscribe?.();
  document.removeEventListener('click', onDocumentClick);
  document.removeEventListener('keydown', onKeydown);
});
</script>

<style scoped>
.theme-toggle { position: relative; display: inline-flex; }
.theme-button {
  display: grid;
  place-items: center;
  width: 2.15rem;
  height: 2.15rem;
  padding: 0;
  border: 1px solid rgb(var(--theme-border, 221 224 240) / 0.9);
  border-radius: 0.6rem;
  background: rgb(var(--theme-surface, 255 255 255) / 0.75);
  color: rgb(var(--theme-text-muted, 71 85 105));
  cursor: pointer;
  transition: color 0.18s ease, border-color 0.18s ease, background 0.18s ease, transform 0.18s ease;
}
.theme-button:hover {
  color: hsl(var(--primary));
  border-color: hsl(var(--primary) / 0.45);
  transform: translateY(-1px);
}
.theme-button svg {
  width: 1.05rem;
  height: 1.05rem;
  fill: none;
  stroke: currentColor;
  stroke-width: 1.8;
  stroke-linecap: round;
  stroke-linejoin: round;
}
.theme-menu {
  position: absolute;
  top: calc(100% + 0.4rem);
  right: 0;
  z-index: 200;
  width: 12.5rem;
  padding: 0.3rem;
  border: 1px solid hsl(var(--border));
  border-radius: 0.75rem;
  background: hsl(var(--background-light, 0 0% 100%));
  box-shadow: 0 16px 44px rgba(15, 23, 42, 0.18);
}
.theme-menu button {
  display: flex;
  align-items: center;
  gap: 0.55rem;
  width: 100%;
  padding: 0.5rem 0.5rem;
  border: 0;
  border-radius: 0.55rem;
  background: transparent;
  color: hsl(var(--foreground));
  text-align: left;
  cursor: pointer;
}
.theme-menu button:hover { background: hsl(var(--muted)); }
.theme-menu button.active { background: hsl(var(--primary) / 0.09); }
.option-icon { display: grid; place-items: center; width: 1.6rem; height: 1.6rem; flex: 0 0 1.6rem; color: hsl(var(--muted-foreground)); }
.option-icon :deep(svg) { width: 1.05rem; height: 1.05rem; fill: none; stroke: currentColor; stroke-width: 1.8; stroke-linecap: round; stroke-linejoin: round; }
.theme-menu button.active .option-icon { color: hsl(var(--primary)); }
.option-copy { flex: 1; min-width: 0; }
.option-copy strong { display: block; font-size: 0.76rem; font-weight: 600; }
.option-copy small { display: block; margin-top: 0.08rem; font-size: 0.63rem; color: hsl(var(--muted-foreground)); }
.check { width: 0.9rem; height: 0.9rem; fill: none; stroke: hsl(var(--primary)); stroke-width: 2.2; stroke-linecap: round; stroke-linejoin: round; }
.theme-pop-enter-active, .theme-pop-leave-active { transition: opacity 0.15s ease, transform 0.15s ease; }
.theme-pop-enter-from, .theme-pop-leave-to { opacity: 0; transform: translateY(-4px) scale(0.98); }
</style>
