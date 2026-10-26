import { reactive } from 'vue';

export type ToastType = 'success' | 'error' | 'info';

export interface ToastItem {
  id: number;
  type: ToastType;
  title: string;
  message?: string;
}

export const toasts = reactive<ToastItem[]>([]);

let seq = 0;

function push(type: ToastType, title: string, message = '', timeout = 3800) {
  const id = ++seq;
  toasts.push({ id, type, title, message });
  window.setTimeout(() => dismissToast(id), timeout);
}

export function dismissToast(id: number) {
  const index = toasts.findIndex((item) => item.id === id);
  if (index >= 0) toasts.splice(index, 1);
}

export const toast = {
  success: (title: string, message = '') => push('success', title, message),
  error: (title: string, message = '') => push('error', title, message, 5000),
  info: (title: string, message = '') => push('info', title, message),
};
