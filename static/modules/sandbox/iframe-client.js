import { render } from "preact";
// import { story } from "../../lib/csf/index.js"; // Remove unused story import if not used directly here
// import { html } from "htm/preact"; // htm is not used directly in this client, stories use it.

console.log("iframe-client.js: Loaded");

let currentAppliedTheme = null;
// let currentAppliedImportMap = null; // Removed: Import map is now part of the initial document

/**
 * Applies the theme to the iframe document if it has changed.
 * @param {string} theme - The theme to apply ("light" or "dark").
 */
function applyTheme(theme) {
  if (theme === currentAppliedTheme) {
    console.log(`iframe-client: Theme '${theme}' already applied.`);
    return;
  }

  if (!document.documentElement || !document.body) {
    console.error(
      "iframe-client: DocumentElement or body not found for applying theme."
    );
    return;
  }

  const classList = document.documentElement.classList;
  const bodyClassList = document.body.classList;

  if (theme === "dark") {
    classList.add("dark-theme");
    classList.remove("light-theme");
    bodyClassList.add("dark-theme");
    bodyClassList.remove("light-theme");
  } else {
    classList.add("light-theme");
    classList.remove("dark-theme");
    bodyClassList.add("light-theme");
    bodyClassList.remove("dark-theme");
  }
  currentAppliedTheme = theme;
  console.log(`iframe-client: Theme applied - ${theme}`);
}

/**
 * Applies the import map to the iframe document if it has changed.
 * The initial import map is expected to be in the srcdoc.
 * This function handles updates if a new map is sent.
 * @param {object} importMapContent - The import map object.
 */
// function applyImportMap(importMapContent) { // Removed function
//   const newImportMapString = JSON.stringify(importMapContent);
//   if (newImportMapString === currentAppliedImportMap) {
//     console.log("iframe-client: Import map already applied/identical.");
//     return;
//   }

//   let mapElement = document.getElementById("dynamic-importmap");
//   if (!mapElement) {
//     // Attempt to find one from srcdoc, or create if absolutely necessary
//     mapElement = document.querySelector('script[type="importmap"]');
//     if (mapElement) {
//       console.log("iframe-client: Found existing import map from srcdoc.");
//       mapElement.id = "dynamic-importmap"; // Ensure it has an ID for future updates
//     } else {
//       console.warn(
//         "iframe-client: No initial import map found in srcdoc with ID 'dynamic-importmap', creating a new one."
//       );
//       mapElement = document.createElement("script");
//       mapElement.id = "dynamic-importmap";
//       mapElement.type = "importmap";
//       // Prepend to head to ensure it's processed early, though ideally it's there from srcdoc
//       document.head.insertBefore(mapElement, document.head.firstChild);
//     }
//   }

//   mapElement.textContent = newImportMapString;
//   currentAppliedImportMap = newImportMapString;
//   console.log("iframe-client: Import map updated.", importMapContent);
// }

/**
 * Displays a formatted error message within the iframe content area.
 * @param {HTMLElement} mountPoint - The DOM element to display the error in.
 * @param {string} title - The main error title.
 * @param {string} details - Specific details about the error.
 * @param {Error} [errorObject] - Optional underlying error object for console logging.
 */
function displayErrorInIframe(mountPoint, title, details, errorObject) {
  if (!mountPoint) return; // Should not happen if called correctly

  console.error(
    `[iframe-client] Error: ${title} - ${details}`,
    errorObject || ""
  );

  // Basic inline styles for visibility. Assumes global.css provides some vars or defaults.
  // mountPoint.innerHTML = `
  //       <div style="border: 1px solid var(--red-7, #e5484d); background-color: var(--red-2, #fff0f0); color: var(--red-11, #c53944); padding: 1em; border-radius: 4px; margin: 1em 0;">
  //           <h3 style="margin-top: 0; color: var(--red-11, #c53944);">${title}</h3>
  //           <p>${details}</p>
  //           ${
  //             errorObject
  //               ? `<p><small>See browser console for more details (Error: ${errorObject.message}).</small></p>`
  //               : ""
  //           }
  //       </div>
  //   `;
  // Enhanced error display
  mountPoint.innerHTML = `
    <div style="
      padding: var(--space-5, 20px); 
      background-color: var(--sage-1, #f9fafb); 
      color: var(--sage-12, #111827); 
      font-family: var(--default-font-family, sans-serif);
      border: 1px solid var(--red-7, #e5484d); 
      border-radius: var(--radius-2, 4px);
      margin: var(--space-5, 20px) 0;
      height: 100%;
      box-sizing: border-box;
    ">
      <h1 style="
        font-size: var(--font-size-5, 1.5rem); 
        color: var(--red-11, #c53944); 
        margin-top: 0; 
        margin-bottom: var(--space-3, 0.75rem);
      ">CSR Error: ${title}</h1>
      <p style="margin-bottom: var(--space-2, 0.5rem); word-wrap: break-word;">${details}</p>
      ${
        errorObject
          ? `<pre style="
        background-color: var(--sage-3, #e4e4e7); 
        color: var(--red-11, #c53944); 
        padding: var(--space-3, 0.75rem); 
        border-radius: var(--radius-1, 2px); 
        overflow-x: auto; 
        font-size: var(--font-size-1, 0.875rem);
        white-space: pre-wrap; 
        word-break: break-all;
      ">Error: ${errorObject.message}${
              errorObject.stack
                ? `\nStack: ${errorObject.stack
                    .replace(/</g, "&lt;")
                    .replace(/>/g, "&gt;")}`
                : ""
            }</pre>`
          : ""
      }
      <p style="font-size: var(--font-size-1, 0.875rem); color: var(--sage-11, #52525b); margin-top: var(--space-3, 0.75rem);">
        Please check the component's story definition, the story key, and the browser console for more technical details.
      </p>
    </div>
  `;
}

/**
 * Loads the specified story module and renders it using Preact.
 * @param {object} payload - The story rendering details from the message.
 * @param {string} payload.storyKey
 * @param {string} payload.storyModulePath
 * @param {string} payload.componentName
 * @param {object} payload.args
 */
async function loadAndRenderStory({
  storyKey,
  storyModulePath,
  componentName,
  args,
}) {
  console.log(
    `iframe-client: Rendering story: '${storyKey}' from '${storyModulePath}'. Args:`,
    args
  );

  const mountPoint = document.getElementById("csr-content-root"); // Changed ID
  if (!mountPoint) {
    console.error("iframe-client: Mount point #csr-content-root not found.");
    if (document.body)
      document.body.innerHTML =
        "<p><em>iframe-client Critical Error: Mount point #csr-content-root not found.</em></p>";
    return;
  }

  // **Clear loading message before attempting to load/render**
  mountPoint.innerHTML = "";

  try {
    // A brief pause might not be necessary if the DOM is stable.
    // await new Promise(resolve => setTimeout(resolve, 0));

    const module = await import(storyModulePath);
    console.log(
      `iframe-client: Module '${storyModulePath}' loaded for story '${storyKey}':`,
      module
    );

    if (!module) {
      throw new Error(
        `Module '${storyModulePath}' loaded as null or undefined.`
      );
    }

    if (!module[storyKey]) {
      const availableKeys = Object.keys(module).join(", ") || "None";
      throw new Error(
        `Story key '${storyKey}' not found in module '${storyModulePath}'. Available exports: ${availableKeys}.`
      );
    }

    const storyObject = module[storyKey];

    if (!storyObject || typeof storyObject.render !== "function") {
      throw new Error(
        `Story '${storyKey}' in module '${storyModulePath}' is missing a valid '.render' function.`
      );
    }

    // --- Success Path ---
    const storyElement = storyObject.render(storyObject.args);

    if (storyElement === undefined || storyElement === null) {
      console.warn(
        `iframe-client: Story '${storyKey}' render function returned null or undefined. Rendering empty.`
      );
    }

    // Use Preact's render function directly on the mount point
    render(storyElement, mountPoint); // Render the VNode (or null/undefined) into the mount point

    console.log(`iframe-client: Story '${storyKey}' successfully rendered.`);
    // --- End Success Path ---
  } catch (err) {
    // --- Error Path ---
    let title = "Client-Side Rendering Error";
    let details = `Failed to load or render story '${storyKey}' from module '${storyModulePath}'.`;

    // Customize details based on error type if possible
    if (err.message.includes("not found in module")) {
      title = "Story Not Found";
      details = err.message; // Use the more specific error message
    } else if (err.message.includes("missing a valid '.render' function")) {
      title = "Invalid Story Format";
      details = err.message;
    } else if (
      err.message.includes("Failed to fetch dynamically imported module")
    ) {
      title = "Module Load Error";
      details = `Could not load the story module '${storyModulePath}'. Check the file path and network requests.`;
    }

    displayErrorInIframe(mountPoint, title, details, err);
    // --- End Error Path ---
  }
}

/**
 * Handles messages received from the parent window.
 * @param {MessageEvent} event
 */
// function handleIncomingMessage(event) { // Removed function
//   // Basic security: Check origin for cross-origin scenarios. For srcdoc, it's often "null" or parent's origin.
//   // if (event.origin !== "expected_parent_origin_if_not_srcdoc_or_srcdoc_known_behavior") {
//   //   console.warn("iframe-client: Message from untrusted origin ignored:", event.origin);
//   //   return;
//   // }
//   console.log("iframe-client: Message received from parent:", event.data);

//   const { type, payload } = event.data || {}; // Ensure event.data exists

//   if (!type || !payload) {
//     console.warn(
//       "iframe-client: Received malformed message (no type or payload).",
//       event.data
//     );
//     return;
//   }

//   switch (type) {
//     case "RENDER_STORY":
//       // The import map from srcdoc should be primary. This function will update if different.
//       // applyImportMap(payload.importMap); // Removed
//       applyTheme(payload.theme);
//       loadAndRenderStory(payload); // Pass the whole payload for convenience
//       break;
//     default:
//       console.warn(`iframe-client: Unknown message type received: '${type}'`);
//   }
// }

// window.addEventListener("message", handleIncomingMessage); // Removed listener

// Optional: Notify parent that the client is ready and listening.
// This can be useful if the parent needs to wait before sending the first message.
// window.parent.postMessage({ type: "IFRAME_CLIENT_READY", payload: { origin: window.location.origin } }, "*"); // Target parent specifically if possible

// --- Self-initialization from embedded config ---
function initializeFromConfig() {
  const mountPoint = document.getElementById("csr-content-root");
  if (!mountPoint) {
    // This error is critical and means the basic iframe HTML structure is wrong.
    console.error(
      "iframe-client: Critical: Mount point #csr-content-root not found on init."
    );
    if (document.body) {
      // Fallback to displaying error in body if mountPoint itself is missing
      document.body.style.margin = "0"; // Reset body margin for full-page error
      document.body.style.height = "100vh";
      displayErrorInIframe(
        document.body,
        "Initialization Error",
        "Core iframe structure is missing: #csr-content-root not found.",
        new Error("Cannot find #csr-content-root")
      );
    }
    return;
  }
  // Clear any initial content (e.g., "Loading story...") from mountPoint
  mountPoint.innerHTML = "";

  const configElement = document.getElementById("sandbox-config");
  if (!configElement) {
    console.error("iframe-client: #sandbox-config script tag not found.");
    displayErrorInIframe(
      mountPoint,
      "Initialization Error",
      "Sandbox configuration data is missing from the iframe.",
      new Error("Cannot find #sandbox-config")
    );
    return;
  }

  try {
    const config = JSON.parse(configElement.textContent);
    console.log("iframe-client: Initializing with embedded config:", config);

    if (!config || typeof config !== "object") {
      throw new Error("Parsed configuration is not a valid object.");
    }

    const theme = config.theme || "light";
    applyTheme(theme);

    // Check if this script is for a fallback page or a story page.
    // The server decides which script to load (iframe-client.js or sandbox-fallback.js).
    // If iframe-client.js is loaded, it expects story details.
    if (!config.storyModulePath || !config.storyKey) {
      console.warn(
        "iframe-client: storyModulePath or storyKey missing in config. This implies an issue if a story was expected."
      );
      displayErrorInIframe(
        mountPoint,
        "Configuration Error",
        "Required story module path or story key is missing in the configuration.",
        new Error("Missing storyModulePath or storyKey in config")
      );
      return;
    }

    loadAndRenderStory({
      storyKey: config.storyKey,
      storyModulePath: config.storyModulePath,
      componentName: config.componentName, // Used for logging/context
      args: config.currentArgs || {}, // Args for the story
    });
  } catch (err) {
    console.error(
      "iframe-client: Error parsing #sandbox-config or initializing:",
      err
    );
    // Ensure mountPoint is cleared again in case of error during parsing but before loadAndRenderStory's own clearing
    if (mountPoint) mountPoint.innerHTML = "";
    displayErrorInIframe(
      mountPoint,
      "Initialization Critical Error",
      "Failed to parse sandbox configuration or a critical error occurred during setup.",
      err
    );
  }
}

// Auto-initialize on script load
if (document.readyState === "loading") {
  document.addEventListener("DOMContentLoaded", initializeFromConfig);
} else {
  initializeFromConfig();
}
