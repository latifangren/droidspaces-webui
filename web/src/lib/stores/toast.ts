import { writable } from 'svelte/store';

export interface ToastItem {
  id: string;
  type: 'success' | 'error' | 'info';
  message: string;
  duration?: number;
}

function createToastStore() {
  const { subscribe, update } = writable<ToastItem[]>([]);

  function add(type: 'success' | 'error' | 'info', message: string, duration = 3500) {
    const id = Math.random().toString(36).substring(2, 9);
    const item: ToastItem = { id, type, message, duration };
    update((items) => [...items, item]);

    if (duration > 0) {
      setTimeout(() => {
        remove(id);
      }, duration);
    }
  }

  function remove(id: string) {
    update((items) => items.filter((t) => t.id !== id));
  }

  return {
    subscribe,
    success: (msg: string, duration = 3500) => add('success', msg, duration),
    error: (msg: string, duration = 4500) => add('error', msg, duration),
    info: (msg: string, duration = 3500) => add('info', msg, duration),
    remove,
  };
}

export const toast = createToastStore();
