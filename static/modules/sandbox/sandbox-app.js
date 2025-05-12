// static/modules/sandbox/sandbox-app.js

import { signal, effect } from "@preact/signals";
import { globalThemeSignal } from "/static/modules/sandbox/utils/reactive-local-storage.js";
import {
  getArgsFromURL,
  syncArgsToURL,
} from "/static/modules/sandbox/utils/url-utils.js";
import {
  initIframeManager,
  getManagedIframe,
} from "/static/modules/sandbox/iframe-manager.js";
import { loadStory } from "./story-loader.js";

let disposeUrlSyncEffect = null;
let disposeIframeSrcUpdateEffect = null;

async function initializeApp() {
  console.log("[SandboxApp] Initializing...");

  // Dispose previous effects
  if (disposeUrlSyncEffect) {
    console.log("[SandboxApp] Disposing previous URL sync effect.");
    disposeUrlSyncEffect();
    disposeUrlSyncEffect = null;
  }
  if (disposeIframeSrcUpdateEffect) {
    console.log("[SandboxApp] Disposing previous iframe src update effect.");
    disposeIframeSrcUpdateEffect();
    disposeIframeSrcUpdateEffect = null;
  }

  const iframe = document.getElementById("sandbox-iframe");
  if (!iframe) {
    console.error("[SandboxApp] Critical: sandbox-iframe element not found.");
    return;
  }

  // Initialize Iframe Manager (only stores reference now)
  initIframeManager(iframe);

  const configElement = document.getElementById("sandbox-config");
  if (!configElement) {
    console.error("[SandboxApp] Critical: sandbox-config element not found.");
    iframe.srcdoc =
      "<p><em>Critical Error: sandbox-config missing. Cannot load story.</em></p>";
    return;
  }
  const config = JSON.parse(configElement.textContent);
  const { storyModulePath, storyKey, componentName, renderMode } = config;

  // --- Check if we are on a story page that needs iframe management ---
  // This logic might need refinement based on how home/fallback pages are configured.
  // If componentName is missing, we might be on the home page.
  if (!componentName) {
    console.log(
      "[SandboxApp] No componentName in config. Assuming home or non-story page. Skipping iframe src management."
    );
    return;
  }
  // If storyModulePath is missing, it might be a fallback or error state.
  if (!storyModulePath) {
    console.log(
      "[SandboxApp] No storyModulePath in config. Cannot manage iframe source."
    );
    // The iframe might already have an error page loaded via its initial src from Go.
    return;
  }
  // Determine the effective renderMode (passed from Go via config, or default)
  // This script assumes the Go handler already determined the correct mode for the *initial* iframe src.
  // The main role here is to update the src when *args* or *theme* change.
  const currentRenderMode = renderMode || "csr"; // Default to csr if not specified in config

  try {
    // Determine initial args (defaults + URL)
    const { storyDefaultArgs } = await loadStory(storyModulePath, storyKey);
    const urlArgs = getArgsFromURL();
    const initialArgs = { ...storyDefaultArgs, ...urlArgs };
    const currentArgsSignal = signal(initialArgs);

    // Setup URL synchronization
    disposeUrlSyncEffect = syncArgsToURL(currentArgsSignal);

    // Effect to update iframe src when args OR theme change
    disposeIframeSrcUpdateEffect = effect(() => {
      const currentArgs = currentArgsSignal.value;
      const currentTheme = globalThemeSignal.value; // Get current theme
      const managedIframe = getManagedIframe(); // Get iframe ref via manager

      if (!managedIframe) {
        console.warn(
          "[SandboxApp] Effect triggered, but iframe not managed. Skipping src update."
        );
        return;
      }

      console.log(
        `[SandboxApp] Args or Theme updated. Rebuilding iframe src for ${componentName}/${
          storyKey || "(no story)"
        }. Mode: ${currentRenderMode}`
      );

      // --- Construct the new iframe src URL with dynamic path ---
      // ComponentName and storyKey are read from the config closure variable
      let newPath = "/sandbox-content/" + componentName;
      if (storyKey) {
        // storyKey might be empty/null from config
        newPath += "/" + storyKey;
      }

      const newSrcParams = new URLSearchParams();
      newSrcParams.set("renderMode", currentRenderMode); // Use the mode determined on initial load
      newSrcParams.set("theme", currentTheme);
      // componentName and storyKey are now in the path, not query params
      // Add all current args to the query string
      for (const [key, value] of Object.entries(currentArgs)) {
        newSrcParams.set(key, String(value));
      }

      const newSrc = `${newPath}?${newSrcParams.toString()}`; // Combine path and query
      // --- End URL construction ---

      // Update the iframe src ONLY if it's different from the current src
      // to avoid unnecessary reloads. Compare path AND search params.
      try {
        const currentFullUrl = new URL(
          managedIframe.src || "about:blank",
          window.location.origin
        );
        const currentSearchParams = new URLSearchParams(currentFullUrl.search);

        // Normalize and compare params (ignoring order)
        const currentParamsString = Array.from(currentSearchParams.entries())
          .sort()
          .toString();
        const newParamsString = Array.from(newSrcParams.entries())
          .sort()
          .toString();

        // Compare path AND params
        if (
          currentFullUrl.pathname === newPath &&
          currentParamsString === newParamsString
        ) {
          console.log(
            "[SandboxApp] Iframe src path and params unchanged, skipping reload."
          );
        } else {
          console.log("[SandboxApp] Updating iframe src to:", newSrc);
          managedIframe.src = newSrc;
        }
      } catch (e) {
        // If current src is invalid (e.g., srcdoc), just set the new one
        console.log(
          "[SandboxApp] Cannot compare current iframe src, setting new src:",
          newSrc
        );
        managedIframe.src = newSrc;
      }
    });
  } catch (error) {
    console.error(
      "[SandboxApp] Error during sandbox initialization:",
      error.message,
      error
    );
    const iframe = getManagedIframe();
    if (iframe) {
      try {
        const errorDisplayHTML = `<p><em>Parent App Error: ${error.message}. Check console for details.</em></p>`;
        iframe.srcdoc = `<!DOCTYPE html><html><head><title>Error</title><link rel="stylesheet" href="/static/styles/global.css"/><style>body{padding:var(--space-5);}</style></head><body>${errorDisplayHTML}</body></html>`;
      } catch (displayError) {
        console.error(
          "[SandboxApp] Could not display error in iframe:",
          displayError
        );
      }
    }
  }
}

// Listen for mach-link navigation completion to re-initialize
document.addEventListener("mach:contentupdated", (event) => {
  console.log(
    "[SandboxApp] 'mach:contentupdated' event received. Re-initializing.",
    event.detail
  );
  initializeApp();
});

// Initial call setup: Wait for the DOM to be ready
if (document.readyState === "loading") {
  document.addEventListener("DOMContentLoaded", initializeApp);
  console.log("[SandboxApp] DOM not ready, waiting for DOMContentLoaded.");
} else {
  console.log("[SandboxApp] DOM already ready, initializing immediately.");
  initializeApp();
}
