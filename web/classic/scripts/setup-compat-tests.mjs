import i18n from 'i18next';
import { initReactI18next } from 'react-i18next';
import { afterEach, vi } from 'vitest';
import { act, cleanup } from '@testing-library/react';

// Node 25+ no longer exposes a localStorage global unless launched with
// --localstorage-file, which breaks every test that reads quota settings.
// Provide an in-memory implementation so the suite runs on any Node version.
if (typeof localStorage === 'undefined' || typeof localStorage?.getItem !== 'function') {
  const store = new Map();
  const memoryStorage = {
    getItem: (key) => (store.has(key) ? store.get(key) : null),
    setItem: (key, value) => store.set(String(key), String(value)),
    removeItem: (key) => store.delete(key),
    clear: () => store.clear(),
    key: (index) => Array.from(store.keys())[index] ?? null,
    get length() {
      return store.size;
    },
  };
  Object.defineProperty(globalThis, 'localStorage', {
    configurable: true,
    writable: true,
    value: memoryStorage,
  });
}
window.matchMedia = () => ({
  matches: false,
  addListener() {},
  removeListener() {},
  addEventListener() {},
  removeEventListener() {},
});
globalThis.ResizeObserver = class {
  observe() {}
  unobserve() {}
  disconnect() {}
};
Element.prototype.scrollIntoView = () => {};
afterEach(async () => {
  const { Toast } = await import('@douyinfe/semi-ui');
  act(() => Toast.destroyAll());
  cleanup();
  localStorage.clear();
  vi.restoreAllMocks();
});

HTMLCanvasElement.prototype.getContext = () => ({
  fillStyle: '',
  fillRect() {},
  clearRect() {},
  measureText: () => ({ width: 0 }),
  getImageData: () => ({ data: new Uint8ClampedArray(4) }),
});
URL.createObjectURL = () => 'blob:classic-test';
URL.revokeObjectURL = () => {};

await i18n.use(initReactI18next).init({
  lng: 'zh',
  fallbackLng: 'zh',
  resources: { zh: { translation: {} } },
  interpolation: { escapeValue: false },
});
