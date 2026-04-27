/**
 * @license
 * Copyright 2019 Google LLC
 * SPDX-License-Identifier: BSD-3-Clause
 */
const X = globalThis, ne = X.ShadowRoot && (X.ShadyCSS === void 0 || X.ShadyCSS.nativeShadow) && "adoptedStyleSheets" in Document.prototype && "replace" in CSSStyleSheet.prototype, oe = Symbol(), ce = /* @__PURE__ */ new WeakMap();
let _e = class {
  constructor(e, s, a) {
    if (this._$cssResult$ = !0, a !== oe) throw Error("CSSResult is not constructable. Use `unsafeCSS` or `css` instead.");
    this.cssText = e, this.t = s;
  }
  get styleSheet() {
    let e = this.o;
    const s = this.t;
    if (ne && e === void 0) {
      const a = s !== void 0 && s.length === 1;
      a && (e = ce.get(s)), e === void 0 && ((this.o = e = new CSSStyleSheet()).replaceSync(this.cssText), a && ce.set(s, e));
    }
    return e;
  }
  toString() {
    return this.cssText;
  }
};
const Ee = (t) => new _e(typeof t == "string" ? t : t + "", void 0, oe), K = (t, ...e) => {
  const s = t.length === 1 ? t[0] : e.reduce((a, i, r) => a + ((n) => {
    if (n._$cssResult$ === !0) return n.cssText;
    if (typeof n == "number") return n;
    throw Error("Value passed to 'css' function must be a 'css' function result: " + n + ". Use 'unsafeCSS' to pass non-literal values, but take care to ensure page security.");
  })(i) + t[r + 1], t[0]);
  return new _e(s, t, oe);
}, Oe = (t, e) => {
  if (ne) t.adoptedStyleSheets = e.map((s) => s instanceof CSSStyleSheet ? s : s.styleSheet);
  else for (const s of e) {
    const a = document.createElement("style"), i = X.litNonce;
    i !== void 0 && a.setAttribute("nonce", i), a.textContent = s.cssText, t.appendChild(a);
  }
}, pe = ne ? (t) => t : (t) => t instanceof CSSStyleSheet ? ((e) => {
  let s = "";
  for (const a of e.cssRules) s += a.cssText;
  return Ee(s);
})(t) : t;
/**
 * @license
 * Copyright 2017 Google LLC
 * SPDX-License-Identifier: BSD-3-Clause
 */
const { is: De, defineProperty: Re, getOwnPropertyDescriptor: Ue, getOwnPropertyNames: Te, getOwnPropertySymbols: ze, getPrototypeOf: je } = Object, C = globalThis, fe = C.trustedTypes, Be = fe ? fe.emptyScript : "", se = C.reactiveElementPolyfillSupport, q = (t, e) => t, Q = { toAttribute(t, e) {
  switch (e) {
    case Boolean:
      t = t ? Be : null;
      break;
    case Object:
    case Array:
      t = t == null ? t : JSON.stringify(t);
  }
  return t;
}, fromAttribute(t, e) {
  let s = t;
  switch (e) {
    case Boolean:
      s = t !== null;
      break;
    case Number:
      s = t === null ? null : Number(t);
      break;
    case Object:
    case Array:
      try {
        s = JSON.parse(t);
      } catch {
        s = null;
      }
  }
  return s;
} }, le = (t, e) => !De(t, e), he = { attribute: !0, type: String, converter: Q, reflect: !1, useDefault: !1, hasChanged: le };
Symbol.metadata ?? (Symbol.metadata = Symbol("metadata")), C.litPropertyMetadata ?? (C.litPropertyMetadata = /* @__PURE__ */ new WeakMap());
let j = class extends HTMLElement {
  static addInitializer(e) {
    this._$Ei(), (this.l ?? (this.l = [])).push(e);
  }
  static get observedAttributes() {
    return this.finalize(), this._$Eh && [...this._$Eh.keys()];
  }
  static createProperty(e, s = he) {
    if (s.state && (s.attribute = !1), this._$Ei(), this.prototype.hasOwnProperty(e) && ((s = Object.create(s)).wrapped = !0), this.elementProperties.set(e, s), !s.noAccessor) {
      const a = Symbol(), i = this.getPropertyDescriptor(e, a, s);
      i !== void 0 && Re(this.prototype, e, i);
    }
  }
  static getPropertyDescriptor(e, s, a) {
    const { get: i, set: r } = Ue(this.prototype, e) ?? { get() {
      return this[s];
    }, set(n) {
      this[s] = n;
    } };
    return { get: i, set(n) {
      const c = i == null ? void 0 : i.call(this);
      r == null || r.call(this, n), this.requestUpdate(e, c, a);
    }, configurable: !0, enumerable: !0 };
  }
  static getPropertyOptions(e) {
    return this.elementProperties.get(e) ?? he;
  }
  static _$Ei() {
    if (this.hasOwnProperty(q("elementProperties"))) return;
    const e = je(this);
    e.finalize(), e.l !== void 0 && (this.l = [...e.l]), this.elementProperties = new Map(e.elementProperties);
  }
  static finalize() {
    if (this.hasOwnProperty(q("finalized"))) return;
    if (this.finalized = !0, this._$Ei(), this.hasOwnProperty(q("properties"))) {
      const s = this.properties, a = [...Te(s), ...ze(s)];
      for (const i of a) this.createProperty(i, s[i]);
    }
    const e = this[Symbol.metadata];
    if (e !== null) {
      const s = litPropertyMetadata.get(e);
      if (s !== void 0) for (const [a, i] of s) this.elementProperties.set(a, i);
    }
    this._$Eh = /* @__PURE__ */ new Map();
    for (const [s, a] of this.elementProperties) {
      const i = this._$Eu(s, a);
      i !== void 0 && this._$Eh.set(i, s);
    }
    this.elementStyles = this.finalizeStyles(this.styles);
  }
  static finalizeStyles(e) {
    const s = [];
    if (Array.isArray(e)) {
      const a = new Set(e.flat(1 / 0).reverse());
      for (const i of a) s.unshift(pe(i));
    } else e !== void 0 && s.push(pe(e));
    return s;
  }
  static _$Eu(e, s) {
    const a = s.attribute;
    return a === !1 ? void 0 : typeof a == "string" ? a : typeof e == "string" ? e.toLowerCase() : void 0;
  }
  constructor() {
    super(), this._$Ep = void 0, this.isUpdatePending = !1, this.hasUpdated = !1, this._$Em = null, this._$Ev();
  }
  _$Ev() {
    var e;
    this._$ES = new Promise((s) => this.enableUpdating = s), this._$AL = /* @__PURE__ */ new Map(), this._$E_(), this.requestUpdate(), (e = this.constructor.l) == null || e.forEach((s) => s(this));
  }
  addController(e) {
    var s;
    (this._$EO ?? (this._$EO = /* @__PURE__ */ new Set())).add(e), this.renderRoot !== void 0 && this.isConnected && ((s = e.hostConnected) == null || s.call(e));
  }
  removeController(e) {
    var s;
    (s = this._$EO) == null || s.delete(e);
  }
  _$E_() {
    const e = /* @__PURE__ */ new Map(), s = this.constructor.elementProperties;
    for (const a of s.keys()) this.hasOwnProperty(a) && (e.set(a, this[a]), delete this[a]);
    e.size > 0 && (this._$Ep = e);
  }
  createRenderRoot() {
    const e = this.shadowRoot ?? this.attachShadow(this.constructor.shadowRootOptions);
    return Oe(e, this.constructor.elementStyles), e;
  }
  connectedCallback() {
    var e;
    this.renderRoot ?? (this.renderRoot = this.createRenderRoot()), this.enableUpdating(!0), (e = this._$EO) == null || e.forEach((s) => {
      var a;
      return (a = s.hostConnected) == null ? void 0 : a.call(s);
    });
  }
  enableUpdating(e) {
  }
  disconnectedCallback() {
    var e;
    (e = this._$EO) == null || e.forEach((s) => {
      var a;
      return (a = s.hostDisconnected) == null ? void 0 : a.call(s);
    });
  }
  attributeChangedCallback(e, s, a) {
    this._$AK(e, a);
  }
  _$ET(e, s) {
    var r;
    const a = this.constructor.elementProperties.get(e), i = this.constructor._$Eu(e, a);
    if (i !== void 0 && a.reflect === !0) {
      const n = (((r = a.converter) == null ? void 0 : r.toAttribute) !== void 0 ? a.converter : Q).toAttribute(s, a.type);
      this._$Em = e, n == null ? this.removeAttribute(i) : this.setAttribute(i, n), this._$Em = null;
    }
  }
  _$AK(e, s) {
    var r, n;
    const a = this.constructor, i = a._$Eh.get(e);
    if (i !== void 0 && this._$Em !== i) {
      const c = a.getPropertyOptions(i), d = typeof c.converter == "function" ? { fromAttribute: c.converter } : ((r = c.converter) == null ? void 0 : r.fromAttribute) !== void 0 ? c.converter : Q;
      this._$Em = i;
      const h = d.fromAttribute(s, c.type);
      this[i] = h ?? ((n = this._$Ej) == null ? void 0 : n.get(i)) ?? h, this._$Em = null;
    }
  }
  requestUpdate(e, s, a, i = !1, r) {
    var n;
    if (e !== void 0) {
      const c = this.constructor;
      if (i === !1 && (r = this[e]), a ?? (a = c.getPropertyOptions(e)), !((a.hasChanged ?? le)(r, s) || a.useDefault && a.reflect && r === ((n = this._$Ej) == null ? void 0 : n.get(e)) && !this.hasAttribute(c._$Eu(e, a)))) return;
      this.C(e, s, a);
    }
    this.isUpdatePending === !1 && (this._$ES = this._$EP());
  }
  C(e, s, { useDefault: a, reflect: i, wrapped: r }, n) {
    a && !(this._$Ej ?? (this._$Ej = /* @__PURE__ */ new Map())).has(e) && (this._$Ej.set(e, n ?? s ?? this[e]), r !== !0 || n !== void 0) || (this._$AL.has(e) || (this.hasUpdated || a || (s = void 0), this._$AL.set(e, s)), i === !0 && this._$Em !== e && (this._$Eq ?? (this._$Eq = /* @__PURE__ */ new Set())).add(e));
  }
  async _$EP() {
    this.isUpdatePending = !0;
    try {
      await this._$ES;
    } catch (s) {
      Promise.reject(s);
    }
    const e = this.scheduleUpdate();
    return e != null && await e, !this.isUpdatePending;
  }
  scheduleUpdate() {
    return this.performUpdate();
  }
  performUpdate() {
    var a;
    if (!this.isUpdatePending) return;
    if (!this.hasUpdated) {
      if (this.renderRoot ?? (this.renderRoot = this.createRenderRoot()), this._$Ep) {
        for (const [r, n] of this._$Ep) this[r] = n;
        this._$Ep = void 0;
      }
      const i = this.constructor.elementProperties;
      if (i.size > 0) for (const [r, n] of i) {
        const { wrapped: c } = n, d = this[r];
        c !== !0 || this._$AL.has(r) || d === void 0 || this.C(r, void 0, n, d);
      }
    }
    let e = !1;
    const s = this._$AL;
    try {
      e = this.shouldUpdate(s), e ? (this.willUpdate(s), (a = this._$EO) == null || a.forEach((i) => {
        var r;
        return (r = i.hostUpdate) == null ? void 0 : r.call(i);
      }), this.update(s)) : this._$EM();
    } catch (i) {
      throw e = !1, this._$EM(), i;
    }
    e && this._$AE(s);
  }
  willUpdate(e) {
  }
  _$AE(e) {
    var s;
    (s = this._$EO) == null || s.forEach((a) => {
      var i;
      return (i = a.hostUpdated) == null ? void 0 : i.call(a);
    }), this.hasUpdated || (this.hasUpdated = !0, this.firstUpdated(e)), this.updated(e);
  }
  _$EM() {
    this._$AL = /* @__PURE__ */ new Map(), this.isUpdatePending = !1;
  }
  get updateComplete() {
    return this.getUpdateComplete();
  }
  getUpdateComplete() {
    return this._$ES;
  }
  shouldUpdate(e) {
    return !0;
  }
  update(e) {
    this._$Eq && (this._$Eq = this._$Eq.forEach((s) => this._$ET(s, this[s]))), this._$EM();
  }
  updated(e) {
  }
  firstUpdated(e) {
  }
};
j.elementStyles = [], j.shadowRootOptions = { mode: "open" }, j[q("elementProperties")] = /* @__PURE__ */ new Map(), j[q("finalized")] = /* @__PURE__ */ new Map(), se == null || se({ ReactiveElement: j }), (C.reactiveElementVersions ?? (C.reactiveElementVersions = [])).push("2.1.2");
/**
 * @license
 * Copyright 2017 Google LLC
 * SPDX-License-Identifier: BSD-3-Clause
 */
const I = globalThis, ue = (t) => t, Y = I.trustedTypes, ge = Y ? Y.createPolicy("lit-html", { createHTML: (t) => t }) : void 0, ke = "$lit$", P = `lit$${Math.random().toFixed(9).slice(2)}$`, xe = "?" + P, Ne = `<${xe}>`, T = document, F = () => T.createComment(""), G = (t) => t === null || typeof t != "object" && typeof t != "function", de = Array.isArray, Me = (t) => de(t) || typeof (t == null ? void 0 : t[Symbol.iterator]) == "function", ie = `[ 	
\f\r]`, L = /<(?:(!--|\/[^a-zA-Z])|(\/?[a-zA-Z][^>\s]*)|(\/?$))/g, be = /-->/g, me = />/g, D = RegExp(`>|${ie}(?:([^\\s"'>=/]+)(${ie}*=${ie}*(?:[^ 	
\f\r"'\`<>=]|("|')|))|$)`, "g"), ve = /'/g, $e = /"/g, Ae = /^(?:script|style|textarea|title)$/i, He = (t) => (e, ...s) => ({ _$litType$: t, strings: e, values: s }), o = He(1), B = Symbol.for("lit-noChange"), l = Symbol.for("lit-nothing"), ye = /* @__PURE__ */ new WeakMap(), R = T.createTreeWalker(T, 129);
function Se(t, e) {
  if (!de(t) || !t.hasOwnProperty("raw")) throw Error("invalid template strings array");
  return ge !== void 0 ? ge.createHTML(e) : e;
}
const We = (t, e) => {
  const s = t.length - 1, a = [];
  let i, r = e === 2 ? "<svg>" : e === 3 ? "<math>" : "", n = L;
  for (let c = 0; c < s; c++) {
    const d = t[c];
    let h, u, f = -1, b = 0;
    for (; b < d.length && (n.lastIndex = b, u = n.exec(d), u !== null); ) b = n.lastIndex, n === L ? u[1] === "!--" ? n = be : u[1] !== void 0 ? n = me : u[2] !== void 0 ? (Ae.test(u[2]) && (i = RegExp("</" + u[2], "g")), n = D) : u[3] !== void 0 && (n = D) : n === D ? u[0] === ">" ? (n = i ?? L, f = -1) : u[1] === void 0 ? f = -2 : (f = n.lastIndex - u[2].length, h = u[1], n = u[3] === void 0 ? D : u[3] === '"' ? $e : ve) : n === $e || n === ve ? n = D : n === be || n === me ? n = L : (n = D, i = void 0);
    const v = n === D && t[c + 1].startsWith("/>") ? " " : "";
    r += n === L ? d + Ne : f >= 0 ? (a.push(h), d.slice(0, f) + ke + d.slice(f) + P + v) : d + P + (f === -2 ? c : v);
  }
  return [Se(t, r + (t[s] || "<?>") + (e === 2 ? "</svg>" : e === 3 ? "</math>" : "")), a];
};
class V {
  constructor({ strings: e, _$litType$: s }, a) {
    let i;
    this.parts = [];
    let r = 0, n = 0;
    const c = e.length - 1, d = this.parts, [h, u] = We(e, s);
    if (this.el = V.createElement(h, a), R.currentNode = this.el.content, s === 2 || s === 3) {
      const f = this.el.content.firstChild;
      f.replaceWith(...f.childNodes);
    }
    for (; (i = R.nextNode()) !== null && d.length < c; ) {
      if (i.nodeType === 1) {
        if (i.hasAttributes()) for (const f of i.getAttributeNames()) if (f.endsWith(ke)) {
          const b = u[n++], v = i.getAttribute(f).split(P), w = /([.?@])?(.*)/.exec(b);
          d.push({ type: 1, index: r, name: w[2], strings: v, ctor: w[1] === "." ? qe : w[1] === "?" ? Ie : w[1] === "@" ? Fe : ee }), i.removeAttribute(f);
        } else f.startsWith(P) && (d.push({ type: 6, index: r }), i.removeAttribute(f));
        if (Ae.test(i.tagName)) {
          const f = i.textContent.split(P), b = f.length - 1;
          if (b > 0) {
            i.textContent = Y ? Y.emptyScript : "";
            for (let v = 0; v < b; v++) i.append(f[v], F()), R.nextNode(), d.push({ type: 2, index: ++r });
            i.append(f[b], F());
          }
        }
      } else if (i.nodeType === 8) if (i.data === xe) d.push({ type: 2, index: r });
      else {
        let f = -1;
        for (; (f = i.data.indexOf(P, f + 1)) !== -1; ) d.push({ type: 7, index: r }), f += P.length - 1;
      }
      r++;
    }
  }
  static createElement(e, s) {
    const a = T.createElement("template");
    return a.innerHTML = e, a;
  }
}
function N(t, e, s = t, a) {
  var n, c;
  if (e === B) return e;
  let i = a !== void 0 ? (n = s._$Co) == null ? void 0 : n[a] : s._$Cl;
  const r = G(e) ? void 0 : e._$litDirective$;
  return (i == null ? void 0 : i.constructor) !== r && ((c = i == null ? void 0 : i._$AO) == null || c.call(i, !1), r === void 0 ? i = void 0 : (i = new r(t), i._$AT(t, s, a)), a !== void 0 ? (s._$Co ?? (s._$Co = []))[a] = i : s._$Cl = i), i !== void 0 && (e = N(t, i._$AS(t, e.values), i, a)), e;
}
class Le {
  constructor(e, s) {
    this._$AV = [], this._$AN = void 0, this._$AD = e, this._$AM = s;
  }
  get parentNode() {
    return this._$AM.parentNode;
  }
  get _$AU() {
    return this._$AM._$AU;
  }
  u(e) {
    const { el: { content: s }, parts: a } = this._$AD, i = ((e == null ? void 0 : e.creationScope) ?? T).importNode(s, !0);
    R.currentNode = i;
    let r = R.nextNode(), n = 0, c = 0, d = a[0];
    for (; d !== void 0; ) {
      if (n === d.index) {
        let h;
        d.type === 2 ? h = new J(r, r.nextSibling, this, e) : d.type === 1 ? h = new d.ctor(r, d.name, d.strings, this, e) : d.type === 6 && (h = new Ge(r, this, e)), this._$AV.push(h), d = a[++c];
      }
      n !== (d == null ? void 0 : d.index) && (r = R.nextNode(), n++);
    }
    return R.currentNode = T, i;
  }
  p(e) {
    let s = 0;
    for (const a of this._$AV) a !== void 0 && (a.strings !== void 0 ? (a._$AI(e, a, s), s += a.strings.length - 2) : a._$AI(e[s])), s++;
  }
}
class J {
  get _$AU() {
    var e;
    return ((e = this._$AM) == null ? void 0 : e._$AU) ?? this._$Cv;
  }
  constructor(e, s, a, i) {
    this.type = 2, this._$AH = l, this._$AN = void 0, this._$AA = e, this._$AB = s, this._$AM = a, this.options = i, this._$Cv = (i == null ? void 0 : i.isConnected) ?? !0;
  }
  get parentNode() {
    let e = this._$AA.parentNode;
    const s = this._$AM;
    return s !== void 0 && (e == null ? void 0 : e.nodeType) === 11 && (e = s.parentNode), e;
  }
  get startNode() {
    return this._$AA;
  }
  get endNode() {
    return this._$AB;
  }
  _$AI(e, s = this) {
    e = N(this, e, s), G(e) ? e === l || e == null || e === "" ? (this._$AH !== l && this._$AR(), this._$AH = l) : e !== this._$AH && e !== B && this._(e) : e._$litType$ !== void 0 ? this.$(e) : e.nodeType !== void 0 ? this.T(e) : Me(e) ? this.k(e) : this._(e);
  }
  O(e) {
    return this._$AA.parentNode.insertBefore(e, this._$AB);
  }
  T(e) {
    this._$AH !== e && (this._$AR(), this._$AH = this.O(e));
  }
  _(e) {
    this._$AH !== l && G(this._$AH) ? this._$AA.nextSibling.data = e : this.T(T.createTextNode(e)), this._$AH = e;
  }
  $(e) {
    var r;
    const { values: s, _$litType$: a } = e, i = typeof a == "number" ? this._$AC(e) : (a.el === void 0 && (a.el = V.createElement(Se(a.h, a.h[0]), this.options)), a);
    if (((r = this._$AH) == null ? void 0 : r._$AD) === i) this._$AH.p(s);
    else {
      const n = new Le(i, this), c = n.u(this.options);
      n.p(s), this.T(c), this._$AH = n;
    }
  }
  _$AC(e) {
    let s = ye.get(e.strings);
    return s === void 0 && ye.set(e.strings, s = new V(e)), s;
  }
  k(e) {
    de(this._$AH) || (this._$AH = [], this._$AR());
    const s = this._$AH;
    let a, i = 0;
    for (const r of e) i === s.length ? s.push(a = new J(this.O(F()), this.O(F()), this, this.options)) : a = s[i], a._$AI(r), i++;
    i < s.length && (this._$AR(a && a._$AB.nextSibling, i), s.length = i);
  }
  _$AR(e = this._$AA.nextSibling, s) {
    var a;
    for ((a = this._$AP) == null ? void 0 : a.call(this, !1, !0, s); e !== this._$AB; ) {
      const i = ue(e).nextSibling;
      ue(e).remove(), e = i;
    }
  }
  setConnected(e) {
    var s;
    this._$AM === void 0 && (this._$Cv = e, (s = this._$AP) == null || s.call(this, e));
  }
}
class ee {
  get tagName() {
    return this.element.tagName;
  }
  get _$AU() {
    return this._$AM._$AU;
  }
  constructor(e, s, a, i, r) {
    this.type = 1, this._$AH = l, this._$AN = void 0, this.element = e, this.name = s, this._$AM = i, this.options = r, a.length > 2 || a[0] !== "" || a[1] !== "" ? (this._$AH = Array(a.length - 1).fill(new String()), this.strings = a) : this._$AH = l;
  }
  _$AI(e, s = this, a, i) {
    const r = this.strings;
    let n = !1;
    if (r === void 0) e = N(this, e, s, 0), n = !G(e) || e !== this._$AH && e !== B, n && (this._$AH = e);
    else {
      const c = e;
      let d, h;
      for (e = r[0], d = 0; d < r.length - 1; d++) h = N(this, c[a + d], s, d), h === B && (h = this._$AH[d]), n || (n = !G(h) || h !== this._$AH[d]), h === l ? e = l : e !== l && (e += (h ?? "") + r[d + 1]), this._$AH[d] = h;
    }
    n && !i && this.j(e);
  }
  j(e) {
    e === l ? this.element.removeAttribute(this.name) : this.element.setAttribute(this.name, e ?? "");
  }
}
class qe extends ee {
  constructor() {
    super(...arguments), this.type = 3;
  }
  j(e) {
    this.element[this.name] = e === l ? void 0 : e;
  }
}
class Ie extends ee {
  constructor() {
    super(...arguments), this.type = 4;
  }
  j(e) {
    this.element.toggleAttribute(this.name, !!e && e !== l);
  }
}
class Fe extends ee {
  constructor(e, s, a, i, r) {
    super(e, s, a, i, r), this.type = 5;
  }
  _$AI(e, s = this) {
    if ((e = N(this, e, s, 0) ?? l) === B) return;
    const a = this._$AH, i = e === l && a !== l || e.capture !== a.capture || e.once !== a.once || e.passive !== a.passive, r = e !== l && (a === l || i);
    i && this.element.removeEventListener(this.name, this, a), r && this.element.addEventListener(this.name, this, e), this._$AH = e;
  }
  handleEvent(e) {
    var s;
    typeof this._$AH == "function" ? this._$AH.call(((s = this.options) == null ? void 0 : s.host) ?? this.element, e) : this._$AH.handleEvent(e);
  }
}
class Ge {
  constructor(e, s, a) {
    this.element = e, this.type = 6, this._$AN = void 0, this._$AM = s, this.options = a;
  }
  get _$AU() {
    return this._$AM._$AU;
  }
  _$AI(e) {
    N(this, e);
  }
}
const ae = I.litHtmlPolyfillSupport;
ae == null || ae(V, J), (I.litHtmlVersions ?? (I.litHtmlVersions = [])).push("3.3.2");
const Ve = (t, e, s) => {
  const a = (s == null ? void 0 : s.renderBefore) ?? e;
  let i = a._$litPart$;
  if (i === void 0) {
    const r = (s == null ? void 0 : s.renderBefore) ?? null;
    a._$litPart$ = i = new J(e.insertBefore(F(), r), r, void 0, s ?? {});
  }
  return i._$AI(t), i;
};
/**
 * @license
 * Copyright 2017 Google LLC
 * SPDX-License-Identifier: BSD-3-Clause
 */
const U = globalThis;
class A extends j {
  constructor() {
    super(...arguments), this.renderOptions = { host: this }, this._$Do = void 0;
  }
  createRenderRoot() {
    var s;
    const e = super.createRenderRoot();
    return (s = this.renderOptions).renderBefore ?? (s.renderBefore = e.firstChild), e;
  }
  update(e) {
    const s = this.render();
    this.hasUpdated || (this.renderOptions.isConnected = this.isConnected), super.update(e), this._$Do = Ve(s, this.renderRoot, this.renderOptions);
  }
  connectedCallback() {
    var e;
    super.connectedCallback(), (e = this._$Do) == null || e.setConnected(!0);
  }
  disconnectedCallback() {
    var e;
    super.disconnectedCallback(), (e = this._$Do) == null || e.setConnected(!1);
  }
  render() {
    return B;
  }
}
var we;
A._$litElement$ = !0, A.finalized = !0, (we = U.litElementHydrateSupport) == null || we.call(U, { LitElement: A });
const re = U.litElementPolyfillSupport;
re == null || re({ LitElement: A });
(U.litElementVersions ?? (U.litElementVersions = [])).push("4.2.2");
/**
 * @license
 * Copyright 2017 Google LLC
 * SPDX-License-Identifier: BSD-3-Clause
 */
const Z = (t) => (e, s) => {
  s !== void 0 ? s.addInitializer(() => {
    customElements.define(t, e);
  }) : customElements.define(t, e);
};
/**
 * @license
 * Copyright 2017 Google LLC
 * SPDX-License-Identifier: BSD-3-Clause
 */
const Ke = { attribute: !0, type: String, converter: Q, reflect: !1, hasChanged: le }, Je = (t = Ke, e, s) => {
  const { kind: a, metadata: i } = s;
  let r = globalThis.litPropertyMetadata.get(i);
  if (r === void 0 && globalThis.litPropertyMetadata.set(i, r = /* @__PURE__ */ new Map()), a === "setter" && ((t = Object.create(t)).wrapped = !0), r.set(s.name, t), a === "accessor") {
    const { name: n } = s;
    return { set(c) {
      const d = e.get.call(this);
      e.set.call(this, c), this.requestUpdate(n, d, t, !0, c);
    }, init(c) {
      return c !== void 0 && this.C(n, void 0, t, c), c;
    } };
  }
  if (a === "setter") {
    const { name: n } = s;
    return function(c) {
      const d = this[n];
      e.call(this, c), this.requestUpdate(n, d, t, !0, c);
    };
  }
  throw Error("Unsupported decorator location: " + a);
};
function z(t) {
  return (e, s) => typeof s == "object" ? Je(t, e, s) : ((a, i, r) => {
    const n = i.hasOwnProperty(r);
    return i.constructor.createProperty(r, a), n ? Object.getOwnPropertyDescriptor(i, r) : void 0;
  })(t, e, s);
}
/**
 * @license
 * Copyright 2017 Google LLC
 * SPDX-License-Identifier: BSD-3-Clause
 */
function p(t) {
  return z({ ...t, state: !0, attribute: !1 });
}
function Ze(t, e) {
  const s = new WebSocket(t);
  return s.onmessage = (a) => {
    var i, r, n, c, d, h, u, f, b, v, w, W;
    try {
      const x = JSON.parse(a.data);
      ((r = (i = x.type) == null ? void 0 : i.startsWith) != null && r.call(i, "build.") || (c = (n = x.type) == null ? void 0 : n.startsWith) != null && c.call(n, "release.") || (h = (d = x.type) == null ? void 0 : d.startsWith) != null && h.call(d, "sdk.") || (f = (u = x.channel) == null ? void 0 : u.startsWith) != null && f.call(u, "build.") || (v = (b = x.channel) == null ? void 0 : b.startsWith) != null && v.call(b, "release.") || (W = (w = x.channel) == null ? void 0 : w.startsWith) != null && W.call(w, "sdk.")) && e(x);
    } catch {
    }
  }, s;
}
class te {
  constructor(e = "") {
    this.baseUrl = e;
  }
  get base() {
    return `${this.baseUrl}/api/v1/build`;
  }
  async request(e, s) {
    var r;
    const i = await (await fetch(`${this.base}${e}`, s)).json();
    if (!i.success)
      throw new Error(((r = i.error) == null ? void 0 : r.message) ?? "Request failed");
    return i.data;
  }
  // -- Build ------------------------------------------------------------------
  config() {
    return this.request("/config");
  }
  discover() {
    return this.request("/discover");
  }
  build() {
    return this.request("/build", { method: "POST" });
  }
  artifacts() {
    return this.request("/artifacts");
  }
  // -- Release ----------------------------------------------------------------
  version() {
    return this.request("/release/version");
  }
  changelog(e, s) {
    const a = new URLSearchParams();
    e && a.set("from", e), s && a.set("to", s);
    const i = a.toString();
    return this.request(`/release/changelog${i ? `?${i}` : ""}`);
  }
  release(e = !1) {
    const s = e ? "?dry_run=true" : "";
    return this.request(`/release${s}`, { method: "POST" });
  }
  releaseWorkflow(e = {}) {
    return this.request("/release/workflow", {
      method: "POST",
      headers: { "Content-Type": "application/json" },
      body: JSON.stringify(e)
    });
  }
  // -- SDK --------------------------------------------------------------------
  sdkDiff(e, s) {
    const a = new URLSearchParams({ base: e, revision: s });
    return this.request(`/sdk/diff?${a.toString()}`);
  }
  sdkGenerate(e) {
    const s = e ? JSON.stringify({ language: e }) : void 0;
    return this.request("/sdk/generate", {
      method: "POST",
      headers: s ? { "Content-Type": "application/json" } : void 0,
      body: s
    });
  }
}
var Xe = Object.defineProperty, Qe = Object.getOwnPropertyDescriptor, M = (t, e, s, a) => {
  for (var i = a > 1 ? void 0 : a ? Qe(e, s) : e, r = t.length - 1, n; r >= 0; r--)
    (n = t[r]) && (i = (a ? n(e, s, i) : n(i)) || i);
  return a && i && Xe(e, s, i), i;
};
let E = class extends A {
  constructor() {
    super(...arguments), this.apiUrl = "", this.configData = null, this.discoverData = null, this.loading = !0, this.error = "";
  }
  connectedCallback() {
    super.connectedCallback(), this.api = new te(this.apiUrl), this.reload();
  }
  async reload() {
    this.loading = !0, this.error = "";
    try {
      const [t, e] = await Promise.all([
        this.api.config(),
        this.api.discover()
      ]);
      this.configData = t, this.discoverData = e;
    } catch (t) {
      this.error = t.message ?? "Failed to load configuration";
    } finally {
      this.loading = !1;
    }
  }
  hasAppleConfig(t) {
    return t ? Object.entries(t).some(([, e]) => e == null ? !1 : Array.isArray(e) ? e.length > 0 : typeof e == "object" ? Object.keys(e).length > 0 : typeof e == "string" ? e.length > 0 : !0) : !1;
  }
  renderToggle(t, e, s = "Enabled", a = "Disabled") {
    return e == null ? l : o`
      <div class="field">
        <span class="field-label">${t}</span>
        <span class="badge ${e ? "present" : "absent"}">
          ${e ? s : a}
        </span>
      </div>
    `;
  }
  renderFlags(t, e) {
    return !e || e.length === 0 ? l : o`
      <div class="field">
        <span class="field-label">${t}</span>
        <div class="flags">
          ${e.map((s) => o`<span class="flag">${s}</span>`)}
        </div>
      </div>
    `;
  }
  render() {
    var a, i, r, n, c, d, h, u, f, b, v, w, W, x;
    if (this.loading)
      return o`<div class="loading">Loading configuration\u2026</div>`;
    if (this.error)
      return o`<div class="error">${this.error}</div>`;
    if (!this.configData)
      return o`<div class="empty">No configuration available.</div>`;
    const t = this.configData.config, e = this.discoverData, s = e ? e.has_subtree_package_json ?? e.has_subtree_npm ?? !1 : !1;
    return o`
      <!-- Discovery -->
      <div class="section">
        <div class="section-title">Project Detection</div>
        <div class="field">
          <span class="field-label">Config file</span>
          <span class="badge ${this.configData.has_config ? "present" : "absent"}">
            ${this.configData.has_config ? "Present" : "Using defaults"}
          </span>
        </div>
        ${e ? o`
              <div class="field">
                <span class="field-label">Primary type</span>
                <span class="badge type-${e.primary || "unknown"}">${e.primary || "none"}</span>
              </div>
              <div class="field">
                <span class="field-label">Suggested stack</span>
                <span class="field-value">${e.suggested_stack || e.primary_stack || e.primary || "none"}</span>
              </div>
              ${e.types.length > 1 ? o`
                    <div class="field">
                      <span class="field-label">Detected types</span>
                      <span class="field-value">${e.types.join(", ")}</span>
                    </div>
                  ` : l}
              <div class="field">
                <span class="field-label">Frontend</span>
                <span class="badge ${e.has_frontend ? "present" : "absent"}">
                  ${e.has_frontend ? "Detected" : "None"}
                </span>
              </div>
              <div class="field">
                <span class="field-label">Nested frontend</span>
                <span class="badge ${s ? "present" : "absent"}">
                  ${s ? "Depth 2" : "None"}
                </span>
              </div>
              ${e.distro ? o`
                    <div class="field">
                      <span class="field-label">Distro</span>
                      <span class="field-value">${e.distro}</span>
                    </div>
                  ` : l}
              ${e.linux_packages && e.linux_packages.length > 0 ? o`
                    <div class="field">
                      <span class="field-label">Linux packages</span>
                      <div class="flags">
                        ${e.linux_packages.map((g) => o`<span class="flag">${g}</span>`)}
                      </div>
                    </div>
                  ` : l}
              ${e.build_options ? o`
                    <div class="field">
                      <span class="field-label">Computed options</span>
                      <span class="field-value">${e.build_options}</span>
                    </div>
                  ` : l}
              ${this.renderToggle("Computed obfuscation", (a = e.options) == null ? void 0 : a.obfuscate)}
              ${this.renderToggle("Computed NSIS", (i = e.options) == null ? void 0 : i.nsis)}
              ${(r = e.options) != null && r.webview2 ? o`
                    <div class="field">
                      <span class="field-label">Computed WebView2</span>
                      <span class="field-value">${e.options.webview2}</span>
                    </div>
                  ` : l}
              ${this.renderFlags("Computed tags", (n = e.options) == null ? void 0 : n.tags)}
              ${this.renderFlags("Computed LD flags", (c = e.options) == null ? void 0 : c.ldflags)}
              ${e.ref ? o`
                    <div class="field">
                      <span class="field-label">Git ref</span>
                      <span class="field-value">${e.ref}</span>
                    </div>
                  ` : l}
              ${e.branch ? o`
                    <div class="field">
                      <span class="field-label">Branch</span>
                      <span class="field-value">${e.branch}</span>
                    </div>
                  ` : l}
              ${e.tag ? o`
                    <div class="field">
                      <span class="field-label">Tag</span>
                      <span class="field-value">${e.tag}</span>
                    </div>
                  ` : l}
              ${e.short_sha ? o`
                    <div class="field">
                      <span class="field-label">Short SHA</span>
                      <span class="field-value">${e.short_sha}</span>
                    </div>
                  ` : l}
              <div class="field">
                <span class="field-label">Directory</span>
                <span class="field-value">${e.dir}</span>
              </div>
            ` : l}
      </div>

      ${e != null && e.setup_plan ? o`
            <div class="section">
              <div class="section-title">Setup Plan</div>
              ${this.renderFlags(
      "Toolchains",
      (d = e.setup_plan.steps) == null ? void 0 : d.map((g) => g.tool)
    )}
              ${this.renderFlags("Frontend dirs", e.setup_plan.frontend_dirs)}
              ${e.setup_plan.linux_packages && e.setup_plan.linux_packages.length > 0 ? o`
                    <div class="field">
                      <span class="field-label">System packages</span>
                      <div class="flags">
                        ${e.setup_plan.linux_packages.map((g) => o`<span class="flag">${g}</span>`)}
                      </div>
                    </div>
                  ` : l}
              ${e.setup_plan.steps && e.setup_plan.steps.length > 0 ? e.setup_plan.steps.map(
      (g) => o`
                      <div class="field">
                        <span class="field-label">${g.tool}</span>
                        <span class="field-value">${g.reason}</span>
                      </div>
                    `
    ) : o`
                    <div class="field">
                      <span class="field-label">Steps</span>
                      <span class="field-value">No setup required</span>
                    </div>
                  `}
            </div>
          ` : l}

      <!-- Project -->
      <div class="section">
        <div class="section-title">Project</div>
        ${t.project.name ? o`
              <div class="field">
                <span class="field-label">Name</span>
                <span class="field-value">${t.project.name}</span>
              </div>
            ` : l}
        ${t.project.description ? o`
              <div class="field">
                <span class="field-label">Description</span>
                <span class="field-value">${t.project.description}</span>
              </div>
            ` : l}
        ${t.project.binary ? o`
              <div class="field">
                <span class="field-label">Binary</span>
                <span class="field-value">${t.project.binary}</span>
              </div>
            ` : l}
        <div class="field">
          <span class="field-label">Main</span>
          <span class="field-value">${t.project.main}</span>
        </div>
      </div>

      <!-- Build Settings -->
      <div class="section">
        <div class="section-title">Build Settings</div>
        ${t.build.type ? o`
              <div class="field">
                <span class="field-label">Type override</span>
                <span class="field-value">${t.build.type}</span>
              </div>
            ` : l}
        <div class="field">
          <span class="field-label">CGO</span>
          <span class="field-value">${t.build.cgo ? "Enabled" : "Disabled"}</span>
        </div>
        ${this.renderToggle("Obfuscation", t.build.obfuscate)}
        ${this.renderToggle("NSIS packaging", t.build.nsis)}
        ${t.build.webview2 ? o`
              <div class="field">
                <span class="field-label">WebView2 mode</span>
                <span class="field-value">${t.build.webview2}</span>
              </div>
            ` : l}
        ${t.build.deno_build ? o`
              <div class="field">
                <span class="field-label">Deno build</span>
                <span class="field-value">${t.build.deno_build}</span>
              </div>
            ` : l}
        ${t.build.archive_format ? o`
              <div class="field">
                <span class="field-label">Archive format</span>
                <span class="field-value">${t.build.archive_format}</span>
              </div>
            ` : l}
        ${this.renderFlags("Build tags", t.build.build_tags)}
        ${t.build.flags && t.build.flags.length > 0 ? o`
              <div class="field">
                <span class="field-label">Flags</span>
                <div class="flags">
                  ${t.build.flags.map((g) => o`<span class="flag">${g}</span>`)}
                </div>
              </div>
            ` : l}
        ${t.build.ldflags && t.build.ldflags.length > 0 ? o`
              <div class="field">
                <span class="field-label">LD flags</span>
                <div class="flags">
                  ${t.build.ldflags.map((g) => o`<span class="flag">${g}</span>`)}
                </div>
              </div>
            ` : l}
        ${this.renderFlags("Environment", t.build.env)}
        ${(h = t.build.cache) != null && h.enabled || (u = t.build.cache) != null && u.path || (f = t.build.cache) != null && f.paths && t.build.cache.paths.length > 0 ? o`
              ${this.renderToggle("Build cache", (b = t.build.cache) == null ? void 0 : b.enabled)}
              ${(v = t.build.cache) != null && v.path ? o`
                    <div class="field">
                      <span class="field-label">Cache path</span>
                      <span class="field-value">${t.build.cache.path}</span>
                    </div>
                  ` : l}
              ${this.renderFlags("Cache paths", (w = t.build.cache) == null ? void 0 : w.paths)}
            ` : l}
        ${t.build.dockerfile ? o`
              <div class="field">
                <span class="field-label">Dockerfile</span>
                <span class="field-value">${t.build.dockerfile}</span>
              </div>
            ` : l}
        ${t.build.image ? o`
              <div class="field">
                <span class="field-label">Image</span>
                <span class="field-value">${t.build.image}</span>
              </div>
            ` : l}
        ${t.build.registry ? o`
              <div class="field">
                <span class="field-label">Registry</span>
                <span class="field-value">${t.build.registry}</span>
              </div>
            ` : l}
        ${this.renderFlags("Image tags", t.build.tags)}
        ${this.renderToggle("Push image", t.build.push)}
        ${this.renderToggle("Load image", t.build.load)}
        ${t.build.linuxkit_config ? o`
              <div class="field">
                <span class="field-label">LinuxKit config</span>
                <span class="field-value">${t.build.linuxkit_config}</span>
              </div>
            ` : l}
        ${this.renderFlags("LinuxKit formats", t.build.formats)}
      </div>

      <!-- Targets -->
      <div class="section">
        <div class="section-title">Targets</div>
        <div class="targets">
          ${t.targets.map(
      (g) => o`<span class="target-badge">${g.os}/${g.arch}</span>`
    )}
        </div>
      </div>

      ${t.apple && this.hasAppleConfig(t.apple) ? o`
            <div class="section">
              <div class="section-title">Apple Pipeline</div>
              ${t.apple.bundle_id ? o`
                    <div class="field">
                      <span class="field-label">Bundle ID</span>
                      <span class="field-value">${t.apple.bundle_id}</span>
                    </div>
                  ` : l}
              ${t.apple.team_id ? o`
                    <div class="field">
                      <span class="field-label">Team ID</span>
                      <span class="field-value">${t.apple.team_id}</span>
                    </div>
                  ` : l}
              ${t.apple.arch ? o`
                    <div class="field">
                      <span class="field-label">Architecture</span>
                      <span class="field-value">${t.apple.arch}</span>
                    </div>
                  ` : l}
              ${t.apple.bundle_display_name ? o`
                    <div class="field">
                      <span class="field-label">Display name</span>
                      <span class="field-value">${t.apple.bundle_display_name}</span>
                    </div>
                  ` : l}
              ${t.apple.min_system_version ? o`
                    <div class="field">
                      <span class="field-label">Minimum macOS</span>
                      <span class="field-value">${t.apple.min_system_version}</span>
                    </div>
                  ` : l}
              ${t.apple.category ? o`
                    <div class="field">
                      <span class="field-label">Category</span>
                      <span class="field-value">${t.apple.category}</span>
                    </div>
                  ` : l}
              ${this.renderToggle("Sign", t.apple.sign)}
              ${this.renderToggle("Notarise", t.apple.notarise)}
              ${this.renderToggle("DMG", t.apple.dmg)}
              ${this.renderToggle("TestFlight", t.apple.testflight)}
              ${this.renderToggle("App Store", t.apple.appstore)}
              ${t.apple.metadata_path ? o`
                    <div class="field">
                      <span class="field-label">Metadata path</span>
                      <span class="field-value">${t.apple.metadata_path}</span>
                    </div>
                  ` : l}
              ${t.apple.privacy_policy_url ? o`
                    <div class="field">
                      <span class="field-label">Privacy policy</span>
                      <span class="field-value">${t.apple.privacy_policy_url}</span>
                    </div>
                  ` : l}
              ${t.apple.dmg_volume_name ? o`
                    <div class="field">
                      <span class="field-label">DMG volume</span>
                      <span class="field-value">${t.apple.dmg_volume_name}</span>
                    </div>
                  ` : l}
              ${t.apple.dmg_background ? o`
                    <div class="field">
                      <span class="field-label">DMG background</span>
                      <span class="field-value">${t.apple.dmg_background}</span>
                    </div>
                  ` : l}
              ${t.apple.entitlements_path ? o`
                    <div class="field">
                      <span class="field-label">Entitlements</span>
                      <span class="field-value">${t.apple.entitlements_path}</span>
                    </div>
                  ` : l}
              ${(W = t.apple.xcode_cloud) != null && W.workflow ? o`
                    <div class="field">
                      <span class="field-label">Xcode Cloud workflow</span>
                      <span class="field-value">${t.apple.xcode_cloud.workflow}</span>
                    </div>
                  ` : l}
              ${(x = t.apple.xcode_cloud) != null && x.triggers && t.apple.xcode_cloud.triggers.length > 0 ? o`
                    <div class="field">
                      <span class="field-label">Xcode Cloud triggers</span>
                      <div class="flags">
                        ${t.apple.xcode_cloud.triggers.map((g) => {
      const Pe = g.branch ? `branch:${g.branch}` : g.tag ? `tag:${g.tag}` : "manual", Ce = g.action ?? "archive";
      return o`<span class="flag">${Pe} → ${Ce}</span>`;
    })}
                      </div>
                    </div>
                  ` : l}
            </div>
          ` : l}
    `;
  }
};
E.styles = K`
    :host {
      display: block;
      font-family: system-ui, -apple-system, sans-serif;
    }

    .section {
      border: 1px solid #e5e7eb;
      border-radius: 0.5rem;
      padding: 1rem;
      background: #fff;
      margin-bottom: 1rem;
    }

    .section-title {
      font-size: 0.75rem;
      font-weight: 700;
      color: #6b7280;
      text-transform: uppercase;
      letter-spacing: 0.025em;
      margin-bottom: 0.75rem;
    }

    .field {
      display: flex;
      justify-content: space-between;
      align-items: baseline;
      padding: 0.375rem 0;
      border-bottom: 1px solid #f3f4f6;
    }

    .field:last-child {
      border-bottom: none;
    }

    .field-label {
      font-size: 0.8125rem;
      font-weight: 500;
      color: #374151;
    }

    .field-value {
      font-size: 0.8125rem;
      font-family: monospace;
      color: #6b7280;
      max-width: 36rem;
      text-align: right;
      word-break: break-word;
    }

    .badge {
      display: inline-block;
      font-size: 0.6875rem;
      font-weight: 600;
      padding: 0.125rem 0.5rem;
      border-radius: 1rem;
    }

    .badge.present {
      background: #dcfce7;
      color: #166534;
    }

    .badge.absent {
      background: #fef3c7;
      color: #92400e;
    }

    .badge.type-go {
      background: #dbeafe;
      color: #1e40af;
    }

    .badge.type-wails {
      background: #f3e8ff;
      color: #6b21a8;
    }

    .badge.type-node {
      background: #dcfce7;
      color: #166534;
    }

    .badge.type-php {
      background: #fef3c7;
      color: #92400e;
    }

    .badge.type-docker {
      background: #e0e7ff;
      color: #3730a3;
    }

    .targets {
      display: flex;
      flex-wrap: wrap;
      gap: 0.375rem;
      margin-top: 0.25rem;
    }

    .target-badge {
      font-size: 0.75rem;
      padding: 0.125rem 0.5rem;
      background: #f3f4f6;
      border-radius: 0.25rem;
      font-family: monospace;
      color: #374151;
    }

    .flags {
      display: flex;
      flex-wrap: wrap;
      gap: 0.25rem;
    }

    .flag {
      font-size: 0.75rem;
      padding: 0.0625rem 0.375rem;
      background: #f9fafb;
      border: 1px solid #e5e7eb;
      border-radius: 0.25rem;
      font-family: monospace;
      color: #6b7280;
    }

    .empty {
      text-align: center;
      padding: 2rem;
      color: #9ca3af;
      font-size: 0.875rem;
    }

    .loading {
      text-align: center;
      padding: 2rem;
      color: #6b7280;
    }

    .error {
      color: #dc2626;
      padding: 0.75rem;
      background: #fef2f2;
      border-radius: 0.375rem;
      font-size: 0.875rem;
    }
  `;
M([
  z({ attribute: "api-url" })
], E.prototype, "apiUrl", 2);
M([
  p()
], E.prototype, "configData", 2);
M([
  p()
], E.prototype, "discoverData", 2);
M([
  p()
], E.prototype, "loading", 2);
M([
  p()
], E.prototype, "error", 2);
E = M([
  Z("core-build-config")
], E);
var Ye = Object.defineProperty, et = Object.getOwnPropertyDescriptor, S = (t, e, s, a) => {
  for (var i = a > 1 ? void 0 : a ? et(e, s) : e, r = t.length - 1, n; r >= 0; r--)
    (n = t[r]) && (i = (a ? n(e, s, i) : n(i)) || i);
  return a && i && Ye(e, s, i), i;
};
let _ = class extends A {
  constructor() {
    super(...arguments), this.apiUrl = "", this.artifacts = [], this.distExists = !1, this.loading = !0, this.error = "", this.building = !1, this.confirmBuild = !1, this.buildSuccess = "";
  }
  connectedCallback() {
    super.connectedCallback(), this.api = new te(this.apiUrl), this.reload();
  }
  async reload() {
    this.loading = !0, this.error = "";
    try {
      const t = await this.api.artifacts();
      this.artifacts = t.artifacts ?? [], this.distExists = t.exists ?? !1;
    } catch (t) {
      this.error = t.message ?? "Failed to load artifacts";
    } finally {
      this.loading = !1;
    }
  }
  handleBuildClick() {
    this.confirmBuild = !0, this.buildSuccess = "";
  }
  handleCancelBuild() {
    this.confirmBuild = !1;
  }
  async handleConfirmBuild() {
    var t;
    this.confirmBuild = !1, this.building = !0, this.error = "", this.buildSuccess = "";
    try {
      const e = await this.api.build();
      this.buildSuccess = `Build complete — ${((t = e.artifacts) == null ? void 0 : t.length) ?? 0} artifact(s) produced (${e.version})`, await this.reload();
    } catch (e) {
      this.error = e.message ?? "Build failed";
    } finally {
      this.building = !1;
    }
  }
  formatSize(t) {
    return t < 1024 ? `${t} B` : t < 1024 * 1024 ? `${(t / 1024).toFixed(1)} KB` : `${(t / (1024 * 1024)).toFixed(1)} MB`;
  }
  render() {
    return this.loading ? o`<div class="loading">Loading artifacts\u2026</div>` : o`
      <div class="toolbar">
        <span class="toolbar-info">
          ${this.distExists ? `${this.artifacts.length} file(s) in dist/` : "No dist/ directory"}
        </span>
        <button
          class="build"
          ?disabled=${this.building}
          @click=${this.handleBuildClick}
        >
          ${this.building ? "Building…" : "Build"}
        </button>
      </div>

      ${this.confirmBuild ? o`
            <div class="confirm">
              <span class="confirm-text">This will run a full build and overwrite dist/. Continue?</span>
              <button class="confirm-yes" @click=${this.handleConfirmBuild}>Build</button>
              <button class="confirm-no" @click=${this.handleCancelBuild}>Cancel</button>
            </div>
          ` : l}

      ${this.error ? o`<div class="error">${this.error}</div>` : l}
      ${this.buildSuccess ? o`<div class="success">${this.buildSuccess}</div>` : l}

      ${this.artifacts.length === 0 ? o`<div class="empty">${this.distExists ? "dist/ is empty." : "Run a build to create artifacts."}</div>` : o`
            <div class="list">
              ${this.artifacts.map(
      (t) => o`
                  <div class="artifact">
                    <span class="artifact-name">${t.name}</span>
                    <span class="artifact-size">${this.formatSize(t.size)}</span>
                  </div>
                `
    )}
            </div>
          `}
    `;
  }
};
_.styles = K`
    :host {
      display: block;
      font-family: system-ui, -apple-system, sans-serif;
    }

    .toolbar {
      display: flex;
      justify-content: space-between;
      align-items: center;
      margin-bottom: 1rem;
    }

    .toolbar-info {
      font-size: 0.8125rem;
      color: #6b7280;
    }

    button.build {
      padding: 0.5rem 1.25rem;
      background: #6366f1;
      color: #fff;
      border: none;
      border-radius: 0.375rem;
      font-size: 0.875rem;
      font-weight: 500;
      cursor: pointer;
      transition: background 0.15s;
    }

    button.build:hover {
      background: #4f46e5;
    }

    button.build:disabled {
      opacity: 0.5;
      cursor: not-allowed;
    }

    .confirm {
      display: flex;
      align-items: center;
      gap: 0.75rem;
      padding: 0.75rem 1rem;
      background: #fffbeb;
      border: 1px solid #fde68a;
      border-radius: 0.375rem;
      margin-bottom: 1rem;
      font-size: 0.8125rem;
    }

    .confirm-text {
      flex: 1;
      color: #92400e;
    }

    button.confirm-yes {
      padding: 0.375rem 1rem;
      background: #dc2626;
      color: #fff;
      border: none;
      border-radius: 0.375rem;
      font-size: 0.8125rem;
      cursor: pointer;
    }

    button.confirm-yes:hover {
      background: #b91c1c;
    }

    button.confirm-no {
      padding: 0.375rem 0.75rem;
      background: #fff;
      border: 1px solid #d1d5db;
      border-radius: 0.375rem;
      font-size: 0.8125rem;
      cursor: pointer;
    }

    .list {
      display: flex;
      flex-direction: column;
      gap: 0.375rem;
    }

    .artifact {
      border: 1px solid #e5e7eb;
      border-radius: 0.375rem;
      padding: 0.625rem 1rem;
      background: #fff;
      display: flex;
      justify-content: space-between;
      align-items: center;
    }

    .artifact-name {
      font-size: 0.875rem;
      font-family: monospace;
      font-weight: 500;
      color: #111827;
    }

    .artifact-size {
      font-size: 0.75rem;
      color: #6b7280;
    }

    .empty {
      text-align: center;
      padding: 2rem;
      color: #9ca3af;
      font-size: 0.875rem;
    }

    .loading {
      text-align: center;
      padding: 2rem;
      color: #6b7280;
    }

    .error {
      color: #dc2626;
      padding: 0.75rem;
      background: #fef2f2;
      border-radius: 0.375rem;
      font-size: 0.875rem;
      margin-bottom: 1rem;
    }

    .success {
      padding: 0.75rem;
      background: #f0fdf4;
      border: 1px solid #bbf7d0;
      border-radius: 0.375rem;
      font-size: 0.875rem;
      color: #166534;
      margin-bottom: 1rem;
    }
  `;
S([
  z({ attribute: "api-url" })
], _.prototype, "apiUrl", 2);
S([
  p()
], _.prototype, "artifacts", 2);
S([
  p()
], _.prototype, "distExists", 2);
S([
  p()
], _.prototype, "loading", 2);
S([
  p()
], _.prototype, "error", 2);
S([
  p()
], _.prototype, "building", 2);
S([
  p()
], _.prototype, "confirmBuild", 2);
S([
  p()
], _.prototype, "buildSuccess", 2);
_ = S([
  Z("core-build-artifacts")
], _);
var tt = Object.defineProperty, st = Object.getOwnPropertyDescriptor, y = (t, e, s, a) => {
  for (var i = a > 1 ? void 0 : a ? st(e, s) : e, r = t.length - 1, n; r >= 0; r--)
    (n = t[r]) && (i = (a ? n(e, s, i) : n(i)) || i);
  return a && i && tt(e, s, i), i;
};
let m = class extends A {
  constructor() {
    super(...arguments), this.apiUrl = "", this.version = "", this.changelog = "", this.loading = !0, this.error = "", this.releasing = !1, this.confirmRelease = !1, this.releaseSuccess = "", this.workflowPath = ".github/workflows/release.yml", this.workflowOutputPath = "", this.generatingWorkflow = !1, this.workflowSuccess = "";
  }
  connectedCallback() {
    super.connectedCallback(), this.api = new te(this.apiUrl), this.reload();
  }
  async reload() {
    this.loading = !0, this.error = "";
    try {
      const [t, e] = await Promise.all([
        this.api.version(),
        this.api.changelog()
      ]);
      this.version = t.version ?? "", this.changelog = e.changelog ?? "";
    } catch (t) {
      this.error = t.message ?? "Failed to load release information";
    } finally {
      this.loading = !1;
    }
  }
  handleReleaseClick() {
    this.confirmRelease = !0, this.releaseSuccess = "";
  }
  handleWorkflowPathInput(t) {
    const e = t.target;
    this.workflowPath = (e == null ? void 0 : e.value) ?? "";
  }
  handleWorkflowOutputPathInput(t) {
    const e = t.target;
    this.workflowOutputPath = (e == null ? void 0 : e.value) ?? "";
  }
  async handleGenerateWorkflow() {
    this.generatingWorkflow = !0, this.error = "", this.workflowSuccess = "";
    try {
      const t = {}, e = this.workflowPath.trim(), s = this.workflowOutputPath.trim();
      e && (t.path = e), e && (t.workflowPath = e, t.workflow_path = e, t["workflow-path"] = e), s && (t.outputPath = s), s && (t["output-path"] = s, t.output_path = s, t.output = s, t.workflowOutputPath = s, t.workflow_output = s, t["workflow-output"] = s, t.workflow_output_path = s, t["workflow-output-path"] = s);
      const i = (await this.api.releaseWorkflow(t)).path ?? s ?? e ?? ".github/workflows/release.yml";
      this.workflowSuccess = `Workflow generated at ${i}`;
    } catch (t) {
      this.error = t.message ?? "Failed to generate release workflow";
    } finally {
      this.generatingWorkflow = !1;
    }
  }
  handleCancelRelease() {
    this.confirmRelease = !1;
  }
  async handleConfirmRelease() {
    this.confirmRelease = !1, await this.doRelease(!1);
  }
  async handleDryRun() {
    await this.doRelease(!0);
  }
  async doRelease(t) {
    var e;
    this.releasing = !0, this.error = "", this.releaseSuccess = "";
    try {
      const s = await this.api.release(t), a = t ? "Dry run complete" : "Release published";
      this.releaseSuccess = `${a} — ${s.version} (${((e = s.artifacts) == null ? void 0 : e.length) ?? 0} artifact(s))`, await this.reload();
    } catch (s) {
      this.error = s.message ?? "Release failed";
    } finally {
      this.releasing = !1;
    }
  }
  render() {
    return this.loading ? o`<div class="loading">Loading release information\u2026</div>` : o`
      ${this.error ? o`<div class="error">${this.error}</div>` : l}
      ${this.releaseSuccess ? o`<div class="success">${this.releaseSuccess}</div>` : l}
      ${this.workflowSuccess ? o`<div class="success">${this.workflowSuccess}</div>` : l}

      <div class="version-bar">
        <div>
          <div class="version-label">Current Version</div>
          <div class="version-value">${this.version || "unknown"}</div>
        </div>
        <div class="actions">
          <button
            class="dry-run"
            ?disabled=${this.releasing}
            @click=${this.handleDryRun}
          >
            Dry Run
          </button>
          <button
            class="release"
            ?disabled=${this.releasing}
            @click=${this.handleReleaseClick}
          >
            ${this.releasing ? "Publishing…" : "Publish Release"}
          </button>
        </div>
      </div>

      <div class="workflow-section">
        <div class="workflow-label">Release Workflow</div>
        <div class="workflow-fields">
          <div class="workflow-field">
            <div class="workflow-field-label">Workflow Path</div>
            <input
              class="workflow-input"
              type="text"
              .value=${this.workflowPath}
              @input=${this.handleWorkflowPathInput}
              placeholder=".github/workflows/release.yml"
              aria-label="Workflow path"
            />
          </div>
          <div class="workflow-field">
            <div class="workflow-field-label">Workflow Output Path</div>
            <input
              class="workflow-input"
              type="text"
              .value=${this.workflowOutputPath}
              @input=${this.handleWorkflowOutputPathInput}
              placeholder="ci/release.yml"
              aria-label="Workflow output path"
            />
          </div>
        </div>
        <div class="workflow-row">
          <button
            class="workflow"
            ?disabled=${this.generatingWorkflow}
            @click=${this.handleGenerateWorkflow}
          >
            ${this.generatingWorkflow ? "Generating…" : "Generate Workflow"}
          </button>
        </div>
      </div>

      ${this.confirmRelease ? o`
            <div class="confirm">
              <span class="confirm-text">This will publish ${this.version} to all configured targets. This action cannot be undone. Continue?</span>
              <button class="confirm-yes" @click=${this.handleConfirmRelease}>Publish</button>
              <button class="confirm-no" @click=${this.handleCancelRelease}>Cancel</button>
            </div>
          ` : l}

      ${this.changelog ? o`
            <div class="changelog-section">
              <div class="changelog-header">Changelog</div>
              <div class="changelog-content">${this.changelog}</div>
            </div>
          ` : o`<div class="empty">No changelog available.</div>`}
    `;
  }
};
m.styles = K`
    :host {
      display: block;
      font-family: system-ui, -apple-system, sans-serif;
    }

    .version-bar {
      display: flex;
      justify-content: space-between;
      align-items: center;
      padding: 1rem;
      background: #fff;
      border: 1px solid #e5e7eb;
      border-radius: 0.5rem;
      margin-bottom: 1rem;
    }

    .version-label {
      font-size: 0.75rem;
      font-weight: 600;
      color: #6b7280;
      text-transform: uppercase;
      letter-spacing: 0.025em;
    }

    .version-value {
      font-size: 1.25rem;
      font-weight: 700;
      font-family: monospace;
      color: #111827;
    }

    .actions {
      display: flex;
      gap: 0.5rem;
      flex-wrap: wrap;
    }

    button {
      padding: 0.5rem 1rem;
      border-radius: 0.375rem;
      font-size: 0.8125rem;
      cursor: pointer;
      transition: background 0.15s;
    }

    button.release {
      background: #6366f1;
      color: #fff;
      border: none;
      font-weight: 500;
    }

    button.release:hover {
      background: #4f46e5;
    }

    button.release:disabled {
      opacity: 0.5;
      cursor: not-allowed;
    }

    button.dry-run {
      background: #fff;
      color: #6366f1;
      border: 1px solid #6366f1;
    }

    button.dry-run:hover {
      background: #eef2ff;
    }

    .workflow-section {
      display: flex;
      flex-direction: column;
      gap: 0.75rem;
      padding: 0.875rem 1rem;
      background: linear-gradient(180deg, #fff, #f8fafc);
      border: 1px solid #e5e7eb;
      border-radius: 0.5rem;
      margin-bottom: 1rem;
    }

    .workflow-fields {
      display: flex;
      flex-direction: column;
      gap: 0.5rem;
    }

    .workflow-field {
      display: flex;
      gap: 0.5rem;
      align-items: center;
      flex-wrap: wrap;
    }

    .workflow-field-label {
      min-width: 9rem;
      font-size: 0.8125rem;
      font-weight: 600;
      color: #374151;
    }

    .workflow-row {
      display: flex;
      gap: 0.5rem;
      align-items: center;
      flex-wrap: wrap;
    }

    .workflow-label {
      font-size: 0.75rem;
      font-weight: 700;
      color: #6b7280;
      text-transform: uppercase;
      letter-spacing: 0.025em;
    }

    .workflow-input {
      flex: 1;
      min-width: 16rem;
      padding: 0.5rem 0.75rem;
      border: 1px solid #d1d5db;
      border-radius: 0.375rem;
      font-size: 0.875rem;
      font-family: monospace;
      color: #111827;
      background: #fff;
    }

    .workflow-input:focus {
      outline: none;
      border-color: #6366f1;
      box-shadow: 0 0 0 3px rgb(99 102 241 / 12%);
    }

    button.workflow {
      background: #111827;
      color: #fff;
      border: none;
      font-weight: 500;
    }

    button.workflow:hover {
      background: #1f2937;
    }

    button.workflow:disabled {
      opacity: 0.5;
      cursor: not-allowed;
    }

    .confirm {
      display: flex;
      align-items: center;
      gap: 0.75rem;
      padding: 0.75rem 1rem;
      background: #fef2f2;
      border: 1px solid #fecaca;
      border-radius: 0.375rem;
      margin-bottom: 1rem;
      font-size: 0.8125rem;
    }

    .confirm-text {
      flex: 1;
      color: #991b1b;
    }

    button.confirm-yes {
      padding: 0.375rem 1rem;
      background: #dc2626;
      color: #fff;
      border: none;
      border-radius: 0.375rem;
      font-size: 0.8125rem;
      cursor: pointer;
    }

    button.confirm-no {
      padding: 0.375rem 0.75rem;
      background: #fff;
      border: 1px solid #d1d5db;
      border-radius: 0.375rem;
      font-size: 0.8125rem;
      cursor: pointer;
    }

    .changelog-section {
      border: 1px solid #e5e7eb;
      border-radius: 0.5rem;
      background: #fff;
    }

    .changelog-header {
      padding: 0.75rem 1rem;
      border-bottom: 1px solid #e5e7eb;
      font-size: 0.75rem;
      font-weight: 700;
      color: #6b7280;
      text-transform: uppercase;
      letter-spacing: 0.025em;
    }

    .changelog-content {
      padding: 1rem;
      font-size: 0.875rem;
      line-height: 1.6;
      white-space: pre-wrap;
      font-family: system-ui, -apple-system, sans-serif;
      color: #374151;
      max-height: 400px;
      overflow-y: auto;
    }

    .empty {
      text-align: center;
      padding: 2rem;
      color: #9ca3af;
      font-size: 0.875rem;
    }

    .loading {
      text-align: center;
      padding: 2rem;
      color: #6b7280;
    }

    .error {
      color: #dc2626;
      padding: 0.75rem;
      background: #fef2f2;
      border-radius: 0.375rem;
      font-size: 0.875rem;
      margin-bottom: 1rem;
    }

    .success {
      padding: 0.75rem;
      background: #f0fdf4;
      border: 1px solid #bbf7d0;
      border-radius: 0.375rem;
      font-size: 0.875rem;
      color: #166534;
      margin-bottom: 1rem;
    }
  `;
y([
  z({ attribute: "api-url" })
], m.prototype, "apiUrl", 2);
y([
  p()
], m.prototype, "version", 2);
y([
  p()
], m.prototype, "changelog", 2);
y([
  p()
], m.prototype, "loading", 2);
y([
  p()
], m.prototype, "error", 2);
y([
  p()
], m.prototype, "releasing", 2);
y([
  p()
], m.prototype, "confirmRelease", 2);
y([
  p()
], m.prototype, "releaseSuccess", 2);
y([
  p()
], m.prototype, "workflowPath", 2);
y([
  p()
], m.prototype, "workflowOutputPath", 2);
y([
  p()
], m.prototype, "generatingWorkflow", 2);
y([
  p()
], m.prototype, "workflowSuccess", 2);
m = y([
  Z("core-build-release")
], m);
var it = Object.defineProperty, at = Object.getOwnPropertyDescriptor, k = (t, e, s, a) => {
  for (var i = a > 1 ? void 0 : a ? at(e, s) : e, r = t.length - 1, n; r >= 0; r--)
    (n = t[r]) && (i = (a ? n(e, s, i) : n(i)) || i);
  return a && i && it(e, s, i), i;
};
let $ = class extends A {
  constructor() {
    super(...arguments), this.apiUrl = "", this.basePath = "", this.revisionPath = "", this.diffResult = null, this.diffing = !1, this.diffError = "", this.selectedLanguage = "", this.generating = !1, this.generateError = "", this.generateSuccess = "";
  }
  connectedCallback() {
    super.connectedCallback(), this.api = new te(this.apiUrl);
  }
  async reload() {
    this.diffResult = null, this.diffError = "", this.generateError = "", this.generateSuccess = "";
  }
  async handleDiff() {
    if (!this.basePath.trim() || !this.revisionPath.trim()) {
      this.diffError = "Both base and revision spec paths are required.";
      return;
    }
    this.diffing = !0, this.diffError = "", this.diffResult = null;
    try {
      this.diffResult = await this.api.sdkDiff(this.basePath.trim(), this.revisionPath.trim());
    } catch (t) {
      this.diffError = t.message ?? "Diff failed";
    } finally {
      this.diffing = !1;
    }
  }
  async handleGenerate() {
    this.generating = !0, this.generateError = "", this.generateSuccess = "";
    try {
      const e = (await this.api.sdkGenerate(this.selectedLanguage || void 0)).language || "all languages";
      this.generateSuccess = `SDK generated successfully for ${e}.`;
    } catch (t) {
      this.generateError = t.message ?? "Generation failed";
    } finally {
      this.generating = !1;
    }
  }
  render() {
    return o`
      <!-- OpenAPI Diff -->
      <div class="section">
        <div class="section-title">OpenAPI Diff</div>
        <div class="diff-form">
          <div class="diff-field">
            <label>Base spec</label>
            <input
              type="text"
              placeholder="path/to/base.yaml"
              .value=${this.basePath}
              @input=${(t) => this.basePath = t.target.value}
            />
          </div>
          <div class="diff-field">
            <label>Revision spec</label>
            <input
              type="text"
              placeholder="path/to/revision.yaml"
              .value=${this.revisionPath}
              @input=${(t) => this.revisionPath = t.target.value}
            />
          </div>
          <button
            class="primary"
            ?disabled=${this.diffing}
            @click=${this.handleDiff}
          >
            ${this.diffing ? "Comparing…" : "Compare"}
          </button>
        </div>

        ${this.diffError ? o`<div class="error">${this.diffError}</div>` : l}

        ${this.diffResult ? o`
              <div class="diff-result ${this.diffResult.Breaking ? "breaking" : "safe"}">
                <div class="diff-summary">${this.diffResult.Summary}</div>
                ${this.diffResult.Changes && this.diffResult.Changes.length > 0 ? o`
                      <ul class="diff-changes">
                        ${this.diffResult.Changes.map(
      (t) => o`<li>${t}</li>`
    )}
                      </ul>
                    ` : l}
              </div>
            ` : l}
      </div>

      <!-- SDK Generation -->
      <div class="section">
        <div class="section-title">SDK Generation</div>

        ${this.generateError ? o`<div class="error">${this.generateError}</div>` : l}
        ${this.generateSuccess ? o`<div class="success">${this.generateSuccess}</div>` : l}

        <div class="generate-form">
          <select
            .value=${this.selectedLanguage}
            @change=${(t) => this.selectedLanguage = t.target.value}
          >
            <option value="">All languages</option>
            <option value="typescript">TypeScript</option>
            <option value="python">Python</option>
            <option value="go">Go</option>
            <option value="php">PHP</option>
          </select>
          <button
            class="primary"
            ?disabled=${this.generating}
            @click=${this.handleGenerate}
          >
            ${this.generating ? "Generating…" : "Generate SDK"}
          </button>
        </div>
      </div>
    `;
  }
};
$.styles = K`
    :host {
      display: block;
      font-family: system-ui, -apple-system, sans-serif;
    }

    .section {
      border: 1px solid #e5e7eb;
      border-radius: 0.5rem;
      padding: 1rem;
      background: #fff;
      margin-bottom: 1rem;
    }

    .section-title {
      font-size: 0.75rem;
      font-weight: 700;
      color: #6b7280;
      text-transform: uppercase;
      letter-spacing: 0.025em;
      margin-bottom: 0.75rem;
    }

    .diff-form {
      display: flex;
      gap: 0.5rem;
      align-items: flex-end;
      margin-bottom: 1rem;
    }

    .diff-field {
      flex: 1;
      display: flex;
      flex-direction: column;
      gap: 0.25rem;
    }

    .diff-field label {
      font-size: 0.75rem;
      font-weight: 500;
      color: #6b7280;
    }

    .diff-field input {
      padding: 0.375rem 0.75rem;
      border: 1px solid #d1d5db;
      border-radius: 0.375rem;
      font-size: 0.8125rem;
      font-family: monospace;
    }

    .diff-field input:focus {
      outline: none;
      border-color: #6366f1;
      box-shadow: 0 0 0 2px rgba(99, 102, 241, 0.2);
    }

    button {
      padding: 0.375rem 1rem;
      border-radius: 0.375rem;
      font-size: 0.8125rem;
      cursor: pointer;
      transition: background 0.15s;
    }

    button.primary {
      background: #6366f1;
      color: #fff;
      border: none;
    }

    button.primary:hover {
      background: #4f46e5;
    }

    button.primary:disabled {
      opacity: 0.5;
      cursor: not-allowed;
    }

    button.secondary {
      background: #fff;
      color: #374151;
      border: 1px solid #d1d5db;
    }

    button.secondary:hover {
      background: #f3f4f6;
    }

    .diff-result {
      padding: 0.75rem;
      border-radius: 0.375rem;
      font-size: 0.875rem;
      margin-top: 0.75rem;
    }

    .diff-result.breaking {
      background: #fef2f2;
      border: 1px solid #fecaca;
      color: #991b1b;
    }

    .diff-result.safe {
      background: #f0fdf4;
      border: 1px solid #bbf7d0;
      color: #166534;
    }

    .diff-summary {
      font-weight: 600;
      margin-bottom: 0.5rem;
    }

    .diff-changes {
      list-style: disc;
      padding-left: 1.25rem;
      margin: 0;
    }

    .diff-changes li {
      font-size: 0.8125rem;
      margin-bottom: 0.25rem;
      font-family: monospace;
    }

    .generate-form {
      display: flex;
      gap: 0.5rem;
      align-items: center;
    }

    .generate-form select {
      padding: 0.375rem 0.75rem;
      border: 1px solid #d1d5db;
      border-radius: 0.375rem;
      font-size: 0.8125rem;
      background: #fff;
    }

    .empty {
      text-align: center;
      padding: 2rem;
      color: #9ca3af;
      font-size: 0.875rem;
    }

    .error {
      color: #dc2626;
      padding: 0.75rem;
      background: #fef2f2;
      border-radius: 0.375rem;
      font-size: 0.875rem;
      margin-bottom: 1rem;
    }

    .success {
      padding: 0.75rem;
      background: #f0fdf4;
      border: 1px solid #bbf7d0;
      border-radius: 0.375rem;
      font-size: 0.875rem;
      color: #166534;
      margin-bottom: 1rem;
    }

    .loading {
      text-align: center;
      padding: 1rem;
      color: #6b7280;
      font-size: 0.875rem;
    }
  `;
k([
  z({ attribute: "api-url" })
], $.prototype, "apiUrl", 2);
k([
  p()
], $.prototype, "basePath", 2);
k([
  p()
], $.prototype, "revisionPath", 2);
k([
  p()
], $.prototype, "diffResult", 2);
k([
  p()
], $.prototype, "diffing", 2);
k([
  p()
], $.prototype, "diffError", 2);
k([
  p()
], $.prototype, "selectedLanguage", 2);
k([
  p()
], $.prototype, "generating", 2);
k([
  p()
], $.prototype, "generateError", 2);
k([
  p()
], $.prototype, "generateSuccess", 2);
$ = k([
  Z("core-build-sdk")
], $);
var rt = Object.defineProperty, nt = Object.getOwnPropertyDescriptor, H = (t, e, s, a) => {
  for (var i = a > 1 ? void 0 : a ? nt(e, s) : e, r = t.length - 1, n; r >= 0; r--)
    (n = t[r]) && (i = (a ? n(e, s, i) : n(i)) || i);
  return a && i && rt(e, s, i), i;
};
let O = class extends A {
  constructor() {
    super(...arguments), this.apiUrl = "", this.wsUrl = "", this.activeTab = "config", this.wsConnected = !1, this.lastEvent = "", this.ws = null, this.tabs = [
      { id: "config", label: "Config" },
      { id: "build", label: "Build" },
      { id: "release", label: "Release" },
      { id: "sdk", label: "SDK" }
    ];
  }
  connectedCallback() {
    super.connectedCallback(), this.wsUrl && this.connectWs();
  }
  disconnectedCallback() {
    super.disconnectedCallback(), this.ws && (this.ws.close(), this.ws = null);
  }
  connectWs() {
    this.ws = Ze(this.wsUrl, (t) => {
      this.lastEvent = t.channel ?? t.type ?? "", this.requestUpdate();
    }), this.ws.onopen = () => {
      this.wsConnected = !0;
    }, this.ws.onclose = () => {
      this.wsConnected = !1;
    };
  }
  handleTabClick(t) {
    this.activeTab = t;
  }
  handleRefresh() {
    var e;
    const t = (e = this.shadowRoot) == null ? void 0 : e.querySelector(".content");
    if (t) {
      const s = t.firstElementChild;
      s && "reload" in s && s.reload();
    }
  }
  renderContent() {
    switch (this.activeTab) {
      case "config":
        return o`<core-build-config api-url=${this.apiUrl}></core-build-config>`;
      case "build":
        return o`<core-build-artifacts api-url=${this.apiUrl}></core-build-artifacts>`;
      case "release":
        return o`<core-build-release api-url=${this.apiUrl}></core-build-release>`;
      case "sdk":
        return o`<core-build-sdk api-url=${this.apiUrl}></core-build-sdk>`;
      default:
        return l;
    }
  }
  render() {
    const t = this.wsUrl ? this.wsConnected ? "connected" : "disconnected" : "idle";
    return o`
      <div class="header">
        <span class="title">Build</span>
        <button class="refresh-btn" @click=${this.handleRefresh}>Refresh</button>
      </div>

      <div class="tabs">
        ${this.tabs.map(
      (e) => o`
            <button
              class="tab ${this.activeTab === e.id ? "active" : ""}"
              @click=${() => this.handleTabClick(e.id)}
            >
              ${e.label}
            </button>
          `
    )}
      </div>

      <div class="content">${this.renderContent()}</div>

      <div class="footer">
        <div class="ws-status">
          <span class="ws-dot ${t}"></span>
          <span>${t === "connected" ? "Connected" : t === "disconnected" ? "Disconnected" : "No WebSocket"}</span>
        </div>
        ${this.lastEvent ? o`<span>Last: ${this.lastEvent}</span>` : l}
      </div>
    `;
  }
};
O.styles = K`
    :host {
      display: flex;
      flex-direction: column;
      font-family: system-ui, -apple-system, sans-serif;
      height: 100%;
      background: #fafafa;
    }

    /* H — Header */
    .header {
      display: flex;
      justify-content: space-between;
      align-items: center;
      padding: 0.75rem 1rem;
      background: #fff;
      border-bottom: 1px solid #e5e7eb;
    }

    .title {
      font-weight: 700;
      font-size: 1rem;
      color: #111827;
    }

    .refresh-btn {
      padding: 0.375rem 0.75rem;
      border: 1px solid #d1d5db;
      border-radius: 0.375rem;
      background: #fff;
      font-size: 0.8125rem;
      cursor: pointer;
      transition: background 0.15s;
    }

    .refresh-btn:hover {
      background: #f3f4f6;
    }

    /* H-L — Tabs */
    .tabs {
      display: flex;
      gap: 0;
      background: #fff;
      border-bottom: 1px solid #e5e7eb;
      padding: 0 1rem;
    }

    .tab {
      padding: 0.625rem 1rem;
      font-size: 0.8125rem;
      font-weight: 500;
      color: #6b7280;
      cursor: pointer;
      border-bottom: 2px solid transparent;
      transition: all 0.15s;
      background: none;
      border-top: none;
      border-left: none;
      border-right: none;
    }

    .tab:hover {
      color: #374151;
    }

    .tab.active {
      color: #6366f1;
      border-bottom-color: #6366f1;
    }

    /* C — Content */
    .content {
      flex: 1;
      padding: 1rem;
      overflow-y: auto;
    }

    /* F — Footer / Status bar */
    .footer {
      display: flex;
      justify-content: space-between;
      align-items: center;
      padding: 0.5rem 1rem;
      background: #fff;
      border-top: 1px solid #e5e7eb;
      font-size: 0.75rem;
      color: #9ca3af;
    }

    .ws-status {
      display: flex;
      align-items: center;
      gap: 0.375rem;
    }

    .ws-dot {
      width: 0.5rem;
      height: 0.5rem;
      border-radius: 50%;
    }

    .ws-dot.connected {
      background: #22c55e;
    }

    .ws-dot.disconnected {
      background: #ef4444;
    }

    .ws-dot.idle {
      background: #d1d5db;
    }
  `;
H([
  z({ attribute: "api-url" })
], O.prototype, "apiUrl", 2);
H([
  z({ attribute: "ws-url" })
], O.prototype, "wsUrl", 2);
H([
  p()
], O.prototype, "activeTab", 2);
H([
  p()
], O.prototype, "wsConnected", 2);
H([
  p()
], O.prototype, "lastEvent", 2);
O = H([
  Z("core-build-panel")
], O);
export {
  te as BuildApi,
  _ as BuildArtifacts,
  E as BuildConfig,
  O as BuildPanel,
  m as BuildRelease,
  $ as BuildSdk,
  Ze as connectBuildEvents
};
