/**
 * @typedef {Function|Object|null|undefined} PossibleRef
 * @description A ref can be a function, an object with a current property, null, or undefined
 */

/**
 * Sets a value to a ref
 * @param {PossibleRef} ref - The ref to set
 * @param {*} value - The value to set the ref to
 */
function setRef(ref, value) {
  if (typeof ref === "function") {
    return ref(value);
  } else if (ref !== null && ref !== undefined) {
    ref.current = value;
  }
}

/**
 * Composes multiple refs into a single ref callback
 * @param {...PossibleRef} refs - The refs to compose
 * @returns {Function} A function that sets all refs to the same value
 */
export function composeRefs(...refs) {
  return (node) => {
    refs.forEach((ref) => {
      setRef(ref, node);
    });
  };
}
