import { signal, effect } from "@preact/signals";

/**
 * Creates a Preact signal that is automatically synced with localStorage.
 *
 * @param {string} storageKey The key to use in localStorage.
 * @param {any} defaultValue The default value to use if nothing is found in localStorage.
 * @returns {import("@preact/signals").Signal<any>} A Preact signal.
 */
export function createReactiveLocalStorageSignal(storageKey, defaultValue) {
  const storedValue = localStorage.getItem(storageKey);
  const initialValue = storedValue ? JSON.parse(storedValue) : defaultValue;
  const reactiveSignal = signal(initialValue);

  effect(() => {
    localStorage.setItem(storageKey, JSON.stringify(reactiveSignal.value));
    console.log(
      `ReactiveLocalStorage: '${storageKey}' updated to`,
      reactiveSignal.value
    );
  });

  return reactiveSignal;
}

// Example Usage (you can create specific global signals here or import this utility elsewhere)

export const globalThemeSignal = createReactiveLocalStorageSignal(
  "globalTheme",
  "light"
); // e.g., 'light' or 'dark'
// export const globalSettingsSignal = createReactiveLocalStorageSignal('globalSettings', { showHints: true, volume: 0.8 });
