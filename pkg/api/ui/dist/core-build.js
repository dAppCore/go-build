/**
 * @license
 * Copyright 2019 Google LLC
 * SPDX-License-Identifier: BSD-3-Clause
 */
const Z = globalThis, ae = Z.ShadowRoot && (Z.ShadyCSS === void 0 || Z.ShadyCSS.nativeShadow) && "adoptedStyleSheets" in Document.prototype && "replace" in CSSStyleSheet.prototype, oe = Symbol(), de = /* @__PURE__ */ new WeakMap();
let we = class {
  constructor(e, s, r) {
    if (this._$cssResult$ = !0, r !== oe) throw Error("CSSResult is not constructable. Use `unsafeCSS` or `css` instead.");
    this.cssText = e, this.t = s;
  }
  get styleSheet() {
    let e = this.o;
    const s = this.t;
    if (ae && e === void 0) {
      const r = s !== void 0 && s.length === 1;
      r && (e = de.get(s)), e === void 0 && ((this.o = e = new CSSStyleSheet()).replaceSync(this.cssText), r && de.set(s, e));
    }
    return e;
  }
  toString() {
    return this.cssText;
  }
};
const Pe = (t) => new we(typeof t == "string" ? t : t + "", void 0, oe), V = (t, ...e) => {
  const s = t.length === 1 ? t[0] : e.reduce((r, i, a) => r + ((o) => {
    if (o._$cssResult$ === !0) return o.cssText;
    if (typeof o == "number") return o;
    throw Error("Value passed to 'css' function must be a 'css' function result: " + o + ". Use 'unsafeCSS' to pass non-literal values, but take care to ensure page security.");
  })(i) + t[a + 1], t[0]);
  return new we(s, t, oe);
}, Ce = (t, e) => {
  if (ae) t.adoptedStyleSheets = e.map((s) => s instanceof CSSStyleSheet ? s : s.styleSheet);
  else for (const s of e) {
    const r = document.createElement("style"), i = Z.litNonce;
    i !== void 0 && r.setAttribute("nonce", i), r.textContent = s.cssText, t.appendChild(r);
  }
}, ce = ae ? (t) => t : (t) => t instanceof CSSStyleSheet ? ((e) => {
  let s = "";
  for (const r of e.cssRules) s += r.cssText;
  return Pe(s);
})(t) : t;
/**
 * @license
 * Copyright 2017 Google LLC
 * SPDX-License-Identifier: BSD-3-Clause
 */
const { is: Ee, defineProperty: Oe, getOwnPropertyDescriptor: De, getOwnPropertyNames: Re, getOwnPropertySymbols: Ue, getPrototypeOf: Te } = Object, C = globalThis, pe = C.trustedTypes, ze = pe ? pe.emptyScript : "", te = C.reactiveElementPolyfillSupport, L = (t, e) => t, X = { toAttribute(t, e) {
  switch (e) {
    case Boolean:
      t = t ? ze : null;
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
} }, ne = (t, e) => !Ee(t, e), fe = { attribute: !0, type: String, converter: X, reflect: !1, useDefault: !1, hasChanged: ne };
Symbol.metadata ?? (Symbol.metadata = Symbol("metadata")), C.litPropertyMetadata ?? (C.litPropertyMetadata = /* @__PURE__ */ new WeakMap());
let j = class extends HTMLElement {
  static addInitializer(e) {
    this._$Ei(), (this.l ?? (this.l = [])).push(e);
  }
  static get observedAttributes() {
    return this.finalize(), this._$Eh && [...this._$Eh.keys()];
  }
  static createProperty(e, s = fe) {
    if (s.state && (s.attribute = !1), this._$Ei(), this.prototype.hasOwnProperty(e) && ((s = Object.create(s)).wrapped = !0), this.elementProperties.set(e, s), !s.noAccessor) {
      const r = Symbol(), i = this.getPropertyDescriptor(e, r, s);
      i !== void 0 && Oe(this.prototype, e, i);
    }
  }
  static getPropertyDescriptor(e, s, r) {
    const { get: i, set: a } = De(this.prototype, e) ?? { get() {
      return this[s];
    }, set(o) {
      this[s] = o;
    } };
    return { get: i, set(o) {
      const c = i == null ? void 0 : i.call(this);
      a == null || a.call(this, o), this.requestUpdate(e, c, r);
    }, configurable: !0, enumerable: !0 };
  }
  static getPropertyOptions(e) {
    return this.elementProperties.get(e) ?? fe;
  }
  static _$Ei() {
    if (this.hasOwnProperty(L("elementProperties"))) return;
    const e = Te(this);
    e.finalize(), e.l !== void 0 && (this.l = [...e.l]), this.elementProperties = new Map(e.elementProperties);
  }
  static finalize() {
    if (this.hasOwnProperty(L("finalized"))) return;
    if (this.finalized = !0, this._$Ei(), this.hasOwnProperty(L("properties"))) {
      const s = this.properties, r = [...Re(s), ...Ue(s)];
      for (const i of r) this.createProperty(i, s[i]);
    }
    const e = this[Symbol.metadata];
    if (e !== null) {
      const s = litPropertyMetadata.get(e);
      if (s !== void 0) for (const [r, i] of s) this.elementProperties.set(r, i);
    }
    this._$Eh = /* @__PURE__ */ new Map();
    for (const [s, r] of this.elementProperties) {
      const i = this._$Eu(s, r);
      i !== void 0 && this._$Eh.set(i, s);
    }
    this.elementStyles = this.finalizeStyles(this.styles);
  }
  static finalizeStyles(e) {
    const s = [];
    if (Array.isArray(e)) {
      const r = new Set(e.flat(1 / 0).reverse());
      for (const i of r) s.unshift(ce(i));
    } else e !== void 0 && s.push(ce(e));
    return s;
  }
  static _$Eu(e, s) {
    const r = s.attribute;
    return r === !1 ? void 0 : typeof r == "string" ? r : typeof e == "string" ? e.toLowerCase() : void 0;
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
    for (const r of s.keys()) this.hasOwnProperty(r) && (e.set(r, this[r]), delete this[r]);
    e.size > 0 && (this._$Ep = e);
  }
  createRenderRoot() {
    const e = this.shadowRoot ?? this.attachShadow(this.constructor.shadowRootOptions);
    return Ce(e, this.constructor.elementStyles), e;
  }
  connectedCallback() {
    var e;
    this.renderRoot ?? (this.renderRoot = this.createRenderRoot()), this.enableUpdating(!0), (e = this._$EO) == null || e.forEach((s) => {
      var r;
      return (r = s.hostConnected) == null ? void 0 : r.call(s);
    });
  }
  enableUpdating(e) {
  }
  disconnectedCallback() {
    var e;
    (e = this._$EO) == null || e.forEach((s) => {
      var r;
      return (r = s.hostDisconnected) == null ? void 0 : r.call(s);
    });
  }
  attributeChangedCallback(e, s, r) {
    this._$AK(e, r);
  }
  _$ET(e, s) {
    var a;
    const r = this.constructor.elementProperties.get(e), i = this.constructor._$Eu(e, r);
    if (i !== void 0 && r.reflect === !0) {
      const o = (((a = r.converter) == null ? void 0 : a.toAttribute) !== void 0 ? r.converter : X).toAttribute(s, r.type);
      this._$Em = e, o == null ? this.removeAttribute(i) : this.setAttribute(i, o), this._$Em = null;
    }
  }
  _$AK(e, s) {
    var a, o;
    const r = this.constructor, i = r._$Eh.get(e);
    if (i !== void 0 && this._$Em !== i) {
      const c = r.getPropertyOptions(i), d = typeof c.converter == "function" ? { fromAttribute: c.converter } : ((a = c.converter) == null ? void 0 : a.fromAttribute) !== void 0 ? c.converter : X;
      this._$Em = i;
      const h = d.fromAttribute(s, c.type);
      this[i] = h ?? ((o = this._$Ej) == null ? void 0 : o.get(i)) ?? h, this._$Em = null;
    }
  }
  requestUpdate(e, s, r, i = !1, a) {
    var o;
    if (e !== void 0) {
      const c = this.constructor;
      if (i === !1 && (a = this[e]), r ?? (r = c.getPropertyOptions(e)), !((r.hasChanged ?? ne)(a, s) || r.useDefault && r.reflect && a === ((o = this._$Ej) == null ? void 0 : o.get(e)) && !this.hasAttribute(c._$Eu(e, r)))) return;
      this.C(e, s, r);
    }
    this.isUpdatePending === !1 && (this._$ES = this._$EP());
  }
  C(e, s, { useDefault: r, reflect: i, wrapped: a }, o) {
    r && !(this._$Ej ?? (this._$Ej = /* @__PURE__ */ new Map())).has(e) && (this._$Ej.set(e, o ?? s ?? this[e]), a !== !0 || o !== void 0) || (this._$AL.has(e) || (this.hasUpdated || r || (s = void 0), this._$AL.set(e, s)), i === !0 && this._$Em !== e && (this._$Eq ?? (this._$Eq = /* @__PURE__ */ new Set())).add(e));
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
    var r;
    if (!this.isUpdatePending) return;
    if (!this.hasUpdated) {
      if (this.renderRoot ?? (this.renderRoot = this.createRenderRoot()), this._$Ep) {
        for (const [a, o] of this._$Ep) this[a] = o;
        this._$Ep = void 0;
      }
      const i = this.constructor.elementProperties;
      if (i.size > 0) for (const [a, o] of i) {
        const { wrapped: c } = o, d = this[a];
        c !== !0 || this._$AL.has(a) || d === void 0 || this.C(a, void 0, o, d);
      }
    }
    let e = !1;
    const s = this._$AL;
    try {
      e = this.shouldUpdate(s), e ? (this.willUpdate(s), (r = this._$EO) == null || r.forEach((i) => {
        var a;
        return (a = i.hostUpdate) == null ? void 0 : a.call(i);
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
    (s = this._$EO) == null || s.forEach((r) => {
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
    this._$Eq && (this._$Eq = this._$Eq.forEach((s) => this._$ET(s, this[s]))), this._$EM();
  }
  updated(e) {
  }
  firstUpdated(e) {
  }
};
j.elementStyles = [], j.shadowRootOptions = { mode: "open" }, j[L("elementProperties")] = /* @__PURE__ */ new Map(), j[L("finalized")] = /* @__PURE__ */ new Map(), te == null || te({ ReactiveElement: j }), (C.reactiveElementVersions ?? (C.reactiveElementVersions = [])).push("2.1.2");
/**
 * @license
 * Copyright 2017 Google LLC
 * SPDX-License-Identifier: BSD-3-Clause
 */
const q = globalThis, he = (t) => t, Q = q.trustedTypes, ue = Q ? Q.createPolicy("lit-html", { createHTML: (t) => t }) : void 0, _e = "$lit$", P = `lit$${Math.random().toFixed(9).slice(2)}$`, ke = "?" + P, je = `<${ke}>`, T = document, I = () => T.createComment(""), F = (t) => t === null || typeof t != "object" && typeof t != "function", le = Array.isArray, Be = (t) => le(t) || typeof (t == null ? void 0 : t[Symbol.iterator]) == "function", se = `[ 	
\f\r]`, W = /<(?:(!--|\/[^a-zA-Z])|(\/?[a-zA-Z][^>\s]*)|(\/?$))/g, ge = /-->/g, be = />/g, D = RegExp(`>|${se}(?:([^\\s"'>=/]+)(${se}*=${se}*(?:[^ 	
\f\r"'\`<>=]|("|')|))|$)`, "g"), me = /'/g, ve = /"/g, xe = /^(?:script|style|textarea|title)$/i, Ne = (t) => (e, ...s) => ({ _$litType$: t, strings: e, values: s }), n = Ne(1), B = Symbol.for("lit-noChange"), l = Symbol.for("lit-nothing"), $e = /* @__PURE__ */ new WeakMap(), R = T.createTreeWalker(T, 129);
function Ae(t, e) {
  if (!le(t) || !t.hasOwnProperty("raw")) throw Error("invalid template strings array");
  return ue !== void 0 ? ue.createHTML(e) : e;
}
const Me = (t, e) => {
  const s = t.length - 1, r = [];
  let i, a = e === 2 ? "<svg>" : e === 3 ? "<math>" : "", o = W;
  for (let c = 0; c < s; c++) {
    const d = t[c];
    let h, u, f = -1, g = 0;
    for (; g < d.length && (o.lastIndex = g, u = o.exec(d), u !== null); ) g = o.lastIndex, o === W ? u[1] === "!--" ? o = ge : u[1] !== void 0 ? o = be : u[2] !== void 0 ? (xe.test(u[2]) && (i = RegExp("</" + u[2], "g")), o = D) : u[3] !== void 0 && (o = D) : o === D ? u[0] === ">" ? (o = i ?? W, f = -1) : u[1] === void 0 ? f = -2 : (f = o.lastIndex - u[2].length, h = u[1], o = u[3] === void 0 ? D : u[3] === '"' ? ve : me) : o === ve || o === me ? o = D : o === ge || o === be ? o = W : (o = D, i = void 0);
    const v = o === D && t[c + 1].startsWith("/>") ? " " : "";
    a += o === W ? d + je : f >= 0 ? (r.push(h), d.slice(0, f) + _e + d.slice(f) + P + v) : d + P + (f === -2 ? c : v);
  }
  return [Ae(t, a + (t[s] || "<?>") + (e === 2 ? "</svg>" : e === 3 ? "</math>" : "")), r];
};
class G {
  constructor({ strings: e, _$litType$: s }, r) {
    let i;
    this.parts = [];
    let a = 0, o = 0;
    const c = e.length - 1, d = this.parts, [h, u] = Me(e, s);
    if (this.el = G.createElement(h, r), R.currentNode = this.el.content, s === 2 || s === 3) {
      const f = this.el.content.firstChild;
      f.replaceWith(...f.childNodes);
    }
    for (; (i = R.nextNode()) !== null && d.length < c; ) {
      if (i.nodeType === 1) {
        if (i.hasAttributes()) for (const f of i.getAttributeNames()) if (f.endsWith(_e)) {
          const g = u[o++], v = i.getAttribute(f).split(P), w = /([.?@])?(.*)/.exec(g);
          d.push({ type: 1, index: a, name: w[2], strings: v, ctor: w[1] === "." ? We : w[1] === "?" ? Le : w[1] === "@" ? qe : Y }), i.removeAttribute(f);
        } else f.startsWith(P) && (d.push({ type: 6, index: a }), i.removeAttribute(f));
        if (xe.test(i.tagName)) {
          const f = i.textContent.split(P), g = f.length - 1;
          if (g > 0) {
            i.textContent = Q ? Q.emptyScript : "";
            for (let v = 0; v < g; v++) i.append(f[v], I()), R.nextNode(), d.push({ type: 2, index: ++a });
            i.append(f[g], I());
          }
        }
      } else if (i.nodeType === 8) if (i.data === ke) d.push({ type: 2, index: a });
      else {
        let f = -1;
        for (; (f = i.data.indexOf(P, f + 1)) !== -1; ) d.push({ type: 7, index: a }), f += P.length - 1;
      }
      a++;
    }
  }
  static createElement(e, s) {
    const r = T.createElement("template");
    return r.innerHTML = e, r;
  }
}
function N(t, e, s = t, r) {
  var o, c;
  if (e === B) return e;
  let i = r !== void 0 ? (o = s._$Co) == null ? void 0 : o[r] : s._$Cl;
  const a = F(e) ? void 0 : e._$litDirective$;
  return (i == null ? void 0 : i.constructor) !== a && ((c = i == null ? void 0 : i._$AO) == null || c.call(i, !1), a === void 0 ? i = void 0 : (i = new a(t), i._$AT(t, s, r)), r !== void 0 ? (s._$Co ?? (s._$Co = []))[r] = i : s._$Cl = i), i !== void 0 && (e = N(t, i._$AS(t, e.values), i, r)), e;
}
class He {
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
    const { el: { content: s }, parts: r } = this._$AD, i = ((e == null ? void 0 : e.creationScope) ?? T).importNode(s, !0);
    R.currentNode = i;
    let a = R.nextNode(), o = 0, c = 0, d = r[0];
    for (; d !== void 0; ) {
      if (o === d.index) {
        let h;
        d.type === 2 ? h = new K(a, a.nextSibling, this, e) : d.type === 1 ? h = new d.ctor(a, d.name, d.strings, this, e) : d.type === 6 && (h = new Ie(a, this, e)), this._$AV.push(h), d = r[++c];
      }
      o !== (d == null ? void 0 : d.index) && (a = R.nextNode(), o++);
    }
    return R.currentNode = T, i;
  }
  p(e) {
    let s = 0;
    for (const r of this._$AV) r !== void 0 && (r.strings !== void 0 ? (r._$AI(e, r, s), s += r.strings.length - 2) : r._$AI(e[s])), s++;
  }
}
class K {
  get _$AU() {
    var e;
    return ((e = this._$AM) == null ? void 0 : e._$AU) ?? this._$Cv;
  }
  constructor(e, s, r, i) {
    this.type = 2, this._$AH = l, this._$AN = void 0, this._$AA = e, this._$AB = s, this._$AM = r, this.options = i, this._$Cv = (i == null ? void 0 : i.isConnected) ?? !0;
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
    e = N(this, e, s), F(e) ? e === l || e == null || e === "" ? (this._$AH !== l && this._$AR(), this._$AH = l) : e !== this._$AH && e !== B && this._(e) : e._$litType$ !== void 0 ? this.$(e) : e.nodeType !== void 0 ? this.T(e) : Be(e) ? this.k(e) : this._(e);
  }
  O(e) {
    return this._$AA.parentNode.insertBefore(e, this._$AB);
  }
  T(e) {
    this._$AH !== e && (this._$AR(), this._$AH = this.O(e));
  }
  _(e) {
    this._$AH !== l && F(this._$AH) ? this._$AA.nextSibling.data = e : this.T(T.createTextNode(e)), this._$AH = e;
  }
  $(e) {
    var a;
    const { values: s, _$litType$: r } = e, i = typeof r == "number" ? this._$AC(e) : (r.el === void 0 && (r.el = G.createElement(Ae(r.h, r.h[0]), this.options)), r);
    if (((a = this._$AH) == null ? void 0 : a._$AD) === i) this._$AH.p(s);
    else {
      const o = new He(i, this), c = o.u(this.options);
      o.p(s), this.T(c), this._$AH = o;
    }
  }
  _$AC(e) {
    let s = $e.get(e.strings);
    return s === void 0 && $e.set(e.strings, s = new G(e)), s;
  }
  k(e) {
    le(this._$AH) || (this._$AH = [], this._$AR());
    const s = this._$AH;
    let r, i = 0;
    for (const a of e) i === s.length ? s.push(r = new K(this.O(I()), this.O(I()), this, this.options)) : r = s[i], r._$AI(a), i++;
    i < s.length && (this._$AR(r && r._$AB.nextSibling, i), s.length = i);
  }
  _$AR(e = this._$AA.nextSibling, s) {
    var r;
    for ((r = this._$AP) == null ? void 0 : r.call(this, !1, !0, s); e !== this._$AB; ) {
      const i = he(e).nextSibling;
      he(e).remove(), e = i;
    }
  }
  setConnected(e) {
    var s;
    this._$AM === void 0 && (this._$Cv = e, (s = this._$AP) == null || s.call(this, e));
  }
}
class Y {
  get tagName() {
    return this.element.tagName;
  }
  get _$AU() {
    return this._$AM._$AU;
  }
  constructor(e, s, r, i, a) {
    this.type = 1, this._$AH = l, this._$AN = void 0, this.element = e, this.name = s, this._$AM = i, this.options = a, r.length > 2 || r[0] !== "" || r[1] !== "" ? (this._$AH = Array(r.length - 1).fill(new String()), this.strings = r) : this._$AH = l;
  }
  _$AI(e, s = this, r, i) {
    const a = this.strings;
    let o = !1;
    if (a === void 0) e = N(this, e, s, 0), o = !F(e) || e !== this._$AH && e !== B, o && (this._$AH = e);
    else {
      const c = e;
      let d, h;
      for (e = a[0], d = 0; d < a.length - 1; d++) h = N(this, c[r + d], s, d), h === B && (h = this._$AH[d]), o || (o = !F(h) || h !== this._$AH[d]), h === l ? e = l : e !== l && (e += (h ?? "") + a[d + 1]), this._$AH[d] = h;
    }
    o && !i && this.j(e);
  }
  j(e) {
    e === l ? this.element.removeAttribute(this.name) : this.element.setAttribute(this.name, e ?? "");
  }
}
class We extends Y {
  constructor() {
    super(...arguments), this.type = 3;
  }
  j(e) {
    this.element[this.name] = e === l ? void 0 : e;
  }
}
class Le extends Y {
  constructor() {
    super(...arguments), this.type = 4;
  }
  j(e) {
    this.element.toggleAttribute(this.name, !!e && e !== l);
  }
}
class qe extends Y {
  constructor(e, s, r, i, a) {
    super(e, s, r, i, a), this.type = 5;
  }
  _$AI(e, s = this) {
    if ((e = N(this, e, s, 0) ?? l) === B) return;
    const r = this._$AH, i = e === l && r !== l || e.capture !== r.capture || e.once !== r.once || e.passive !== r.passive, a = e !== l && (r === l || i);
    i && this.element.removeEventListener(this.name, this, r), a && this.element.addEventListener(this.name, this, e), this._$AH = e;
  }
  handleEvent(e) {
    var s;
    typeof this._$AH == "function" ? this._$AH.call(((s = this.options) == null ? void 0 : s.host) ?? this.element, e) : this._$AH.handleEvent(e);
  }
}
class Ie {
  constructor(e, s, r) {
    this.element = e, this.type = 6, this._$AN = void 0, this._$AM = s, this.options = r;
  }
  get _$AU() {
    return this._$AM._$AU;
  }
  _$AI(e) {
    N(this, e);
  }
}
const ie = q.litHtmlPolyfillSupport;
ie == null || ie(G, K), (q.litHtmlVersions ?? (q.litHtmlVersions = [])).push("3.3.2");
const Fe = (t, e, s) => {
  const r = (s == null ? void 0 : s.renderBefore) ?? e;
  let i = r._$litPart$;
  if (i === void 0) {
    const a = (s == null ? void 0 : s.renderBefore) ?? null;
    r._$litPart$ = i = new K(e.insertBefore(I(), a), a, void 0, s ?? {});
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
    this.hasUpdated || (this.renderOptions.isConnected = this.isConnected), super.update(e), this._$Do = Fe(s, this.renderRoot, this.renderOptions);
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
A._$litElement$ = !0, A.finalized = !0, (ye = U.litElementHydrateSupport) == null || ye.call(U, { LitElement: A });
const re = U.litElementPolyfillSupport;
re == null || re({ LitElement: A });
(U.litElementVersions ?? (U.litElementVersions = [])).push("4.2.2");
/**
 * @license
 * Copyright 2017 Google LLC
 * SPDX-License-Identifier: BSD-3-Clause
 */
const J = (t) => (e, s) => {
  s !== void 0 ? s.addInitializer(() => {
    customElements.define(t, e);
  }) : customElements.define(t, e);
};
/**
 * @license
 * Copyright 2017 Google LLC
 * SPDX-License-Identifier: BSD-3-Clause
 */
const Ge = { attribute: !0, type: String, converter: X, reflect: !1, hasChanged: ne }, Ve = (t = Ge, e, s) => {
  const { kind: r, metadata: i } = s;
  let a = globalThis.litPropertyMetadata.get(i);
  if (a === void 0 && globalThis.litPropertyMetadata.set(i, a = /* @__PURE__ */ new Map()), r === "setter" && ((t = Object.create(t)).wrapped = !0), a.set(s.name, t), r === "accessor") {
    const { name: o } = s;
    return { set(c) {
      const d = e.get.call(this);
      e.set.call(this, c), this.requestUpdate(o, d, t, !0, c);
    }, init(c) {
      return c !== void 0 && this.C(o, void 0, t, c), c;
    } };
  }
  if (r === "setter") {
    const { name: o } = s;
    return function(c) {
      const d = this[o];
      e.call(this, c), this.requestUpdate(o, d, t, !0, c);
    };
  }
  throw Error("Unsupported decorator location: " + r);
};
function z(t) {
  return (e, s) => typeof s == "object" ? Ve(t, e, s) : ((r, i, a) => {
    const o = i.hasOwnProperty(a);
    return i.constructor.createProperty(a, r), o ? Object.getOwnPropertyDescriptor(i, a) : void 0;
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
function Ke(t, e) {
  const s = new WebSocket(t);
  return s.onmessage = (r) => {
    var i, a, o, c, d, h, u, f, g, v, w, b;
    try {
      const x = JSON.parse(r.data);
      ((a = (i = x.type) == null ? void 0 : i.startsWith) != null && a.call(i, "build.") || (c = (o = x.type) == null ? void 0 : o.startsWith) != null && c.call(o, "release.") || (h = (d = x.type) == null ? void 0 : d.startsWith) != null && h.call(d, "sdk.") || (f = (u = x.channel) == null ? void 0 : u.startsWith) != null && f.call(u, "build.") || (v = (g = x.channel) == null ? void 0 : g.startsWith) != null && v.call(g, "release.") || (b = (w = x.channel) == null ? void 0 : w.startsWith) != null && b.call(w, "sdk.")) && e(x);
    } catch {
    }
  }, s;
}
class ee {
  constructor(e = "") {
    this.baseUrl = e;
  }
  get base() {
    return `${this.baseUrl}/api/v1/build`;
  }
  async request(e, s) {
    var a;
    const i = await (await fetch(`${this.base}${e}`, s)).json();
    if (!i.success)
      throw new Error(((a = i.error) == null ? void 0 : a.message) ?? "Request failed");
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
    const r = new URLSearchParams();
    e && r.set("from", e), s && r.set("to", s);
    const i = r.toString();
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
    const r = new URLSearchParams({ base: e, revision: s });
    return this.request(`/sdk/diff?${r.toString()}`);
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
var Je = Object.defineProperty, Ze = Object.getOwnPropertyDescriptor, M = (t, e, s, r) => {
  for (var i = r > 1 ? void 0 : r ? Ze(e, s) : e, a = t.length - 1, o; a >= 0; a--)
    (o = t[a]) && (i = (r ? o(e, s, i) : o(i)) || i);
  return r && i && Je(e, s, i), i;
};
let E = class extends A {
  constructor() {
    super(...arguments), this.apiUrl = "", this.configData = null, this.discoverData = null, this.loading = !0, this.error = "";
  }
  connectedCallback() {
    super.connectedCallback(), this.api = new ee(this.apiUrl), this.reload();
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
  renderToggle(t, e, s = "Enabled", r = "Disabled") {
    return e == null ? l : n`
      <div class="field">
        <span class="field-label">${t}</span>
        <span class="badge ${e ? "present" : "absent"}">
          ${e ? s : r}
        </span>
      </div>
    `;
  }
  renderFlags(t, e) {
    return !e || e.length === 0 ? l : n`
      <div class="field">
        <span class="field-label">${t}</span>
        <div class="flags">
          ${e.map((s) => n`<span class="flag">${s}</span>`)}
        </div>
      </div>
    `;
  }
  render() {
    var s, r, i, a, o, c, d, h, u, f, g, v, w;
    if (this.loading)
      return n`<div class="loading">Loading configuration\u2026</div>`;
    if (this.error)
      return n`<div class="error">${this.error}</div>`;
    if (!this.configData)
      return n`<div class="empty">No configuration available.</div>`;
    const t = this.configData.config, e = this.discoverData;
    return n`
      <!-- Discovery -->
      <div class="section">
        <div class="section-title">Project Detection</div>
        <div class="field">
          <span class="field-label">Config file</span>
          <span class="badge ${this.configData.has_config ? "present" : "absent"}">
            ${this.configData.has_config ? "Present" : "Using defaults"}
          </span>
        </div>
        ${e ? n`
              <div class="field">
                <span class="field-label">Primary type</span>
                <span class="badge type-${e.primary || "unknown"}">${e.primary || "none"}</span>
              </div>
              <div class="field">
                <span class="field-label">Suggested stack</span>
                <span class="field-value">${e.suggested_stack || e.primary_stack || e.primary || "none"}</span>
              </div>
              ${e.types.length > 1 ? n`
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
                <span class="badge ${e.has_subtree_npm ? "present" : "absent"}">
                  ${e.has_subtree_npm ? "Depth 2" : "None"}
                </span>
              </div>
              ${e.distro ? n`
                    <div class="field">
                      <span class="field-label">Distro</span>
                      <span class="field-value">${e.distro}</span>
                    </div>
                  ` : l}
              ${e.linux_packages && e.linux_packages.length > 0 ? n`
                    <div class="field">
                      <span class="field-label">Linux packages</span>
                      <div class="flags">
                        ${e.linux_packages.map((b) => n`<span class="flag">${b}</span>`)}
                      </div>
                    </div>
                  ` : l}
              ${e.build_options ? n`
                    <div class="field">
                      <span class="field-label">Computed options</span>
                      <span class="field-value">${e.build_options}</span>
                    </div>
                  ` : l}
              ${this.renderToggle("Computed obfuscation", (s = e.options) == null ? void 0 : s.obfuscate)}
              ${this.renderToggle("Computed NSIS", (r = e.options) == null ? void 0 : r.nsis)}
              ${(i = e.options) != null && i.webview2 ? n`
                    <div class="field">
                      <span class="field-label">Computed WebView2</span>
                      <span class="field-value">${e.options.webview2}</span>
                    </div>
                  ` : l}
              ${this.renderFlags("Computed tags", (a = e.options) == null ? void 0 : a.tags)}
              ${this.renderFlags("Computed LD flags", (o = e.options) == null ? void 0 : o.ldflags)}
              ${e.ref ? n`
                    <div class="field">
                      <span class="field-label">Git ref</span>
                      <span class="field-value">${e.ref}</span>
                    </div>
                  ` : l}
              ${e.branch ? n`
                    <div class="field">
                      <span class="field-label">Branch</span>
                      <span class="field-value">${e.branch}</span>
                    </div>
                  ` : l}
              ${e.tag ? n`
                    <div class="field">
                      <span class="field-label">Tag</span>
                      <span class="field-value">${e.tag}</span>
                    </div>
                  ` : l}
              ${e.short_sha ? n`
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

      <!-- Project -->
      <div class="section">
        <div class="section-title">Project</div>
        ${t.project.name ? n`
              <div class="field">
                <span class="field-label">Name</span>
                <span class="field-value">${t.project.name}</span>
              </div>
            ` : l}
        ${t.project.description ? n`
              <div class="field">
                <span class="field-label">Description</span>
                <span class="field-value">${t.project.description}</span>
              </div>
            ` : l}
        ${t.project.binary ? n`
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
        ${t.build.type ? n`
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
        ${t.build.webview2 ? n`
              <div class="field">
                <span class="field-label">WebView2 mode</span>
                <span class="field-value">${t.build.webview2}</span>
              </div>
            ` : l}
        ${t.build.deno_build ? n`
              <div class="field">
                <span class="field-label">Deno build</span>
                <span class="field-value">${t.build.deno_build}</span>
              </div>
            ` : l}
        ${t.build.archive_format ? n`
              <div class="field">
                <span class="field-label">Archive format</span>
                <span class="field-value">${t.build.archive_format}</span>
              </div>
            ` : l}
        ${this.renderFlags("Build tags", t.build.build_tags)}
        ${t.build.flags && t.build.flags.length > 0 ? n`
              <div class="field">
                <span class="field-label">Flags</span>
                <div class="flags">
                  ${t.build.flags.map((b) => n`<span class="flag">${b}</span>`)}
                </div>
              </div>
            ` : l}
        ${t.build.ldflags && t.build.ldflags.length > 0 ? n`
              <div class="field">
                <span class="field-label">LD flags</span>
                <div class="flags">
                  ${t.build.ldflags.map((b) => n`<span class="flag">${b}</span>`)}
                </div>
              </div>
            ` : l}
        ${this.renderFlags("Environment", t.build.env)}
        ${(c = t.build.cache) != null && c.enabled || (d = t.build.cache) != null && d.path || (h = t.build.cache) != null && h.paths && t.build.cache.paths.length > 0 ? n`
              ${this.renderToggle("Build cache", (u = t.build.cache) == null ? void 0 : u.enabled)}
              ${(f = t.build.cache) != null && f.path ? n`
                    <div class="field">
                      <span class="field-label">Cache path</span>
                      <span class="field-value">${t.build.cache.path}</span>
                    </div>
                  ` : l}
              ${this.renderFlags("Cache paths", (g = t.build.cache) == null ? void 0 : g.paths)}
            ` : l}
        ${t.build.dockerfile ? n`
              <div class="field">
                <span class="field-label">Dockerfile</span>
                <span class="field-value">${t.build.dockerfile}</span>
              </div>
            ` : l}
        ${t.build.image ? n`
              <div class="field">
                <span class="field-label">Image</span>
                <span class="field-value">${t.build.image}</span>
              </div>
            ` : l}
        ${t.build.registry ? n`
              <div class="field">
                <span class="field-label">Registry</span>
                <span class="field-value">${t.build.registry}</span>
              </div>
            ` : l}
        ${this.renderFlags("Image tags", t.build.tags)}
        ${this.renderToggle("Push image", t.build.push)}
        ${this.renderToggle("Load image", t.build.load)}
        ${t.build.linuxkit_config ? n`
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
      (b) => n`<span class="target-badge">${b.os}/${b.arch}</span>`
    )}
        </div>
      </div>

      ${t.apple && this.hasAppleConfig(t.apple) ? n`
            <div class="section">
              <div class="section-title">Apple Pipeline</div>
              ${t.apple.bundle_id ? n`
                    <div class="field">
                      <span class="field-label">Bundle ID</span>
                      <span class="field-value">${t.apple.bundle_id}</span>
                    </div>
                  ` : l}
              ${t.apple.team_id ? n`
                    <div class="field">
                      <span class="field-label">Team ID</span>
                      <span class="field-value">${t.apple.team_id}</span>
                    </div>
                  ` : l}
              ${t.apple.arch ? n`
                    <div class="field">
                      <span class="field-label">Architecture</span>
                      <span class="field-value">${t.apple.arch}</span>
                    </div>
                  ` : l}
              ${t.apple.bundle_display_name ? n`
                    <div class="field">
                      <span class="field-label">Display name</span>
                      <span class="field-value">${t.apple.bundle_display_name}</span>
                    </div>
                  ` : l}
              ${t.apple.min_system_version ? n`
                    <div class="field">
                      <span class="field-label">Minimum macOS</span>
                      <span class="field-value">${t.apple.min_system_version}</span>
                    </div>
                  ` : l}
              ${t.apple.category ? n`
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
              ${t.apple.metadata_path ? n`
                    <div class="field">
                      <span class="field-label">Metadata path</span>
                      <span class="field-value">${t.apple.metadata_path}</span>
                    </div>
                  ` : l}
              ${t.apple.privacy_policy_url ? n`
                    <div class="field">
                      <span class="field-label">Privacy policy</span>
                      <span class="field-value">${t.apple.privacy_policy_url}</span>
                    </div>
                  ` : l}
              ${t.apple.dmg_volume_name ? n`
                    <div class="field">
                      <span class="field-label">DMG volume</span>
                      <span class="field-value">${t.apple.dmg_volume_name}</span>
                    </div>
                  ` : l}
              ${t.apple.dmg_background ? n`
                    <div class="field">
                      <span class="field-label">DMG background</span>
                      <span class="field-value">${t.apple.dmg_background}</span>
                    </div>
                  ` : l}
              ${t.apple.entitlements_path ? n`
                    <div class="field">
                      <span class="field-label">Entitlements</span>
                      <span class="field-value">${t.apple.entitlements_path}</span>
                    </div>
                  ` : l}
              ${(v = t.apple.xcode_cloud) != null && v.workflow ? n`
                    <div class="field">
                      <span class="field-label">Xcode Cloud workflow</span>
                      <span class="field-value">${t.apple.xcode_cloud.workflow}</span>
                    </div>
                  ` : l}
              ${(w = t.apple.xcode_cloud) != null && w.triggers && t.apple.xcode_cloud.triggers.length > 0 ? n`
                    <div class="field">
                      <span class="field-label">Xcode Cloud triggers</span>
                      <div class="flags">
                        ${t.apple.xcode_cloud.triggers.map((b) => {
      const x = b.branch ? `branch:${b.branch}` : b.tag ? `tag:${b.tag}` : "manual", Se = b.action ?? "archive";
      return n`<span class="flag">${x} → ${Se}</span>`;
    })}
                      </div>
                    </div>
                  ` : l}
            </div>
          ` : l}
    `;
  }
};
E.styles = V`
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
  J("core-build-config")
], E);
var Xe = Object.defineProperty, Qe = Object.getOwnPropertyDescriptor, S = (t, e, s, r) => {
  for (var i = r > 1 ? void 0 : r ? Qe(e, s) : e, a = t.length - 1, o; a >= 0; a--)
    (o = t[a]) && (i = (r ? o(e, s, i) : o(i)) || i);
  return r && i && Xe(e, s, i), i;
};
let _ = class extends A {
  constructor() {
    super(...arguments), this.apiUrl = "", this.artifacts = [], this.distExists = !1, this.loading = !0, this.error = "", this.building = !1, this.confirmBuild = !1, this.buildSuccess = "";
  }
  connectedCallback() {
    super.connectedCallback(), this.api = new ee(this.apiUrl), this.reload();
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
    return this.loading ? n`<div class="loading">Loading artifacts\u2026</div>` : n`
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

      ${this.confirmBuild ? n`
            <div class="confirm">
              <span class="confirm-text">This will run a full build and overwrite dist/. Continue?</span>
              <button class="confirm-yes" @click=${this.handleConfirmBuild}>Build</button>
              <button class="confirm-no" @click=${this.handleCancelBuild}>Cancel</button>
            </div>
          ` : l}

      ${this.error ? n`<div class="error">${this.error}</div>` : l}
      ${this.buildSuccess ? n`<div class="success">${this.buildSuccess}</div>` : l}

      ${this.artifacts.length === 0 ? n`<div class="empty">${this.distExists ? "dist/ is empty." : "Run a build to create artifacts."}</div>` : n`
            <div class="list">
              ${this.artifacts.map(
      (t) => n`
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
_.styles = V`
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
  J("core-build-artifacts")
], _);
var Ye = Object.defineProperty, et = Object.getOwnPropertyDescriptor, y = (t, e, s, r) => {
  for (var i = r > 1 ? void 0 : r ? et(e, s) : e, a = t.length - 1, o; a >= 0; a--)
    (o = t[a]) && (i = (r ? o(e, s, i) : o(i)) || i);
  return r && i && Ye(e, s, i), i;
};
let m = class extends A {
  constructor() {
    super(...arguments), this.apiUrl = "", this.version = "", this.changelog = "", this.loading = !0, this.error = "", this.releasing = !1, this.confirmRelease = !1, this.releaseSuccess = "", this.workflowPath = ".github/workflows/release.yml", this.workflowOutputPath = "", this.generatingWorkflow = !1, this.workflowSuccess = "";
  }
  connectedCallback() {
    super.connectedCallback(), this.api = new ee(this.apiUrl), this.reload();
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
      const s = await this.api.release(t), r = t ? "Dry run complete" : "Release published";
      this.releaseSuccess = `${r} — ${s.version} (${((e = s.artifacts) == null ? void 0 : e.length) ?? 0} artifact(s))`, await this.reload();
    } catch (s) {
      this.error = s.message ?? "Release failed";
    } finally {
      this.releasing = !1;
    }
  }
  render() {
    return this.loading ? n`<div class="loading">Loading release information\u2026</div>` : n`
      ${this.error ? n`<div class="error">${this.error}</div>` : l}
      ${this.releaseSuccess ? n`<div class="success">${this.releaseSuccess}</div>` : l}
      ${this.workflowSuccess ? n`<div class="success">${this.workflowSuccess}</div>` : l}

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

      ${this.confirmRelease ? n`
            <div class="confirm">
              <span class="confirm-text">This will publish ${this.version} to all configured targets. This action cannot be undone. Continue?</span>
              <button class="confirm-yes" @click=${this.handleConfirmRelease}>Publish</button>
              <button class="confirm-no" @click=${this.handleCancelRelease}>Cancel</button>
            </div>
          ` : l}

      ${this.changelog ? n`
            <div class="changelog-section">
              <div class="changelog-header">Changelog</div>
              <div class="changelog-content">${this.changelog}</div>
            </div>
          ` : n`<div class="empty">No changelog available.</div>`}
    `;
  }
};
m.styles = V`
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
  J("core-build-release")
], m);
var tt = Object.defineProperty, st = Object.getOwnPropertyDescriptor, k = (t, e, s, r) => {
  for (var i = r > 1 ? void 0 : r ? st(e, s) : e, a = t.length - 1, o; a >= 0; a--)
    (o = t[a]) && (i = (r ? o(e, s, i) : o(i)) || i);
  return r && i && tt(e, s, i), i;
};
let $ = class extends A {
  constructor() {
    super(...arguments), this.apiUrl = "", this.basePath = "", this.revisionPath = "", this.diffResult = null, this.diffing = !1, this.diffError = "", this.selectedLanguage = "", this.generating = !1, this.generateError = "", this.generateSuccess = "";
  }
  connectedCallback() {
    super.connectedCallback(), this.api = new ee(this.apiUrl);
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
    return n`
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

        ${this.diffError ? n`<div class="error">${this.diffError}</div>` : l}

        ${this.diffResult ? n`
              <div class="diff-result ${this.diffResult.Breaking ? "breaking" : "safe"}">
                <div class="diff-summary">${this.diffResult.Summary}</div>
                ${this.diffResult.Changes && this.diffResult.Changes.length > 0 ? n`
                      <ul class="diff-changes">
                        ${this.diffResult.Changes.map(
      (t) => n`<li>${t}</li>`
    )}
                      </ul>
                    ` : l}
              </div>
            ` : l}
      </div>

      <!-- SDK Generation -->
      <div class="section">
        <div class="section-title">SDK Generation</div>

        ${this.generateError ? n`<div class="error">${this.generateError}</div>` : l}
        ${this.generateSuccess ? n`<div class="success">${this.generateSuccess}</div>` : l}

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
$.styles = V`
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
  J("core-build-sdk")
], $);
var it = Object.defineProperty, rt = Object.getOwnPropertyDescriptor, H = (t, e, s, r) => {
  for (var i = r > 1 ? void 0 : r ? rt(e, s) : e, a = t.length - 1, o; a >= 0; a--)
    (o = t[a]) && (i = (r ? o(e, s, i) : o(i)) || i);
  return r && i && it(e, s, i), i;
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
    this.ws = Ke(this.wsUrl, (t) => {
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
        return n`<core-build-config api-url=${this.apiUrl}></core-build-config>`;
      case "build":
        return n`<core-build-artifacts api-url=${this.apiUrl}></core-build-artifacts>`;
      case "release":
        return n`<core-build-release api-url=${this.apiUrl}></core-build-release>`;
      case "sdk":
        return n`<core-build-sdk api-url=${this.apiUrl}></core-build-sdk>`;
      default:
        return l;
    }
  }
  render() {
    const t = this.wsUrl ? this.wsConnected ? "connected" : "disconnected" : "idle";
    return n`
      <div class="header">
        <span class="title">Build</span>
        <button class="refresh-btn" @click=${this.handleRefresh}>Refresh</button>
      </div>

      <div class="tabs">
        ${this.tabs.map(
      (e) => n`
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
        ${this.lastEvent ? n`<span>Last: ${this.lastEvent}</span>` : l}
      </div>
    `;
  }
};
O.styles = V`
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
  J("core-build-panel")
], O);
export {
  ee as BuildApi,
  _ as BuildArtifacts,
  E as BuildConfig,
  O as BuildPanel,
  m as BuildRelease,
  $ as BuildSdk,
  Ke as connectBuildEvents
};
