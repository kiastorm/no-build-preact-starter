// Store the iframe element reference
let iframeElement = null;

/**
 * Initializes the iframe manager with the iframe element.
 * @param {HTMLIFrameElement} iframe - The iframe DOM element.
 */
export function initIframeManager(iframe) {
  if (!iframe) {
    console.error("[IframeManager] Init called with null iframe element.");
    iframeElement = null; // Ensure it's null if invalid input
    return;
  }
  if (iframeElement === iframe) {
    console.log("[IframeManager] Already initialized with this iframe.");
    return;
  }
  iframeElement = iframe;
  console.log("[IframeManager] Initialized with iframe:", iframeElement);
}

/**
 * Returns the managed iframe element.
 * @returns {HTMLIFrameElement | null}
 */
export function getManagedIframe() {
  return iframeElement;
}

// Removed functions:
// - sendToIframe
// - updateCurrentStoryArgs
// - setupAndRenderCurrentStoryInIframe
// - rerenderStoryInIframe
// - Effect for theme changes (this should be handled by reloading iframe src if needed)
