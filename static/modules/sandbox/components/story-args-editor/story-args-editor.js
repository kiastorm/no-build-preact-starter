import { html } from "htm/preact";
import styles from "./story-args-editor.css" with { type: "css" };

document.adoptedStyleSheets.push(styles);

export const StoryArgsEditor = ({
  initialArgs,
  argsSignalForEditor,
  // componentPropsDef, // REMOVED
}) => {
  console.log("[StoryArgsEditor] Received props. initialArgs:", initialArgs, "argsSignalForEditor:", argsSignalForEditor);

  const argKeys = Object.keys(initialArgs);

  const handleChange = (key, value) => {
    if (!argsSignalForEditor) {
      console.error("[StoryArgsEditor] handleChange called but argsSignalForEditor is missing!");
      return;
    }
    argsSignalForEditor.value = {
      ...(argsSignalForEditor.value || {}),
      [key]: value,
    };
  };

  return html`
    <div class="story-args-editor">
      <h4>Args</h4>
      ${argKeys.map((key) => {
        const initialValue = initialArgs[key];
        let inputElement;

        const currentSignalValue = argsSignalForEditor ? argsSignalForEditor.value : undefined;
        const currentValue = (currentSignalValue && typeof currentSignalValue === 'object' && currentSignalValue.hasOwnProperty(key))
                           ? currentSignalValue[key]
                           : initialValue;

        if (typeof initialValue === "boolean") {
          inputElement = html`<input
            id="arg-${key}"
            type="checkbox"
            checked=${currentValue}
            onChange=${(e) => handleChange(key, e.target.checked)}
          />`;
        } else if (typeof initialValue === "string") {
          inputElement = html`<input
            id="arg-${key}"
            type="text"
            value=${currentValue}
            onInput=${(e) => handleChange(key, e.target.value)}
          />`;
        } else if (typeof initialValue === "number") {
          inputElement = html`<input
            id="arg-${key}"
            type="number"
            value=${currentValue}
            onInput=${(e) => handleChange(key, parseFloat(e.target.value) || 0)}
          />`;
        } else {
          inputElement = html`<span>Unsupported arg type (key: ${key})</span>`;
        }

        return html`
          <div class="arg-control" key=${key}>
            <label for="arg-${key}">${key}:</label>
            ${inputElement}
          </div>
        `;
      })}
    </div>
  `;
};
