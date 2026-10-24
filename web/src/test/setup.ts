import '@testing-library/jest-dom/vitest';
import { cleanup } from '@testing-library/react';
import { afterEach } from 'vitest';

// jsdom has no modal dialogs yet; this is just enough for <dialog> to open and close.
if (!('showModal' in HTMLDialogElement.prototype)) {
  Object.assign(HTMLDialogElement.prototype, {
    showModal(this: HTMLDialogElement) {
      this.setAttribute('open', '');
      // Like a browser's dialog focusing steps: the first [autofocus] element, else the first control.
      this.querySelector<HTMLElement>('[autofocus], button, input, select, textarea, a[href]')?.focus();
    },
    close(this: HTMLDialogElement) {
      this.removeAttribute('open');
      this.dispatchEvent(new Event('close'));
    },
  });
}

afterEach(() => {
  cleanup();
});
