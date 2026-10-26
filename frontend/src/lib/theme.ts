export type ThemeMode = 'light' | 'dark' | 'auto';

const STORAGE_KEY = 'pansou_theme';
const media = typeof window !== 'undefined' ? window.matchMedia('(prefers-color-scheme: dark)') : null;
const listeners = new Set<(mode: ThemeMode) => void>();

export function getStoredTheme(): ThemeMode {
  const value = localStorage.getItem(STORAGE_KEY);
  return value === 'light' || value === 'dark' || value === 'auto' ? value : 'auto';
}

function resolvedTheme(mode: ThemeMode): 'light' | 'dark' {
  if (mode === 'auto') return media?.matches ? 'dark' : 'light';
  return mode;
}

export function applyTheme(mode: ThemeMode = getStoredTheme()) {
  const resolved = resolvedTheme(mode);
  const root = document.documentElement;
  root.classList.toggle('dark', resolved === 'dark');
  root.dataset.theme = resolved;
  root.dataset.themeMode = mode;
  root.style.colorScheme = resolved;
  document
    .querySelector('meta[name="theme-color"]')
    ?.setAttribute('content', resolved === 'dark' ? '#0f111a' : '#eef2ff');
  listeners.forEach((fn) => fn(mode));
}

export function setTheme(mode: ThemeMode) {
  localStorage.setItem(STORAGE_KEY, mode);
  applyTheme(mode);
}

/** 在 Vue 组件卸载时注销，避免热更新重复注册。 */
export function onThemeChange(fn: (mode: ThemeMode) => void): () => void {
  listeners.add(fn);
  return () => listeners.delete(fn);
}

export function initTheme() {
  applyTheme(getStoredTheme());
  const handler = () => {
    if (getStoredTheme() === 'auto') applyTheme('auto');
  };
  if (media?.addEventListener) media.addEventListener('change', handler);
  else media?.addListener?.(handler);
}
