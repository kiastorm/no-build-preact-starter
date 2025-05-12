import { render } from "preact";
import { html } from "htm/preact";
// import { StoryArgsEditor } from "./components/story-args-editor/story-args-editor.js"; // No longer rendered by ui.js
// import { SandboxToolbar } from "./components/sandbox-toolbar/sandbox-toolbar.js"; // No longer rendered by ui.js

export function renderSandboxUI(
  {
    // storyDefaultArgs,
    // initialArgs,
    // currentArgsSignal,
    // componentName,
    // storyKey,
    // storyModulePath,
  }
) {
  // const toolbarMountPoint = document.getElementById("toolbar-root"); // Removed from index.html
  // if (toolbarMountPoint) {
  //   toolbarMountPoint.innerHTML = ""; // Clear previous content
  //   render(
  //     html`<${SandboxToolbar}
  //       initialArgs=${storyDefaultArgs}
  //       currentArgsSignal=${currentArgsSignal}
  //       componentName=${componentName}
  //       storyKey=${storyKey}
  //       storyModulePath=${storyModulePath}
  //     />`,
  //     toolbarMountPoint
  //   );
  // } else {
  //   console.warn("Sandbox UI: Toolbar mount point 'toolbar-root' not found.");
  // }

  // const argsEditorMountPoint = document.getElementById(
  //   "story-args-editor-root" // Removed from index.html
  // );
  // if (argsEditorMountPoint) {
  //   argsEditorMountPoint.innerHTML = ""; // Clear previous content
  //   render(
  //     html`<${StoryArgsEditor}
  //       initialArgs=${initialArgs}
  //       argsSignalForEditor=${currentArgsSignal}
  //     />`,
  //     argsEditorMountPoint
  //   );
  // } else {
  //   console.warn(
  //     "Sandbox UI: Args editor mount point 'story-args-editor-root' not found."
  //   );
  // }
  console.log(
    "[Sandbox UI] renderSandboxUI called, but Toolbar and ArgsEditor are now Server-Side Rendered."
  );
}

export function clearSandboxUI() {
  // const argsEditorMountPoint = document.getElementById(
  //   "story-args-editor-root" // Removed
  // );
  // if (argsEditorMountPoint) argsEditorMountPoint.innerHTML = "";

  // const toolbarMountPoint = document.getElementById("toolbar-root"); // Removed
  // if (toolbarMountPoint) toolbarMountPoint.innerHTML = "";
  console.log(
    "[Sandbox UI] clearSandboxUI called. Toolbar and ArgsEditor mount points are no longer managed by client-side JS for clearing."
  );
}
