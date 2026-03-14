/**
 * @license
 * Copyright 2019 Google LLC
 * SPDX-License-Identifier: BSD-3-Clause
 */
const J = globalThis, re = J.ShadowRoot && (J.ShadyCSS === void 0 || J.ShadyCSS.nativeShadow) && "adoptedStyleSheets" in Document.prototype && "replace" in CSSStyleSheet.prototype, oe = Symbol(), de = /* @__PURE__ */ new WeakMap();
let _e = class {
  constructor(e, t, r) {
    if (this._$cssResult$ = !0, r !== oe) throw Error("CSSResult is not constructable. Use `unsafeCSS` or `css` instead.");
    this.cssText = e, this.t = t;
  }
  get styleSheet() {
    let e = this.o;
    const t = this.t;
    if (re && e === void 0) {
      const r = t !== void 0 && t.length === 1;
      r && (e = de.get(t)), e === void 0 && ((this.o = e = new CSSStyleSheet()).replaceSync(this.cssText), r && de.set(t, e));
    }
    return e;
  }
  toString() {
    return this.cssText;
  }
};
const Ee = (s) => new _e(typeof s == "string" ? s : s + "", void 0, oe), F = (s, ...e) => {
  const t = s.length === 1 ? s[0] : e.reduce((r, i, o) => r + ((n) => {
    if (n._$cssResult$ === !0) return n.cssText;
    if (typeof n == "number") return n;
    throw Error("Value passed to 'css' function must be a 'css' function result: " + n + ". Use 'unsafeCSS' to pass non-literal values, but take care to ensure page security.");
  })(i) + s[o + 1], s[0]);
  return new _e(t, s, oe);
}, Ce = (s, e) => {
  if (re) s.adoptedStyleSheets = e.map((t) => t instanceof CSSStyleSheet ? t : t.styleSheet);
  else for (const t of e) {
    const r = document.createElement("style"), i = J.litNonce;
    i !== void 0 && r.setAttribute("nonce", i), r.textContent = t.cssText, s.appendChild(r);
  }
}, ce = re ? (s) => s : (s) => s instanceof CSSStyleSheet ? ((e) => {
  let t = "";
  for (const r of e.cssRules) t += r.cssText;
  return Ee(t);
})(s) : s;
/**
 * @license
 * Copyright 2017 Google LLC
 * SPDX-License-Identifier: BSD-3-Clause
 */
const { is: Pe, defineProperty: ke, getOwnPropertyDescriptor: Ue, getOwnPropertyNames: Re, getOwnPropertySymbols: Oe, getPrototypeOf: De } = Object, E = globalThis, he = E.trustedTypes, ze = he ? he.emptyScript : "", ee = E.reactiveElementPolyfillSupport, L = (s, e) => s, Z = { toAttribute(s, e) {
  switch (e) {
    case Boolean:
      s = s ? ze : null;
      break;
    case Object:
    case Array:
      s = s == null ? s : JSON.stringify(s);
  }
  return s;
}, fromAttribute(s, e) {
  let t = s;
  switch (e) {
    case Boolean:
      t = s !== null;
      break;
    case Number:
      t = s === null ? null : Number(s);
      break;
    case Object:
    case Array:
      try {
        t = JSON.parse(s);
      } catch {
        t = null;
      }
  }
  return t;
} }, ne = (s, e) => !Pe(s, e), fe = { attribute: !0, type: String, converter: Z, reflect: !1, useDefault: !1, hasChanged: ne };
Symbol.metadata ?? (Symbol.metadata = Symbol("metadata")), E.litPropertyMetadata ?? (E.litPropertyMetadata = /* @__PURE__ */ new WeakMap());
let T = class extends HTMLElement {
  static addInitializer(e) {
    this._$Ei(), (this.l ?? (this.l = [])).push(e);
  }
  static get observedAttributes() {
    return this.finalize(), this._$Eh && [...this._$Eh.keys()];
  }
  static createProperty(e, t = fe) {
    if (t.state && (t.attribute = !1), this._$Ei(), this.prototype.hasOwnProperty(e) && ((t = Object.create(t)).wrapped = !0), this.elementProperties.set(e, t), !t.noAccessor) {
      const r = Symbol(), i = this.getPropertyDescriptor(e, r, t);
      i !== void 0 && ke(this.prototype, e, i);
    }
  }
  static getPropertyDescriptor(e, t, r) {
    const { get: i, set: o } = Ue(this.prototype, e) ?? { get() {
      return this[t];
    }, set(n) {
      this[t] = n;
    } };
    return { get: i, set(n) {
      const d = i == null ? void 0 : i.call(this);
      o == null || o.call(this, n), this.requestUpdate(e, d, r);
    }, configurable: !0, enumerable: !0 };
  }
  static getPropertyOptions(e) {
    return this.elementProperties.get(e) ?? fe;
  }
  static _$Ei() {
    if (this.hasOwnProperty(L("elementProperties"))) return;
    const e = De(this);
    e.finalize(), e.l !== void 0 && (this.l = [...e.l]), this.elementProperties = new Map(e.elementProperties);
  }
  static finalize() {
    if (this.hasOwnProperty(L("finalized"))) return;
    if (this.finalized = !0, this._$Ei(), this.hasOwnProperty(L("properties"))) {
      const t = this.properties, r = [...Re(t), ...Oe(t)];
      for (const i of r) this.createProperty(i, t[i]);
    }
    const e = this[Symbol.metadata];
    if (e !== null) {
      const t = litPropertyMetadata.get(e);
      if (t !== void 0) for (const [r, i] of t) this.elementProperties.set(r, i);
    }
    this._$Eh = /* @__PURE__ */ new Map();
    for (const [t, r] of this.elementProperties) {
      const i = this._$Eu(t, r);
      i !== void 0 && this._$Eh.set(i, t);
    }
    this.elementStyles = this.finalizeStyles(this.styles);
  }
  static finalizeStyles(e) {
    const t = [];
    if (Array.isArray(e)) {
      const r = new Set(e.flat(1 / 0).reverse());
      for (const i of r) t.unshift(ce(i));
    } else e !== void 0 && t.push(ce(e));
    return t;
  }
  static _$Eu(e, t) {
    const r = t.attribute;
    return r === !1 ? void 0 : typeof r == "string" ? r : typeof e == "string" ? e.toLowerCase() : void 0;
  }
  constructor() {
    super(), this._$Ep = void 0, this.isUpdatePending = !1, this.hasUpdated = !1, this._$Em = null, this._$Ev();
  }
  _$Ev() {
    var e;
    this._$ES = new Promise((t) => this.enableUpdating = t), this._$AL = /* @__PURE__ */ new Map(), this._$E_(), this.requestUpdate(), (e = this.constructor.l) == null || e.forEach((t) => t(this));
  }
  addController(e) {
    var t;
    (this._$EO ?? (this._$EO = /* @__PURE__ */ new Set())).add(e), this.renderRoot !== void 0 && this.isConnected && ((t = e.hostConnected) == null || t.call(e));
  }
  removeController(e) {
    var t;
    (t = this._$EO) == null || t.delete(e);
  }
  _$E_() {
    const e = /* @__PURE__ */ new Map(), t = this.constructor.elementProperties;
    for (const r of t.keys()) this.hasOwnProperty(r) && (e.set(r, this[r]), delete this[r]);
    e.size > 0 && (this._$Ep = e);
  }
  createRenderRoot() {
    const e = this.shadowRoot ?? this.attachShadow(this.constructor.shadowRootOptions);
    return Ce(e, this.constructor.elementStyles), e;
  }
  connectedCallback() {
    var e;
    this.renderRoot ?? (this.renderRoot = this.createRenderRoot()), this.enableUpdating(!0), (e = this._$EO) == null || e.forEach((t) => {
      var r;
      return (r = t.hostConnected) == null ? void 0 : r.call(t);
    });
  }
  enableUpdating(e) {
  }
  disconnectedCallback() {
    var e;
    (e = this._$EO) == null || e.forEach((t) => {
      var r;
      return (r = t.hostDisconnected) == null ? void 0 : r.call(t);
    });
  }
  attributeChangedCallback(e, t, r) {
    this._$AK(e, r);
  }
  _$ET(e, t) {
    var o;
    const r = this.constructor.elementProperties.get(e), i = this.constructor._$Eu(e, r);
    if (i !== void 0 && r.reflect === !0) {
      const n = (((o = r.converter) == null ? void 0 : o.toAttribute) !== void 0 ? r.converter : Z).toAttribute(t, r.type);
      this._$Em = e, n == null ? this.removeAttribute(i) : this.setAttribute(i, n), this._$Em = null;
    }
  }
  _$AK(e, t) {
    var o, n;
    const r = this.constructor, i = r._$Eh.get(e);
    if (i !== void 0 && this._$Em !== i) {
      const d = r.getPropertyOptions(i), a = typeof d.converter == "function" ? { fromAttribute: d.converter } : ((o = d.converter) == null ? void 0 : o.fromAttribute) !== void 0 ? d.converter : Z;
      this._$Em = i;
      const u = a.fromAttribute(t, d.type);
      this[i] = u ?? ((n = this._$Ej) == null ? void 0 : n.get(i)) ?? u, this._$Em = null;
    }
  }
  requestUpdate(e, t, r, i = !1, o) {
    var n;
    if (e !== void 0) {
      const d = this.constructor;
      if (i === !1 && (o = this[e]), r ?? (r = d.getPropertyOptions(e)), !((r.hasChanged ?? ne)(o, t) || r.useDefault && r.reflect && o === ((n = this._$Ej) == null ? void 0 : n.get(e)) && !this.hasAttribute(d._$Eu(e, r)))) return;
      this.C(e, t, r);
    }
    this.isUpdatePending === !1 && (this._$ES = this._$EP());
  }
  C(e, t, { useDefault: r, reflect: i, wrapped: o }, n) {
    r && !(this._$Ej ?? (this._$Ej = /* @__PURE__ */ new Map())).has(e) && (this._$Ej.set(e, n ?? t ?? this[e]), o !== !0 || n !== void 0) || (this._$AL.has(e) || (this.hasUpdated || r || (t = void 0), this._$AL.set(e, t)), i === !0 && this._$Em !== e && (this._$Eq ?? (this._$Eq = /* @__PURE__ */ new Set())).add(e));
  }
  async _$EP() {
    this.isUpdatePending = !0;
    try {
      await this._$ES;
    } catch (t) {
      Promise.reject(t);
    }
    const e = this.scheduleUpdate();
    return e != null && await e, !this.isUpdatePending;
  }
  scheduleUpdate() {
    return this.performUpdate();
  }
  performUpdate() {
    var r;
    if (!this.isUpdatePending) return;
    if (!this.hasUpdated) {
      if (this.renderRoot ?? (this.renderRoot = this.createRenderRoot()), this._$Ep) {
        for (const [o, n] of this._$Ep) this[o] = n;
        this._$Ep = void 0;
      }
      const i = this.constructor.elementProperties;
      if (i.size > 0) for (const [o, n] of i) {
        const { wrapped: d } = n, a = this[o];
        d !== !0 || this._$AL.has(o) || a === void 0 || this.C(o, void 0, n, a);
      }
    }
    let e = !1;
    const t = this._$AL;
    try {
      e = this.shouldUpdate(t), e ? (this.willUpdate(t), (r = this._$EO) == null || r.forEach((i) => {
        var o;
        return (o = i.hostUpdate) == null ? void 0 : o.call(i);
      }), this.update(t)) : this._$EM();
    } catch (i) {
      throw e = !1, this._$EM(), i;
    }
    e && this._$AE(t);
  }
  willUpdate(e) {
  }
  _$AE(e) {
    var t;
    (t = this._$EO) == null || t.forEach((r) => {
      var i;
      return (i = r.hostUpdated) == null ? void 0 : i.call(r);
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
    this._$Eq && (this._$Eq = this._$Eq.forEach((t) => this._$ET(t, this[t]))), this._$EM();
  }
  updated(e) {
  }
  firstUpdated(e) {
  }
};
T.elementStyles = [], T.shadowRootOptions = { mode: "open" }, T[L("elementProperties")] = /* @__PURE__ */ new Map(), T[L("finalized")] = /* @__PURE__ */ new Map(), ee == null || ee({ ReactiveElement: T }), (E.reactiveElementVersions ?? (E.reactiveElementVersions = [])).push("2.1.2");
/**
 * @license
 * Copyright 2017 Google LLC
 * SPDX-License-Identifier: BSD-3-Clause
 */
const q = globalThis, ue = (s) => s, Q = q.trustedTypes, pe = Q ? Q.createPolicy("lit-html", { createHTML: (s) => s }) : void 0, we = "$lit$", S = `lit$${Math.random().toFixed(9).slice(2)}$`, Ae = "?" + S, Te = `<${Ae}>`, D = document, W = () => D.createComment(""), I = (s) => s === null || typeof s != "object" && typeof s != "function", ae = Array.isArray, Be = (s) => ae(s) || typeof (s == null ? void 0 : s[Symbol.iterator]) == "function", te = `[ 	
\f\r]`, M = /<(?:(!--|\/[^a-zA-Z])|(\/?[a-zA-Z][^>\s]*)|(\/?$))/g, ge = /-->/g, me = />/g, U = RegExp(`>|${te}(?:([^\\s"'>=/]+)(${te}*=${te}*(?:[^ 	
\f\r"'\`<>=]|("|')|))|$)`, "g"), be = /'/g, ve = /"/g, xe = /^(?:script|style|textarea|title)$/i, je = (s) => (e, ...t) => ({ _$litType$: s, strings: e, values: t }), l = je(1), B = Symbol.for("lit-noChange"), c = Symbol.for("lit-nothing"), $e = /* @__PURE__ */ new WeakMap(), R = D.createTreeWalker(D, 129);
function Se(s, e) {
  if (!ae(s) || !s.hasOwnProperty("raw")) throw Error("invalid template strings array");
  return pe !== void 0 ? pe.createHTML(e) : e;
}
const He = (s, e) => {
  const t = s.length - 1, r = [];
  let i, o = e === 2 ? "<svg>" : e === 3 ? "<math>" : "", n = M;
  for (let d = 0; d < t; d++) {
    const a = s[d];
    let u, p, f = -1, g = 0;
    for (; g < a.length && (n.lastIndex = g, p = n.exec(a), p !== null); ) g = n.lastIndex, n === M ? p[1] === "!--" ? n = ge : p[1] !== void 0 ? n = me : p[2] !== void 0 ? (xe.test(p[2]) && (i = RegExp("</" + p[2], "g")), n = U) : p[3] !== void 0 && (n = U) : n === U ? p[0] === ">" ? (n = i ?? M, f = -1) : p[1] === void 0 ? f = -2 : (f = n.lastIndex - p[2].length, u = p[1], n = p[3] === void 0 ? U : p[3] === '"' ? ve : be) : n === ve || n === be ? n = U : n === ge || n === me ? n = M : (n = U, i = void 0);
    const b = n === U && s[d + 1].startsWith("/>") ? " " : "";
    o += n === M ? a + Te : f >= 0 ? (r.push(u), a.slice(0, f) + we + a.slice(f) + S + b) : a + S + (f === -2 ? d : b);
  }
  return [Se(s, o + (s[t] || "<?>") + (e === 2 ? "</svg>" : e === 3 ? "</math>" : "")), r];
};
class G {
  constructor({ strings: e, _$litType$: t }, r) {
    let i;
    this.parts = [];
    let o = 0, n = 0;
    const d = e.length - 1, a = this.parts, [u, p] = He(e, t);
    if (this.el = G.createElement(u, r), R.currentNode = this.el.content, t === 2 || t === 3) {
      const f = this.el.content.firstChild;
      f.replaceWith(...f.childNodes);
    }
    for (; (i = R.nextNode()) !== null && a.length < d; ) {
      if (i.nodeType === 1) {
        if (i.hasAttributes()) for (const f of i.getAttributeNames()) if (f.endsWith(we)) {
          const g = p[n++], b = i.getAttribute(f).split(S), _ = /([.?@])?(.*)/.exec(g);
          a.push({ type: 1, index: o, name: _[2], strings: b, ctor: _[1] === "." ? Me : _[1] === "?" ? Le : _[1] === "@" ? qe : X }), i.removeAttribute(f);
        } else f.startsWith(S) && (a.push({ type: 6, index: o }), i.removeAttribute(f));
        if (xe.test(i.tagName)) {
          const f = i.textContent.split(S), g = f.length - 1;
          if (g > 0) {
            i.textContent = Q ? Q.emptyScript : "";
            for (let b = 0; b < g; b++) i.append(f[b], W()), R.nextNode(), a.push({ type: 2, index: ++o });
            i.append(f[g], W());
          }
        }
      } else if (i.nodeType === 8) if (i.data === Ae) a.push({ type: 2, index: o });
      else {
        let f = -1;
        for (; (f = i.data.indexOf(S, f + 1)) !== -1; ) a.push({ type: 7, index: o }), f += S.length - 1;
      }
      o++;
    }
  }
  static createElement(e, t) {
    const r = D.createElement("template");
    return r.innerHTML = e, r;
  }
}
function j(s, e, t = s, r) {
  var n, d;
  if (e === B) return e;
  let i = r !== void 0 ? (n = t._$Co) == null ? void 0 : n[r] : t._$Cl;
  const o = I(e) ? void 0 : e._$litDirective$;
  return (i == null ? void 0 : i.constructor) !== o && ((d = i == null ? void 0 : i._$AO) == null || d.call(i, !1), o === void 0 ? i = void 0 : (i = new o(s), i._$AT(s, t, r)), r !== void 0 ? (t._$Co ?? (t._$Co = []))[r] = i : t._$Cl = i), i !== void 0 && (e = j(s, i._$AS(s, e.values), i, r)), e;
}
class Ne {
  constructor(e, t) {
    this._$AV = [], this._$AN = void 0, this._$AD = e, this._$AM = t;
  }
  get parentNode() {
    return this._$AM.parentNode;
  }
  get _$AU() {
    return this._$AM._$AU;
  }
  u(e) {
    const { el: { content: t }, parts: r } = this._$AD, i = ((e == null ? void 0 : e.creationScope) ?? D).importNode(t, !0);
    R.currentNode = i;
    let o = R.nextNode(), n = 0, d = 0, a = r[0];
    for (; a !== void 0; ) {
      if (n === a.index) {
        let u;
        a.type === 2 ? u = new V(o, o.nextSibling, this, e) : a.type === 1 ? u = new a.ctor(o, a.name, a.strings, this, e) : a.type === 6 && (u = new We(o, this, e)), this._$AV.push(u), a = r[++d];
      }
      n !== (a == null ? void 0 : a.index) && (o = R.nextNode(), n++);
    }
    return R.currentNode = D, i;
  }
  p(e) {
    let t = 0;
    for (const r of this._$AV) r !== void 0 && (r.strings !== void 0 ? (r._$AI(e, r, t), t += r.strings.length - 2) : r._$AI(e[t])), t++;
  }
}
class V {
  get _$AU() {
    var e;
    return ((e = this._$AM) == null ? void 0 : e._$AU) ?? this._$Cv;
  }
  constructor(e, t, r, i) {
    this.type = 2, this._$AH = c, this._$AN = void 0, this._$AA = e, this._$AB = t, this._$AM = r, this.options = i, this._$Cv = (i == null ? void 0 : i.isConnected) ?? !0;
  }
  get parentNode() {
    let e = this._$AA.parentNode;
    const t = this._$AM;
    return t !== void 0 && (e == null ? void 0 : e.nodeType) === 11 && (e = t.parentNode), e;
  }
  get startNode() {
    return this._$AA;
  }
  get endNode() {
    return this._$AB;
  }
  _$AI(e, t = this) {
    e = j(this, e, t), I(e) ? e === c || e == null || e === "" ? (this._$AH !== c && this._$AR(), this._$AH = c) : e !== this._$AH && e !== B && this._(e) : e._$litType$ !== void 0 ? this.$(e) : e.nodeType !== void 0 ? this.T(e) : Be(e) ? this.k(e) : this._(e);
  }
  O(e) {
    return this._$AA.parentNode.insertBefore(e, this._$AB);
  }
  T(e) {
    this._$AH !== e && (this._$AR(), this._$AH = this.O(e));
  }
  _(e) {
    this._$AH !== c && I(this._$AH) ? this._$AA.nextSibling.data = e : this.T(D.createTextNode(e)), this._$AH = e;
  }
  $(e) {
    var o;
    const { values: t, _$litType$: r } = e, i = typeof r == "number" ? this._$AC(e) : (r.el === void 0 && (r.el = G.createElement(Se(r.h, r.h[0]), this.options)), r);
    if (((o = this._$AH) == null ? void 0 : o._$AD) === i) this._$AH.p(t);
    else {
      const n = new Ne(i, this), d = n.u(this.options);
      n.p(t), this.T(d), this._$AH = n;
    }
  }
  _$AC(e) {
    let t = $e.get(e.strings);
    return t === void 0 && $e.set(e.strings, t = new G(e)), t;
  }
  k(e) {
    ae(this._$AH) || (this._$AH = [], this._$AR());
    const t = this._$AH;
    let r, i = 0;
    for (const o of e) i === t.length ? t.push(r = new V(this.O(W()), this.O(W()), this, this.options)) : r = t[i], r._$AI(o), i++;
    i < t.length && (this._$AR(r && r._$AB.nextSibling, i), t.length = i);
  }
  _$AR(e = this._$AA.nextSibling, t) {
    var r;
    for ((r = this._$AP) == null ? void 0 : r.call(this, !1, !0, t); e !== this._$AB; ) {
      const i = ue(e).nextSibling;
      ue(e).remove(), e = i;
    }
  }
  setConnected(e) {
    var t;
    this._$AM === void 0 && (this._$Cv = e, (t = this._$AP) == null || t.call(this, e));
  }
}
class X {
  get tagName() {
    return this.element.tagName;
  }
  get _$AU() {
    return this._$AM._$AU;
  }
  constructor(e, t, r, i, o) {
    this.type = 1, this._$AH = c, this._$AN = void 0, this.element = e, this.name = t, this._$AM = i, this.options = o, r.length > 2 || r[0] !== "" || r[1] !== "" ? (this._$AH = Array(r.length - 1).fill(new String()), this.strings = r) : this._$AH = c;
  }
  _$AI(e, t = this, r, i) {
    const o = this.strings;
    let n = !1;
    if (o === void 0) e = j(this, e, t, 0), n = !I(e) || e !== this._$AH && e !== B, n && (this._$AH = e);
    else {
      const d = e;
      let a, u;
      for (e = o[0], a = 0; a < o.length - 1; a++) u = j(this, d[r + a], t, a), u === B && (u = this._$AH[a]), n || (n = !I(u) || u !== this._$AH[a]), u === c ? e = c : e !== c && (e += (u ?? "") + o[a + 1]), this._$AH[a] = u;
    }
    n && !i && this.j(e);
  }
  j(e) {
    e === c ? this.element.removeAttribute(this.name) : this.element.setAttribute(this.name, e ?? "");
  }
}
class Me extends X {
  constructor() {
    super(...arguments), this.type = 3;
  }
  j(e) {
    this.element[this.name] = e === c ? void 0 : e;
  }
}
class Le extends X {
  constructor() {
    super(...arguments), this.type = 4;
  }
  j(e) {
    this.element.toggleAttribute(this.name, !!e && e !== c);
  }
}
class qe extends X {
  constructor(e, t, r, i, o) {
    super(e, t, r, i, o), this.type = 5;
  }
  _$AI(e, t = this) {
    if ((e = j(this, e, t, 0) ?? c) === B) return;
    const r = this._$AH, i = e === c && r !== c || e.capture !== r.capture || e.once !== r.once || e.passive !== r.passive, o = e !== c && (r === c || i);
    i && this.element.removeEventListener(this.name, this, r), o && this.element.addEventListener(this.name, this, e), this._$AH = e;
  }
  handleEvent(e) {
    var t;
    typeof this._$AH == "function" ? this._$AH.call(((t = this.options) == null ? void 0 : t.host) ?? this.element, e) : this._$AH.handleEvent(e);
  }
}
class We {
  constructor(e, t, r) {
    this.element = e, this.type = 6, this._$AN = void 0, this._$AM = t, this.options = r;
  }
  get _$AU() {
    return this._$AM._$AU;
  }
  _$AI(e) {
    j(this, e);
  }
}
const se = q.litHtmlPolyfillSupport;
se == null || se(G, V), (q.litHtmlVersions ?? (q.litHtmlVersions = [])).push("3.3.2");
const Ie = (s, e, t) => {
  const r = (t == null ? void 0 : t.renderBefore) ?? e;
  let i = r._$litPart$;
  if (i === void 0) {
    const o = (t == null ? void 0 : t.renderBefore) ?? null;
    r._$litPart$ = i = new V(e.insertBefore(W(), o), o, void 0, t ?? {});
  }
  return i._$AI(s), i;
};
/**
 * @license
 * Copyright 2017 Google LLC
 * SPDX-License-Identifier: BSD-3-Clause
 */
const O = globalThis;
class w extends T {
  constructor() {
    super(...arguments), this.renderOptions = { host: this }, this._$Do = void 0;
  }
  createRenderRoot() {
    var t;
    const e = super.createRenderRoot();
    return (t = this.renderOptions).renderBefore ?? (t.renderBefore = e.firstChild), e;
  }
  update(e) {
    const t = this.render();
    this.hasUpdated || (this.renderOptions.isConnected = this.isConnected), super.update(e), this._$Do = Ie(t, this.renderRoot, this.renderOptions);
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
var ye;
w._$litElement$ = !0, w.finalized = !0, (ye = O.litElementHydrateSupport) == null || ye.call(O, { LitElement: w });
const ie = O.litElementPolyfillSupport;
ie == null || ie({ LitElement: w });
(O.litElementVersions ?? (O.litElementVersions = [])).push("4.2.2");
/**
 * @license
 * Copyright 2017 Google LLC
 * SPDX-License-Identifier: BSD-3-Clause
 */
const K = (s) => (e, t) => {
  t !== void 0 ? t.addInitializer(() => {
    customElements.define(s, e);
  }) : customElements.define(s, e);
};
/**
 * @license
 * Copyright 2017 Google LLC
 * SPDX-License-Identifier: BSD-3-Clause
 */
const Ge = { attribute: !0, type: String, converter: Z, reflect: !1, hasChanged: ne }, Fe = (s = Ge, e, t) => {
  const { kind: r, metadata: i } = t;
  let o = globalThis.litPropertyMetadata.get(i);
  if (o === void 0 && globalThis.litPropertyMetadata.set(i, o = /* @__PURE__ */ new Map()), r === "setter" && ((s = Object.create(s)).wrapped = !0), o.set(t.name, s), r === "accessor") {
    const { name: n } = t;
    return { set(d) {
      const a = e.get.call(this);
      e.set.call(this, d), this.requestUpdate(n, a, s, !0, d);
    }, init(d) {
      return d !== void 0 && this.C(n, void 0, s, d), d;
    } };
  }
  if (r === "setter") {
    const { name: n } = t;
    return function(d) {
      const a = this[n];
      e.call(this, d), this.requestUpdate(n, a, s, !0, d);
    };
  }
  throw Error("Unsupported decorator location: " + r);
};
function z(s) {
  return (e, t) => typeof t == "object" ? Fe(s, e, t) : ((r, i, o) => {
    const n = i.hasOwnProperty(o);
    return i.constructor.createProperty(o, r), n ? Object.getOwnPropertyDescriptor(i, o) : void 0;
  })(s, e, t);
}
/**
 * @license
 * Copyright 2017 Google LLC
 * SPDX-License-Identifier: BSD-3-Clause
 */
function h(s) {
  return z({ ...s, state: !0, attribute: !1 });
}
function Ve(s, e) {
  const t = new WebSocket(s);
  return t.onmessage = (r) => {
    var i, o, n, d, a, u, p, f, g, b, _, le;
    try {
      const k = JSON.parse(r.data);
      ((o = (i = k.type) == null ? void 0 : i.startsWith) != null && o.call(i, "build.") || (d = (n = k.type) == null ? void 0 : n.startsWith) != null && d.call(n, "release.") || (u = (a = k.type) == null ? void 0 : a.startsWith) != null && u.call(a, "sdk.") || (f = (p = k.channel) == null ? void 0 : p.startsWith) != null && f.call(p, "build.") || (b = (g = k.channel) == null ? void 0 : g.startsWith) != null && b.call(g, "release.") || (le = (_ = k.channel) == null ? void 0 : _.startsWith) != null && le.call(_, "sdk.")) && e(k);
    } catch {
    }
  }, t;
}
class Y {
  constructor(e = "") {
    this.baseUrl = e;
  }
  get base() {
    return `${this.baseUrl}/api/v1/build`;
  }
  async request(e, t) {
    var o;
    const i = await (await fetch(`${this.base}${e}`, t)).json();
    if (!i.success)
      throw new Error(((o = i.error) == null ? void 0 : o.message) ?? "Request failed");
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
  changelog(e, t) {
    const r = new URLSearchParams();
    e && r.set("from", e), t && r.set("to", t);
    const i = r.toString();
    return this.request(`/release/changelog${i ? `?${i}` : ""}`);
  }
  release(e = !1) {
    const t = e ? "?dry_run=true" : "";
    return this.request(`/release${t}`, { method: "POST" });
  }
  // -- SDK --------------------------------------------------------------------
  sdkDiff(e, t) {
    const r = new URLSearchParams({ base: e, revision: t });
    return this.request(`/sdk/diff?${r.toString()}`);
  }
  sdkGenerate(e) {
    const t = e ? JSON.stringify({ language: e }) : void 0;
    return this.request("/sdk/generate", {
      method: "POST",
      headers: t ? { "Content-Type": "application/json" } : void 0,
      body: t
    });
  }
}
var Ke = Object.defineProperty, Je = Object.getOwnPropertyDescriptor, H = (s, e, t, r) => {
  for (var i = r > 1 ? void 0 : r ? Je(e, t) : e, o = s.length - 1, n; o >= 0; o--)
    (n = s[o]) && (i = (r ? n(e, t, i) : n(i)) || i);
  return r && i && Ke(e, t, i), i;
};
let C = class extends w {
  constructor() {
    super(...arguments), this.apiUrl = "", this.configData = null, this.discoverData = null, this.loading = !0, this.error = "";
  }
  connectedCallback() {
    super.connectedCallback(), this.api = new Y(this.apiUrl), this.reload();
  }
  async reload() {
    this.loading = !0, this.error = "";
    try {
      const [s, e] = await Promise.all([
        this.api.config(),
        this.api.discover()
      ]);
      this.configData = s, this.discoverData = e;
    } catch (s) {
      this.error = s.message ?? "Failed to load configuration";
    } finally {
      this.loading = !1;
    }
  }
  render() {
    if (this.loading)
      return l`<div class="loading">Loading configuration\u2026</div>`;
    if (this.error)
      return l`<div class="error">${this.error}</div>`;
    if (!this.configData)
      return l`<div class="empty">No configuration available.</div>`;
    const s = this.configData.config, e = this.discoverData;
    return l`
      <!-- Discovery -->
      <div class="section">
        <div class="section-title">Project Detection</div>
        <div class="field">
          <span class="field-label">Config file</span>
          <span class="badge ${this.configData.has_config ? "present" : "absent"}">
            ${this.configData.has_config ? "Present" : "Using defaults"}
          </span>
        </div>
        ${e ? l`
              <div class="field">
                <span class="field-label">Primary type</span>
                <span class="badge type-${e.primary || "unknown"}">${e.primary || "none"}</span>
              </div>
              ${e.types.length > 1 ? l`
                    <div class="field">
                      <span class="field-label">Detected types</span>
                      <span class="field-value">${e.types.join(", ")}</span>
                    </div>
                  ` : c}
              <div class="field">
                <span class="field-label">Directory</span>
                <span class="field-value">${e.dir}</span>
              </div>
            ` : c}
      </div>

      <!-- Project -->
      <div class="section">
        <div class="section-title">Project</div>
        ${s.project.name ? l`
              <div class="field">
                <span class="field-label">Name</span>
                <span class="field-value">${s.project.name}</span>
              </div>
            ` : c}
        ${s.project.binary ? l`
              <div class="field">
                <span class="field-label">Binary</span>
                <span class="field-value">${s.project.binary}</span>
              </div>
            ` : c}
        <div class="field">
          <span class="field-label">Main</span>
          <span class="field-value">${s.project.main}</span>
        </div>
      </div>

      <!-- Build Settings -->
      <div class="section">
        <div class="section-title">Build Settings</div>
        ${s.build.type ? l`
              <div class="field">
                <span class="field-label">Type override</span>
                <span class="field-value">${s.build.type}</span>
              </div>
            ` : c}
        <div class="field">
          <span class="field-label">CGO</span>
          <span class="field-value">${s.build.cgo ? "Enabled" : "Disabled"}</span>
        </div>
        ${s.build.flags && s.build.flags.length > 0 ? l`
              <div class="field">
                <span class="field-label">Flags</span>
                <div class="flags">
                  ${s.build.flags.map((t) => l`<span class="flag">${t}</span>`)}
                </div>
              </div>
            ` : c}
        ${s.build.ldflags && s.build.ldflags.length > 0 ? l`
              <div class="field">
                <span class="field-label">LD flags</span>
                <div class="flags">
                  ${s.build.ldflags.map((t) => l`<span class="flag">${t}</span>`)}
                </div>
              </div>
            ` : c}
      </div>

      <!-- Targets -->
      <div class="section">
        <div class="section-title">Targets</div>
        <div class="targets">
          ${s.targets.map(
      (t) => l`<span class="target-badge">${t.os}/${t.arch}</span>`
    )}
        </div>
      </div>
    `;
  }
};
C.styles = F`
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
      colour: #6b7280;
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
      colour: #374151;
    }

    .field-value {
      font-size: 0.8125rem;
      font-family: monospace;
      colour: #6b7280;
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
      colour: #166534;
    }

    .badge.absent {
      background: #fef3c7;
      colour: #92400e;
    }

    .badge.type-go {
      background: #dbeafe;
      colour: #1e40af;
    }

    .badge.type-wails {
      background: #f3e8ff;
      colour: #6b21a8;
    }

    .badge.type-node {
      background: #dcfce7;
      colour: #166534;
    }

    .badge.type-php {
      background: #fef3c7;
      colour: #92400e;
    }

    .badge.type-docker {
      background: #e0e7ff;
      colour: #3730a3;
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
      colour: #374151;
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
      colour: #6b7280;
    }

    .empty {
      text-align: center;
      padding: 2rem;
      colour: #9ca3af;
      font-size: 0.875rem;
    }

    .loading {
      text-align: center;
      padding: 2rem;
      colour: #6b7280;
    }

    .error {
      colour: #dc2626;
      padding: 0.75rem;
      background: #fef2f2;
      border-radius: 0.375rem;
      font-size: 0.875rem;
    }
  `;
H([
  z({ attribute: "api-url" })
], C.prototype, "apiUrl", 2);
H([
  h()
], C.prototype, "configData", 2);
H([
  h()
], C.prototype, "discoverData", 2);
H([
  h()
], C.prototype, "loading", 2);
H([
  h()
], C.prototype, "error", 2);
C = H([
  K("core-build-config")
], C);
var Ze = Object.defineProperty, Qe = Object.getOwnPropertyDescriptor, A = (s, e, t, r) => {
  for (var i = r > 1 ? void 0 : r ? Qe(e, t) : e, o = s.length - 1, n; o >= 0; o--)
    (n = s[o]) && (i = (r ? n(e, t, i) : n(i)) || i);
  return r && i && Ze(e, t, i), i;
};
let v = class extends w {
  constructor() {
    super(...arguments), this.apiUrl = "", this.artifacts = [], this.distExists = !1, this.loading = !0, this.error = "", this.building = !1, this.confirmBuild = !1, this.buildSuccess = "";
  }
  connectedCallback() {
    super.connectedCallback(), this.api = new Y(this.apiUrl), this.reload();
  }
  async reload() {
    this.loading = !0, this.error = "";
    try {
      const s = await this.api.artifacts();
      this.artifacts = s.artifacts ?? [], this.distExists = s.exists ?? !1;
    } catch (s) {
      this.error = s.message ?? "Failed to load artifacts";
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
    var s;
    this.confirmBuild = !1, this.building = !0, this.error = "", this.buildSuccess = "";
    try {
      const e = await this.api.build();
      this.buildSuccess = `Build complete — ${((s = e.artifacts) == null ? void 0 : s.length) ?? 0} artifact(s) produced (${e.version})`, await this.reload();
    } catch (e) {
      this.error = e.message ?? "Build failed";
    } finally {
      this.building = !1;
    }
  }
  formatSize(s) {
    return s < 1024 ? `${s} B` : s < 1024 * 1024 ? `${(s / 1024).toFixed(1)} KB` : `${(s / (1024 * 1024)).toFixed(1)} MB`;
  }
  render() {
    return this.loading ? l`<div class="loading">Loading artifacts\u2026</div>` : l`
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

      ${this.confirmBuild ? l`
            <div class="confirm">
              <span class="confirm-text">This will run a full build and overwrite dist/. Continue?</span>
              <button class="confirm-yes" @click=${this.handleConfirmBuild}>Build</button>
              <button class="confirm-no" @click=${this.handleCancelBuild}>Cancel</button>
            </div>
          ` : c}

      ${this.error ? l`<div class="error">${this.error}</div>` : c}
      ${this.buildSuccess ? l`<div class="success">${this.buildSuccess}</div>` : c}

      ${this.artifacts.length === 0 ? l`<div class="empty">${this.distExists ? "dist/ is empty." : "Run a build to create artifacts."}</div>` : l`
            <div class="list">
              ${this.artifacts.map(
      (s) => l`
                  <div class="artifact">
                    <span class="artifact-name">${s.name}</span>
                    <span class="artifact-size">${this.formatSize(s.size)}</span>
                  </div>
                `
    )}
            </div>
          `}
    `;
  }
};
v.styles = F`
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
      colour: #6b7280;
    }

    button.build {
      padding: 0.5rem 1.25rem;
      background: #6366f1;
      colour: #fff;
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
      colour: #92400e;
    }

    button.confirm-yes {
      padding: 0.375rem 1rem;
      background: #dc2626;
      colour: #fff;
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
      colour: #111827;
    }

    .artifact-size {
      font-size: 0.75rem;
      colour: #6b7280;
    }

    .empty {
      text-align: center;
      padding: 2rem;
      colour: #9ca3af;
      font-size: 0.875rem;
    }

    .loading {
      text-align: center;
      padding: 2rem;
      colour: #6b7280;
    }

    .error {
      colour: #dc2626;
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
      colour: #166534;
      margin-bottom: 1rem;
    }
  `;
A([
  z({ attribute: "api-url" })
], v.prototype, "apiUrl", 2);
A([
  h()
], v.prototype, "artifacts", 2);
A([
  h()
], v.prototype, "distExists", 2);
A([
  h()
], v.prototype, "loading", 2);
A([
  h()
], v.prototype, "error", 2);
A([
  h()
], v.prototype, "building", 2);
A([
  h()
], v.prototype, "confirmBuild", 2);
A([
  h()
], v.prototype, "buildSuccess", 2);
v = A([
  K("core-build-artifacts")
], v);
var Xe = Object.defineProperty, Ye = Object.getOwnPropertyDescriptor, x = (s, e, t, r) => {
  for (var i = r > 1 ? void 0 : r ? Ye(e, t) : e, o = s.length - 1, n; o >= 0; o--)
    (n = s[o]) && (i = (r ? n(e, t, i) : n(i)) || i);
  return r && i && Xe(e, t, i), i;
};
let $ = class extends w {
  constructor() {
    super(...arguments), this.apiUrl = "", this.version = "", this.changelog = "", this.loading = !0, this.error = "", this.releasing = !1, this.confirmRelease = !1, this.releaseSuccess = "";
  }
  connectedCallback() {
    super.connectedCallback(), this.api = new Y(this.apiUrl), this.reload();
  }
  async reload() {
    this.loading = !0, this.error = "";
    try {
      const [s, e] = await Promise.all([
        this.api.version(),
        this.api.changelog()
      ]);
      this.version = s.version ?? "", this.changelog = e.changelog ?? "";
    } catch (s) {
      this.error = s.message ?? "Failed to load release information";
    } finally {
      this.loading = !1;
    }
  }
  handleReleaseClick() {
    this.confirmRelease = !0, this.releaseSuccess = "";
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
  async doRelease(s) {
    var e;
    this.releasing = !0, this.error = "", this.releaseSuccess = "";
    try {
      const t = await this.api.release(s), r = s ? "Dry run complete" : "Release published";
      this.releaseSuccess = `${r} — ${t.version} (${((e = t.artifacts) == null ? void 0 : e.length) ?? 0} artifact(s))`, await this.reload();
    } catch (t) {
      this.error = t.message ?? "Release failed";
    } finally {
      this.releasing = !1;
    }
  }
  render() {
    return this.loading ? l`<div class="loading">Loading release information\u2026</div>` : l`
      ${this.error ? l`<div class="error">${this.error}</div>` : c}
      ${this.releaseSuccess ? l`<div class="success">${this.releaseSuccess}</div>` : c}

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

      ${this.confirmRelease ? l`
            <div class="confirm">
              <span class="confirm-text">This will publish ${this.version} to all configured targets. This action cannot be undone. Continue?</span>
              <button class="confirm-yes" @click=${this.handleConfirmRelease}>Publish</button>
              <button class="confirm-no" @click=${this.handleCancelRelease}>Cancel</button>
            </div>
          ` : c}

      ${this.changelog ? l`
            <div class="changelog-section">
              <div class="changelog-header">Changelog</div>
              <div class="changelog-content">${this.changelog}</div>
            </div>
          ` : l`<div class="empty">No changelog available.</div>`}
    `;
  }
};
$.styles = F`
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
      colour: #6b7280;
      text-transform: uppercase;
      letter-spacing: 0.025em;
    }

    .version-value {
      font-size: 1.25rem;
      font-weight: 700;
      font-family: monospace;
      colour: #111827;
    }

    .actions {
      display: flex;
      gap: 0.5rem;
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
      colour: #fff;
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
      colour: #6366f1;
      border: 1px solid #6366f1;
    }

    button.dry-run:hover {
      background: #eef2ff;
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
      colour: #991b1b;
    }

    button.confirm-yes {
      padding: 0.375rem 1rem;
      background: #dc2626;
      colour: #fff;
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
      colour: #6b7280;
      text-transform: uppercase;
      letter-spacing: 0.025em;
    }

    .changelog-content {
      padding: 1rem;
      font-size: 0.875rem;
      line-height: 1.6;
      white-space: pre-wrap;
      font-family: system-ui, -apple-system, sans-serif;
      colour: #374151;
      max-height: 400px;
      overflow-y: auto;
    }

    .empty {
      text-align: center;
      padding: 2rem;
      colour: #9ca3af;
      font-size: 0.875rem;
    }

    .loading {
      text-align: center;
      padding: 2rem;
      colour: #6b7280;
    }

    .error {
      colour: #dc2626;
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
      colour: #166534;
      margin-bottom: 1rem;
    }
  `;
x([
  z({ attribute: "api-url" })
], $.prototype, "apiUrl", 2);
x([
  h()
], $.prototype, "version", 2);
x([
  h()
], $.prototype, "changelog", 2);
x([
  h()
], $.prototype, "loading", 2);
x([
  h()
], $.prototype, "error", 2);
x([
  h()
], $.prototype, "releasing", 2);
x([
  h()
], $.prototype, "confirmRelease", 2);
x([
  h()
], $.prototype, "releaseSuccess", 2);
$ = x([
  K("core-build-release")
], $);
var et = Object.defineProperty, tt = Object.getOwnPropertyDescriptor, y = (s, e, t, r) => {
  for (var i = r > 1 ? void 0 : r ? tt(e, t) : e, o = s.length - 1, n; o >= 0; o--)
    (n = s[o]) && (i = (r ? n(e, t, i) : n(i)) || i);
  return r && i && et(e, t, i), i;
};
let m = class extends w {
  constructor() {
    super(...arguments), this.apiUrl = "", this.basePath = "", this.revisionPath = "", this.diffResult = null, this.diffing = !1, this.diffError = "", this.selectedLanguage = "", this.generating = !1, this.generateError = "", this.generateSuccess = "";
  }
  connectedCallback() {
    super.connectedCallback(), this.api = new Y(this.apiUrl);
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
    } catch (s) {
      this.diffError = s.message ?? "Diff failed";
    } finally {
      this.diffing = !1;
    }
  }
  async handleGenerate() {
    this.generating = !0, this.generateError = "", this.generateSuccess = "";
    try {
      const e = (await this.api.sdkGenerate(this.selectedLanguage || void 0)).language || "all languages";
      this.generateSuccess = `SDK generated successfully for ${e}.`;
    } catch (s) {
      this.generateError = s.message ?? "Generation failed";
    } finally {
      this.generating = !1;
    }
  }
  render() {
    return l`
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
              @input=${(s) => this.basePath = s.target.value}
            />
          </div>
          <div class="diff-field">
            <label>Revision spec</label>
            <input
              type="text"
              placeholder="path/to/revision.yaml"
              .value=${this.revisionPath}
              @input=${(s) => this.revisionPath = s.target.value}
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

        ${this.diffError ? l`<div class="error">${this.diffError}</div>` : c}

        ${this.diffResult ? l`
              <div class="diff-result ${this.diffResult.Breaking ? "breaking" : "safe"}">
                <div class="diff-summary">${this.diffResult.Summary}</div>
                ${this.diffResult.Changes && this.diffResult.Changes.length > 0 ? l`
                      <ul class="diff-changes">
                        ${this.diffResult.Changes.map(
      (s) => l`<li>${s}</li>`
    )}
                      </ul>
                    ` : c}
              </div>
            ` : c}
      </div>

      <!-- SDK Generation -->
      <div class="section">
        <div class="section-title">SDK Generation</div>

        ${this.generateError ? l`<div class="error">${this.generateError}</div>` : c}
        ${this.generateSuccess ? l`<div class="success">${this.generateSuccess}</div>` : c}

        <div class="generate-form">
          <select
            .value=${this.selectedLanguage}
            @change=${(s) => this.selectedLanguage = s.target.value}
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
m.styles = F`
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
      colour: #6b7280;
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
      colour: #6b7280;
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
      border-colour: #6366f1;
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
      colour: #fff;
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
      colour: #374151;
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
      colour: #991b1b;
    }

    .diff-result.safe {
      background: #f0fdf4;
      border: 1px solid #bbf7d0;
      colour: #166534;
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
      colour: #9ca3af;
      font-size: 0.875rem;
    }

    .error {
      colour: #dc2626;
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
      colour: #166534;
      margin-bottom: 1rem;
    }

    .loading {
      text-align: center;
      padding: 1rem;
      colour: #6b7280;
      font-size: 0.875rem;
    }
  `;
y([
  z({ attribute: "api-url" })
], m.prototype, "apiUrl", 2);
y([
  h()
], m.prototype, "basePath", 2);
y([
  h()
], m.prototype, "revisionPath", 2);
y([
  h()
], m.prototype, "diffResult", 2);
y([
  h()
], m.prototype, "diffing", 2);
y([
  h()
], m.prototype, "diffError", 2);
y([
  h()
], m.prototype, "selectedLanguage", 2);
y([
  h()
], m.prototype, "generating", 2);
y([
  h()
], m.prototype, "generateError", 2);
y([
  h()
], m.prototype, "generateSuccess", 2);
m = y([
  K("core-build-sdk")
], m);
var st = Object.defineProperty, it = Object.getOwnPropertyDescriptor, N = (s, e, t, r) => {
  for (var i = r > 1 ? void 0 : r ? it(e, t) : e, o = s.length - 1, n; o >= 0; o--)
    (n = s[o]) && (i = (r ? n(e, t, i) : n(i)) || i);
  return r && i && st(e, t, i), i;
};
let P = class extends w {
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
    this.ws = Ve(this.wsUrl, (s) => {
      this.lastEvent = s.channel ?? s.type ?? "", this.requestUpdate();
    }), this.ws.onopen = () => {
      this.wsConnected = !0;
    }, this.ws.onclose = () => {
      this.wsConnected = !1;
    };
  }
  handleTabClick(s) {
    this.activeTab = s;
  }
  handleRefresh() {
    var e;
    const s = (e = this.shadowRoot) == null ? void 0 : e.querySelector(".content");
    if (s) {
      const t = s.firstElementChild;
      t && "reload" in t && t.reload();
    }
  }
  renderContent() {
    switch (this.activeTab) {
      case "config":
        return l`<core-build-config api-url=${this.apiUrl}></core-build-config>`;
      case "build":
        return l`<core-build-artifacts api-url=${this.apiUrl}></core-build-artifacts>`;
      case "release":
        return l`<core-build-release api-url=${this.apiUrl}></core-build-release>`;
      case "sdk":
        return l`<core-build-sdk api-url=${this.apiUrl}></core-build-sdk>`;
      default:
        return c;
    }
  }
  render() {
    const s = this.wsUrl ? this.wsConnected ? "connected" : "disconnected" : "idle";
    return l`
      <div class="header">
        <span class="title">Build</span>
        <button class="refresh-btn" @click=${this.handleRefresh}>Refresh</button>
      </div>

      <div class="tabs">
        ${this.tabs.map(
      (e) => l`
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
          <span class="ws-dot ${s}"></span>
          <span>${s === "connected" ? "Connected" : s === "disconnected" ? "Disconnected" : "No WebSocket"}</span>
        </div>
        ${this.lastEvent ? l`<span>Last: ${this.lastEvent}</span>` : c}
      </div>
    `;
  }
};
P.styles = F`
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
      colour: #111827;
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
      colour: #6b7280;
      cursor: pointer;
      border-bottom: 2px solid transparent;
      transition: all 0.15s;
      background: none;
      border-top: none;
      border-left: none;
      border-right: none;
    }

    .tab:hover {
      colour: #374151;
    }

    .tab.active {
      colour: #6366f1;
      border-bottom-colour: #6366f1;
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
      colour: #9ca3af;
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
N([
  z({ attribute: "api-url" })
], P.prototype, "apiUrl", 2);
N([
  z({ attribute: "ws-url" })
], P.prototype, "wsUrl", 2);
N([
  h()
], P.prototype, "activeTab", 2);
N([
  h()
], P.prototype, "wsConnected", 2);
N([
  h()
], P.prototype, "lastEvent", 2);
P = N([
  K("core-build-panel")
], P);
export {
  Y as BuildApi,
  v as BuildArtifacts,
  C as BuildConfig,
  P as BuildPanel,
  $ as BuildRelease,
  m as BuildSdk,
  Ve as connectBuildEvents
};
