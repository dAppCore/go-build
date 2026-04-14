/**
 * @license
 * Copyright 2019 Google LLC
 * SPDX-License-Identifier: BSD-3-Clause
 */
const J = globalThis, re = J.ShadowRoot && (J.ShadyCSS === void 0 || J.ShadyCSS.nativeShadow) && "adoptedStyleSheets" in Document.prototype && "replace" in CSSStyleSheet.prototype, oe = Symbol(), de = /* @__PURE__ */ new WeakMap();
let we = class {
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
const Se = (s) => new we(typeof s == "string" ? s : s + "", void 0, oe), F = (s, ...e) => {
  const t = s.length === 1 ? s[0] : e.reduce((r, i, o) => r + ((a) => {
    if (a._$cssResult$ === !0) return a.cssText;
    if (typeof a == "number") return a;
    throw Error("Value passed to 'css' function must be a 'css' function result: " + a + ". Use 'unsafeCSS' to pass non-literal values, but take care to ensure page security.");
  })(i) + s[o + 1], s[0]);
  return new we(t, s, oe);
}, Pe = (s, e) => {
  if (re) s.adoptedStyleSheets = e.map((t) => t instanceof CSSStyleSheet ? t : t.styleSheet);
  else for (const t of e) {
    const r = document.createElement("style"), i = J.litNonce;
    i !== void 0 && r.setAttribute("nonce", i), r.textContent = t.cssText, s.appendChild(r);
  }
}, ce = re ? (s) => s : (s) => s instanceof CSSStyleSheet ? ((e) => {
  let t = "";
  for (const r of e.cssRules) t += r.cssText;
  return Se(t);
})(s) : s;
/**
 * @license
 * Copyright 2017 Google LLC
 * SPDX-License-Identifier: BSD-3-Clause
 */
const { is: Ee, defineProperty: Ce, getOwnPropertyDescriptor: Oe, getOwnPropertyNames: Re, getOwnPropertySymbols: Ue, getPrototypeOf: ze } = Object, S = globalThis, fe = S.trustedTypes, De = fe ? fe.emptyScript : "", ee = S.reactiveElementPolyfillSupport, W = (s, e) => s, Z = { toAttribute(s, e) {
  switch (e) {
    case Boolean:
      s = s ? De : null;
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
} }, ae = (s, e) => !Ee(s, e), he = { attribute: !0, type: String, converter: Z, reflect: !1, useDefault: !1, hasChanged: ae };
Symbol.metadata ?? (Symbol.metadata = Symbol("metadata")), S.litPropertyMetadata ?? (S.litPropertyMetadata = /* @__PURE__ */ new WeakMap());
let T = class extends HTMLElement {
  static addInitializer(e) {
    this._$Ei(), (this.l ?? (this.l = [])).push(e);
  }
  static get observedAttributes() {
    return this.finalize(), this._$Eh && [...this._$Eh.keys()];
  }
  static createProperty(e, t = he) {
    if (t.state && (t.attribute = !1), this._$Ei(), this.prototype.hasOwnProperty(e) && ((t = Object.create(t)).wrapped = !0), this.elementProperties.set(e, t), !t.noAccessor) {
      const r = Symbol(), i = this.getPropertyDescriptor(e, r, t);
      i !== void 0 && Ce(this.prototype, e, i);
    }
  }
  static getPropertyDescriptor(e, t, r) {
    const { get: i, set: o } = Oe(this.prototype, e) ?? { get() {
      return this[t];
    }, set(a) {
      this[t] = a;
    } };
    return { get: i, set(a) {
      const d = i == null ? void 0 : i.call(this);
      o == null || o.call(this, a), this.requestUpdate(e, d, r);
    }, configurable: !0, enumerable: !0 };
  }
  static getPropertyOptions(e) {
    return this.elementProperties.get(e) ?? he;
  }
  static _$Ei() {
    if (this.hasOwnProperty(W("elementProperties"))) return;
    const e = ze(this);
    e.finalize(), e.l !== void 0 && (this.l = [...e.l]), this.elementProperties = new Map(e.elementProperties);
  }
  static finalize() {
    if (this.hasOwnProperty(W("finalized"))) return;
    if (this.finalized = !0, this._$Ei(), this.hasOwnProperty(W("properties"))) {
      const t = this.properties, r = [...Re(t), ...Ue(t)];
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
    return Pe(e, this.constructor.elementStyles), e;
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
      const a = (((o = r.converter) == null ? void 0 : o.toAttribute) !== void 0 ? r.converter : Z).toAttribute(t, r.type);
      this._$Em = e, a == null ? this.removeAttribute(i) : this.setAttribute(i, a), this._$Em = null;
    }
  }
  _$AK(e, t) {
    var o, a;
    const r = this.constructor, i = r._$Eh.get(e);
    if (i !== void 0 && this._$Em !== i) {
      const d = r.getPropertyOptions(i), n = typeof d.converter == "function" ? { fromAttribute: d.converter } : ((o = d.converter) == null ? void 0 : o.fromAttribute) !== void 0 ? d.converter : Z;
      this._$Em = i;
      const u = n.fromAttribute(t, d.type);
      this[i] = u ?? ((a = this._$Ej) == null ? void 0 : a.get(i)) ?? u, this._$Em = null;
    }
  }
  requestUpdate(e, t, r, i = !1, o) {
    var a;
    if (e !== void 0) {
      const d = this.constructor;
      if (i === !1 && (o = this[e]), r ?? (r = d.getPropertyOptions(e)), !((r.hasChanged ?? ae)(o, t) || r.useDefault && r.reflect && o === ((a = this._$Ej) == null ? void 0 : a.get(e)) && !this.hasAttribute(d._$Eu(e, r)))) return;
      this.C(e, t, r);
    }
    this.isUpdatePending === !1 && (this._$ES = this._$EP());
  }
  C(e, t, { useDefault: r, reflect: i, wrapped: o }, a) {
    r && !(this._$Ej ?? (this._$Ej = /* @__PURE__ */ new Map())).has(e) && (this._$Ej.set(e, a ?? t ?? this[e]), o !== !0 || a !== void 0) || (this._$AL.has(e) || (this.hasUpdated || r || (t = void 0), this._$AL.set(e, t)), i === !0 && this._$Em !== e && (this._$Eq ?? (this._$Eq = /* @__PURE__ */ new Set())).add(e));
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
        for (const [o, a] of this._$Ep) this[o] = a;
        this._$Ep = void 0;
      }
      const i = this.constructor.elementProperties;
      if (i.size > 0) for (const [o, a] of i) {
        const { wrapped: d } = a, n = this[o];
        d !== !0 || this._$AL.has(o) || n === void 0 || this.C(o, void 0, a, n);
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
T.elementStyles = [], T.shadowRootOptions = { mode: "open" }, T[W("elementProperties")] = /* @__PURE__ */ new Map(), T[W("finalized")] = /* @__PURE__ */ new Map(), ee == null || ee({ ReactiveElement: T }), (S.reactiveElementVersions ?? (S.reactiveElementVersions = [])).push("2.1.2");
/**
 * @license
 * Copyright 2017 Google LLC
 * SPDX-License-Identifier: BSD-3-Clause
 */
const L = globalThis, ue = (s) => s, Q = L.trustedTypes, pe = Q ? Q.createPolicy("lit-html", { createHTML: (s) => s }) : void 0, _e = "$lit$", A = `lit$${Math.random().toFixed(9).slice(2)}$`, ke = "?" + A, Te = `<${ke}>`, z = document, q = () => z.createComment(""), I = (s) => s === null || typeof s != "object" && typeof s != "function", ne = Array.isArray, je = (s) => ne(s) || typeof (s == null ? void 0 : s[Symbol.iterator]) == "function", te = `[ 	
\f\r]`, M = /<(?:(!--|\/[^a-zA-Z])|(\/?[a-zA-Z][^>\s]*)|(\/?$))/g, ge = /-->/g, be = />/g, O = RegExp(`>|${te}(?:([^\\s"'>=/]+)(${te}*=${te}*(?:[^ 	
\f\r"'\`<>=]|("|')|))|$)`, "g"), me = /'/g, ve = /"/g, xe = /^(?:script|style|textarea|title)$/i, Be = (s) => (e, ...t) => ({ _$litType$: s, strings: e, values: t }), l = Be(1), j = Symbol.for("lit-noChange"), c = Symbol.for("lit-nothing"), $e = /* @__PURE__ */ new WeakMap(), R = z.createTreeWalker(z, 129);
function Ae(s, e) {
  if (!ne(s) || !s.hasOwnProperty("raw")) throw Error("invalid template strings array");
  return pe !== void 0 ? pe.createHTML(e) : e;
}
const Ne = (s, e) => {
  const t = s.length - 1, r = [];
  let i, o = e === 2 ? "<svg>" : e === 3 ? "<math>" : "", a = M;
  for (let d = 0; d < t; d++) {
    const n = s[d];
    let u, p, h = -1, b = 0;
    for (; b < n.length && (a.lastIndex = b, p = a.exec(n), p !== null); ) b = a.lastIndex, a === M ? p[1] === "!--" ? a = ge : p[1] !== void 0 ? a = be : p[2] !== void 0 ? (xe.test(p[2]) && (i = RegExp("</" + p[2], "g")), a = O) : p[3] !== void 0 && (a = O) : a === O ? p[0] === ">" ? (a = i ?? M, h = -1) : p[1] === void 0 ? h = -2 : (h = a.lastIndex - p[2].length, u = p[1], a = p[3] === void 0 ? O : p[3] === '"' ? ve : me) : a === ve || a === me ? a = O : a === ge || a === be ? a = M : (a = O, i = void 0);
    const $ = a === O && s[d + 1].startsWith("/>") ? " " : "";
    o += a === M ? n + Te : h >= 0 ? (r.push(u), n.slice(0, h) + _e + n.slice(h) + A + $) : n + A + (h === -2 ? d : $);
  }
  return [Ae(s, o + (s[t] || "<?>") + (e === 2 ? "</svg>" : e === 3 ? "</math>" : "")), r];
};
class G {
  constructor({ strings: e, _$litType$: t }, r) {
    let i;
    this.parts = [];
    let o = 0, a = 0;
    const d = e.length - 1, n = this.parts, [u, p] = Ne(e, t);
    if (this.el = G.createElement(u, r), R.currentNode = this.el.content, t === 2 || t === 3) {
      const h = this.el.content.firstChild;
      h.replaceWith(...h.childNodes);
    }
    for (; (i = R.nextNode()) !== null && n.length < d; ) {
      if (i.nodeType === 1) {
        if (i.hasAttributes()) for (const h of i.getAttributeNames()) if (h.endsWith(_e)) {
          const b = p[a++], $ = i.getAttribute(h).split(A), _ = /([.?@])?(.*)/.exec(b);
          n.push({ type: 1, index: o, name: _[2], strings: $, ctor: _[1] === "." ? Me : _[1] === "?" ? We : _[1] === "@" ? Le : X }), i.removeAttribute(h);
        } else h.startsWith(A) && (n.push({ type: 6, index: o }), i.removeAttribute(h));
        if (xe.test(i.tagName)) {
          const h = i.textContent.split(A), b = h.length - 1;
          if (b > 0) {
            i.textContent = Q ? Q.emptyScript : "";
            for (let $ = 0; $ < b; $++) i.append(h[$], q()), R.nextNode(), n.push({ type: 2, index: ++o });
            i.append(h[b], q());
          }
        }
      } else if (i.nodeType === 8) if (i.data === ke) n.push({ type: 2, index: o });
      else {
        let h = -1;
        for (; (h = i.data.indexOf(A, h + 1)) !== -1; ) n.push({ type: 7, index: o }), h += A.length - 1;
      }
      o++;
    }
  }
  static createElement(e, t) {
    const r = z.createElement("template");
    return r.innerHTML = e, r;
  }
}
function B(s, e, t = s, r) {
  var a, d;
  if (e === j) return e;
  let i = r !== void 0 ? (a = t._$Co) == null ? void 0 : a[r] : t._$Cl;
  const o = I(e) ? void 0 : e._$litDirective$;
  return (i == null ? void 0 : i.constructor) !== o && ((d = i == null ? void 0 : i._$AO) == null || d.call(i, !1), o === void 0 ? i = void 0 : (i = new o(s), i._$AT(s, t, r)), r !== void 0 ? (t._$Co ?? (t._$Co = []))[r] = i : t._$Cl = i), i !== void 0 && (e = B(s, i._$AS(s, e.values), i, r)), e;
}
class He {
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
    const { el: { content: t }, parts: r } = this._$AD, i = ((e == null ? void 0 : e.creationScope) ?? z).importNode(t, !0);
    R.currentNode = i;
    let o = R.nextNode(), a = 0, d = 0, n = r[0];
    for (; n !== void 0; ) {
      if (a === n.index) {
        let u;
        n.type === 2 ? u = new V(o, o.nextSibling, this, e) : n.type === 1 ? u = new n.ctor(o, n.name, n.strings, this, e) : n.type === 6 && (u = new qe(o, this, e)), this._$AV.push(u), n = r[++d];
      }
      a !== (n == null ? void 0 : n.index) && (o = R.nextNode(), a++);
    }
    return R.currentNode = z, i;
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
    e = B(this, e, t), I(e) ? e === c || e == null || e === "" ? (this._$AH !== c && this._$AR(), this._$AH = c) : e !== this._$AH && e !== j && this._(e) : e._$litType$ !== void 0 ? this.$(e) : e.nodeType !== void 0 ? this.T(e) : je(e) ? this.k(e) : this._(e);
  }
  O(e) {
    return this._$AA.parentNode.insertBefore(e, this._$AB);
  }
  T(e) {
    this._$AH !== e && (this._$AR(), this._$AH = this.O(e));
  }
  _(e) {
    this._$AH !== c && I(this._$AH) ? this._$AA.nextSibling.data = e : this.T(z.createTextNode(e)), this._$AH = e;
  }
  $(e) {
    var o;
    const { values: t, _$litType$: r } = e, i = typeof r == "number" ? this._$AC(e) : (r.el === void 0 && (r.el = G.createElement(Ae(r.h, r.h[0]), this.options)), r);
    if (((o = this._$AH) == null ? void 0 : o._$AD) === i) this._$AH.p(t);
    else {
      const a = new He(i, this), d = a.u(this.options);
      a.p(t), this.T(d), this._$AH = a;
    }
  }
  _$AC(e) {
    let t = $e.get(e.strings);
    return t === void 0 && $e.set(e.strings, t = new G(e)), t;
  }
  k(e) {
    ne(this._$AH) || (this._$AH = [], this._$AR());
    const t = this._$AH;
    let r, i = 0;
    for (const o of e) i === t.length ? t.push(r = new V(this.O(q()), this.O(q()), this, this.options)) : r = t[i], r._$AI(o), i++;
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
    let a = !1;
    if (o === void 0) e = B(this, e, t, 0), a = !I(e) || e !== this._$AH && e !== j, a && (this._$AH = e);
    else {
      const d = e;
      let n, u;
      for (e = o[0], n = 0; n < o.length - 1; n++) u = B(this, d[r + n], t, n), u === j && (u = this._$AH[n]), a || (a = !I(u) || u !== this._$AH[n]), u === c ? e = c : e !== c && (e += (u ?? "") + o[n + 1]), this._$AH[n] = u;
    }
    a && !i && this.j(e);
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
class We extends X {
  constructor() {
    super(...arguments), this.type = 4;
  }
  j(e) {
    this.element.toggleAttribute(this.name, !!e && e !== c);
  }
}
class Le extends X {
  constructor(e, t, r, i, o) {
    super(e, t, r, i, o), this.type = 5;
  }
  _$AI(e, t = this) {
    if ((e = B(this, e, t, 0) ?? c) === j) return;
    const r = this._$AH, i = e === c && r !== c || e.capture !== r.capture || e.once !== r.once || e.passive !== r.passive, o = e !== c && (r === c || i);
    i && this.element.removeEventListener(this.name, this, r), o && this.element.addEventListener(this.name, this, e), this._$AH = e;
  }
  handleEvent(e) {
    var t;
    typeof this._$AH == "function" ? this._$AH.call(((t = this.options) == null ? void 0 : t.host) ?? this.element, e) : this._$AH.handleEvent(e);
  }
}
class qe {
  constructor(e, t, r) {
    this.element = e, this.type = 6, this._$AN = void 0, this._$AM = t, this.options = r;
  }
  get _$AU() {
    return this._$AM._$AU;
  }
  _$AI(e) {
    B(this, e);
  }
}
const se = L.litHtmlPolyfillSupport;
se == null || se(G, V), (L.litHtmlVersions ?? (L.litHtmlVersions = [])).push("3.3.2");
const Ie = (s, e, t) => {
  const r = (t == null ? void 0 : t.renderBefore) ?? e;
  let i = r._$litPart$;
  if (i === void 0) {
    const o = (t == null ? void 0 : t.renderBefore) ?? null;
    r._$litPart$ = i = new V(e.insertBefore(q(), o), o, void 0, t ?? {});
  }
  return i._$AI(s), i;
};
/**
 * @license
 * Copyright 2017 Google LLC
 * SPDX-License-Identifier: BSD-3-Clause
 */
const U = globalThis;
class k extends T {
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
    return j;
  }
}
var ye;
k._$litElement$ = !0, k.finalized = !0, (ye = U.litElementHydrateSupport) == null || ye.call(U, { LitElement: k });
const ie = U.litElementPolyfillSupport;
ie == null || ie({ LitElement: k });
(U.litElementVersions ?? (U.litElementVersions = [])).push("4.2.2");
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
const Ge = { attribute: !0, type: String, converter: Z, reflect: !1, hasChanged: ae }, Fe = (s = Ge, e, t) => {
  const { kind: r, metadata: i } = t;
  let o = globalThis.litPropertyMetadata.get(i);
  if (o === void 0 && globalThis.litPropertyMetadata.set(i, o = /* @__PURE__ */ new Map()), r === "setter" && ((s = Object.create(s)).wrapped = !0), o.set(t.name, s), r === "accessor") {
    const { name: a } = t;
    return { set(d) {
      const n = e.get.call(this);
      e.set.call(this, d), this.requestUpdate(a, n, s, !0, d);
    }, init(d) {
      return d !== void 0 && this.C(a, void 0, s, d), d;
    } };
  }
  if (r === "setter") {
    const { name: a } = t;
    return function(d) {
      const n = this[a];
      e.call(this, d), this.requestUpdate(a, n, s, !0, d);
    };
  }
  throw Error("Unsupported decorator location: " + r);
};
function D(s) {
  return (e, t) => typeof t == "object" ? Fe(s, e, t) : ((r, i, o) => {
    const a = i.hasOwnProperty(o);
    return i.constructor.createProperty(o, r), a ? Object.getOwnPropertyDescriptor(i, o) : void 0;
  })(s, e, t);
}
/**
 * @license
 * Copyright 2017 Google LLC
 * SPDX-License-Identifier: BSD-3-Clause
 */
function f(s) {
  return D({ ...s, state: !0, attribute: !1 });
}
function Ve(s, e) {
  const t = new WebSocket(s);
  return t.onmessage = (r) => {
    var i, o, a, d, n, u, p, h, b, $, _, le;
    try {
      const C = JSON.parse(r.data);
      ((o = (i = C.type) == null ? void 0 : i.startsWith) != null && o.call(i, "build.") || (d = (a = C.type) == null ? void 0 : a.startsWith) != null && d.call(a, "release.") || (u = (n = C.type) == null ? void 0 : n.startsWith) != null && u.call(n, "sdk.") || (h = (p = C.channel) == null ? void 0 : p.startsWith) != null && h.call(p, "build.") || ($ = (b = C.channel) == null ? void 0 : b.startsWith) != null && $.call(b, "release.") || (le = (_ = C.channel) == null ? void 0 : _.startsWith) != null && le.call(_, "sdk.")) && e(C);
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
  releaseWorkflow(e = {}) {
    return this.request("/release/workflow", {
      method: "POST",
      headers: { "Content-Type": "application/json" },
      body: JSON.stringify(e)
    });
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
var Ke = Object.defineProperty, Je = Object.getOwnPropertyDescriptor, N = (s, e, t, r) => {
  for (var i = r > 1 ? void 0 : r ? Je(e, t) : e, o = s.length - 1, a; o >= 0; o--)
    (a = s[o]) && (i = (r ? a(e, t, i) : a(i)) || i);
  return r && i && Ke(e, t, i), i;
};
let P = class extends k {
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
              <div class="field">
                <span class="field-label">Suggested stack</span>
                <span class="field-value">${e.suggested_stack || e.primary_stack || e.primary || "none"}</span>
              </div>
              ${e.types.length > 1 ? l`
                    <div class="field">
                      <span class="field-label">Detected types</span>
                      <span class="field-value">${e.types.join(", ")}</span>
                    </div>
                  ` : c}
              <div class="field">
                <span class="field-label">Frontend</span>
                <span class="badge ${e.has_frontend ? "present" : "absent"}">
                  ${e.has_frontend ? "Detected" : "None"}
                </span>
              </div>
              <div class="field">
                <span class="field-label">Nested frontend</span>
                <span class="badge ${e.has_subtree_npm ? "present" : "absent"}">
                  ${e.has_subtree_npm ? "Depth 2" : "None"}
                </span>
              </div>
              ${e.linux_packages && e.linux_packages.length > 0 ? l`
                    <div class="field">
                      <span class="field-label">Linux packages</span>
                      <div class="flags">
                        ${e.linux_packages.map((t) => l`<span class="flag">${t}</span>`)}
                      </div>
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
P.styles = F`
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
N([
  D({ attribute: "api-url" })
], P.prototype, "apiUrl", 2);
N([
  f()
], P.prototype, "configData", 2);
N([
  f()
], P.prototype, "discoverData", 2);
N([
  f()
], P.prototype, "loading", 2);
N([
  f()
], P.prototype, "error", 2);
P = N([
  K("core-build-config")
], P);
var Ze = Object.defineProperty, Qe = Object.getOwnPropertyDescriptor, x = (s, e, t, r) => {
  for (var i = r > 1 ? void 0 : r ? Qe(e, t) : e, o = s.length - 1, a; o >= 0; o--)
    (a = s[o]) && (i = (r ? a(e, t, i) : a(i)) || i);
  return r && i && Ze(e, t, i), i;
};
let y = class extends k {
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
y.styles = F`
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
x([
  D({ attribute: "api-url" })
], y.prototype, "apiUrl", 2);
x([
  f()
], y.prototype, "artifacts", 2);
x([
  f()
], y.prototype, "distExists", 2);
x([
  f()
], y.prototype, "loading", 2);
x([
  f()
], y.prototype, "error", 2);
x([
  f()
], y.prototype, "building", 2);
x([
  f()
], y.prototype, "confirmBuild", 2);
x([
  f()
], y.prototype, "buildSuccess", 2);
y = x([
  K("core-build-artifacts")
], y);
var Xe = Object.defineProperty, Ye = Object.getOwnPropertyDescriptor, v = (s, e, t, r) => {
  for (var i = r > 1 ? void 0 : r ? Ye(e, t) : e, o = s.length - 1, a; o >= 0; o--)
    (a = s[o]) && (i = (r ? a(e, t, i) : a(i)) || i);
  return r && i && Xe(e, t, i), i;
};
let g = class extends k {
  constructor() {
    super(...arguments), this.apiUrl = "", this.version = "", this.changelog = "", this.loading = !0, this.error = "", this.releasing = !1, this.confirmRelease = !1, this.releaseSuccess = "", this.workflowPath = ".github/workflows/release.yml", this.workflowOutputPath = "", this.generatingWorkflow = !1, this.workflowSuccess = "";
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
  handleWorkflowPathInput(s) {
    const e = s.target;
    this.workflowPath = (e == null ? void 0 : e.value) ?? "";
  }
  handleWorkflowOutputPathInput(s) {
    const e = s.target;
    this.workflowOutputPath = (e == null ? void 0 : e.value) ?? "";
  }
  async handleGenerateWorkflow() {
    this.generatingWorkflow = !0, this.error = "", this.workflowSuccess = "";
    try {
      const s = {}, e = this.workflowPath.trim(), t = this.workflowOutputPath.trim();
      e && (s.path = e), e && (s.workflowPath = e, s.workflow_path = e, s["workflow-path"] = e), t && (s.outputPath = t), t && (s["output-path"] = t, s.output_path = t, s.output = t, s.workflowOutputPath = t, s.workflow_output = t, s["workflow-output"] = t, s.workflow_output_path = t, s["workflow-output-path"] = t);
      const i = (await this.api.releaseWorkflow(s)).path ?? t ?? e ?? ".github/workflows/release.yml";
      this.workflowSuccess = `Workflow generated at ${i}`;
    } catch (s) {
      this.error = s.message ?? "Failed to generate release workflow";
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
      ${this.workflowSuccess ? l`<div class="success">${this.workflowSuccess}</div>` : c}

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
g.styles = F`
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
v([
  D({ attribute: "api-url" })
], g.prototype, "apiUrl", 2);
v([
  f()
], g.prototype, "version", 2);
v([
  f()
], g.prototype, "changelog", 2);
v([
  f()
], g.prototype, "loading", 2);
v([
  f()
], g.prototype, "error", 2);
v([
  f()
], g.prototype, "releasing", 2);
v([
  f()
], g.prototype, "confirmRelease", 2);
v([
  f()
], g.prototype, "releaseSuccess", 2);
v([
  f()
], g.prototype, "workflowPath", 2);
v([
  f()
], g.prototype, "workflowOutputPath", 2);
v([
  f()
], g.prototype, "generatingWorkflow", 2);
v([
  f()
], g.prototype, "workflowSuccess", 2);
g = v([
  K("core-build-release")
], g);
var et = Object.defineProperty, tt = Object.getOwnPropertyDescriptor, w = (s, e, t, r) => {
  for (var i = r > 1 ? void 0 : r ? tt(e, t) : e, o = s.length - 1, a; o >= 0; o--)
    (a = s[o]) && (i = (r ? a(e, t, i) : a(i)) || i);
  return r && i && et(e, t, i), i;
};
let m = class extends k {
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
w([
  D({ attribute: "api-url" })
], m.prototype, "apiUrl", 2);
w([
  f()
], m.prototype, "basePath", 2);
w([
  f()
], m.prototype, "revisionPath", 2);
w([
  f()
], m.prototype, "diffResult", 2);
w([
  f()
], m.prototype, "diffing", 2);
w([
  f()
], m.prototype, "diffError", 2);
w([
  f()
], m.prototype, "selectedLanguage", 2);
w([
  f()
], m.prototype, "generating", 2);
w([
  f()
], m.prototype, "generateError", 2);
w([
  f()
], m.prototype, "generateSuccess", 2);
m = w([
  K("core-build-sdk")
], m);
var st = Object.defineProperty, it = Object.getOwnPropertyDescriptor, H = (s, e, t, r) => {
  for (var i = r > 1 ? void 0 : r ? it(e, t) : e, o = s.length - 1, a; o >= 0; o--)
    (a = s[o]) && (i = (r ? a(e, t, i) : a(i)) || i);
  return r && i && st(e, t, i), i;
};
let E = class extends k {
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
E.styles = F`
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
H([
  D({ attribute: "api-url" })
], E.prototype, "apiUrl", 2);
H([
  D({ attribute: "ws-url" })
], E.prototype, "wsUrl", 2);
H([
  f()
], E.prototype, "activeTab", 2);
H([
  f()
], E.prototype, "wsConnected", 2);
H([
  f()
], E.prototype, "lastEvent", 2);
E = H([
  K("core-build-panel")
], E);
export {
  Y as BuildApi,
  y as BuildArtifacts,
  P as BuildConfig,
  E as BuildPanel,
  g as BuildRelease,
  m as BuildSdk,
  Ve as connectBuildEvents
};
