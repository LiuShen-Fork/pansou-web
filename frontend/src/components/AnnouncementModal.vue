<template>
  <Teleport to="body">
    <Transition name="announce">
      <div v-if="visible" class="announce-overlay" @click.self="close">
        <section class="announce-card" role="dialog" aria-modal="true" aria-labelledby="announce-title">
          <div class="announce-accent" aria-hidden="true" />
          <header class="announce-head">
            <div class="announce-badge" aria-hidden="true">
              <svg viewBox="0 0 24 24">
                <path d="M3 11v2a1 1 0 0 0 1 1h2l4 4V6L6 10H4a1 1 0 0 0-1 1Z" />
                <path d="M15 8a5 5 0 0 1 0 8M18 5a9 9 0 0 1 0 14" />
              </svg>
            </div>
            <div class="announce-heading">
              <span class="announce-eyebrow">NOTICE</span>
              <h2 id="announce-title">站点公告</h2>
            </div>
            <button class="announce-close" type="button" aria-label="关闭公告" @click="close">
              <svg viewBox="0 0 24 24"><path d="M6 6l12 12M18 6 6 18" /></svg>
            </button>
          </header>

          <div class="announce-body">
            <div class="markdown-body" v-html="html" />
          </div>

          <footer class="announce-foot">
            <span>公告由站长发布 · 5 分钟内不会重复显示</span>
            <button class="announce-ok" type="button" @click="close">我知道了</button>
          </footer>
        </section>
      </div>
    </Transition>
  </Teleport>
</template>

<script setup lang="ts">
import { computed, onBeforeUnmount, onMounted, watch } from 'vue';
import { renderMarkdown } from '@/lib/markdown';

const props = defineProps<{ visible: boolean; content: string }>();
const emit = defineEmits<{ close: [] }>();

const html = computed(() => renderMarkdown(props.content));

const close = () => emit('close');

const onKeydown = (event: KeyboardEvent) => {
  if (event.key === 'Escape' && props.visible) close();
};

let previousOverflow = '';

watch(
  () => props.visible,
  (visible) => {
    if (visible) {
      previousOverflow = document.body.style.overflow;
      document.body.style.overflow = 'hidden';
    } else {
      document.body.style.overflow = previousOverflow;
    }
  },
);

onMounted(() => document.addEventListener('keydown', onKeydown));
onBeforeUnmount(() => {
  document.removeEventListener('keydown', onKeydown);
  document.body.style.overflow = previousOverflow;
});
</script>

<style scoped>
.announce-overlay {
  position: fixed;
  inset: 0;
  z-index: 300;
  display: flex;
  align-items: center;
  justify-content: center;
  padding: 1rem;
  background: rgba(15, 23, 42, 0.5);
  backdrop-filter: blur(8px);
}
.announce-card {
  position: relative;
  display: flex;
  flex-direction: column;
  width: min(100%, 34rem);
  max-height: min(82vh, 40rem);
  overflow: hidden;
  border: 1px solid hsl(var(--border));
  border-radius: 1.1rem;
  background: hsl(var(--background-light, 0 0% 100%));
  box-shadow: 0 30px 80px rgba(15, 23, 42, 0.28);
}
.announce-accent {
  height: 0.28rem;
  flex: 0 0 auto;
  background: linear-gradient(90deg, hsl(var(--primary)), hsl(var(--accent)), hsl(var(--primary)));
}
.announce-head {
  display: flex;
  align-items: center;
  gap: 0.75rem;
  padding: 1.1rem 1.2rem 0.85rem;
}
.announce-badge {
  display: grid;
  place-items: center;
  width: 2.5rem;
  height: 2.5rem;
  flex: 0 0 2.5rem;
  border-radius: 0.8rem;
  background: hsl(var(--primary) / 0.1);
  color: hsl(var(--primary));
}
.announce-badge svg { width: 1.35rem; height: 1.35rem; fill: none; stroke: currentColor; stroke-width: 1.8; stroke-linecap: round; stroke-linejoin: round; }
.announce-heading { flex: 1; min-width: 0; }
.announce-eyebrow { display: block; font-size: 0.58rem; font-weight: 750; letter-spacing: 0.18em; color: hsl(var(--primary)); }
.announce-heading h2 { margin: 0.1rem 0 0; font-size: 1.15rem; font-weight: 750; letter-spacing: -0.02em; }
.announce-close {
  display: grid;
  place-items: center;
  width: 1.9rem;
  height: 1.9rem;
  flex: 0 0 1.9rem;
  border: 0;
  border-radius: 999px;
  background: hsl(var(--muted));
  color: hsl(var(--muted-foreground));
  cursor: pointer;
  transition: background 0.15s ease, color 0.15s ease;
}
.announce-close:hover { background: hsl(var(--destructive) / 0.1); color: hsl(var(--destructive)); }
.announce-close svg { width: 0.9rem; height: 0.9rem; fill: none; stroke: currentColor; stroke-width: 2; stroke-linecap: round; }
.announce-body {
  padding: 0.3rem 1.2rem 1rem;
  overflow-y: auto;
  scrollbar-width: thin;
}
.announce-foot {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 0.8rem;
  flex-wrap: wrap;
  padding: 0.75rem 1.2rem;
  border-top: 1px solid hsl(var(--border) / 0.75);
  background: hsl(var(--muted) / 0.3);
}
.announce-foot span { font-size: 0.66rem; color: hsl(var(--muted-foreground)); }
.announce-ok {
  padding: 0.45rem 0.95rem;
  border: 0;
  border-radius: 0.5rem;
  background: hsl(var(--primary));
  color: hsl(var(--primary-foreground));
  font-size: 0.76rem;
  font-weight: 650;
  cursor: pointer;
  box-shadow: 0 6px 16px hsl(var(--primary) / 0.24);
}
.announce-ok:hover { filter: brightness(1.06); }

/* Markdown 排版 */
.markdown-body { font-size: 0.83rem; line-height: 1.75; color: hsl(var(--foreground)); }
.markdown-body :deep(> :first-child) { margin-top: 0; }
.markdown-body :deep(> :last-child) { margin-bottom: 0; }
.markdown-body :deep(h1),
.markdown-body :deep(h2),
.markdown-body :deep(h3),
.markdown-body :deep(h4) { margin: 1rem 0 0.4rem; font-weight: 700; letter-spacing: -0.01em; line-height: 1.35; }
.markdown-body :deep(h1) { font-size: 1.2rem; }
.markdown-body :deep(h2) { font-size: 1.05rem; }
.markdown-body :deep(h3) { font-size: 0.95rem; }
.markdown-body :deep(p) { margin: 0.45rem 0; }
.markdown-body :deep(a) { color: hsl(var(--primary)); font-weight: 600; text-decoration: underline; text-underline-offset: 2px; }
.markdown-body :deep(strong) { font-weight: 700; }
.markdown-body :deep(ul),
.markdown-body :deep(ol) { margin: 0.45rem 0; padding-left: 1.15rem; }
.markdown-body :deep(ul) { list-style: disc; }
.markdown-body :deep(ol) { list-style: decimal; }
.markdown-body :deep(li) { margin: 0.2rem 0; }
.markdown-body :deep(blockquote) {
  margin: 0.6rem 0;
  padding: 0.45rem 0.75rem;
  border-left: 3px solid hsl(var(--primary) / 0.55);
  border-radius: 0 0.45rem 0.45rem 0;
  background: hsl(var(--primary) / 0.06);
  color: hsl(var(--muted-foreground));
}
.markdown-body :deep(code) {
  padding: 0.1rem 0.3rem;
  border-radius: 0.3rem;
  background: hsl(var(--muted));
  font-family: ui-monospace, SFMono-Regular, Menlo, monospace;
  font-size: 0.78em;
}
.markdown-body :deep(pre.md-code) {
  margin: 0.6rem 0;
  padding: 0.7rem 0.8rem;
  overflow-x: auto;
  border: 1px solid hsl(var(--border));
  border-radius: 0.55rem;
  background: hsl(var(--muted));
  font-size: 0.75rem;
  line-height: 1.6;
}
.markdown-body :deep(pre.md-code code) { padding: 0; background: transparent; font-size: inherit; }
.markdown-body :deep(hr) { margin: 0.9rem 0; border: 0; border-top: 1px solid hsl(var(--border)); }
.markdown-body :deep(del) { opacity: 0.7; }

.announce-enter-active, .announce-leave-active { transition: opacity 0.2s ease; }
.announce-enter-active .announce-card, .announce-leave-active .announce-card { transition: transform 0.22s cubic-bezier(0.22, 1, 0.36, 1), opacity 0.2s ease; }
.announce-enter-from, .announce-leave-to { opacity: 0; }
.announce-enter-from .announce-card, .announce-leave-to .announce-card { opacity: 0; transform: translateY(14px) scale(0.97); }

@media (max-width: 560px) {
  .announce-overlay { padding: 0.7rem; align-items: flex-end; }
  .announce-card { max-height: 86vh; border-radius: 1rem; }
  .announce-head { padding: 0.95rem 1rem 0.7rem; }
  .announce-body { padding: 0.2rem 1rem 0.9rem; }
  .announce-foot { padding: 0.7rem 1rem; }
  .announce-foot span { display: none; }
  .announce-ok { width: 100%; }
}
</style>
