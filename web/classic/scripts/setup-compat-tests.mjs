import i18n from 'i18next';
import { initReactI18next } from 'react-i18next';
import { afterEach, vi } from 'vitest';
import { act, cleanup } from '@testing-library/react';
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
