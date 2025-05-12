// Utility functions for handling URL parameters and argument strings.
import { effect } from "@preact/signals"; // Import effect

/**
 * Parses a custom arguments string (e.g., "key1:val1;key2:val2") into an object.
 * Handles boolean values (true, !true, false, !false) and decodes URI components.
 * @param {string | null | undefined} argsString - The string to parse.
 * @returns {object} The parsed arguments object.
 */
export function parseCustomArgsString(argsString) {
  const args = {};
  if (!argsString) {
    return args;
  }
  argsString.split(";").forEach((pairStr) => {
    if (!pairStr) return;
    const parts = pairStr.split(/:(.+)/); // Split only on the first colon
    if (parts.length >= 2) {
      const key = parts[0].trim();
      let valueStr = parts[1].trim();

      if (valueStr === "true") {
        args[key] = true;
      } else if (valueStr === "!true") {
        // This implies false, as per current logic.
        // Consider if "!true" should distinctly mean "explicitly false" vs. just "false".
        // For now, aligning with existing behavior.
        args[key] = false;
      } else if (valueStr === "false") {
        args[key] = false;
      } else if (valueStr === "!false") {
        // This implies true.
        args[key] = true;
      } else {
        let decodedValueStr;
        try {
          decodedValueStr = decodeURIComponent(valueStr);
        } catch (e) {
          console.warn(
            `[URLUtils] Failed to decode URI component: ${valueStr}`,
            e
          );
          decodedValueStr = valueStr; // Fallback to original if decoding fails
        }

        // Check if it might be a number after decoding
        if (
          !isNaN(decodedValueStr) &&
          decodedValueStr.trim() !== "" &&
          !/\\s/.test(decodedValueStr) && // ensure no internal spaces for numbers
          String(Number(decodedValueStr)) === decodedValueStr.trim()
        ) {
          args[key] = Number(decodedValueStr);
        } else {
          args[key] = decodedValueStr; // Treat as string
        }
      }
    } else if (parts.length === 1 && parts[0].trim() !== "") {
      console.warn(`[URLUtils] Malformed arg pair: ${pairStr}`);
    }
  });
  return args;
}

/**
 * Formats an arguments object into a custom string (e.g., "key1:val1;key2:val2").
 * Encodes string values and handles boolean values with "true" or "!true".
 * @param {object} argsObject - The arguments object to format.
 * @returns {string} The formatted arguments string.
 */
export function formatArgsToCustomString(argsObject) {
  const pairs = [];
  for (const [key, value] of Object.entries(argsObject)) {
    let valueStr;
    if (typeof value === "boolean") {
      // Using "true" for true, and "!true" for false, as per original logic in index.html
      // This seems a bit unconventional; typically one might use "true" and "false".
      // Sticking to existing pattern for now.
      valueStr = value ? "true" : "!true";
    } else if (typeof value === "number") {
      valueStr = String(value);
    } else if (value === null || value === undefined) {
      valueStr = ""; // Represent null/undefined as empty string in the query
    } else {
      valueStr = encodeURIComponent(String(value)); // encodeURIComponent does not encode '!'
    }
    pairs.push(`${key}:${valueStr}`);
  }
  return pairs.join(";");
}

/**
 * Retrieves arguments from the 'args' URL parameter and parses them.
 * @returns {object} The parsed arguments object from the URL.
 */
export function getArgsFromURL() {
  const params = new URLSearchParams(window.location.search);
  const argsQueryParam = params.get("args");
  if (argsQueryParam) {
    return parseCustomArgsString(argsQueryParam);
  }
  return {};
}

/**
 * Sets up a Preact effect to synchronize an arguments signal with the URL.
 * @param {import("@preact/signals").Signal} argsSignal - The signal containing the arguments object.
 * @returns {() => void} A function to dispose of the effect.
 */
export function syncArgsToURL(argsSignal) {
  const dispose = effect(() => {
    const currentArgs = argsSignal.value;
    if (currentArgs === undefined || currentArgs === null) {
      // Avoid running if signal is not yet populated or reset
      return;
    }
    const otherParams = new URLSearchParams();
    const currentQueryString = window.location.search;
    const currentFullParams = new URLSearchParams(currentQueryString);

    for (const [paramKey, paramValue] of currentFullParams) {
      if (paramKey !== "args") {
        otherParams.append(paramKey, paramValue);
      }
    }

    let newSearchString = otherParams.toString();
    const formattedArgs = formatArgsToCustomString(currentArgs);

    if (formattedArgs) {
      if (newSearchString) {
        newSearchString += "&";
      }
      newSearchString += "args=" + formattedArgs;
    }

    const newUrl = `${window.location.pathname}${
      newSearchString ? "?" + newSearchString : ""
    }${window.location.hash}`;

    if (window.location.href !== newUrl) {
      history.replaceState({}, "", newUrl);
      console.log("[URLUtils] URL updated with new args:", currentArgs);
    }
  });
  return dispose; // Return the dispose function
}
