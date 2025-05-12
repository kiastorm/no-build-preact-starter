import {
  createElement,
  render as preactRender,
  cloneElement as preactCloneElement,
  createRef,
  Component,
  createContext,
  Fragment,
  options,
  toChildArray,
} from "preact";

import {
  useState,
  useId,
  useReducer,
  useEffect,
  useLayoutEffect,
  useRef,
  useImperativeHandle,
  useMemo,
  useCallback,
  useContext,
  useDebugValue,
} from "preact/hooks";

// Utility functions
function assign(obj, props) {
  for (let i in props) obj[i] = props[i];
  return obj;
}

function shallowDiffers(a, b) {
  for (let i in a) if (i !== "__source" && !(i in b)) return true;
  for (let i in b) if (i !== "__source" && a[i] !== b[i]) return true;
  return false;
}

function is(x, y) {
  return (x === y && (x !== 0 || 1 / x === 1 / y)) || (x !== x && y !== y);
}

// Children API
const mapFn = (children, fn) => {
  if (children == null) return null;
  return toChildArray(toChildArray(children).map(fn));
};

const Children = {
  map: mapFn,
  forEach: mapFn,
  count(children) {
    return children ? toChildArray(children).length : 0;
  },
  only(children) {
    const normalized = toChildArray(children);
    if (normalized.length !== 1) throw "Children.only";
    return normalized[0];
  },
  toArray: toChildArray,
};

// ForwardRef implementation
let oldDiffHook = options._diff;
options._diff = (vnode) => {
  if (vnode.type && vnode.type._forwarded && vnode.ref) {
    vnode.props.ref = vnode.ref;
    vnode.ref = null;
  }
  if (oldDiffHook) oldDiffHook(vnode);
};

const REACT_FORWARD_SYMBOL =
  (typeof Symbol != "undefined" &&
    Symbol.for &&
    Symbol.for("react.forward_ref")) ||
  0xf47;

function forwardRef(fn) {
  function Forwarded(props) {
    let clone = assign({}, props);
    delete clone.ref;
    return fn(clone, props.ref || null);
  }

  Forwarded.$$typeof = REACT_FORWARD_SYMBOL;
  Forwarded.render = Forwarded;
  Forwarded.prototype.isReactComponent = Forwarded._forwarded = true;
  Forwarded.displayName = "ForwardRef(" + (fn.displayName || fn.name) + ")";
  return Forwarded;
}

// Hooks
function useSyncExternalStore(subscribe, getSnapshot) {
  const value = getSnapshot();

  const [{ _instance }, forceUpdate] = useState({
    _instance: { _value: value, _getSnapshot: getSnapshot },
  });

  useLayoutEffect(() => {
    _instance._value = value;
    _instance._getSnapshot = getSnapshot;

    if (didSnapshotChange(_instance)) {
      forceUpdate({ _instance });
    }
  }, [subscribe, value, getSnapshot]);

  useEffect(() => {
    if (didSnapshotChange(_instance)) {
      forceUpdate({ _instance });
    }

    return subscribe(() => {
      if (didSnapshotChange(_instance)) {
        forceUpdate({ _instance });
      }
    });
  }, [subscribe]);

  return value;
}

function didSnapshotChange(inst) {
  const latestGetSnapshot = inst._getSnapshot;
  const prevValue = inst._value;
  try {
    const nextValue = latestGetSnapshot();
    return !is(prevValue, nextValue);
  } catch (error) {
    return true;
  }
}

function startTransition(cb) {
  cb();
}

function useDeferredValue(val) {
  return val;
}

function useTransition() {
  return [false, startTransition];
}

const useInsertionEffect = useLayoutEffect;

// PureComponent
function PureComponent(p, c) {
  this.props = p;
  this.context = c;
}
PureComponent.prototype = new Component();
PureComponent.prototype.isPureReactComponent = true;
PureComponent.prototype.shouldComponentUpdate = function (props, state) {
  return shallowDiffers(this.props, props) || shallowDiffers(this.state, state);
};

// Memo
function memo(c, comparer) {
  function shouldUpdate(nextProps) {
    let ref = this.props.ref;
    let updateRef = ref == nextProps.ref;
    if (!updateRef && ref) {
      ref.call ? ref(null) : (ref.current = null);
    }

    if (!comparer) {
      return shallowDiffers(this.props, nextProps);
    }

    return !comparer(this.props, nextProps) || !updateRef;
  }

  function Memoed(props) {
    this.shouldComponentUpdate = shouldUpdate;
    return createElement(c, props);
  }
  Memoed.displayName = "Memo(" + (c.displayName || c.name) + ")";
  Memoed.prototype.isReactComponent = true;
  Memoed._forwarded = true;
  return Memoed;
}

// Portal
function ContextProvider(props) {
  this.getChildContext = () => props.context;
  return props.children;
}

function Portal(props) {
  const _this = this;
  let container = props._container;

  _this.componentWillUnmount = function () {
    preactRender(null, _this._temp);
    _this._temp = null;
    _this._container = null;
  };

  if (_this._container && _this._container !== container) {
    _this.componentWillUnmount();
  }

  if (!_this._temp) {
    _this._container = container;

    _this._temp = {
      nodeType: 1,
      parentNode: container,
      childNodes: [],
      contains: () => true,
      appendChild(child) {
        this.childNodes.push(child);
        _this._container.appendChild(child);
      },
      insertBefore(child, before) {
        this.childNodes.push(child);
        _this._container.insertBefore(child, before);
      },
      removeChild(child) {
        this.childNodes.splice(this.childNodes.indexOf(child) >>> 1, 1);
        _this._container.removeChild(child);
      },
    };
  }

  preactRender(
    createElement(ContextProvider, { context: _this.context }, props._vnode),
    _this._temp
  );
}

function createPortal(vnode, container) {
  const el = createElement(Portal, { _vnode: vnode, _container: container });
  el.containerInfo = container;
  return el;
}

// Suspense
function detachedClone(vnode, detachedParent, parentDom) {
  if (vnode) {
    if (vnode._component && vnode._component.__hooks) {
      vnode._component.__hooks._list.forEach((effect) => {
        if (typeof effect._cleanup == "function") effect._cleanup();
      });

      vnode._component.__hooks = null;
    }

    vnode = assign({}, vnode);
    if (vnode._component != null) {
      if (vnode._component._parentDom === parentDom) {
        vnode._component._parentDom = detachedParent;
      }
      vnode._component = null;
    }

    vnode._children =
      vnode._children &&
      vnode._children.map((child) =>
        detachedClone(child, detachedParent, parentDom)
      );
  }

  return vnode;
}

function removeOriginal(vnode, detachedParent, originalParent) {
  if (vnode && originalParent) {
    vnode._original = null;
    vnode._children =
      vnode._children &&
      vnode._children.map((child) =>
        removeOriginal(child, detachedParent, originalParent)
      );

    if (vnode._component) {
      if (vnode._component._parentDom === detachedParent) {
        if (vnode._dom) {
          originalParent.appendChild(vnode._dom);
        }
        vnode._component._force = true;
        vnode._component._parentDom = originalParent;
      }
    }
  }

  return vnode;
}

function suspended(vnode) {
  let component = vnode._parent._component;
  return component && component._suspended && component._suspended(vnode);
}

function Suspense() {
  this._pendingSuspensionCount = 0;
  this._suspenders = null;
  this._detachOnNextRender = null;
}

Suspense.prototype = new Component();

Suspense.prototype._childDidSuspend = function (promise, suspendingVNode) {
  const suspendingComponent = suspendingVNode._component;
  const c = this;

  if (c._suspenders == null) {
    c._suspenders = [];
  }
  c._suspenders.push(suspendingComponent);

  const resolve = suspended(c._vnode);

  let resolved = false;
  const onResolved = () => {
    if (resolved) return;

    resolved = true;
    suspendingComponent._onResolve = null;

    if (resolve) {
      resolve(onSuspensionComplete);
    } else {
      onSuspensionComplete();
    }
  };

  suspendingComponent._onResolve = onResolved;

  const onSuspensionComplete = () => {
    if (!--c._pendingSuspensionCount) {
      if (c.state._suspended) {
        const suspendedVNode = c.state._suspended;
        c._vnode._children[0] = removeOriginal(
          suspendedVNode,
          suspendedVNode._component._parentDom,
          suspendedVNode._component._originalParentDom
        );
      }

      c.setState({ _suspended: (c._detachOnNextRender = null) });

      let suspended;
      while ((suspended = c._suspenders.pop())) {
        suspended.forceUpdate();
      }
    }
  };

  if (!c._pendingSuspensionCount++) {
    c.setState({ _suspended: (c._detachOnNextRender = c._vnode._children[0]) });
  }
  promise.then(onResolved, onResolved);
};

Suspense.prototype.componentWillUnmount = function () {
  this._suspenders = [];
};

Suspense.prototype.render = function (props, state) {
  if (this._detachOnNextRender) {
    if (this._vnode._children) {
      const detachedParent = document.createElement("div");
      const detachedComponent = this._vnode._children[0]._component;
      this._vnode._children[0] = detachedClone(
        this._detachOnNextRender,
        detachedParent,
        (detachedComponent._originalParentDom = detachedComponent._parentDom)
      );
    }

    this._detachOnNextRender = null;
  }

  const fallback =
    state._suspended && createElement(Fragment, null, props.fallback);

  return [
    createElement(Fragment, null, state._suspended ? null : props.children),
    fallback,
  ];
};

function lazy(loader) {
  let prom;
  let component;
  let error;

  function Lazy(props) {
    if (!prom) {
      prom = loader();
      prom.then(
        (exports) => {
          component = exports.default || exports;
        },
        (e) => {
          error = e;
        }
      );
    }

    if (error) {
      throw error;
    }

    if (!component) {
      throw prom;
    }

    return createElement(component, props);
  }

  Lazy.displayName = "Lazy";
  Lazy._forwarded = true;
  return Lazy;
}

// SuspenseList
function SuspenseList() {
  this._next = null;
  this._map = null;
}

const resolve = (list, child, node) => {
  if (++node[1] === node[0]) {
    list._map.delete(child);
  }

  if (
    !list.props.revealOrder ||
    (list.props.revealOrder[0] === "t" && list._map.size)
  ) {
    return;
  }

  node = list._next;
  while (node) {
    while (node.length > 3) {
      node.pop()();
    }
    if (node[1] < node[0]) {
      break;
    }
    list._next = node = node[2];
  }
};

SuspenseList.prototype = new Component();

SuspenseList.prototype._suspended = function (child) {
  const list = this;
  const delegated = suspended(list._vnode);

  let node = list._map.get(child);
  node[0]++;

  return (unsuspend) => {
    const wrappedUnsuspend = () => {
      if (!list.props.revealOrder) {
        unsuspend();
      } else {
        node.push(unsuspend);
        resolve(list, child, node);
      }
    };
    if (delegated) {
      delegated(wrappedUnsuspend);
    } else {
      wrappedUnsuspend();
    }
  };
};

SuspenseList.prototype.render = function (props) {
  this._next = null;
  this._map = new Map();

  const children = toChildArray(props.children);
  if (props.revealOrder && props.revealOrder[0] === "b") {
    children.reverse();
  }
  for (let i = children.length; i--; ) {
    this._map.set(children[i], (this._next = [1, 0, this._next]));
  }
  return props.children;
};

SuspenseList.prototype.componentDidUpdate =
  SuspenseList.prototype.componentDidMount = function () {
    this._map.forEach((node, child) => {
      resolve(this, child, node);
    });
  };

// Render
const REACT_ELEMENT_TYPE =
  (typeof Symbol != "undefined" && Symbol.for && Symbol.for("react.element")) ||
  0xeac7;

const CAMEL_PROPS =
  /^(?:accent|alignment|arabic|baseline|cap|clip(?!PathU)|color|dominant|fill|flood|font|glyph(?!R)|horiz|image(!S)|letter|lighting|marker(?!H|W|U)|overline|paint|pointer|shape|stop|strikethrough|stroke|text(?!L)|transform|underline|unicode|units|v|vector|vert|word|writing|x(?!C))[A-Z]/;
const ON_ANI = /^on(Ani|Tra|Tou|BeforeInp|Compo)/;
const CAMEL_REPLACE = /[A-Z0-9]/g;
const IS_DOM = typeof document !== "undefined";

const onChangeInputType = (type) =>
  (typeof Symbol != "undefined" && typeof Symbol() == "symbol"
    ? /fil|che|rad/
    : /fil|che|ra/
  ).test(type);

Component.prototype.isReactComponent = {};

[
  "componentWillMount",
  "componentWillReceiveProps",
  "componentWillUpdate",
].forEach((key) => {
  Object.defineProperty(Component.prototype, key, {
    configurable: true,
    get() {
      return this["UNSAFE_" + key];
    },
    set(v) {
      Object.defineProperty(this, key, {
        configurable: true,
        writable: true,
        value: v,
      });
    },
  });
});

function render(vnode, parent, callback) {
  if (parent._children == null) {
    parent.textContent = "";
  }

  preactRender(vnode, parent);
  if (typeof callback == "function") callback();

  return vnode ? vnode._component : null;
}

function hydrate(vnode, parent, callback) {
  preactHydrate(vnode, parent);
  if (typeof callback == "function") callback();

  return vnode ? vnode._component : null;
}

let oldEventHook = options.event;
options.event = (e) => {
  if (oldEventHook) e = oldEventHook(e);

  e.persist = empty;
  e.isPropagationStopped = isPropagationStopped;
  e.isDefaultPrevented = isDefaultPrevented;
  return (e.nativeEvent = e);
};

function empty() {}

function isPropagationStopped() {
  return this.cancelBubble;
}

function isDefaultPrevented() {
  return this.defaultPrevented;
}

const classNameDescriptorNonEnumberable = {
  enumerable: false,
  configurable: true,
  get() {
    return this.class;
  },
};

function handleDomVNode(vnode) {
  let props = vnode.props,
    type = vnode.type,
    normalizedProps = {};

  let isNonDashedType = type.indexOf("-") === -1;
  for (let i in props) {
    let value = props[i];

    if (
      (i === "value" && "defaultValue" in props && value == null) ||
      (IS_DOM && i === "children" && type === "noscript") ||
      i === "class" ||
      i === "className"
    ) {
      continue;
    }

    let lowerCased = i.toLowerCase();
    if (i === "defaultValue" && "value" in props && props.value == null) {
      i = "value";
    } else if (i === "download" && value === true) {
      value = "";
    } else if (lowerCased === "translate" && value === "no") {
      value = false;
    } else if (lowerCased[0] === "o" && lowerCased[1] === "n") {
      if (lowerCased === "ondoubleclick") {
        i = "ondblclick";
      } else if (
        lowerCased === "onchange" &&
        (type === "input" || type === "textarea") &&
        !onChangeInputType(props.type)
      ) {
        lowerCased = i = "oninput";
      } else if (lowerCased === "onfocus") {
        i = "onfocusin";
      } else if (lowerCased === "onblur") {
        i = "onfocusout";
      } else if (ON_ANI.test(i)) {
        i = lowerCased;
      }
    } else if (isNonDashedType && CAMEL_PROPS.test(i)) {
      i = i.replace(CAMEL_REPLACE, "-$&").toLowerCase();
    } else if (value === null) {
      value = undefined;
    }

    if (lowerCased === "oninput") {
      i = lowerCased;
      if (normalizedProps[i]) {
        i = "oninputCapture";
      }
    }

    normalizedProps[i] = value;
  }

  if (
    type == "select" &&
    normalizedProps.multiple &&
    Array.isArray(normalizedProps.value)
  ) {
    normalizedProps.value = toChildArray(props.children).forEach((child) => {
      child.props.selected =
        normalizedProps.value.indexOf(child.props.value) != -1;
    });
  }

  if (type == "select" && normalizedProps.defaultValue != null) {
    normalizedProps.value = toChildArray(props.children).forEach((child) => {
      if (normalizedProps.multiple) {
        child.props.selected =
          normalizedProps.defaultValue.indexOf(child.props.value) != -1;
      } else {
        child.props.selected =
          normalizedProps.defaultValue == child.props.value;
      }
    });
  }

  if (props.class && !props.className) {
    normalizedProps.class = props.class;
    Object.defineProperty(
      normalizedProps,
      "className",
      classNameDescriptorNonEnumberable
    );
  } else if (props.className && !props.class) {
    normalizedProps.class = normalizedProps.className = props.className;
  } else if (props.class && props.className) {
    normalizedProps.class = normalizedProps.className = props.className;
  }

  vnode.props = normalizedProps;
}

let oldVNodeHook = options.vnode;
options.vnode = (vnode) => {
  if (typeof vnode.type === "string") {
    handleDomVNode(vnode);
  }

  vnode.$$typeof = REACT_ELEMENT_TYPE;

  if (oldVNodeHook) oldVNodeHook(vnode);
};

let currentComponent;
const oldBeforeRender = options._render;
options._render = function (vnode) {
  if (oldBeforeRender) {
    oldBeforeRender(vnode);
  }
  currentComponent = vnode._component;
};

const oldDiffed = options.diffed;
options.diffed = function (vnode) {
  if (oldDiffed) {
    oldDiffed(vnode);
  }

  const props = vnode.props;
  const dom = vnode._dom;

  if (
    dom != null &&
    vnode.type === "textarea" &&
    "value" in props &&
    props.value !== dom.value
  ) {
    dom.value = props.value == null ? "" : props.value;
  }

  currentComponent = null;
};

const oldCatchError = options._catchError;
options._catchError = function (error, newVNode, oldVNode, errorInfo) {
  if (error.then) {
    let component;
    let vnode = newVNode;

    for (; (vnode = vnode._parent); ) {
      if ((component = vnode._component) && component._childDidSuspend) {
        if (newVNode._dom == null) {
          newVNode._dom = oldVNode._dom;
          newVNode._children = oldVNode._children;
        }
        return component._childDidSuspend(error, newVNode);
      }
    }
  }
  oldCatchError(error, newVNode, oldVNode, errorInfo);
};

const oldUnmount = options.unmount;
options.unmount = function (vnode) {
  const component = vnode._component;
  if (component && component._onResolve) {
    component._onResolve();
  }

  if (component && vnode._flags & 1) {
    vnode.type = null;
  }

  if (oldUnmount) oldUnmount(vnode);
};

// React API
function createFactory(type) {
  return createElement.bind(null, type);
}

function isValidElement(element) {
  return !!element && element.$$typeof === REACT_ELEMENT_TYPE;
}

function isFragment(element) {
  return isValidElement(element) && element.type === Fragment;
}

function isMemo(element) {
  return (
    !!element &&
    !!element.displayName &&
    (typeof element.displayName === "string" ||
      element.displayName instanceof String) &&
    element.displayName.startsWith("Memo(")
  );
}

function cloneElement(element) {
  if (!isValidElement(element)) return element;
  return preactCloneElement.apply(null, arguments);
}

function unmountComponentAtNode(container) {
  if (container._children) {
    preactRender(null, container);
    return true;
  }
  return false;
}

function findDOMNode(component) {
  return (
    (component &&
      (component.base || (component.nodeType === 1 && component))) ||
    null
  );
}

const unstable_batchedUpdates = (callback, arg) => callback(arg);

const flushSync = (callback, arg) => callback(arg);

const StrictMode = Fragment;

const __SECRET_INTERNALS_DO_NOT_USE_OR_YOU_WILL_BE_FIRED = {
  ReactCurrentDispatcher: {
    current: {
      readContext(context) {
        return currentComponent._globalContext[context._id].props.value;
      },
      useCallback,
      useContext,
      useDebugValue,
      useDeferredValue,
      useEffect,
      useId,
      useImperativeHandle,
      useInsertionEffect,
      useLayoutEffect,
      useMemo,
      useReducer,
      useRef,
      useState,
      useSyncExternalStore,
      useTransition,
    },
  },
};

const version = "18.3.1";

// Export
export {
  version,
  Children,
  render,
  hydrate,
  unmountComponentAtNode,
  createPortal,
  createElement,
  createContext,
  createFactory,
  cloneElement,
  createRef,
  Fragment,
  isValidElement,
  isFragment,
  isMemo,
  findDOMNode,
  Component,
  PureComponent,
  memo,
  forwardRef,
  flushSync,
  useInsertionEffect,
  startTransition,
  useDeferredValue,
  useSyncExternalStore,
  useTransition,
  unstable_batchedUpdates,
  StrictMode,
  Suspense,
  SuspenseList,
  lazy,
  __SECRET_INTERNALS_DO_NOT_USE_OR_YOU_WILL_BE_FIRED,
};

// React copies the named exports to the default one.
export default {
  useState,
  useId,
  useReducer,
  useEffect,
  useLayoutEffect,
  useInsertionEffect,
  useTransition,
  useDeferredValue,
  useSyncExternalStore,
  startTransition,
  useRef,
  useImperativeHandle,
  useMemo,
  useCallback,
  useContext,
  useDebugValue,
  version,
  Children,
  render,
  hydrate,
  unmountComponentAtNode,
  createPortal,
  createElement,
  createContext,
  createFactory,
  cloneElement,
  createRef,
  Fragment,
  isValidElement,
  isFragment,
  isMemo,
  findDOMNode,
  Component,
  PureComponent,
  memo,
  forwardRef,
  flushSync,
  unstable_batchedUpdates,
  StrictMode,
  Suspense,
  SuspenseList,
  lazy,
  __SECRET_INTERNALS_DO_NOT_USE_OR_YOU_WILL_BE_FIRED,
};
