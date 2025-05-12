import { createContext as createReactContext } from "preact";
import { useContext as useReactContext } from "preact/hooks";

/**
 * @typedef {Object} CreateContextOptions
 * @property {boolean} [strict]
 * @property {string} [hookName]
 * @property {string} [providerName]
 * @property {string} [errorMessage]
 * @property {string} [name]
 * @property {*} [defaultValue]
 */

/**
 * @typedef {[any, function():*, any]} CreateContextReturn
 */

/**
 * @param {string} hook
 * @param {string} provider
 * @returns {string}
 */
function getErrorMessage(hook, provider) {
  return `${hook} returned \`undefined\`. Seems you forgot to wrap component within ${provider}`;
}

/**
 * @param {CreateContextOptions} [options={}]
 * @returns {CreateContextReturn}
 */
export function createContext(options = {}) {
  const {
    name,
    strict = true,
    hookName = "useContext",
    providerName = "Provider",
    errorMessage,
    defaultValue,
  } = options;

  const Context = createReactContext(defaultValue);

  Context.displayName = name;

  function useContext() {
    const context = useReactContext(Context);

    if (!context && strict) {
      const error = new Error(
        errorMessage ?? getErrorMessage(hookName, providerName)
      );
      error.name = "ContextError";
      Error.captureStackTrace?.(error, useContext);
      throw error;
    }

    return context;
  }

  return [Context.Provider, useContext, Context];
}
