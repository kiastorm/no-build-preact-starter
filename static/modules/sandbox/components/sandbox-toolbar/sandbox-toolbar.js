import { html } from "htm/preact";
import styles from "./sandbox-toolbar.css" with { type: "css" };
import { globalThemeSignal } from "/static/modules/sandbox/utils/reactive-local-storage.js";
import { Button } from "/static/components/button/button.js";
import { signal } from "@preact/signals";
import { setupAndRenderCurrentStoryInIframe } from "../../iframe-manager.js";

document.adoptedStyleSheets.push(styles);

// Signal to track JS sandbox state, true if JS is allowed
const jsSandboxEnabled = signal(true); 

export const SandboxToolbar = ({
  initialArgs,
  currentArgsSignal,
  componentName,
  storyKey,
  storyModulePath,
}) => {
  const handleResetArgs = () => {
    currentArgsSignal.value = JSON.parse(JSON.stringify(initialArgs));
    console.log("Arguments reset to initial state:", currentArgsSignal.value);
  };

  const toggleTheme = () => {
    globalThemeSignal.value = globalThemeSignal.value === 'light' ? 'dark' : 'light';
  };

  const toggleJsSandbox = () => {
    const iframe = document.getElementById("registry-root");
    if (!iframe) {
      console.error("Could not find iframe #registry-root");
      return;
    }

    let currentSandbox = iframe.getAttribute('sandbox') || "";
    let tokens = currentSandbox.split(' ').map(t => t.trim()).filter(t => t.length > 0);
    
    const allowScriptsToken = "allow-scripts";
    const hasAllowScripts = tokens.includes(allowScriptsToken);

    if (jsSandboxEnabled.value) { // JS is currently enabled, so disable it
      if (hasAllowScripts) {
        tokens = tokens.filter(t => t !== allowScriptsToken);
      }
    } else { // JS is currently disabled, so enable it
      if (!hasAllowScripts) {
        tokens.push(allowScriptsToken);
      }
    }
    
    iframe.setAttribute('sandbox', tokens.join(' '));
    jsSandboxEnabled.value = !jsSandboxEnabled.value;
    console.log(`[SandboxToolbar] Sandbox set to: "${tokens.join(' ')}". JS enabled: ${jsSandboxEnabled.value}`);

    // Force iframe re-initialization with new sandbox policy and current args
    if (typeof setupAndRenderCurrentStoryInIframe === "function") {
      console.log("[SandboxToolbar] Forcing iframe re-initialization for JS toggle.");
      setupAndRenderCurrentStoryInIframe(
        storyKey, // Ensure order matches function definition
        storyModulePath,
        componentName,
        currentArgsSignal.value // Pass current args
      );
    } else {
      console.error("[SandboxToolbar] setupAndRenderCurrentStoryInIframe function not found.");
    }
  };

  return html`
    <div class="sandbox-toolbar ${globalThemeSignal.value}">
      <div class="toolbar-info">
        <span class="component-name">${componentName}</span>
        <span class="story-key">${storyKey}</span>
      </div>
      <div class="toolbar-actions">
        <${Button} 
          variant="neutral" 
          size="2" 
          onClick=${toggleJsSandbox}
        >
          Toggle JS (Currently: ${jsSandboxEnabled.value ? 'Enabled' : 'Disabled'})
        </${Button}>
        <${Button} 
          variant="neutral" 
          size="2" 
          onClick=${toggleTheme}
        >
          Toggle Theme (Current: ${globalThemeSignal.value})
        </${Button}>
        <${Button} 
          variant="neutral" 
          size="2" 
          onClick=${handleResetArgs}
        >
          Reset Args
        </${Button}>
      </div>
    </div>
  `;
}; 
