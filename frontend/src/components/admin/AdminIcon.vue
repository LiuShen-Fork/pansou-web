<template>
  <svg
    :width="size"
    :height="size"
    viewBox="0 0 24 24"
    fill="none"
    stroke="currentColor"
    :stroke-width="strokeWidth"
    stroke-linecap="round"
    stroke-linejoin="round"
    aria-hidden="true"
  >
    <template v-for="(d, index) in paths" :key="index">
      <path v-if="d.type !== 'circle' && d.type !== 'rect'" :d="d.d" />
      <circle v-else-if="d.type === 'circle'" :cx="d.cx" :cy="d.cy" :r="d.r" />
      <rect v-else :x="d.x" :y="d.y" :width="d.w" :height="d.h" :rx="d.rx" />
    </template>
  </svg>
</template>

<script setup lang="ts">
import { computed } from 'vue';

const props = withDefaults(
  defineProps<{
    name: string;
    size?: number | string;
    strokeWidth?: number;
  }>(),
  { size: 18, strokeWidth: 1.8 }
);

type Shape = { type?: 'path' | 'circle' | 'rect'; d?: string; cx?: number; cy?: number; r?: number; x?: number; y?: number; w?: number; h?: number; rx?: number };

const icons: Record<string, Shape[]> = {
  dashboard: [
    { d: 'M3 3h7v7H3zM14 3h7v4h-7zM14 10h7v11h-7zM3 14h7v7H3z' },
  ],
  users: [
    { d: 'M16 21v-2a4 4 0 0 0-4-4H6a4 4 0 0 0-4 4v2' },
    { type: 'circle', cx: 9, cy: 7, r: 4 },
    { d: 'M22 21v-2a4 4 0 0 0-3-3.87M16 3.13a4 4 0 0 1 0 7.75' },
  ],
  logs: [
    { d: 'M4 5h16M4 12h16M4 19h10' },
    { type: 'circle', cx: 18, cy: 19, r: 2 },
  ],
  settings: [
    { type: 'circle', cx: 12, cy: 12, r: 3 },
    { d: 'M19.4 15a1.65 1.65 0 0 0 .33 1.82l.06.06a2 2 0 1 1-2.83 2.83l-.06-.06a1.65 1.65 0 0 0-1.82-.33 1.65 1.65 0 0 0-1 1.51V21a2 2 0 1 1-4 0v-.09A1.65 1.65 0 0 0 9 19.4a1.65 1.65 0 0 0-1.82.33l-.06.06a2 2 0 1 1-2.83-2.83l.06-.06A1.65 1.65 0 0 0 4.6 15a1.65 1.65 0 0 0-1.51-1H3a2 2 0 1 1 0-4h.09A1.65 1.65 0 0 0 4.6 9a1.65 1.65 0 0 0-.33-1.82l-.06-.06a2 2 0 1 1 2.83-2.83l.06.06A1.65 1.65 0 0 0 9 4.6a1.65 1.65 0 0 0 1-1.51V3a2 2 0 1 1 4 0v.09a1.65 1.65 0 0 0 1 1.51 1.65 1.65 0 0 0 1.82-.33l.06-.06a2 2 0 1 1 2.83 2.83l-.06.06A1.65 1.65 0 0 0 19.4 9a1.65 1.65 0 0 0 1.51 1H21a2 2 0 1 1 0 4h-.09a1.65 1.65 0 0 0-1.51 1z' },
  ],
  key: [
    { type: 'circle', cx: 7.5, cy: 15.5, r: 4.5 },
    { d: 'M10.7 12.3L21 2M16 7l3 3M19 4l3 3' },
  ],
  upstream: [
    { d: 'M21 12a9 9 0 1 1-2.64-6.36M21 3v6h-6' },
  ],
  system: [
    { d: 'M3 12h4l3 8 4-16 3 8h4' },
  ],
  search: [
    { type: 'circle', cx: 11, cy: 11, r: 7 },
    { d: 'M21 21l-4.35-4.35' },
  ],
  plus: [{ d: 'M12 5v14M5 12h14' }],
  edit: [{ d: 'M12 20h9M16.5 3.5a2.1 2.1 0 0 1 3 3L7 19l-4 1 1-4z' }],
  trash: [{ d: 'M3 6h18M8 6V4h8v2M19 6l-1 14H6L5 6M10 11v6M14 11v6' }],
  copy: [{ type: 'rect', x: 9, y: 9, w: 12, h: 12, rx: 2 }, { d: 'M5 15H4a1 1 0 0 1-1-1V4a1 1 0 0 1 1-1h10a1 1 0 0 1 1 1v1' }],
  check: [{ d: 'M20 6L9 17l-5-5' }],
  close: [{ d: 'M18 6L6 18M6 6l12 12' }],
  menu: [{ d: 'M3 6h18M3 12h18M3 18h18' }],
  home: [{ d: 'M3 10.5L12 3l9 7.5V21H3zM9 21v-6h6v6' }],
  arrowLeft: [{ d: 'M19 12H5M12 19l-7-7 7-7' }],
  user: [{ type: 'circle', cx: 12, cy: 8, r: 4 }, { d: 'M4 21a8 8 0 0 1 16 0' }],
  lock: [{ type: 'rect', x: 4, y: 10, w: 16, h: 11, rx: 2 }, { d: 'M8 10V7a4 4 0 0 1 8 0v3' }],
  logout: [{ d: 'M9 21H5a2 2 0 0 1-2-2V5a2 2 0 0 1 2-2h4M16 17l5-5-5-5M21 12H9' }],
  external: [{ d: 'M15 3h6v6M10 14L21 3M21 14v5a2 2 0 0 1-2 2H5a2 2 0 0 1-2-2V5a2 2 0 0 1 2-2h5' }],
  alert: [{ d: 'M10.3 3.9L1.8 18a2 2 0 0 0 1.7 3h17a2 2 0 0 0 1.7-3L13.7 3.9a2 2 0 0 0-3.4 0zM12 9v4M12 17h.01' }],
  clock: [{ type: 'circle', cx: 12, cy: 12, r: 9 }, { d: 'M12 7v5l3 2' }],
  server: [{ type: 'rect', x: 3, y: 4, w: 18, h: 7, rx: 2 }, { type: 'rect', x: 3, y: 13, w: 18, h: 7, rx: 2 }, { d: 'M7 7.5h.01M7 16.5h.01' }],
  mail: [{ type: 'rect', x: 3, y: 5, w: 18, h: 14, rx: 2 }, { d: 'M3 7l9 6 9-6' }],
  shield: [{ d: 'M12 22s8-4 8-10V5l-8-3-8 3v7c0 6 8 10 8 10z' }, { d: 'M9 12l2 2 4-4' }],
  chevronRight: [{ d: 'M9 18l6-6-6-6' }],
  filter: [{ d: 'M22 3H2l8 9.5V19l4 2v-8.5z' }],
  refresh: [{ d: 'M21 12a9 9 0 1 1-3-6.7M21 3v6h-6' }],
  database: [{ d: 'M12 3c4.97 0 9 1.34 9 3s-4.03 3-9 3-9-1.34-9-3 4.03-3 9-3zM3 6v6c0 1.66 4.03 3 9 3s9-1.34 9-3V6M3 12v6c0 1.66 4.03 3 9 3s9-1.34 9-3v-6' }],
  chart: [{ d: 'M3 3v18h18M7 15l4-4 3 3 5-7' }],
};

const paths = computed(() => icons[props.name] || icons.dashboard);
</script>
