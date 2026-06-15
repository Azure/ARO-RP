function clampProp(e, n, t, o, r) {
  return ba(n, ((e, n) => {
    const t = e[n];
    if (void 0 === t) {
      throw new TypeError(missingField(n));
    }
    return t;
  })(e, n), t, o, r);
}

function ba(e, n, t, o, r, i) {
  const a = clampNumber(n, t, o);
  if (r && n !== a) {
    throw new RangeError(numberOutOfRange(e, n, t, o, i));
  }
  return a;
}

function s(e) {
  return null !== e && /object|function/.test(typeof e);
}

function on(e, n = Map) {
  const t = new n;
  return (n, ...o) => {
    if (t.has(n)) {
      return t.get(n);
    }
    const r = e(n, ...o);
    return t.set(n, r), r;
  };
}

function r(e) {
  return n({
    name: e
  }, 1);
}

function n(n, t) {
  return e((e => ({
    value: e,
    configurable: 1,
    writable: !t
  })), n);
}

function t(n) {
  return e((e => ({
    get: e,
    configurable: 1
  })), n);
}

function o(e) {
  return {
    [Symbol.toStringTag]: {
      value: e,
      configurable: 1
    }
  };
}

function zipProps(e, n) {
  const t = {};
  let o = e.length;
  for (const r of n) {
    t[e[--o]] = r;
  }
  return t;
}

function e(e, n, t) {
  const o = {};
  for (const r in n) {
    o[r] = e(n[r], r, t);
  }
  return o;
}

function P(e, n, t) {
  const o = {};
  for (let r = 0; r < n.length; r++) {
    const i = n[r];
    o[i] = e(i, r, t);
  }
  return o;
}

function remapProps(e, n, t) {
  const o = {};
  for (let r = 0; r < e.length; r++) {
    o[n[r]] = t[e[r]];
  }
  return o;
}

function nn(e, n) {
  const t = Object.create(null);
  for (const o of e) {
    t[o] = n[o];
  }
  return t;
}

function hasAnyPropsByName(e, n) {
  for (const t of n) {
    if (t in e) {
      return 1;
    }
  }
  return 0;
}

function allPropsEqual(e, n, t) {
  for (const o of e) {
    if (n[o] !== t[o]) {
      return 0;
    }
  }
  return 1;
}

function zeroOutProps(e, n, t) {
  const o = {
    ...t
  };
  for (let t = 0; t < n; t++) {
    o[e[t]] = 0;
  }
  return o;
}

function gt(e, ...n) {
  return (...t) => e(...n, ...t);
}

function Io(e) {
  return e;
}

function noop() {}

function capitalize(e) {
  return e[0].toUpperCase() + e.substring(1);
}

function sortStrings(e) {
  return e.slice().sort();
}

function padNumber(e, n) {
  return String(n).padStart(e, "0");
}

function compareNumbers(e, n) {
  return Math.sign(e - n);
}

function clampNumber(e, n, t) {
  return Math.min(Math.max(e, n), t);
}

function divModFloor(e, n) {
  return [ Math.floor(e / n), modFloor(e, n) ];
}

function modFloor(e, n) {
  return (e % n + n) % n;
}

function divModTrunc(e, n) {
  return [ divTrunc(e, n), modTrunc(e, n) ];
}

function divTrunc(e, n) {
  return Math.trunc(e / n) || 0;
}

function modTrunc(e, n) {
  return e % n || 0;
}

function hasHalf(e) {
  return .5 === Math.abs(e % 1);
}

function givenFieldsToBigNano(e, n, t) {
  let o = 0, r = 0;
  for (let i = 0; i <= n; i++) {
    const n = e[t[i]], a = Zu[i], s = go / a, [c, u] = divModTrunc(n, s);
    o += u * a, r += c;
  }
  const [i, a] = divModTrunc(o, go);
  return [ r + i, a ];
}

function nanoToGivenFields(e, n, t) {
  const o = {};
  for (let r = n; r >= 0; r--) {
    const n = Zu[r];
    o[t[r]] = divTrunc(e, n), e = modTrunc(e, n);
  }
  return o;
}

function m(e) {
  if (void 0 !== e) {
    return d(e);
  }
}

function g(e) {
  if (void 0 !== e) {
    return h(e);
  }
}

function S(e) {
  if (void 0 !== e) {
    return T(e);
  }
}

function h(e) {
  return requireNumberIsPositive(T(e));
}

function T(e) {
  return _e(rl(e));
}

function requirePropDefined(e, n) {
  if (null == n) {
    throw new RangeError(missingField(e));
  }
  return n;
}

function oa(e) {
  if (!s(e)) {
    throw new TypeError(ru);
  }
  return e;
}

function requireType(e, n, t = e) {
  if (typeof n !== e) {
    throw new TypeError(invalidEntity(t, n));
  }
  return n;
}

function _e(e, n = "number") {
  if (!Number.isInteger(e)) {
    throw new RangeError(expectedInteger(n, e));
  }
  return e || 0;
}

function requireNumberIsPositive(e, n = "number") {
  if (e <= 0) {
    throw new RangeError(expectedPositive(n, e));
  }
  return e;
}

function tu(e) {
  if ("symbol" == typeof e) {
    throw new TypeError(ou);
  }
  return String(e);
}

function toStringViaPrimitive(e, n) {
  return s(e) ? String(e) : d(e, n);
}

function toBigInt(e) {
  if ("string" == typeof e) {
    return BigInt(e);
  }
  if ("bigint" != typeof e) {
    throw new TypeError(invalidBigInt(e));
  }
  return e;
}

function toNumber(e, n = "number") {
  if ("bigint" == typeof e) {
    throw new TypeError(forbiddenBigIntToNumber(n));
  }
  if (e = Number(e), !Number.isFinite(e)) {
    throw new RangeError(expectedFinite(n, e));
  }
  return e;
}

function Za(e, n) {
  return Math.trunc(toNumber(e, n)) || 0;
}

function Ba(e, n) {
  return _e(toNumber(e, n), n);
}

function toPositiveInteger(e, n) {
  return requireNumberIsPositive(Za(e, n), n);
}

function createBigNano(e, n) {
  let [t, o] = divModTrunc(n, go), r = e + t;
  const i = Math.sign(r);
  return i && i === -Math.sign(o) && (r -= i, o += i * go), [ r, o ];
}

function so(e, n, t = 1) {
  return createBigNano(e[0] + n[0] * t, e[1] + n[1] * t);
}

function Ta(e, n) {
  return createBigNano(e[0], e[1] + n);
}

function va(e, n) {
  return so(n, e, -1);
}

function pa(e, n) {
  return compareNumbers(e[0], n[0]) || compareNumbers(e[1], n[1]);
}

function bigNanoOutside(e, n, t) {
  return -1 === pa(e, n) || 1 === pa(e, t);
}

function bigIntToBigNano(e, n = 1) {
  const t = BigInt(go / n);
  return [ Number(e / t), Number(e % t) * n ];
}

function Ge(e, n = 1) {
  const t = go / n, [o, r] = divModTrunc(e, t);
  return [ o, r * n ];
}

function bigNanoToBigInt(e, n = 1) {
  const [t, o] = e, r = Math.floor(o / n), i = go / n;
  return BigInt(t) * BigInt(i) + BigInt(r);
}

function La(e, n = 1, t) {
  const [o, r] = e, [i, a] = divModTrunc(r, n);
  return o * (go / n) + (i + (t ? a / n : 0));
}

function Oa(e) {
  return e[0] + e[1] / go;
}

function divModBigNano(e, n, t = divModFloor) {
  const [o, r] = e, [i, a] = t(r, n);
  return [ o * (go / n) + i, a ];
}

function checkIsoYearMonthInBounds(e) {
  return clampProp(e, "isoYear", Nl, yl, 1), e.isoYear === Nl ? clampProp(e, "isoMonth", 4, 12, 1) : e.isoYear === yl && clampProp(e, "isoMonth", 1, 9, 1), 
  e;
}

function To(e) {
  return Do({
    ...e,
    ...At,
    isoHour: 12
  }), e;
}

function Do(e) {
  const n = clampProp(e, "isoYear", Nl, yl, 1), t = n === Nl ? 1 : n === yl ? -1 : 0;
  return t && io(ma({
    ...e,
    isoDay: e.isoDay + t,
    isoNanosecond: e.isoNanosecond - t
  })), e;
}

function io(e) {
  if (!e || bigNanoOutside(e, Ml, Tl)) {
    throw new RangeError(Mu);
  }
  return e;
}

function isoTimeFieldsToNano(e) {
  return givenFieldsToBigNano(e, 5, w)[1];
}

function nanoToIsoTimeAndDay(e) {
  const [n, t] = divModFloor(e, go);
  return [ nanoToGivenFields(t, 5, w), n ];
}

function epochNanoToSec(e) {
  return epochNanoToSecMod(e)[0];
}

function epochNanoToSecMod(e) {
  return divModBigNano(e, oo);
}

function isoToEpochMilli(e) {
  return isoArgsToEpochMilli(e.isoYear, e.isoMonth, e.isoDay, e.isoHour, e.isoMinute, e.isoSecond, e.isoMillisecond);
}

function ma(e) {
  const n = isoToEpochMilli(e);
  if (void 0 !== n) {
    const [t, o] = divModTrunc(n, Cu);
    return [ t, o * Ke + (e.isoMicrosecond || 0) * ro + (e.isoNanosecond || 0) ];
  }
}

function isoToEpochNanoWithOffset(e, n) {
  const [t, o] = nanoToIsoTimeAndDay(isoTimeFieldsToNano(e) - n);
  return io(ma({
    ...e,
    isoDay: e.isoDay + o,
    ...t
  }));
}

function isoArgsToEpochSec(...e) {
  return isoArgsToEpochMilli(...e) / ku;
}

function isoArgsToEpochMilli(...e) {
  const [n, t] = isoToLegacyDate(...e), o = n.valueOf();
  if (!isNaN(o)) {
    return o - t * Cu;
  }
}

function isoToLegacyDate(e, n = 1, t = 1, o = 0, r = 0, i = 0, a = 0) {
  const s = e === Nl ? 1 : e === yl ? -1 : 0, c = new Date;
  return c.setUTCHours(o, r, i, a), c.setUTCFullYear(e, n - 1, t + s), [ c, s ];
}

function So(e, n) {
  let [t, o] = Ta(e, n);
  o < 0 && (o += go, t -= 1);
  const [r, i] = divModFloor(o, Ke), [a, s] = divModFloor(i, ro);
  return Pa(t * Cu + r, a, s);
}

function Pa(e, n = 0, t = 0) {
  const o = Math.ceil(Math.max(0, Math.abs(e) - gl) / Cu) * Math.sign(e), r = new Date(e - o * Cu);
  return zipProps(pl, [ r.getUTCFullYear(), r.getUTCMonth() + 1, r.getUTCDate() + o, r.getUTCHours(), r.getUTCMinutes(), r.getUTCSeconds(), r.getUTCMilliseconds(), n, t ]);
}

function hashIntlFormatParts(e, n) {
  if (n < -gl) {
    throw new RangeError(Mu);
  }
  const t = e.formatToParts(n), o = {};
  for (const e of t) {
    o[e.type] = e.value;
  }
  return o;
}

function computeIsoDay(e) {
  return e.isoDay;
}

function computeIsoDateParts(e) {
  return [ e.isoYear, e.isoMonth, e.isoDay ];
}

function computeIsoMonthCodeParts(e, n) {
  return [ n, 0 ];
}

function computeIsoYearMonthForMonthDay(e, n) {
  if (!n) {
    return [ Pl, e ];
  }
}

function computeIsoFieldsFromParts(e, n, t) {
  return {
    isoYear: e,
    isoMonth: n,
    isoDay: t
  };
}

function fo() {
  return 7;
}

function computeIsoMonthsInYear() {
  return Fl;
}

function computeIsoDaysInMonth(e, n) {
  switch (n) {
   case 2:
    return computeIsoInLeapYear(e) ? 29 : 28;

   case 4:
   case 6:
   case 9:
   case 11:
    return 30;
  }
  return 31;
}

function computeIsoDaysInYear(e) {
  return computeIsoInLeapYear(e) ? 366 : 365;
}

function computeIsoInLeapYear(e) {
  return e % 4 == 0 && (e % 100 != 0 || e % 400 == 0);
}

function Ha(e) {
  const [n, t] = isoToLegacyDate(e.isoYear, e.isoMonth, e.isoDay);
  return modFloor(n.getUTCDay() - t, 7) || 7;
}

function computeIsoEraParts(e) {
  return this.id === Xu ? (({isoYear: e}) => e < 1 ? [ "gregory-inverse", 1 - e ] : [ "gregory", e ])(e) : this.id === el ? Ol(e) : [];
}

function computeJapaneseEraParts(e) {
  const n = isoToEpochMilli(e);
  if (n < El) {
    const {isoYear: n} = e;
    return n < 1 ? [ "japanese-inverse", 1 - n ] : [ "japanese", n ];
  }
  const t = hashIntlFormatParts(bf(el), n), {era: o, eraYear: r} = parseIntlYear(t, el);
  return [ o, r ];
}

function checkIsoDateTimeFields(e) {
  return checkIsoDateFields(e), constrainIsoTimeFields(e, 1), e;
}

function checkIsoDateFields(e) {
  return constrainIsoDateFields(e, 1), e;
}

function isIsoDateFieldsValid(e) {
  return allPropsEqual(ml, e, constrainIsoDateFields(e));
}

function constrainIsoDateFields(e, n) {
  const {isoYear: t} = e, o = clampProp(e, "isoMonth", 1, computeIsoMonthsInYear(), n);
  return {
    isoYear: t,
    isoMonth: o,
    isoDay: clampProp(e, "isoDay", 1, computeIsoDaysInMonth(t, o), n)
  };
}

function constrainIsoTimeFields(e, n) {
  return zipProps(w, [ clampProp(e, "isoHour", 0, 23, n), clampProp(e, "isoMinute", 0, 59, n), clampProp(e, "isoSecond", 0, 59, n), clampProp(e, "isoMillisecond", 0, 999, n), clampProp(e, "isoMicrosecond", 0, 999, n), clampProp(e, "isoNanosecond", 0, 999, n) ]);
}

function dt(e) {
  return void 0 === e ? 0 : Gl(oa(e));
}

function je(e, n = 0) {
  e = normalizeOptions(e);
  const t = Vl(e), o = _l(e, n);
  return [ Gl(e), o, t ];
}

function refineDiffOptions(e, n, t, o = 9, r = 0, i = 4) {
  n = normalizeOptions(n);
  let a = $l(n, o, r), s = parseRoundingIncInteger(n), c = Xl(n, i);
  const u = xl(n, o, r, 1);
  return null == a ? a = Math.max(t, u) : checkLargestSmallestUnit(a, u), s = refineRoundingInc(s, u, 1), 
  e && (c = (e => e < 4 ? (e + 2) % 4 : e)(c)), [ a, u, s, c ];
}

function refineRoundingOptions(e, n = 6, t) {
  let o = parseRoundingIncInteger(e = normalizeOptionsOrString(e, bl));
  const r = Xl(e, 7);
  let i = xl(e, n);
  return i = requirePropDefined(bl, i), o = refineRoundingInc(o, i, void 0, t), [ i, o, r ];
}

function refineRoundingMathOptions(e, n, t) {
  let o = parseRoundingIncInteger(n = normalizeOptionsOrString(n, wl));
  const r = Xl(n, 7);
  return o = refineRoundingInc(o, e, t), [ o, r ];
}

function Ma(e, n) {
  return void 0 !== n ? refineRoundingMathOptions(e, n, 1) : [];
}

function co(e, n) {
  return void 0 !== n ? refineRoundingMathOptions(e, n) : [ 1, 7 ];
}

function refineDateDisplayOptions(e) {
  return Jl(normalizeOptions(e));
}

function refineTimeDisplayOptions(e, n) {
  return refineTimeDisplayTuple(normalizeOptions(e), n);
}

function Ze(e) {
  const n = normalizeOptionsOrString(e, kl), t = refineChoiceOption(kl, Wl, n, 0);
  if (!t) {
    throw new RangeError(invalidEntity(kl, t));
  }
  return t;
}

function refineTimeDisplayTuple(e, n = 4) {
  const t = refineSubsecDigits(e);
  return [ Xl(e, 4), ...refineSmallestUnitAndSubsecDigits(xl(e, n), t) ];
}

function refineSmallestUnitAndSubsecDigits(e, n) {
  return null != e ? [ Zu[e], e < 4 ? 9 - 3 * e : -1 ] : [ void 0 === n ? 1 : 10 ** (9 - n), n ];
}

function parseRoundingIncInteger(e) {
  const n = e[Bl];
  return void 0 === n ? 1 : Za(n, Bl);
}

function refineRoundingInc(e, n, t, o) {
  const r = o ? go : Zu[n + 1];
  if (r) {
    const t = Zu[n];
    if (r % ((e = ba(Bl, e, 1, r / t - (o ? 0 : 1), 1)) * t)) {
      throw new RangeError(invalidEntity(Bl, e));
    }
  } else {
    e = ba(Bl, e, 1, t ? 10 ** 9 : 1, 1);
  }
  return e;
}

function refineSubsecDigits(e) {
  let n = e[Yl];
  if (void 0 !== n) {
    if ("number" != typeof n) {
      if ("auto" === tu(n)) {
        return;
      }
      throw new RangeError(invalidEntity(Yl, n));
    }
    n = ba(Yl, Math.floor(n), 0, 9, 1);
  }
  return n;
}

function normalizeOptions(e) {
  return void 0 === e ? {} : oa(e);
}

function normalizeOptionsOrString(e, n) {
  return "string" == typeof e ? {
    [n]: e
  } : oa(e);
}

function fabricateOverflowOptions(e) {
  return {
    overflow: Rl[e]
  };
}

function refineUnitOption(e, n, t = 9, o = 0, r) {
  let i = n[e];
  if (void 0 === i) {
    return r ? o : void 0;
  }
  if (i = tu(i), "auto" === i) {
    return r ? o : null;
  }
  let a = Bu[i];
  if (void 0 === a && (a = ul[i]), void 0 === a) {
    throw new RangeError(invalidChoice(e, i, Bu));
  }
  return ba(e, a, o, t, 1, Yu), a;
}

function refineChoiceOption(e, n, t, o = 0) {
  const r = t[e];
  if (void 0 === r) {
    return o;
  }
  const i = tu(r), a = n[i];
  if (void 0 === a) {
    throw new RangeError(invalidChoice(e, i, n));
  }
  return a;
}

function checkLargestSmallestUnit(e, n) {
  if (n > e) {
    throw new RangeError(Eu);
  }
}

function xe(e) {
  return {
    branding: Re,
    epochNanoseconds: e
  };
}

function Xe(e, n, t) {
  return {
    branding: _,
    calendar: t,
    timeZone: n,
    epochNanoseconds: e
  };
}

function jt(e, n = e.calendar) {
  return {
    branding: x,
    calendar: n,
    ...nn(Il, e)
  };
}

function W(e, n = e.calendar) {
  return {
    branding: G,
    calendar: n,
    ...nn(Ca, e)
  };
}

function createPlainYearMonthSlots(e, n = e.calendar) {
  return {
    branding: Qt,
    calendar: n,
    ...nn(Ca, e)
  };
}

function createPlainMonthDaySlots(e, n = e.calendar) {
  return {
    branding: qt,
    calendar: n,
    ...nn(Ca, e)
  };
}

function St(e) {
  return {
    branding: ft,
    ...nn(hl, e)
  };
}

function pe(e) {
  return {
    branding: A,
    sign: computeDurationSign(e),
    ...nn(il, e)
  };
}

function ta(e) {
  return epochNanoToSec(e.epochNanoseconds);
}

function I(e) {
  return divModBigNano(e.epochNanoseconds, Ke)[0];
}

function aa(e) {
  return bigNanoToBigInt(e.epochNanoseconds, ro);
}

function b(e) {
  return bigNanoToBigInt(e.epochNanoseconds);
}

function fa(e) {
  return e.epochNanoseconds;
}

function J(e, n, t, o, r) {
  const i = getMaxDurationUnit(o), [a, s] = ((e, n) => {
    const t = n((e = normalizeOptionsOrString(e, Sl))[Cl]);
    let o = Hl(e);
    return o = requirePropDefined(Sl, o), [ o, t ];
  })(r, e), c = Math.max(a, i);
  if (!s && isUniformUnit(c, s)) {
    return totalDayTimeDuration(o, a);
  }
  if (!s) {
    throw new RangeError(vu);
  }
  if (!o.sign) {
    return 0;
  }
  const [u, l, f] = createMarkerSystem(n, t, s), d = createMarkerToEpochNano(f), m = createMoveMarker(f), p = createDiffMarkers(f), h = m(l, u, o);
  isZonedEpochSlots(s) || (Do(u), Do(h));
  const I = p(l, u, h, a);
  return isUniformUnit(a, s) ? totalDayTimeDuration(I, a) : ya(I, d(h), a, l, u, d, m);
}

function ya(e, n, t, o, r, i, a) {
  const s = computeDurationSign(e), [c, u] = clampRelativeDuration(o, dl(t, e), t, s, r, i, a), l = ja(n, c, u);
  return e[O[t]] + l * s;
}

function totalDayTimeDuration(e, n) {
  return La(durationFieldsToBigNano(e), Zu[n], 1);
}

function clampRelativeDuration(e, n, t, o, r, i, a) {
  const s = O[t], c = {
    ...n,
    [s]: n[s] + o
  }, u = a(e, r, n), l = a(e, r, c);
  return [ i(u), i(l) ];
}

function ja(e, n, t) {
  const o = La(va(n, t));
  if (!o) {
    throw new RangeError(du);
  }
  return La(va(n, e)) / o;
}

function Le(e, n) {
  const [t, o, r] = refineRoundingOptions(n, 5, 1);
  return xe(roundBigNano(e.epochNanoseconds, t, o, r, 1));
}

function Ie(e, n, t) {
  let {epochNanoseconds: o, timeZone: r, calendar: i} = n;
  const [a, s, c] = refineRoundingOptions(t);
  if (0 === a && 1 === s) {
    return n;
  }
  const u = e(r);
  if (6 === a) {
    o = uo(computeDayInterval, u, n, c);
  } else {
    const e = u.N(o);
    o = getMatchingInstantFor(u, roundDateTime(So(o, e), a, s, c), e, 2, 0, 1);
  }
  return Xe(o, r, i);
}

function bt(e, n) {
  return jt(roundDateTime(e, ...refineRoundingOptions(n)), e.calendar);
}

function lt(e, n) {
  const [t, o, r] = refineRoundingOptions(n, 5);
  var i;
  return St((i = r, roundTimeToNano(e, computeNanoInc(t, o), i)[0]));
}

function Te(e, n) {
  const t = e(n.timeZone), o = he(n, t), [r, i] = computeDayInterval(o), a = La(va(getStartOfDayInstantFor(t, r), getStartOfDayInstantFor(t, i)), no, 1);
  if (a <= 0) {
    throw new RangeError(du);
  }
  return a;
}

function be(e, n) {
  const {timeZone: t, calendar: o} = n;
  return Xe(lo(ho, e(t), n), t, o);
}

function lo(e, n, t) {
  return getStartOfDayInstantFor(n, e(he(t, n)));
}

function uo(e, n, t, o) {
  const r = he(t, n), [i, a] = e(r), s = t.epochNanoseconds, c = getStartOfDayInstantFor(n, i), u = getStartOfDayInstantFor(n, a);
  if (bigNanoOutside(s, c, u)) {
    throw new RangeError(du);
  }
  return Ea(ja(s, c, u), o) ? u : c;
}

function roundDateTime(e, n, t, o) {
  return roundDateTimeToNano(e, computeNanoInc(n, t), o);
}

function roundDateTimeToNano(e, n, t) {
  const [o, r] = roundTimeToNano(e, n, t);
  return Do({
    ...Ua(e, r),
    ...o
  });
}

function roundTimeToNano(e, n, t) {
  return nanoToIsoTimeAndDay(Da(isoTimeFieldsToNano(e), n, t));
}

function roundToMinute(e) {
  return Da(e, ao, 7);
}

function computeNanoInc(e, n) {
  return Zu[e] * n;
}

function computeDayInterval(e) {
  const n = ho(e);
  return [ n, Ua(n, 1) ];
}

function ho(e) {
  return Ra(6, e);
}

function roundDayTimeDurationByInc(e, n, t) {
  const o = Math.min(getMaxDurationUnit(e), 6);
  return nanoToDurationDayTimeFields(Ya(durationFieldsToBigNano(e, o), n, t), o);
}

function roundRelativeDuration(e, n, t, o, r, i, a, s, c, u) {
  if (0 === o && 1 === r) {
    return e;
  }
  const l = isUniformUnit(o, s) ? isZonedEpochSlots(s) && o < 6 && t >= 6 ? nudgeZonedTimeDuration : nudgeDayTimeDuration : nudgeRelativeDuration;
  let [f, d, m] = l(e, n, t, o, r, i, a, s, c, u);
  return m && 7 !== o && (f = ((e, n, t, o, r, i, a, s) => {
    const c = computeDurationSign(e);
    for (let u = o + 1; u <= t; u++) {
      if (7 === u && 7 !== t) {
        continue;
      }
      const o = dl(u, e);
      o[O[u]] += c;
      const l = La(va(a(s(r, i, o)), n));
      if (l && Math.sign(l) !== c) {
        break;
      }
      e = o;
    }
    return e;
  })(f, d, t, Math.max(6, o), a, s, c, u)), f;
}

function roundBigNano(e, n, t, o, r) {
  return 6 === n ? [ Da(Oa(e), t, o), 0 ] : Ya(e, computeNanoInc(n, t), o, r);
}

function Ya(e, n, t, o) {
  let [r, i] = e;
  o && i < 0 && (i += go, r -= 1);
  const [a, s] = divModFloor(Da(i, n, t), go);
  return createBigNano(r + a, s);
}

function Da(e, n, t) {
  return Ea(e / n, t) * n;
}

function Ea(e, n) {
  return ef[n](e);
}

function nudgeDayTimeDuration(e, n, t, o, r, i) {
  const a = computeDurationSign(e), s = durationFieldsToBigNano(e), c = roundBigNano(s, o, r, i), u = va(s, c), l = Math.sign(c[0] - s[0]) === a, f = nanoToDurationDayTimeFields(c, Math.min(t, 6));
  return [ {
    ...e,
    ...f
  }, so(n, u), l ];
}

function nudgeZonedTimeDuration(e, n, t, o, r, i, a, s, c, u) {
  const l = computeDurationSign(e) || 1, f = La(durationFieldsToBigNano(e, 5)), d = computeNanoInc(o, r);
  let m = Da(f, d, i);
  const [p, h] = clampRelativeDuration(a, {
    ...e,
    ...fl
  }, 6, l, s, c, u), I = m - La(va(p, h));
  let D = 0;
  I && Math.sign(I) !== l ? n = Ta(p, m) : (D += l, m = Da(I, d, i), n = Ta(h, m));
  const g = nanoToDurationTimeFields(m);
  return [ {
    ...e,
    ...g,
    days: e.days + D
  }, n, Boolean(D) ];
}

function nudgeRelativeDuration(e, n, t, o, r, i, a, s, c, u) {
  const l = computeDurationSign(e), f = O[o], d = dl(o, e);
  7 === o && (e = {
    ...e,
    weeks: e.weeks + Math.trunc(e.days / 7)
  });
  const m = divTrunc(e[f], r) * r;
  d[f] = m;
  const [p, h] = clampRelativeDuration(a, d, o, r * l, s, c, u), I = m + ja(n, p, h) * l * r, D = Da(I, r, i), g = Math.sign(D - I) === l;
  return d[f] = D, [ d, g ? h : p, g ];
}

function ke(e, n, t, o) {
  const [r, i, a, s] = (e => {
    const n = refineTimeDisplayTuple(e = normalizeOptions(e));
    return [ e.timeZone, ...n ];
  })(o), c = void 0 !== r;
  return ((e, n, t, o, r, i) => {
    t = Ya(t, r, o, 1);
    const a = n.N(t);
    return formatIsoDateTimeFields(So(t, a), i) + (e ? Se(roundToMinute(a)) : "Z");
  })(c, n(c ? e(r) : nf), t.epochNanoseconds, i, a, s);
}

function Fe(e, n, t) {
  const [o, r, i, a, s, c] = (e => {
    e = normalizeOptions(e);
    const n = Jl(e), t = refineSubsecDigits(e), o = Ql(e), r = Xl(e, 4), i = xl(e, 4);
    return [ n, Kl(e), o, r, ...refineSmallestUnitAndSubsecDigits(i, t) ];
  })(t);
  return ((e, n, t, o, r, i, a, s, c, u) => {
    o = Ya(o, c, s, 1);
    const l = e(t).N(o);
    return formatIsoDateTimeFields(So(o, l), u) + Se(roundToMinute(l), a) + ((e, n) => 1 !== n ? "[" + (2 === n ? "!" : "") + e + "]" : "")(t, i) + formatCalendar(n, r);
  })(e, n.calendar, n.timeZone, n.epochNanoseconds, o, r, i, a, s, c);
}

function Ft(e, n) {
  const [t, o, r, i] = (e => (e = normalizeOptions(e), [ Jl(e), ...refineTimeDisplayTuple(e) ]))(n);
  return a = e.calendar, s = t, c = i, formatIsoDateTimeFields(roundDateTimeToNano(e, r, o), c) + formatCalendar(a, s);
  var a, s, c;
}

function ce(e, n) {
  return t = e.calendar, o = e, r = refineDateDisplayOptions(n), formatIsoDateFields(o) + formatCalendar(t, r);
  var t, o, r;
}

function Ht(e, n) {
  return formatDateLikeIso(e.calendar, formatIsoYearMonthFields, e, refineDateDisplayOptions(n));
}

function Jt(e, n) {
  return formatDateLikeIso(e.calendar, formatIsoMonthDayFields, e, refineDateDisplayOptions(n));
}

function ct(e, n) {
  const [t, o, r] = refineTimeDisplayOptions(n);
  return i = r, formatIsoTimeFields(roundTimeToNano(e, o, t)[0], i);
  var i;
}

function k(e, n) {
  const [t, o, r] = refineTimeDisplayOptions(n, 3);
  return o > 1 && checkDurationUnits(e = {
    ...e,
    ...roundDayTimeDurationByInc(e, o, t)
  }), ((e, n) => {
    const {sign: t} = e, o = -1 === t ? negateDurationFields(e) : e, {hours: r, minutes: i} = o, [a, s] = divModBigNano(durationFieldsToBigNano(o, 3), oo, divModTrunc);
    checkDurationTimeUnit(a);
    const c = formatSubsecNano(s, n), u = n >= 0 || !t || c;
    return (t < 0 ? "-" : "") + "P" + formatDurationFragments({
      Y: formatDurationNumber(o.years),
      M: formatDurationNumber(o.months),
      W: formatDurationNumber(o.weeks),
      D: formatDurationNumber(o.days)
    }) + (r || i || a || u ? "T" + formatDurationFragments({
      H: formatDurationNumber(r),
      M: formatDurationNumber(i),
      S: formatDurationNumber(a, u) + c
    }) : "");
  })(e, r);
}

function formatDateLikeIso(e, n, t, o) {
  const r = o > 1 || 0 === o && e !== l;
  return 1 === o ? e === l ? n(t) : formatIsoDateFields(t) : r ? formatIsoDateFields(t) + formatCalendarId(e, 2 === o) : n(t);
}

function formatDurationFragments(e) {
  const n = [];
  for (const t in e) {
    const o = e[t];
    o && n.push(o, t);
  }
  return n.join("");
}

function formatIsoDateTimeFields(e, n) {
  return formatIsoDateFields(e) + "T" + formatIsoTimeFields(e, n);
}

function formatIsoDateFields(e) {
  return formatIsoYearMonthFields(e) + "-" + wu(e.isoDay);
}

function formatIsoYearMonthFields(e) {
  const {isoYear: n} = e;
  return (n < 0 || n > 9999 ? getSignStr(n) + padNumber(6, Math.abs(n)) : padNumber(4, n)) + "-" + wu(e.isoMonth);
}

function formatIsoMonthDayFields(e) {
  return wu(e.isoMonth) + "-" + wu(e.isoDay);
}

function formatIsoTimeFields(e, n) {
  const t = [ wu(e.isoHour), wu(e.isoMinute) ];
  return -1 !== n && t.push(wu(e.isoSecond) + ((e, n, t, o) => formatSubsecNano(e * Ke + n * ro + t, o))(e.isoMillisecond, e.isoMicrosecond, e.isoNanosecond, n)), 
  t.join(":");
}

function Se(e, n = 0) {
  if (1 === n) {
    return "";
  }
  const [t, o] = divModFloor(Math.abs(e), no), [r, i] = divModFloor(o, ao), [a, s] = divModFloor(i, oo);
  return getSignStr(e) + wu(t) + ":" + wu(r) + (a || s ? ":" + wu(a) + formatSubsecNano(s) : "");
}

function formatCalendar(e, n) {
  return 1 !== n && (n > 1 || 0 === n && e !== l) ? formatCalendarId(e, 2 === n) : "";
}

function formatCalendarId(e, n) {
  return "[" + (n ? "!" : "") + "u-ca=" + e + "]";
}

function formatSubsecNano(e, n) {
  let t = padNumber(9, e);
  return t = void 0 === n ? t.replace(af, "") : t.slice(0, n), t ? "." + t : "";
}

function getSignStr(e) {
  return e < 0 ? "-" : "+";
}

function formatDurationNumber(e, n) {
  return e || n ? e.toLocaleString("fullwide", {
    useGrouping: 0
  }) : "";
}

function _zonedEpochSlotsToIso(e, n) {
  const {epochNanoseconds: t} = e, o = (n.N ? n : n(e.timeZone)).N(t), r = So(t, o);
  return {
    calendar: e.calendar,
    ...r,
    offsetNanoseconds: o
  };
}

function Ja(e, n) {
  const t = he(n, e);
  return {
    calendar: n.calendar,
    ...nn(Il, t),
    offset: Se(t.offsetNanoseconds),
    timeZone: n.timeZone
  };
}

function getMatchingInstantFor(e, n, t, o = 0, r = 0, i, a) {
  if (void 0 !== t && 1 === o && (1 === o || a)) {
    return isoToEpochNanoWithOffset(n, t);
  }
  const s = e.v(n);
  if (void 0 !== t && 3 !== o) {
    const e = ((e, n, t, o) => {
      const r = ma(n);
      o && (t = roundToMinute(t));
      for (const n of e) {
        let e = La(va(n, r));
        if (o && (e = roundToMinute(e)), e === t) {
          return n;
        }
      }
    })(s, n, t, i);
    if (void 0 !== e) {
      return e;
    }
    if (0 === o) {
      throw new RangeError(gu);
    }
  }
  return a ? ma(n) : $o(e, n, r, s);
}

function $o(e, n, t = 0, o = e.v(n)) {
  if (1 === o.length) {
    return o[0];
  }
  if (1 === t) {
    throw new RangeError(Tu);
  }
  if (o.length) {
    return o[3 === t ? 1 : 0];
  }
  const r = ma(n), i = ((e, n) => {
    const t = e.N(Ta(n, -go));
    return (e => {
      if (e > go) {
        throw new RangeError(Du);
      }
      return e;
    })(e.N(Ta(n, go)) - t);
  })(e, r), a = i * (2 === t ? -1 : 1);
  return (o = e.v(So(r, a)))[2 === t ? 0 : o.length - 1];
}

function getStartOfDayInstantFor(e, n) {
  const t = e.v(n);
  if (t.length) {
    return t[0];
  }
  const o = Ta(ma(n), -go);
  return e.l(o, 1);
}

function Ye(e, n, t) {
  return xe(io(so(n.epochNanoseconds, (e => {
    if (durationHasDateParts(e)) {
      throw new RangeError(Pu);
    }
    return durationFieldsToBigNano(e, 5);
  })(e ? negateDurationFields(t) : t))));
}

function Oe(e, n, t, o, r, i = Object.create(null)) {
  const a = n(o.timeZone), s = e(o.calendar);
  return {
    ...o,
    ...Fa(a, s, o, t ? negateDurationFields(r) : r, i)
  };
}

function wt(e, n, t, o, r = Object.create(null)) {
  const {calendar: i} = t;
  return jt(ka(e(i), t, n ? negateDurationFields(o) : o, r), i);
}

function ne(e, n, t, o, r) {
  const {calendar: i} = t;
  return W(moveDate(e(i), t, n ? negateDurationFields(o) : o, r), i);
}

function Gt(e, n, t, o, r) {
  const i = t.calendar, a = e(i);
  let s = To(Na(a, t));
  n && (o = B(o)), o.sign < 0 && (s = a.P(s, {
    ...ll,
    months: 1
  }), s = Ua(s, -1));
  const c = a.P(s, o, r);
  return createPlainYearMonthSlots(Na(a, c), i);
}

function at(e, n, t) {
  return St(moveTime(n, e ? negateDurationFields(t) : t)[0]);
}

function Fa(e, n, t, o, r) {
  const i = durationFieldsToBigNano(o, 5);
  let a = t.epochNanoseconds;
  if (durationHasDateParts(o)) {
    const s = he(t, e);
    a = so($o(e, {
      ...moveDate(n, s, {
        ...o,
        ...fl
      }, r),
      ...nn(w, s)
    }), i);
  } else {
    a = so(a, i), dt(r);
  }
  return {
    epochNanoseconds: io(a)
  };
}

function ka(e, n, t, o) {
  const [r, i] = moveTime(n, t);
  return Do({
    ...moveDate(e, n, {
      ...t,
      ...fl,
      days: t.days + i
    }, o),
    ...r
  });
}

function moveDate(e, n, t, o) {
  if (t.years || t.months || t.weeks) {
    return e.P(n, t, o);
  }
  dt(o);
  const r = t.days + durationFieldsToBigNano(t, 5)[0];
  return r ? To(Ua(n, r)) : n;
}

function Na(e, n, t = 1) {
  return Ua(n, t - e.day(n));
}

function moveTime(e, n) {
  const [t, o] = durationFieldsToBigNano(n, 5), [r, i] = nanoToIsoTimeAndDay(isoTimeFieldsToNano(e) + o);
  return [ r, t + i ];
}

function nativeDateAdd(e, n, t) {
  const o = dt(t);
  let r, {years: i, months: a, weeks: s, days: c} = n;
  if (c += durationFieldsToBigNano(n, 5)[0], i || a) {
    r = wa(this, e, i, a, o);
  } else {
    if (!s && !c) {
      return e;
    }
    r = isoToEpochMilli(e);
  }
  if (void 0 === r) {
    throw new RangeError(Mu);
  }
  return r += (7 * s + c) * Cu, To(Pa(r));
}

function wa(e, n, t, o, r) {
  let [i, a, s] = e.u(n);
  if (t) {
    const [n, o] = e.m(i, a);
    i += t, a = monthCodeNumberToMonth(n, o, e.F(i)), a = ba("month", a, 1, e.O(i), r);
  }
  return o && ([i, a] = e.p(i, a, o)), s = ba("day", s, 1, e.B(i, a), r), e.M(i, a, s);
}

function isoMonthAdd(e, n, t) {
  return e += divTrunc(t, Fl), (n += modTrunc(t, Fl)) < 1 ? (e--, n += Fl) : n > Fl && (e++, 
  n -= Fl), [ e, n ];
}

function intlMonthAdd(e, n, t) {
  if (t) {
    if (n += t, !Number.isSafeInteger(n)) {
      throw new RangeError(Mu);
    }
    if (t < 0) {
      for (;n < 1; ) {
        n += computeIntlMonthsInYear.call(this, --e);
      }
    } else {
      let t;
      for (;n > (t = computeIntlMonthsInYear.call(this, e)); ) {
        n -= t, e++;
      }
    }
  }
  return [ e, n ];
}

function Ua(e, n) {
  return n ? {
    ...e,
    ...Pa(isoToEpochMilli(e) + n * Cu)
  } : e;
}

function createMarkerSystem(e, n, t) {
  const o = e(t.calendar);
  return isZonedEpochSlots(t) ? [ t, o, n(t.timeZone) ] : [ {
    ...t,
    ...At
  }, o ];
}

function createMarkerToEpochNano(e) {
  return e ? fa : ma;
}

function createMoveMarker(e) {
  return e ? gt(Fa, e) : ka;
}

function createDiffMarkers(e) {
  return e ? gt(diffZonedEpochsExact, e) : diffDateTimesExact;
}

function isZonedEpochSlots(e) {
  return e && e.epochNanoseconds;
}

function isUniformUnit(e, n) {
  return e <= 6 - (isZonedEpochSlots(n) ? 1 : 0);
}

function E(e, n, t, o, r, i, a) {
  const s = e(normalizeOptions(a).relativeTo), c = Math.max(getMaxDurationUnit(r), getMaxDurationUnit(i));
  if (isUniformUnit(c, s)) {
    return pe(checkDurationUnits(((e, n, t, o) => {
      const r = so(durationFieldsToBigNano(e), durationFieldsToBigNano(n), o ? -1 : 1);
      if (!Number.isFinite(r[0])) {
        throw new RangeError(Mu);
      }
      return {
        ...ll,
        ...nanoToDurationDayTimeFields(r, t)
      };
    })(r, i, c, o)));
  }
  if (!s) {
    throw new RangeError(vu);
  }
  o && (i = negateDurationFields(i));
  const [u, l, f] = createMarkerSystem(n, t, s), d = createMoveMarker(f), m = createDiffMarkers(f), p = d(l, u, r);
  return pe(m(l, u, d(l, p, i), c));
}

function V(e, n, t, o, r) {
  const i = getMaxDurationUnit(o), [a, s, c, u, l] = ((e, n, t) => {
    e = normalizeOptionsOrString(e, bl);
    let o = $l(e);
    const r = t(e[Cl]);
    let i = parseRoundingIncInteger(e);
    const a = Xl(e, 7);
    let s = xl(e);
    if (void 0 === o && void 0 === s) {
      throw new RangeError(Fu);
    }
    if (null == s && (s = 0), null == o && (o = Math.max(s, n)), checkLargestSmallestUnit(o, s), 
    i = refineRoundingInc(i, s, 1), i > 1 && s > 5 && o !== s) {
      throw new RangeError("For calendar units with roundingIncrement > 1, use largestUnit = smallestUnit");
    }
    return [ o, s, i, a, r ];
  })(r, i, e), f = Math.max(i, a);
  if (!l && f <= 6) {
    return pe(checkDurationUnits(((e, n, t, o, r) => {
      const i = roundBigNano(durationFieldsToBigNano(e), t, o, r);
      return {
        ...ll,
        ...nanoToDurationDayTimeFields(i, n)
      };
    })(o, a, s, c, u)));
  }
  if (!isZonedEpochSlots(l) && !o.sign) {
    return o;
  }
  if (!l) {
    throw new RangeError(vu);
  }
  const [d, m, p] = createMarkerSystem(n, t, l), h = createMarkerToEpochNano(p), I = createMoveMarker(p), D = createDiffMarkers(p), g = I(m, d, o);
  isZonedEpochSlots(l) || (Do(d), Do(g));
  let T = D(m, d, g, a);
  const M = o.sign, y = computeDurationSign(T);
  if (M && y && M !== y) {
    throw new RangeError(du);
  }
  return T = roundRelativeDuration(T, h(g), a, s, c, u, m, d, h, I), pe(T);
}

function Y(e) {
  return -1 === e.sign ? B(e) : e;
}

function B(e) {
  return pe(negateDurationFields(e));
}

function negateDurationFields(e) {
  const n = {};
  for (const t of O) {
    n[t] = -1 * e[t] || 0;
  }
  return n;
}

function y(e) {
  return !e.sign;
}

function computeDurationSign(e, n = O) {
  let t = 0;
  for (const o of n) {
    const n = Math.sign(e[o]);
    if (n) {
      if (t && t !== n) {
        throw new RangeError(Nu);
      }
      t = n;
    }
  }
  return t;
}

function checkDurationUnits(e) {
  for (const n of cl) {
    ba(n, e[n], -sf, sf, 1);
  }
  return checkDurationTimeUnit(La(durationFieldsToBigNano(e), oo)), e;
}

function checkDurationTimeUnit(e) {
  if (!Number.isSafeInteger(e)) {
    throw new RangeError(yu);
  }
}

function durationFieldsToBigNano(e, n = 6) {
  return givenFieldsToBigNano(e, n, O);
}

function nanoToDurationDayTimeFields(e, n = 6) {
  const [t, o] = e, r = nanoToGivenFields(o, n, O);
  if (r[O[n]] += t * (go / Zu[n]), !Number.isFinite(r[O[n]])) {
    throw new RangeError(Mu);
  }
  return r;
}

function nanoToDurationTimeFields(e, n = 5) {
  return nanoToGivenFields(e, n, O);
}

function durationHasDateParts(e) {
  return Boolean(computeDurationSign(e, sl));
}

function getMaxDurationUnit(e) {
  let n = 9;
  for (;n > 0 && !e[O[n]]; n--) {}
  return n;
}

function createSplitTuple(e, n) {
  return [ e, n ];
}

function computePeriod(e) {
  const n = Math.floor(e / tf) * tf;
  return [ n, n + tf ];
}

function We(e) {
  const n = parseDateTimeLike(e = toStringViaPrimitive(e));
  if (!n) {
    throw new RangeError(failedParse(e));
  }
  let t;
  if (n.C) {
    t = 0;
  } else {
    if (!n.offset) {
      throw new RangeError(failedParse(e));
    }
    t = parseOffsetNano(n.offset);
  }
  return n.timeZone && parseOffsetNanoMaybe(n.timeZone, 1), xe(isoToEpochNanoWithOffset(checkIsoDateTimeFields(n), t));
}

function $(e) {
  const n = parseDateTimeLike(d(e));
  if (!n) {
    throw new RangeError(failedParse(e));
  }
  if (n.timeZone) {
    return finalizeZonedDateTime(n, n.offset ? parseOffsetNano(n.offset) : void 0);
  }
  if (n.C) {
    throw new RangeError(failedParse(e));
  }
  return finalizeDate(n);
}

function Ne(e, n) {
  const t = parseDateTimeLike(d(e));
  if (!t || !t.timeZone) {
    throw new RangeError(failedParse(e));
  }
  const {offset: o} = t, r = o ? parseOffsetNano(o) : void 0, [, i, a] = je(n);
  return finalizeZonedDateTime(t, r, i, a);
}

function parseOffsetNano(e) {
  const n = parseOffsetNanoMaybe(e);
  if (void 0 === n) {
    throw new RangeError(failedParse(e));
  }
  return n;
}

function Bt(e) {
  const n = parseDateTimeLike(d(e));
  if (!n || n.C) {
    throw new RangeError(failedParse(e));
  }
  return jt(finalizeDateTime(n));
}

function me(e, n, t) {
  let o = parseDateTimeLike(d(e));
  if (!o || o.C) {
    throw new RangeError(failedParse(e));
  }
  return n ? o.calendar === l && (o = -271821 === o.isoYear && 4 === o.isoMonth ? {
    ...o,
    isoDay: 20,
    ...At
  } : {
    ...o,
    isoDay: 1,
    ...At
  }) : t && o.calendar === l && (o = {
    ...o,
    isoYear: Pl
  }), W(o.k ? finalizeDateTime(o) : finalizeDate(o));
}

function Xt(e, n) {
  const t = parseYearMonthOnly(d(n));
  if (t) {
    return requireIsoCalendar(t), createPlainYearMonthSlots(checkIsoYearMonthInBounds(checkIsoDateFields(t)));
  }
  const o = me(n, 1);
  return createPlainYearMonthSlots(Na(e(o.calendar), o));
}

function requireIsoCalendar(e) {
  if (e.calendar !== l) {
    throw new RangeError(invalidSubstring(e.calendar));
  }
}

function xt(e, n) {
  const t = parseMonthDayOnly(d(n));
  if (t) {
    return requireIsoCalendar(t), createPlainMonthDaySlots(checkIsoDateFields(t));
  }
  const o = me(n, 0, 1), {calendar: r} = o, i = e(r), [a, s, c] = i.u(o), [u, l] = i.m(a, s), [f, m] = i.R(u, l, c);
  return createPlainMonthDaySlots(To(i.U(f, m, c)), r);
}

function ht(e) {
  let n, t = (e => {
    const n = Tf.exec(e);
    return n ? (organizeAnnotationParts(n[10]), organizeTimeParts(n)) : void 0;
  })(d(e));
  if (!t) {
    if (t = parseDateTimeLike(e), !t) {
      throw new RangeError(failedParse(e));
    }
    if (!t.k) {
      throw new RangeError(failedParse(e));
    }
    if (t.C) {
      throw new RangeError(invalidSubstring("Z"));
    }
    requireIsoCalendar(t);
  }
  if ((n = parseYearMonthOnly(e)) && isIsoDateFieldsValid(n)) {
    throw new RangeError(failedParse(e));
  }
  if ((n = parseMonthDayOnly(e)) && isIsoDateFieldsValid(n)) {
    throw new RangeError(failedParse(e));
  }
  return St(constrainIsoTimeFields(t, 1));
}

function R(e) {
  const n = (e => {
    const n = Nf.exec(e);
    return n ? (e => {
      function parseUnit(e, r, i) {
        let a = 0, s = 0;
        if (i && ([a, o] = divModFloor(o, Zu[i])), void 0 !== e) {
          if (t) {
            throw new RangeError(invalidSubstring(e));
          }
          s = (e => {
            const n = parseInt(e);
            if (!Number.isFinite(n)) {
              throw new RangeError(invalidSubstring(e));
            }
            return n;
          })(e), n = 1, r && (o = parseSubsecNano(r) * (Zu[i] / oo), t = 1);
        }
        return a + s;
      }
      let n = 0, t = 0, o = 0, r = {
        ...zipProps(O, [ parseUnit(e[2]), parseUnit(e[3]), parseUnit(e[4]), parseUnit(e[5]), parseUnit(e[6], e[7], 5), parseUnit(e[8], e[9], 4), parseUnit(e[10], e[11], 3) ]),
        ...nanoToGivenFields(o, 2, O)
      };
      if (!n) {
        throw new RangeError(noValidFields(O));
      }
      return parseSign(e[1]) < 0 && (r = negateDurationFields(r)), r;
    })(n) : void 0;
  })(d(e));
  if (!n) {
    throw new RangeError(failedParse(e));
  }
  return pe(checkDurationUnits(n));
}

function f(e) {
  const n = parseDateTimeLike(e) || parseYearMonthOnly(e) || parseMonthDayOnly(e);
  return n ? n.calendar : e;
}

function M(e) {
  const n = parseDateTimeLike(e);
  return n && (n.timeZone || n.C && nf || n.offset) || e;
}

function finalizeZonedDateTime(e, n, t = 0, o = 0) {
  const r = Z(e.timeZone), i = L(r);
  let a;
  return checkIsoDateTimeFields(e), a = e.k ? getMatchingInstantFor(i, e, n, t, o, !i.j, e.C) : getStartOfDayInstantFor(i, e), 
  Xe(a, r, u(e.calendar));
}

function finalizeDateTime(e) {
  return resolveSlotsCalendar(Do(checkIsoDateTimeFields(e)));
}

function finalizeDate(e) {
  return resolveSlotsCalendar(To(checkIsoDateFields(e)));
}

function resolveSlotsCalendar(e) {
  return {
    ...e,
    calendar: u(e.calendar)
  };
}

function parseDateTimeLike(e) {
  const n = gf.exec(e);
  return n ? (e => {
    const n = e[10], t = "Z" === (n || "").toUpperCase();
    return {
      isoYear: organizeIsoYearParts(e),
      isoMonth: parseInt(e[4]),
      isoDay: parseInt(e[5]),
      ...organizeTimeParts(e.slice(5)),
      ...organizeAnnotationParts(e[16]),
      k: Boolean(e[6]),
      C: t,
      offset: t ? void 0 : n
    };
  })(n) : void 0;
}

function parseYearMonthOnly(e) {
  const n = If.exec(e);
  return n ? (e => ({
    isoYear: organizeIsoYearParts(e),
    isoMonth: parseInt(e[4]),
    isoDay: 1,
    ...organizeAnnotationParts(e[5])
  }))(n) : void 0;
}

function parseMonthDayOnly(e) {
  const n = Df.exec(e);
  return n ? (e => ({
    isoYear: Pl,
    isoMonth: parseInt(e[1]),
    isoDay: parseInt(e[2]),
    ...organizeAnnotationParts(e[3])
  }))(n) : void 0;
}

function parseOffsetNanoMaybe(e, n) {
  const t = Mf.exec(e);
  return t ? ((e, n) => {
    const t = e[4] || e[5];
    if (n && t) {
      throw new RangeError(invalidSubstring(t));
    }
    return (e => {
      if (Math.abs(e) >= go) {
        throw new RangeError(Iu);
      }
      return e;
    })((parseInt0(e[2]) * no + parseInt0(e[3]) * ao + parseInt0(e[4]) * oo + parseSubsecNano(e[5] || "")) * parseSign(e[1]));
  })(t, n) : void 0;
}

function organizeIsoYearParts(e) {
  const n = parseSign(e[1]), t = parseInt(e[2] || e[3]);
  if (n < 0 && !t) {
    throw new RangeError(invalidSubstring(-0));
  }
  return n * t;
}

function organizeTimeParts(e) {
  const n = parseInt0(e[3]);
  return {
    ...nanoToIsoTimeAndDay(parseSubsecNano(e[4] || ""))[0],
    isoHour: parseInt0(e[1]),
    isoMinute: parseInt0(e[2]),
    isoSecond: 60 === n ? 59 : n
  };
}

function organizeAnnotationParts(e) {
  let n, t;
  const o = [];
  if (e.replace(yf, ((e, r, i) => {
    const a = Boolean(r), [s, c] = i.split("=").reverse();
    if (c) {
      if ("u-ca" === c) {
        o.push(s), n || (n = a);
      } else if (a || /[A-Z]/.test(c)) {
        throw new RangeError(invalidSubstring(e));
      }
    } else {
      if (t) {
        throw new RangeError(invalidSubstring(e));
      }
      t = s;
    }
    return "";
  })), o.length > 1 && n) {
    throw new RangeError(invalidSubstring(e));
  }
  return {
    timeZone: t,
    calendar: o[0] || l
  };
}

function parseSubsecNano(e) {
  return parseInt(e.padEnd(9, "0"));
}

function createRegExp(e) {
  return new RegExp(`^${e}$`, "i");
}

function parseSign(e) {
  return e && "+" !== e ? -1 : 1;
}

function parseInt0(e) {
  return void 0 === e ? 0 : parseInt(e);
}

function Me(e) {
  return Z(d(e));
}

function Z(e) {
  const n = getTimeZoneEssence(e);
  return "number" == typeof n ? Se(n) : n ? (e => {
    if (Ff.test(e)) {
      throw new RangeError(F(e));
    }
    if (Pf.test(e)) {
      throw new RangeError(hu);
    }
    return e.toLowerCase().split("/").map(((e, n) => (e.length <= 3 || /\d/.test(e)) && !/etc|yap/.test(e) ? e.toUpperCase() : e.replace(/baja|dumont|[a-z]+/g, ((e, t) => e.length <= 2 && !n || "in" === e || "chat" === e ? e.toUpperCase() : e.length > 2 || !t ? capitalize(e).replace(/island|noronha|murdo|rivadavia|urville/, capitalize) : e)))).join("/");
  })(e) : nf;
}

function getTimeZoneAtomic(e) {
  const n = getTimeZoneEssence(e);
  return "number" == typeof n ? n : n ? n.resolvedOptions().timeZone : nf;
}

function getTimeZoneEssence(e) {
  const n = parseOffsetNanoMaybe(e = e.toUpperCase(), 1);
  return void 0 !== n ? n : e !== nf ? vf(e) : void 0;
}

function He(e, n) {
  return pa(e.epochNanoseconds, n.epochNanoseconds);
}

function Be(e, n) {
  return pa(e.epochNanoseconds, n.epochNanoseconds);
}

function H(e, n, t, o, r, i) {
  const a = e(normalizeOptions(i).relativeTo), s = Math.max(getMaxDurationUnit(o), getMaxDurationUnit(r));
  if (allPropsEqual(O, o, r)) {
    return 0;
  }
  if (isUniformUnit(s, a)) {
    return pa(durationFieldsToBigNano(o), durationFieldsToBigNano(r));
  }
  if (!a) {
    throw new RangeError(vu);
  }
  const [c, u, l] = createMarkerSystem(n, t, a), f = createMarkerToEpochNano(l), d = createMoveMarker(l);
  return pa(f(d(u, c, o)), f(d(u, c, r)));
}

function Yt(e, n) {
  return te(e, n) || Dt(e, n);
}

function te(e, n) {
  return compareNumbers(isoToEpochMilli(e), isoToEpochMilli(n));
}

function Dt(e, n) {
  return compareNumbers(isoTimeFieldsToNano(e), isoTimeFieldsToNano(n));
}

function Ve(e, n) {
  return !He(e, n);
}

function ve(e, n) {
  return !Be(e, n) && !!isTimeZoneIdsEqual(e.timeZone, n.timeZone) && e.calendar === n.calendar;
}

function vt(e, n) {
  return !Yt(e, n) && e.calendar === n.calendar;
}

function re(e, n) {
  return !te(e, n) && e.calendar === n.calendar;
}

function zt(e, n) {
  return !te(e, n) && e.calendar === n.calendar;
}

function Lt(e, n) {
  return !te(e, n) && e.calendar === n.calendar;
}

function st(e, n) {
  return !Dt(e, n);
}

function isTimeZoneIdsEqual(e, n) {
  if (e === n) {
    return 1;
  }
  try {
    return getTimeZoneAtomic(e) === getTimeZoneAtomic(n);
  } catch (e) {}
}

function Ee(e, n, t, o) {
  const r = refineDiffOptions(e, o, 3, 5), i = diffEpochNanos(n.epochNanoseconds, t.epochNanoseconds, ...r);
  return pe(e ? negateDurationFields(i) : i);
}

function we(e, n, t, o, r, i) {
  const a = ha(o.calendar, r.calendar), [s, c, u, l] = refineDiffOptions(t, i, 5), f = o.epochNanoseconds, d = r.epochNanoseconds, m = pa(d, f);
  let p;
  if (m) {
    if (s < 6) {
      p = diffEpochNanos(f, d, s, c, u, l);
    } else {
      const t = n(ga(o.timeZone, r.timeZone)), f = e(a);
      p = diffZonedEpochsBig(f, t, o, r, m, s, i), p = roundRelativeDuration(p, d, s, c, u, l, f, o, fa, gt(Fa, t));
    }
  } else {
    p = ll;
  }
  return pe(t ? negateDurationFields(p) : p);
}

function It(e, n, t, o, r) {
  const i = ha(t.calendar, o.calendar), [a, s, c, u] = refineDiffOptions(n, r, 6), l = ma(t), f = ma(o), d = pa(f, l);
  let m;
  if (d) {
    if (a <= 6) {
      m = diffEpochNanos(l, f, a, s, c, u);
    } else {
      const n = e(i);
      m = diffDateTimesBig(n, t, o, d, a, r), m = roundRelativeDuration(m, f, a, s, c, u, n, t, ma, ka);
    }
  } else {
    m = ll;
  }
  return pe(n ? negateDurationFields(m) : m);
}

function oe(e, n, t, o, r) {
  const i = ha(t.calendar, o.calendar);
  return diffDateLike(n, (() => e(i)), t, o, ...refineDiffOptions(n, r, 6, 9, 6));
}

function _t(e, n, t, o, r) {
  const i = ha(t.calendar, o.calendar), a = refineDiffOptions(n, r, 9, 9, 8), s = e(i), c = Na(s, t), u = Na(s, o);
  return c.isoYear === u.isoYear && c.isoMonth === u.isoMonth && c.isoDay === u.isoDay ? pe(ll) : diffDateLike(n, (() => s), To(c), To(u), ...a, 8);
}

function diffDateLike(e, n, t, o, r, i, a, s, c = 6) {
  const u = ma(t), l = ma(o);
  if (void 0 === u || void 0 === l) {
    throw new RangeError(Mu);
  }
  let f;
  if (pa(l, u)) {
    if (6 === r) {
      f = diffEpochNanos(u, l, r, i, a, s);
    } else {
      const e = n();
      f = e.h(t, o, r), i === c && 1 === a || (f = roundRelativeDuration(f, l, r, i, a, s, e, t, ma, moveDate));
    }
  } else {
    f = ll;
  }
  return pe(e ? negateDurationFields(f) : f);
}

function it(e, n, t, o) {
  const [r, i, a, s] = refineDiffOptions(e, o, 5, 5), c = Da(diffTimes(n, t), computeNanoInc(i, a), s), u = {
    ...ll,
    ...nanoToDurationTimeFields(c, r)
  };
  return pe(e ? negateDurationFields(u) : u);
}

function diffZonedEpochsExact(e, n, t, o, r, i) {
  const a = pa(o.epochNanoseconds, t.epochNanoseconds);
  return a ? r < 6 ? diffEpochNanosExact(t.epochNanoseconds, o.epochNanoseconds, r) : diffZonedEpochsBig(n, e, t, o, a, r, i) : ll;
}

function diffDateTimesExact(e, n, t, o, r) {
  const i = ma(n), a = ma(t), s = pa(a, i);
  return s ? o <= 6 ? diffEpochNanosExact(i, a, o) : diffDateTimesBig(e, n, t, s, o, r) : ll;
}

function diffZonedEpochsBig(e, n, t, o, r, i, a) {
  const [s, c, u] = Sa(n, t, o, r);
  var l, f;
  return {
    ...6 === i ? (l = s, f = c, {
      ...ll,
      days: td(l, f)
    }) : e.h(s, c, i, a),
    ...nanoToDurationTimeFields(u)
  };
}

function diffDateTimesBig(e, n, t, o, r, i) {
  const [a, s, c] = ((e, n, t) => {
    let o = n, r = diffTimes(e, n);
    return Math.sign(r) === -t && (o = Ua(n, -t), r += go * t), [ e, o, r ];
  })(n, t, o);
  return {
    ...e.h(a, s, r, i),
    ...nanoToDurationTimeFields(c)
  };
}

function Sa(e, n, t, o) {
  function updateMid() {
    return l = {
      ...Ua(a, c++ * -o),
      ...i
    }, f = $o(e, l), pa(s, f) === -o;
  }
  const r = he(n, e), i = nn(w, r), a = he(t, e), s = t.epochNanoseconds;
  let c = 0;
  const u = diffTimes(r, a);
  let l, f;
  if (Math.sign(u) === -o && c++, updateMid() && (-1 === o || updateMid())) {
    throw new RangeError(du);
  }
  const d = La(va(f, s));
  return [ r, l, d ];
}

function diffEpochNanos(e, n, t, o, r, i) {
  return {
    ...ll,
    ...nanoToDurationDayTimeFields(roundBigNano(va(e, n), o, r, i), t)
  };
}

function diffEpochNanosExact(e, n, t) {
  return {
    ...ll,
    ...nanoToDurationDayTimeFields(va(e, n), t)
  };
}

function td(e, n) {
  return diffEpochMilliByDay(isoToEpochMilli(e), isoToEpochMilli(n));
}

function diffEpochMilliByDay(e, n) {
  return Math.trunc((n - e) / Cu);
}

function diffTimes(e, n) {
  return isoTimeFieldsToNano(n) - isoTimeFieldsToNano(e);
}

function nativeDateUntil(e, n, t) {
  if (t <= 7) {
    let o = 0, r = td({
      ...e,
      ...At
    }, {
      ...n,
      ...At
    });
    return 7 === t && ([o, r] = divModTrunc(r, 7)), {
      ...ll,
      weeks: o,
      days: r
    };
  }
  const o = this.u(e), r = this.u(n);
  let [i, a, s] = ((e, n, t, o, r, i, a) => {
    let s = r - n, c = i - t, u = a - o;
    if (s || c) {
      const l = Math.sign(s || c);
      let f = e.B(r, i), d = 0;
      if (Math.sign(u) === -l) {
        const o = f;
        [r, i] = e.p(r, i, -l), s = r - n, c = i - t, f = e.B(r, i), d = l < 0 ? -o : f;
      }
      if (u = a - Math.min(o, f) + d, s) {
        const [o, a] = e.m(n, t), [u, f] = e.m(r, i);
        if (c = u - o || Number(f) - Number(a), Math.sign(c) === -l) {
          const t = l < 0 && -e.O(r);
          s = (r -= l) - n, c = i - monthCodeNumberToMonth(o, a, e.F(r)) + (t || e.O(r));
        }
      }
    }
    return [ s, c, u ];
  })(this, ...o, ...r);
  return 8 === t && (a += this.q(i, o[0]), i = 0), {
    ...ll,
    years: i,
    months: a,
    days: s
  };
}

function computeIsoMonthsInYearSpan(e) {
  return e * Fl;
}

function computeIntlMonthsInYearSpan(e, n) {
  const t = n + e, o = Math.sign(e), r = o < 0 ? -1 : 0;
  let i = 0;
  for (let e = n; e !== t; e += o) {
    i += computeIntlMonthsInYear.call(this, e + r);
  }
  return i;
}

function ha(e, n) {
  if (e !== n) {
    throw new RangeError(mu);
  }
  return e;
}

function ga(e, n) {
  if (!isTimeZoneIdsEqual(e, n)) {
    throw new RangeError(pu);
  }
  return e;
}

function computeNativeWeekOfYear(e) {
  return this.I(e)[0];
}

function computeNativeYearOfWeek(e) {
  return this.I(e)[1];
}

function computeNativeInLeapYear(e) {
  const [n] = this.u(e);
  return this.L(n);
}

function computeNativeMonthsInYear(e) {
  const [n] = this.u(e);
  return this.O(n);
}

function computeNativeDaysInMonth(e) {
  const [n, t] = this.u(e);
  return this.B(n, t);
}

function computeNativeDaysInYear(e) {
  const [n] = this.u(e);
  return this.G(n);
}

function computeNativeDayOfYear(e) {
  const [n] = this.u(e);
  return diffEpochMilliByDay(this.M(n), isoToEpochMilli(e)) + 1;
}

function parseMonthCode(e) {
  const n = Ef.exec(e);
  if (!n) {
    throw new RangeError(invalidMonthCode(e));
  }
  return [ parseInt(n[1]), Boolean(n[2]) ];
}

function sa(e, n) {
  return "M" + wu(e) + (n ? "L" : "");
}

function monthCodeNumberToMonth(e, n, t) {
  return e + (n || t && e >= t ? 1 : 0);
}

function monthToMonthCodeNumber(e, n) {
  return e - (n && e >= n ? 1 : 0);
}

function eraYearToYear(e, n) {
  return (n + e) * (Math.sign(n) || 1) || 0;
}

function getCalendarEraOrigins(e) {
  return nl[getCalendarIdBase(e)];
}

function getCalendarLeapMonthMeta(e) {
  return ol[getCalendarIdBase(e)];
}

function getCalendarIdBase(e) {
  return computeCalendarIdBase(e.id || l);
}

function createIntlCalendar(e) {
  function epochMilliToIntlFields(e) {
    return ((e, n) => ({
      ...parseIntlYear(e, n),
      V: e.month,
      day: parseInt(e.day)
    }))(hashIntlFormatParts(n, e), t);
  }
  const n = bf(e), t = computeCalendarIdBase(e);
  return {
    id: e,
    _: createIntlFieldCache(epochMilliToIntlFields),
    J: createIntlYearDataCache(epochMilliToIntlFields)
  };
}

function createIntlFieldCache(e) {
  return on((n => {
    const t = isoToEpochMilli(n);
    return e(t);
  }), WeakMap);
}

function createIntlYearDataCache(e) {
  const n = e(0).year - vl;
  return on((t => {
    let o, r = isoArgsToEpochMilli(t - n), i = 0;
    const a = [], s = [];
    do {
      r += 400 * Cu;
    } while ((o = e(r)).year <= t);
    do {
      if (r += (1 - o.day) * Cu, o.year === t && (a.push(r), s.push(o.V)), r -= Cu, ++i > 100 || r < -gl) {
        throw new RangeError(du);
      }
    } while ((o = e(r)).year >= t);
    return {
      K: a.reverse(),
      X: bu(s.reverse())
    };
  }));
}

function parseIntlYear(e, n) {
  let t, o, r = parseIntlPartsYear(e);
  if (e.era) {
    const i = nl[n], a = tl[n] || {};
    void 0 !== i && (t = "islamic" === n ? "ah" : e.era.normalize("NFD").toLowerCase().replace(/[^a-z0-9]/g, ""), 
    "bc" === t || "b" === t ? t = "bce" : "ad" === t || "a" === t ? t = "ce" : "beforeroc" === t && (t = "broc"), 
    t = a[t] || t, o = r, r = eraYearToYear(o, i[t] || 0));
  }
  return {
    era: t,
    eraYear: o,
    year: r
  };
}

function parseIntlPartsYear(e) {
  return parseInt(e.relatedYear || e.year);
}

function computeIntlDay(e) {
  return this._(e).day;
}

function computeIntlDateParts(e) {
  const {year: n, V: t, day: o} = this._(e), {X: r} = this.J(n);
  return [ n, r[t] + 1, o ];
}

function computeIsoFieldsFromIntlParts(e, n, t) {
  return Pa(computeIntlEpochMilli.call(this, e, n, t));
}

function computeIntlEpochMilli(e, n = 1, t = 1) {
  return this.J(e).K[n - 1] + (t - 1) * Cu;
}

function computeIntlMonthCodeParts(e, n) {
  const t = computeIntlLeapMonth.call(this, e);
  return [ monthToMonthCodeNumber(n, t), t === n ];
}

function computeIntlLeapMonth(e) {
  const n = queryMonthStrings(this, e), t = queryMonthStrings(this, e - 1), o = n.length;
  if (o > t.length) {
    const e = getCalendarLeapMonthMeta(this);
    if (e < 0) {
      return -e;
    }
    for (let e = 0; e < o; e++) {
      if (n[e] !== t[e]) {
        return e + 1;
      }
    }
  }
}

function computeIntlInLeapYear(e) {
  const n = computeIntlDaysInYear.call(this, e);
  return n > computeIntlDaysInYear.call(this, e - 1) && n > computeIntlDaysInYear.call(this, e + 1);
}

function computeIntlDaysInYear(e) {
  return diffEpochMilliByDay(computeIntlEpochMilli.call(this, e), computeIntlEpochMilli.call(this, e + 1));
}

function computeIntlDaysInMonth(e, n) {
  const {K: t} = this.J(e);
  let o = n + 1, r = t;
  return o > t.length && (o = 1, r = this.J(e + 1).K), diffEpochMilliByDay(t[n - 1], r[o - 1]);
}

function computeIntlMonthsInYear(e) {
  return this.J(e).K.length;
}

function computeIntlEraParts(e) {
  const n = this._(e);
  return [ n.era, n.eraYear ];
}

function computeIntlYearMonthForMonthDay(e, n, t) {
  const o = this.id && "chinese" === computeCalendarIdBase(this.id) ? ((e, n, t) => {
    if (n) {
      switch (e) {
       case 1:
        return 1651;

       case 2:
        return t < 30 ? 1947 : 1765;

       case 3:
        return t < 30 ? 1966 : 1955;

       case 4:
        return t < 30 ? 1963 : 1944;

       case 5:
        return t < 30 ? 1971 : 1952;

       case 6:
        return t < 30 ? 1960 : 1941;

       case 7:
        return t < 30 ? 1968 : 1938;

       case 8:
        return t < 30 ? 1957 : 1718;

       case 9:
        return 1832;

       case 10:
        return 1870;

       case 11:
        return 1814;

       case 12:
        return 1890;
      }
    }
    return 1972;
  })(e, n, t) : Pl;
  let [r, i, a] = computeIntlDateParts.call(this, {
    isoYear: o,
    isoMonth: Fl,
    isoDay: 31
  });
  const s = computeIntlLeapMonth.call(this, r), c = i === s;
  1 === (compareNumbers(e, monthToMonthCodeNumber(i, s)) || compareNumbers(Number(n), Number(c)) || compareNumbers(t, a)) && r--;
  for (let o = 0; o < 100; o++) {
    const i = r - o, a = computeIntlLeapMonth.call(this, i), s = monthCodeNumberToMonth(e, n, a);
    if (n === (s === a) && t <= computeIntlDaysInMonth.call(this, i, s)) {
      return [ i, s ];
    }
  }
}

function queryMonthStrings(e, n) {
  return Object.keys(e.J(n).X);
}

function Zt(e) {
  return u(d(e));
}

function u(e) {
  if ((e = e.toLowerCase()) !== l && e !== Xu) {
    const n = bf(e).resolvedOptions().calendar;
    if (computeCalendarIdBase(e) !== computeCalendarIdBase(n)) {
      throw new RangeError(c(e));
    }
    return n;
  }
  return e;
}

function computeCalendarIdBase(e) {
  return "islamicc" === e && (e = "islamic"), e.split("-")[0];
}

function createNativeOpsCreator(e, n) {
  return t => t === l ? e : t === Xu || t === el ? Object.assign(Object.create(e), {
    id: t
  }) : Object.assign(Object.create(n), Of(t));
}

function z(e, n, t, o) {
  const r = refineCalendarFields(t, o, _u, [], ju);
  if (void 0 !== r.timeZone) {
    const o = t.ee(r), i = refineTimeBag(r), a = e(r.timeZone);
    return {
      epochNanoseconds: getMatchingInstantFor(n(a), {
        ...o,
        ...i
      }, void 0 !== r.offset ? parseOffsetNano(r.offset) : void 0),
      timeZone: a
    };
  }
  return {
    ...t.ee(r),
    ...At
  };
}

function Ae(e, n, t, o, r, i) {
  const a = refineCalendarFields(t, r, _u, Au, ju), s = e(a.timeZone), [c, u, l] = je(i), f = t.ee(a, fabricateOverflowOptions(c)), d = refineTimeBag(a, c);
  return Xe(getMatchingInstantFor(n(s), {
    ...f,
    ...d
  }, void 0 !== a.offset ? parseOffsetNano(a.offset) : void 0, u, l), s, o);
}

function Nt(e, n, t) {
  const o = refineCalendarFields(e, n, _u, [], p), r = dt(t);
  return jt(Do({
    ...e.ee(o, fabricateOverflowOptions(r)),
    ...refineTimeBag(o, r)
  }));
}

function de(e, n, t, o = []) {
  const r = refineCalendarFields(e, n, _u, o);
  return e.ee(r, t);
}

function Ut(e, n, t, o) {
  const r = refineCalendarFields(e, n, Gu, o);
  return e.ne(r, t);
}

function Rt(e, n, t, o) {
  const r = refineCalendarFields(e, t, _u, Hu);
  return n && void 0 !== r.month && void 0 === r.monthCode && void 0 === r.year && (r.year = Pl), 
  e.te(r, o);
}

function Tt(e, n) {
  return St(refineTimeBag(refineFields(e, Ru, [], 1), dt(n)));
}

function q(e) {
  const n = refineFields(e, il);
  return pe(checkDurationUnits({
    ...ll,
    ...n
  }));
}

function refineCalendarFields(e, n, t, o = [], r = []) {
  return refineFields(n, [ ...e.fields(t), ...r ].sort(), o);
}

function refineFields(e, n, t, o = !t) {
  const r = {};
  let i, a = 0;
  for (const o of n) {
    if (o === i) {
      throw new RangeError(duplicateFields(o));
    }
    if ("constructor" === o || "__proto__" === o) {
      throw new RangeError(forbiddenField(o));
    }
    let n = e[o];
    if (void 0 !== n) {
      a = 1, Rm[o] && (n = Rm[o](n, o)), r[o] = n;
    } else if (t) {
      if (t.includes(o)) {
        throw new TypeError(missingField(o));
      }
      r[o] = Qu[o];
    }
    i = o;
  }
  if (o && !a) {
    throw new TypeError(noValidFields(n));
  }
  return r;
}

function refineTimeBag(e, n) {
  return constrainIsoTimeFields(zm({
    ...Qu,
    ...e
  }), n);
}

function De(e, n, t, o, r) {
  const {calendar: i, timeZone: a} = t, s = e(i), c = n(a), u = [ ...s.fields(_u), ...Uu ].sort(), l = (e => {
    const n = he(e, L), t = Se(n.offsetNanoseconds), o = ra(e.calendar), [r, i, a] = o.u(n), [s, c] = o.m(r, i), u = sa(s, c);
    return {
      ...Ga(n),
      year: r,
      monthCode: u,
      day: a,
      offset: t
    };
  })(t), f = refineFields(o, u), d = s.oe(l, f), m = {
    ...l,
    ...f
  }, [p, h, I] = je(r, 2);
  return Xe(getMatchingInstantFor(c, {
    ...s.ee(d, fabricateOverflowOptions(p)),
    ...constrainIsoTimeFields(zm(m), p)
  }, parseOffsetNano(m.offset), h, I), a, i);
}

function Pt(e, n, t, o) {
  const r = e(n.calendar), i = [ ...r.fields(_u), ...p ].sort(), a = {
    ...computeDateEssentials(s = n),
    hour: s.isoHour,
    minute: s.isoMinute,
    second: s.isoSecond,
    millisecond: s.isoMillisecond,
    microsecond: s.isoMicrosecond,
    nanosecond: s.isoNanosecond
  };
  var s;
  const c = refineFields(t, i), u = dt(o), l = r.oe(a, c), f = {
    ...a,
    ...c
  };
  return jt(Do({
    ...r.ee(l, fabricateOverflowOptions(u)),
    ...constrainIsoTimeFields(zm(f), u)
  }));
}

function ee(e, n, t, o) {
  const r = e(n.calendar), i = r.fields(_u).sort(), a = computeDateEssentials(n), s = refineFields(t, i), c = r.oe(a, s);
  return r.ee(c, o);
}

function Wt(e, n, t, o) {
  const r = e(n.calendar), i = r.fields(Gu).sort(), a = (e => {
    const n = ra(e.calendar), [t, o] = n.u(e), [r, i] = n.m(t, o);
    return {
      year: t,
      monthCode: sa(r, i)
    };
  })(n), s = refineFields(t, i), c = r.oe(a, s);
  return r.ne(c, o);
}

function Et(e, n, t, o) {
  const r = e(n.calendar), i = r.fields(_u).sort(), a = (e => {
    const n = ra(e.calendar), [t, o, r] = n.u(e), [i, a] = n.m(t, o);
    return {
      monthCode: sa(i, a),
      day: r
    };
  })(n), s = refineFields(t, i), c = r.oe(a, s);
  return r.te(c, o);
}

function rt(e, n, t) {
  return St(((e, n, t) => refineTimeBag({
    ...nn(Ru, e),
    ...refineFields(n, Ru)
  }, dt(t)))(e, n, t));
}

function N(e, n) {
  return pe((t = e, o = n, checkDurationUnits({
    ...t,
    ...refineFields(o, il)
  })));
  var t, o;
}

function convertToPlainMonthDay(e, n) {
  const t = refineCalendarFields(e, n, Ku);
  return e.te(t);
}

function convertToPlainYearMonth(e, n, t) {
  const o = refineCalendarFields(e, n, Vu);
  return e.ne(o, t);
}

function convertToIso(e, n, t, o, r) {
  n = nn(t = e.fields(t), n), o = refineFields(o, r = e.fields(r), []);
  let i = e.oe(n, o);
  return i = refineFields(i, [ ...t, ...r ].sort(), []), e.ee(i);
}

function nativeDateFromFields(e, n) {
  const t = dt(n), o = refineYear(this, e), r = refineMonth(this, e, o, t), i = refineDay(this, e, r, o, t);
  return W(To(this.U(o, r, i)), this.id || l);
}

function nativeYearMonthFromFields(e, n) {
  const t = dt(n), o = refineYear(this, e), r = refineMonth(this, e, o, t);
  return createPlainYearMonthSlots(checkIsoYearMonthInBounds(this.U(o, r, 1)), this.id || l);
}

function nativeMonthDayFromFields(e, n) {
  const t = dt(n);
  let o, r, i, a = void 0 !== e.eraYear || void 0 !== e.year ? refineYear(this, e) : void 0;
  const s = !this.id;
  if (void 0 === a && s && (a = Pl), void 0 !== a) {
    const n = refineMonth(this, e, a, t);
    o = refineDay(this, e, n, a, t);
    const s = this.F(a);
    r = monthToMonthCodeNumber(n, s), i = n === s;
  } else {
    if (void 0 === e.monthCode) {
      throw new TypeError(lu);
    }
    if ([r, i] = parseMonthCode(e.monthCode), this.id && this.id !== Xu && this.id !== el) {
      if (this.id && "coptic" === computeCalendarIdBase(this.id) && 0 === t) {
        const n = i || 13 !== r ? 30 : 6;
        o = e.day, o = clampNumber(o, 1, n);
      } else if (this.id && "chinese" === computeCalendarIdBase(this.id) && 0 === t) {
        const n = !i || 1 !== r && 9 !== r && 10 !== r && 11 !== r && 12 !== r ? 30 : 29;
        o = e.day, o = clampNumber(o, 1, n);
      } else {
        o = e.day;
      }
    } else {
      o = refineDay(this, e, refineMonth(this, e, Pl, t), Pl, t);
    }
  }
  const c = this.R(r, i, o);
  if (!c) {
    throw new RangeError("Cannot guess year");
  }
  const [u, f] = c;
  return createPlainMonthDaySlots(To(this.U(u, f, o)), this.id || l);
}

function nativeFieldsMethod(e) {
  return getCalendarEraOrigins(this) && e.includes("year") ? [ ...e, ...qu ] : e;
}

function nativeMergeFields(e, n) {
  const t = Object.assign(Object.create(null), e);
  return spliceFields(t, n, $u), getCalendarEraOrigins(this) && (spliceFields(t, n, Lu), 
  this.id === el && spliceFields(t, n, Ju, qu)), t;
}

function refineYear(e, n) {
  const t = getCalendarEraOrigins(e), o = tl[e.id || ""] || {};
  let {era: r, eraYear: i, year: a} = n;
  if (void 0 !== r || void 0 !== i) {
    if (void 0 === r || void 0 === i) {
      throw new TypeError(su);
    }
    if (!t) {
      throw new RangeError(iu);
    }
    const e = t[o[r] || r];
    if (void 0 === e) {
      throw new RangeError(invalidEra(r));
    }
    const n = eraYearToYear(i, e);
    if (void 0 !== a && a !== n) {
      throw new RangeError(cu);
    }
    a = n;
  } else if (void 0 === a) {
    throw new TypeError(missingYear(t));
  }
  return a;
}

function refineMonth(e, n, t, o) {
  let {month: r, monthCode: i} = n;
  if (void 0 !== i) {
    const n = ((e, n, t, o) => {
      const r = e.F(t), [i, a] = parseMonthCode(n);
      let s = monthCodeNumberToMonth(i, a, r);
      if (a) {
        const n = getCalendarLeapMonthMeta(e);
        if (void 0 === n) {
          throw new RangeError(fu);
        }
        if (n > 0) {
          if (s > n) {
            throw new RangeError(fu);
          }
          if (void 0 === r) {
            if (1 === o) {
              throw new RangeError(fu);
            }
            s--;
          }
        } else {
          if (s !== -n) {
            throw new RangeError(fu);
          }
          if (void 0 === r && 1 === o) {
            throw new RangeError(fu);
          }
        }
      }
      return s;
    })(e, i, t, o);
    if (void 0 !== r && r !== n) {
      throw new RangeError(uu);
    }
    r = n, o = 1;
  } else if (void 0 === r) {
    throw new TypeError(lu);
  }
  return ba("month", r, 1, e.O(t), o);
}

function refineDay(e, n, t, o, r) {
  return clampProp(n, "day", 1, e.B(o, t), r);
}

function spliceFields(e, n, t, o) {
  let r = 0;
  const i = [];
  for (const e of t) {
    void 0 !== n[e] ? r = 1 : i.push(e);
  }
  if (Object.assign(e, n), r) {
    for (const n of o || i) {
      delete e[n];
    }
  }
}

function computeDateEssentials(e) {
  const n = ra(e.calendar), [t, o, r] = n.u(e), [i, a] = n.m(t, o);
  return {
    year: t,
    monthCode: sa(i, a),
    day: r
  };
}

function qe(e) {
  return xe(io(bigIntToBigNano(toBigInt(e))));
}

function ye(e, n, t, o, r = l) {
  return Xe(io(bigIntToBigNano(toBigInt(t))), n(o), e(r));
}

function Mt(n, t, o, r, i = 0, a = 0, s = 0, c = 0, u = 0, f = 0, d = l) {
  return jt(Do(checkIsoDateTimeFields(e(Za, zipProps(pl, [ t, o, r, i, a, s, c, u, f ])))), n(d));
}

function ue(n, t, o, r, i = l) {
  return W(To(checkIsoDateFields(e(Za, {
    isoYear: t,
    isoMonth: o,
    isoDay: r
  }))), n(i));
}

function Kt(e, n, t, o = l, r = 1) {
  const i = Za(n), a = Za(t), s = e(o);
  return createPlainYearMonthSlots(checkIsoYearMonthInBounds(checkIsoDateFields({
    isoYear: i,
    isoMonth: a,
    isoDay: Za(r)
  })), s);
}

function kt(e, n, t, o = l, r = Pl) {
  const i = Za(n), a = Za(t), s = e(o);
  return createPlainMonthDaySlots(To(checkIsoDateFields({
    isoYear: Za(r),
    isoMonth: i,
    isoDay: a
  })), s);
}

function ut(n = 0, t = 0, o = 0, r = 0, i = 0, a = 0) {
  return St(constrainIsoTimeFields(e(Za, zipProps(w, [ n, t, o, r, i, a ])), 1));
}

function j(n = 0, t = 0, o = 0, r = 0, i = 0, a = 0, s = 0, c = 0, u = 0, l = 0) {
  return pe(checkDurationUnits(e(Ba, zipProps(O, [ n, t, o, r, i, a, s, c, u, l ]))));
}

function Je(e, n, t = l) {
  return Xe(e.epochNanoseconds, n, t);
}

function Ce(e) {
  return xe(e.epochNanoseconds);
}

function yt(e, n) {
  return jt(he(n, e));
}

function fe(e, n) {
  return W(he(n, e));
}

function Qa(e, n, t) {
  return convertToPlainYearMonth(e(n.calendar), t);
}

function Xa(e, n, t) {
  return convertToPlainMonthDay(e(n.calendar), t);
}

function mt(e, n) {
  return St(he(n, e));
}

function Ct(e, n, t, o) {
  const r = ((e, n, t, o) => {
    const r = (e => Vl(normalizeOptions(e)))(o);
    return $o(e(n), t, r);
  })(e, t, n, o);
  return Xe(io(r), t, n.calendar);
}

function po(e, n, t) {
  const o = e(n.calendar);
  return createPlainYearMonthSlots({
    ...n,
    ...convertToPlainYearMonth(o, t)
  });
}

function yo(e, n, t) {
  return convertToPlainMonthDay(e(n.calendar), t);
}

function ae(e, n, t, o, r) {
  const i = e(r.timeZone), a = r.plainTime, s = void 0 !== a ? n(a) : void 0, c = t(i);
  let u;
  return u = s ? $o(c, {
    ...o,
    ...s
  }) : getStartOfDayInstantFor(c, {
    ...o,
    ...At
  }), Xe(u, i, o.calendar);
}

function ie(e, n = At) {
  return jt(Do({
    ...e,
    ...n
  }));
}

function le(e, n, t) {
  return convertToPlainYearMonth(e(n.calendar), t);
}

function se(e, n, t) {
  return convertToPlainMonthDay(e(n.calendar), t);
}

function $t(e, n, t, o) {
  return ((e, n, t) => convertToIso(e, n, Vu, oa(t), Hu))(e(n.calendar), t, o);
}

function Vt(e, n, t, o) {
  return ((e, n, t) => convertToIso(e, n, Ku, oa(t), Wu))(e(n.calendar), t, o);
}

function vo(e, n, t, o, r) {
  const i = oa(r), a = n(i.re), s = e(i.timeZone);
  return Xe($o(t(s), {
    ...a,
    ...o
  }), s, a.calendar);
}

function Oo(e, n) {
  return jt(Do({
    ...e,
    ...n
  }));
}

function ea(e) {
  return xe(io(Ge(Ba(e), oo)));
}

function ze(e) {
  return xe(io(Ge(Ba(e), Ke)));
}

function na(e) {
  return xe(io(bigIntToBigNano(toBigInt(e), ro)));
}

function $e(e) {
  return xe(io(bigIntToBigNano(toBigInt(e))));
}

function createOptionsTransformer(e, n, t) {
  const o = new Set(t);
  return (r, i) => {
    const a = t && hasAnyPropsByName(r, t);
    if (!hasAnyPropsByName(r = ((e, n) => {
      const t = {};
      for (const o in n) {
        e.has(o) || (t[o] = n[o]);
      }
      return t;
    })(o, r), e)) {
      if (i && a) {
        throw new TypeError("Invalid formatting options");
      }
      r = {
        ...n,
        ...r
      };
    }
    return t && (r.timeZone = nf, [ "full", "long" ].includes(r.ie) && (r.ie = "medium")), 
    r;
  };
}

function K(e, n = an, t = 0) {
  const [o, , , r] = e;
  return (i, a = mp, ...s) => {
    const c = n(r && r(...s), i, a, o, t), u = c.resolvedOptions();
    return [ c, ...toEpochMillis(e, u, s) ];
  };
}

function an(e, n, t, o, r) {
  if (t = o(t, r), e) {
    if (void 0 !== t.timeZone) {
      throw new TypeError(Ou);
    }
    t.timeZone = e;
  }
  return new en(n, t);
}

function computeNonBuggyIsoResolve() {
  return new en(void 0, {
    calendar: l
  }).resolvedOptions().calendar === l;
}

function toEpochMillis(e, n, t) {
  const [, o, r] = e;
  return t.map((e => (e.calendar && ((e, n, t) => {
    if ((t || e !== l) && e !== n) {
      throw new RangeError(mu);
    }
  })(e.calendar, n.calendar, r), o(e, n))));
}

function Pe(e, n, t) {
  const o = n.timeZone, r = e(o), i = {
    ...he(n, r),
    ...t || At
  };
  let a;
  return a = t ? getMatchingInstantFor(r, i, i.offsetNanoseconds, 2) : getStartOfDayInstantFor(r, i), 
  Xe(a, o, n.calendar);
}

function Ka(e, n, t) {
  const o = n.timeZone, r = e(o), i = {
    ...he(n, r),
    ...t
  }, a = getPreferredCalendarId(n.calendar, t.calendar);
  return Xe(getMatchingInstantFor(r, i, i.offsetNanoseconds, 2), o, a);
}

function pt(e, n = At) {
  return jt(Do({
    ...e,
    ...n
  }));
}

function Mo(e, n) {
  return jt({
    ...e,
    ...n
  }, getPreferredCalendarId(e.calendar, n.calendar));
}

function Ot(e, n) {
  return {
    ...e,
    calendar: n
  };
}

function ge(e, n) {
  return {
    ...e,
    timeZone: n
  };
}

function getPreferredCalendarId(e, n) {
  if (e === n) {
    return e;
  }
  if (e === n || e === l) {
    return n;
  }
  if (n === l) {
    return e;
  }
  throw new RangeError(mu);
}

function tn(e) {
  const n = Ue();
  return So(n, e.N(n));
}

function Ue() {
  return Ge(Date.now(), Ke);
}

function Qe() {
  return (new en).resolvedOptions().timeZone;
}

const expectedInteger = (e, n) => `Non-integer ${e}: ${n}`, expectedPositive = (e, n) => `Non-positive ${e}: ${n}`, expectedFinite = (e, n) => `Non-finite ${e}: ${n}`, forbiddenBigIntToNumber = e => `Cannot convert bigint to ${e}`, invalidBigInt = e => `Invalid bigint: ${e}`, ou = "Cannot convert Symbol to string", ru = "Invalid object", numberOutOfRange = (e, n, t, o, r) => r ? numberOutOfRange(e, r[n], r[t], r[o]) : invalidEntity(e, n) + `; must be between ${t}-${o}`, invalidEntity = (e, n) => `Invalid ${e}: ${n}`, missingField = e => `Missing ${e}`, forbiddenField = e => `Invalid field ${e}`, duplicateFields = e => `Duplicate field ${e}`, noValidFields = e => "No valid fields: " + e.join(), i = "Invalid bag", invalidChoice = (e, n, t) => invalidEntity(e, n) + "; must be " + Object.keys(t).join(), C = "Cannot use valueOf", a = "Invalid calling context", iu = "Forbidden era/eraYear", su = "Mismatching era/eraYear", cu = "Mismatching year/eraYear", invalidEra = e => `Invalid era: ${e}`, missingYear = e => "Missing year" + (e ? "/era/eraYear" : ""), invalidMonthCode = e => `Invalid monthCode: ${e}`, uu = "Mismatching month/monthCode", lu = "Missing month/monthCode", fu = "Invalid leap month", du = "Invalid protocol results", c = e => invalidEntity("Calendar", e), mu = "Mismatching Calendars", qa = "Calendar week operations forbidden", F = e => invalidEntity("TimeZone", e), pu = "Mismatching TimeZones", hu = "Forbidden ICU TimeZone", Iu = "Out-of-bounds offset", Du = "Out-of-bounds TimeZone gap", gu = "Invalid TimeZone offset", Tu = "Ambiguous offset", Mu = "Out-of-bounds date", yu = "Out-of-bounds duration", Nu = "Cannot mix duration signs", vu = "Missing relativeTo", Pu = "Cannot use large units", Fu = "Required smallestUnit or largestUnit", Eu = "smallestUnit > largestUnit", failedParse = e => `Cannot parse: ${e}`, invalidSubstring = e => `Invalid substring: ${e}`, rn = e => `Cannot format ${e}`, ln = "Mismatching types for formatting", Ou = "Cannot specify TimeZone", bu = /*@__PURE__*/ gt(P, ((e, n) => n)), Su = /*@__PURE__*/ gt(P, ((e, n, t) => t)), wu = /*@__PURE__*/ gt(padNumber, 2), Bu = {
  nanosecond: 0,
  microsecond: 1,
  millisecond: 2,
  second: 3,
  minute: 4,
  hour: 5,
  day: 6,
  week: 7,
  month: 8,
  year: 9
}, Yu = /*@__PURE__*/ Object.keys(Bu), Cu = 864e5, ku = 1e3, ro = 1e3, Ke = 1e6, oo = 1e9, ao = 6e10, no = 36e11, go = 864e11, Zu = [ 1, ro, Ke, oo, ao, no, go ], p = /*@__PURE__*/ Yu.slice(0, 6), Ru = /*@__PURE__*/ sortStrings(p), zu = [ "offset" ], Au = [ "timeZone" ], Uu = /*@__PURE__*/ p.concat(zu), ju = /*@__PURE__*/ Uu.concat(Au), qu = [ "era", "eraYear" ], Lu = /*@__PURE__*/ qu.concat([ "year" ]), Wu = [ "year" ], xu = [ "monthCode" ], $u = /*@__PURE__*/ [ "month" ].concat(xu), Hu = [ "day" ], Gu = /*@__PURE__*/ $u.concat(Wu), Vu = /*@__PURE__*/ xu.concat(Wu), _u = /*@__PURE__*/ Hu.concat(Gu), Ju = /*@__PURE__*/ Hu.concat($u), Ku = /*@__PURE__*/ Hu.concat(xu), Qu = /*@__PURE__*/ Su(p, 0), l = "iso8601", Xu = "gregory", el = "japanese", nl = {
  [Xu]: {
    "gregory-inverse": -1,
    gregory: 0
  },
  [el]: {
    "japanese-inverse": -1,
    japanese: 0,
    meiji: 1867,
    taisho: 1911,
    showa: 1925,
    heisei: 1988,
    reiwa: 2018
  },
  ethiopic: {
    ethioaa: 0,
    ethiopic: 5500
  },
  coptic: {
    "coptic-inverse": -1,
    coptic: 0
  },
  roc: {
    "roc-inverse": -1,
    roc: 0
  },
  buddhist: {
    be: 0
  },
  islamic: {
    ah: 0
  },
  indian: {
    saka: 0
  },
  persian: {
    ap: 0
  }
}, tl = {
  [Xu]: {
    bce: "gregory-inverse",
    ce: "gregory"
  },
  [el]: {
    bce: "japanese-inverse",
    ce: "japanese"
  },
  ethiopic: {
    era0: "ethioaa",
    era1: "ethiopic"
  },
  coptic: {
    era0: "coptic-inverse",
    era1: "coptic"
  },
  roc: {
    broc: "roc-inverse",
    minguo: "roc"
  }
}, ol = {
  chinese: 13,
  dangi: 13,
  hebrew: -6
}, d = /*@__PURE__*/ gt(requireType, "string"), D = /*@__PURE__*/ gt(requireType, "boolean"), rl = /*@__PURE__*/ gt(requireType, "number"), O = /*@__PURE__*/ Yu.map((e => e + "s")), il = /*@__PURE__*/ sortStrings(O), al = /*@__PURE__*/ O.slice(0, 6), sl = /*@__PURE__*/ O.slice(6), cl = /*@__PURE__*/ sl.slice(1), ul = /*@__PURE__*/ bu(O), ll = /*@__PURE__*/ Su(O, 0), fl = /*@__PURE__*/ Su(al, 0), dl = /*@__PURE__*/ gt(zeroOutProps, O), w = [ "isoNanosecond", "isoMicrosecond", "isoMillisecond", "isoSecond", "isoMinute", "isoHour" ], ml = [ "isoDay", "isoMonth", "isoYear" ], pl = /*@__PURE__*/ w.concat(ml), Ca = /*@__PURE__*/ sortStrings(ml), hl = /*@__PURE__*/ sortStrings(w), Il = /*@__PURE__*/ sortStrings(pl), At = /*@__PURE__*/ Su(hl, 0), Ra = /*@__PURE__*/ gt(zeroOutProps, pl), Dl = 1e8, gl = Dl * Cu, Tl = [ Dl, 0 ], Ml = [ -Dl, 0 ], yl = 275760, Nl = -271821, en = Intl.DateTimeFormat, vl = 1970, Pl = 1972, Fl = 12, El = /*@__PURE__*/ isoArgsToEpochMilli(1868, 9, 8), Ol = /*@__PURE__*/ on(computeJapaneseEraParts, WeakMap), bl = "smallestUnit", Sl = "unit", wl = "roundingMode", Bl = "roundingIncrement", Yl = "fractionalSecondDigits", Cl = "relativeTo", kl = "direction", Zl = {
  constrain: 0,
  reject: 1
}, Rl = /*@__PURE__*/ Object.keys(Zl), zl = {
  compatible: 0,
  reject: 1,
  earlier: 2,
  later: 3
}, Al = {
  reject: 0,
  use: 1,
  prefer: 2,
  ignore: 3
}, Ul = {
  auto: 0,
  never: 1,
  critical: 2,
  always: 3
}, jl = {
  auto: 0,
  never: 1,
  critical: 2
}, ql = {
  auto: 0,
  never: 1
}, Ll = {
  floor: 0,
  halfFloor: 1,
  ceil: 2,
  halfCeil: 3,
  trunc: 4,
  halfTrunc: 5,
  expand: 6,
  halfExpand: 7,
  halfEven: 8
}, Wl = {
  previous: -1,
  next: 1
}, xl = /*@__PURE__*/ gt(refineUnitOption, bl), $l = /*@__PURE__*/ gt(refineUnitOption, "largestUnit"), Hl = /*@__PURE__*/ gt(refineUnitOption, Sl), Gl = /*@__PURE__*/ gt(refineChoiceOption, "overflow", Zl), Vl = /*@__PURE__*/ gt(refineChoiceOption, "disambiguation", zl), _l = /*@__PURE__*/ gt(refineChoiceOption, "offset", Al), Jl = /*@__PURE__*/ gt(refineChoiceOption, "calendarName", Ul), Kl = /*@__PURE__*/ gt(refineChoiceOption, "timeZoneName", jl), Ql = /*@__PURE__*/ gt(refineChoiceOption, "offset", ql), Xl = /*@__PURE__*/ gt(refineChoiceOption, wl, Ll), Qt = "PlainYearMonth", qt = "PlainMonthDay", G = "PlainDate", x = "PlainDateTime", ft = "PlainTime", _ = "ZonedDateTime", Re = "Instant", A = "Duration", ef = [ Math.floor, e => hasHalf(e) ? Math.floor(e) : Math.round(e), Math.ceil, e => hasHalf(e) ? Math.ceil(e) : Math.round(e), Math.trunc, e => hasHalf(e) ? Math.trunc(e) || 0 : Math.round(e), e => e < 0 ? Math.floor(e) : Math.ceil(e), e => Math.sign(e) * Math.round(Math.abs(e)) || 0, e => hasHalf(e) ? (e = Math.trunc(e) || 0) + e % 2 : Math.round(e) ], nf = "UTC", tf = 5184e3, of = /*@__PURE__*/ isoArgsToEpochSec(1847), rf = /*@__PURE__*/ isoArgsToEpochSec((() => {
  const e = new Date;
  return (0 === e.getTime() ? 2040 : e.getUTCFullYear()) + 10;
})()), af = /0+$/, he = /*@__PURE__*/ on(_zonedEpochSlotsToIso, WeakMap), sf = 2 ** 32 - 1, L = /*@__PURE__*/ on((e => {
  const n = getTimeZoneEssence(e);
  return "object" == typeof n ? new IntlTimeZone(n) : new FixedTimeZone(n || 0);
}));

class FixedTimeZone {
  constructor(e) {
    this.j = e;
  }
  N() {
    return this.j;
  }
  v(e) {
    return (e => {
      const n = ma({
        ...e,
        ...At
      });
      if (!n || Math.abs(n[0]) > 1e8) {
        throw new RangeError(Mu);
      }
    })(e), [ isoToEpochNanoWithOffset(e, this.j) ];
  }
  l() {}
}

class IntlTimeZone {
  constructor(e) {
    this.ae = (e => {
      function getOffsetSec(e) {
        const i = clampNumber(e, o, r), [a, s] = computePeriod(i), c = n(a), u = n(s);
        return c === u ? c : pinch(t(a, s), c, u, e);
      }
      function pinch(n, t, o, r) {
        let i, a;
        for (;(void 0 === r || void 0 === (i = r < n[0] ? t : r >= n[1] ? o : void 0)) && (a = n[1] - n[0]); ) {
          const t = n[0] + Math.floor(a / 2);
          e(t) === o ? n[1] = t : n[0] = t + 1;
        }
        return i;
      }
      const n = on(e), t = on(createSplitTuple);
      let o = of, r = rf;
      return {
        se(e) {
          const n = getOffsetSec(e - 86400), t = getOffsetSec(e + 86400), o = e - n, r = e - t;
          if (n === t) {
            return [ o ];
          }
          const i = getOffsetSec(o);
          return i === getOffsetSec(r) ? [ e - i ] : n > t ? [ o, r ] : [];
        },
        ue: getOffsetSec,
        l(e, i) {
          const a = clampNumber(e, o, r);
          let [s, c] = computePeriod(a);
          const u = tf * i, l = i < 0 ? () => c > o || (o = a, 0) : () => s < r || (r = a, 
          0);
          for (;l(); ) {
            const o = n(s), r = n(c);
            if (o !== r) {
              const n = t(s, c);
              pinch(n, o, r);
              const a = n[0];
              if ((compareNumbers(a, e) || 1) === i) {
                return a;
              }
            }
            s += u, c += u;
          }
        }
      };
    })((e => n => {
      const t = hashIntlFormatParts(e, n * ku);
      return isoArgsToEpochSec(parseIntlPartsYear(t), parseInt(t.month), parseInt(t.day), parseInt(t.hour), parseInt(t.minute), parseInt(t.second)) - n;
    })(e));
  }
  N(e) {
    return this.ae.ue(epochNanoToSec(e)) * oo;
  }
  v(e) {
    const [n, t] = [ isoArgsToEpochSec((o = e).isoYear, o.isoMonth, o.isoDay, o.isoHour, o.isoMinute, o.isoSecond), o.isoMillisecond * Ke + o.isoMicrosecond * ro + o.isoNanosecond ];
    var o;
    return this.ae.se(n).map((e => io(Ta(Ge(e, oo), t))));
  }
  l(e, n) {
    const [t, o] = epochNanoToSecMod(e), r = this.ae.l(t + (n > 0 || o ? 1 : 0), n);
    if (void 0 !== r) {
      return Ge(r, oo);
    }
  }
}

const cf = "([+-])", uf = "(?:[.,](\\d{1,9}))?", lf = `(?:(?:${cf}(\\d{6}))|(\\d{4}))-?(\\d{2})`, ff = "(\\d{2})(?::?(\\d{2})(?::?(\\d{2})" + uf + ")?)?", df = cf + ff, mf = lf + "-?(\\d{2})(?:[T ]" + ff + "(Z|" + df + ")?)?", pf = "\\[(!?)([^\\]]*)\\]", hf = `((?:${pf}){0,9})`, If = /*@__PURE__*/ createRegExp(lf + hf), Df = /*@__PURE__*/ createRegExp("(?:--)?(\\d{2})-?(\\d{2})" + hf), gf = /*@__PURE__*/ createRegExp(mf + hf), Tf = /*@__PURE__*/ createRegExp("T?" + ff + "(?:" + df + ")?" + hf), Mf = /*@__PURE__*/ createRegExp(df), yf = /*@__PURE__*/ new RegExp(pf, "g"), Nf = /*@__PURE__*/ createRegExp(`${cf}?P(\\d+Y)?(\\d+M)?(\\d+W)?(\\d+D)?(?:T(?:(\\d+)${uf}H)?(?:(\\d+)${uf}M)?(?:(\\d+)${uf}S)?)?`), vf = /*@__PURE__*/ on((e => new en("en", {
  calendar: l,
  timeZone: e,
  era: "short",
  year: "numeric",
  month: "numeric",
  day: "numeric",
  hour: "numeric",
  minute: "numeric",
  second: "numeric",
  hour12: 0
}))), Pf = /^(AC|AE|AG|AR|AS|BE|BS|CA|CN|CS|CT|EA|EC|IE|IS|JS|MI|NE|NS|PL|PN|PR|PS|SS|VS)T$/, Ff = /[^\w\/:+-]+/, Ef = /^M(\d{2})(L?)$/, Of = /*@__PURE__*/ on(createIntlCalendar), bf = /*@__PURE__*/ on((e => new en("en", {
  calendar: e,
  timeZone: nf,
  era: "short",
  year: "numeric",
  month: "short",
  day: "numeric",
  hour12: 0
}))), Sf = {
  ne: nativeYearMonthFromFields,
  fields: nativeFieldsMethod
}, wf = {
  ee: nativeDateFromFields,
  fields: nativeFieldsMethod
}, Bf = {
  te: nativeMonthDayFromFields,
  fields: nativeFieldsMethod
}, Yf = {
  P: nativeDateAdd
}, Cf = {
  P: nativeDateAdd,
  h: nativeDateUntil
}, kf = {
  P: nativeDateAdd,
  h: nativeDateUntil,
  ee: nativeDateFromFields,
  ne: nativeYearMonthFromFields,
  te: nativeMonthDayFromFields,
  fields: nativeFieldsMethod,
  oe: nativeMergeFields,
  inLeapYear: computeNativeInLeapYear,
  monthsInYear: computeNativeMonthsInYear,
  daysInMonth: computeNativeDaysInMonth,
  daysInYear: computeNativeDaysInYear,
  dayOfYear: computeNativeDayOfYear,
  era(e) {
    return this.$(e)[0];
  },
  eraYear(e) {
    return this.$(e)[1];
  },
  monthCode(e) {
    const [n, t] = this.u(e), [o, r] = this.m(n, t);
    return sa(o, r);
  },
  dayOfWeek: Ha,
  daysInWeek: fo
}, Zf = {
  F: noop,
  O: computeIsoMonthsInYear,
  U: computeIsoFieldsFromParts
}, Rf = /*@__PURE__*/ Object.assign({}, Zf, {
  B: computeIsoDaysInMonth
}), zf = /*@__PURE__*/ Object.assign({}, Rf, {
  R: computeIsoYearMonthForMonthDay
}), Af = /*@__PURE__*/ Object.assign({}, Sf, Zf), Uf = /*@__PURE__*/ Object.assign({}, wf, zf), jf = /*@__PURE__*/ Object.assign({}, Bf, zf), qf = /*@__PURE__*/ Object.assign({}, Af, {
  oe: nativeMergeFields
}), Lf = /*@__PURE__*/ Object.assign({}, Uf, {
  oe: nativeMergeFields
}), Wf = /*@__PURE__*/ Object.assign({}, jf, {
  oe: nativeMergeFields
}), xf = {
  u: computeIsoDateParts,
  M: isoArgsToEpochMilli,
  p: isoMonthAdd
}, $f = /*@__PURE__*/ Object.assign({}, xf, {
  m: computeIsoMonthCodeParts,
  O: computeIsoMonthsInYear,
  B: computeIsoDaysInMonth,
  F: noop
}), Hf = /*@__PURE__*/ Object.assign({}, Yf, $f), Gf = /*@__PURE__*/ Object.assign({}, Cf, $f, {
  q: computeIsoMonthsInYearSpan
}), Vf = {
  day: computeIsoDay
}, _f = /*@__PURE__*/ Object.assign({}, Hf, Vf), Jf = /*@__PURE__*/ Object.assign({}, Gf, Vf), Kf = {
  u: computeIsoDateParts,
  $: computeIsoEraParts,
  m: computeIsoMonthCodeParts
}, Qf = {
  inLeapYear: computeNativeInLeapYear,
  u: computeIsoDateParts,
  L: computeIsoInLeapYear
}, Xf = {
  monthsInYear: computeNativeMonthsInYear,
  u: computeIsoDateParts,
  O: computeIsoMonthsInYear
}, em = {
  daysInMonth: computeNativeDaysInMonth,
  u: computeIsoDateParts,
  B: computeIsoDaysInMonth
}, nm = {
  daysInYear: computeNativeDaysInYear,
  u: computeIsoDateParts,
  G: computeIsoDaysInYear
}, tm = {
  dayOfYear: computeNativeDayOfYear,
  u: computeIsoDateParts,
  M: isoArgsToEpochMilli
}, om = /*@__PURE__*/ Object.assign({}, tm, {
  weekOfYear: computeNativeWeekOfYear,
  yearOfWeek: computeNativeYearOfWeek,
  I(e) {
    function computeWeekShift(e) {
      return (7 - e < n ? 7 : 0) - e;
    }
    function computeWeeksInYear(e) {
      const n = computeIsoDaysInYear(l + e), t = e || 1, o = computeWeekShift(modFloor(a + n * t, 7));
      return c = (n + (o - s) * t) / 7;
    }
    const n = this.id ? 1 : 4, t = Ha(e), o = this.dayOfYear(e), r = modFloor(t - 1, 7), i = o - 1, a = modFloor(r - i, 7), s = computeWeekShift(a);
    let c, u = Math.floor((i - s) / 7) + 1, l = e.isoYear;
    return u ? u > computeWeeksInYear(0) && (u = 1, l++) : (u = computeWeeksInYear(-1), 
    l--), [ u, l, c ];
  }
}), rm = {
  u: computeIsoDateParts,
  m: computeIsoMonthCodeParts,
  R: computeIsoYearMonthForMonthDay,
  U: computeIsoFieldsFromParts
}, im = /*@__PURE__*/ Object.assign({}, kf, om, {
  u: computeIsoDateParts,
  $: computeIsoEraParts,
  m: computeIsoMonthCodeParts,
  R: computeIsoYearMonthForMonthDay,
  L: computeIsoInLeapYear,
  F: noop,
  O: computeIsoMonthsInYear,
  q: computeIsoMonthsInYearSpan,
  B: computeIsoDaysInMonth,
  G: computeIsoDaysInYear,
  U: computeIsoFieldsFromParts,
  M: isoArgsToEpochMilli,
  p: isoMonthAdd,
  year(e) {
    return e.isoYear;
  },
  month(e) {
    return e.isoMonth;
  },
  day: computeIsoDay
}), am = {
  F: computeIntlLeapMonth,
  O: computeIntlMonthsInYear,
  U: computeIsoFieldsFromIntlParts
}, sm = /*@__PURE__*/ Object.assign({}, am, {
  B: computeIntlDaysInMonth
}), cm = /*@__PURE__*/ Object.assign({}, sm, {
  R: computeIntlYearMonthForMonthDay
}), um = /*@__PURE__*/ Object.assign({}, Sf, am), lm = /*@__PURE__*/ Object.assign({}, wf, sm), fm = /*@__PURE__*/ Object.assign({}, Bf, cm), dm = /*@__PURE__*/ Object.assign({}, um, {
  oe: nativeMergeFields
}), mm = /*@__PURE__*/ Object.assign({}, lm, {
  oe: nativeMergeFields
}), pm = /*@__PURE__*/ Object.assign({}, fm, {
  oe: nativeMergeFields
}), hm = {
  u: computeIntlDateParts,
  M: computeIntlEpochMilli,
  p: intlMonthAdd
}, Im = /*@__PURE__*/ Object.assign({}, hm, {
  m: computeIntlMonthCodeParts,
  O: computeIntlMonthsInYear,
  B: computeIntlDaysInMonth,
  F: computeIntlLeapMonth
}), Dm = /*@__PURE__*/ Object.assign({}, Yf, Im), gm = /*@__PURE__*/ Object.assign({}, Cf, Im, {
  q: computeIntlMonthsInYearSpan
}), Tm = {
  day: computeIntlDay
}, Mm = /*@__PURE__*/ Object.assign({}, Dm, Tm), ym = /*@__PURE__*/ Object.assign({}, gm, Tm), Nm = {
  u: computeIntlDateParts,
  $: computeIntlEraParts,
  m: computeIntlMonthCodeParts
}, vm = {
  inLeapYear: computeNativeInLeapYear,
  u: computeIntlDateParts,
  L: computeIntlInLeapYear
}, Pm = {
  monthsInYear: computeNativeMonthsInYear,
  u: computeIntlDateParts,
  O: computeIntlMonthsInYear
}, Fm = {
  daysInMonth: computeNativeDaysInMonth,
  u: computeIntlDateParts,
  B: computeIntlDaysInMonth
}, Em = {
  daysInYear: computeNativeDaysInYear,
  u: computeIntlDateParts,
  G: computeIntlDaysInYear
}, Om = {
  dayOfYear: computeNativeDayOfYear,
  u: computeIntlDateParts,
  M: computeIntlEpochMilli
}, bm = {
  I() {
    return [];
  }
}, Sm = /*@__PURE__*/ Object.assign({}, Om, bm, {
  weekOfYear: computeNativeWeekOfYear,
  yearOfWeek: computeNativeYearOfWeek
}), wm = {
  u: computeIntlDateParts,
  m: computeIntlMonthCodeParts,
  R: computeIntlYearMonthForMonthDay,
  U: computeIsoFieldsFromIntlParts
}, Bm = /*@__PURE__*/ Object.assign({}, kf, Sm, {
  u: computeIntlDateParts,
  $: computeIntlEraParts,
  m: computeIntlMonthCodeParts,
  R: computeIntlYearMonthForMonthDay,
  L: computeIntlInLeapYear,
  F: computeIntlLeapMonth,
  O: computeIntlMonthsInYear,
  q: computeIntlMonthsInYearSpan,
  B: computeIntlDaysInMonth,
  G: computeIntlDaysInYear,
  U: computeIsoFieldsFromIntlParts,
  M: computeIntlEpochMilli,
  p: intlMonthAdd,
  year(e) {
    return this._(e).year;
  },
  month(e) {
    const {year: n, V: t} = this._(e), {X: o} = this.J(n);
    return o[t] + 1;
  },
  day: computeIntlDay
}), Va = /*@__PURE__*/ createNativeOpsCreator(Af, um), Aa = /*@__PURE__*/ createNativeOpsCreator(Uf, lm), _a = /*@__PURE__*/ createNativeOpsCreator(jf, fm), Fo = /*@__PURE__*/ createNativeOpsCreator(qf, dm), mo = /*@__PURE__*/ createNativeOpsCreator(Lf, mm), Wo = /*@__PURE__*/ createNativeOpsCreator(Wf, pm), xa = /*@__PURE__*/ createNativeOpsCreator(xf, hm), Wa = /*@__PURE__*/ createNativeOpsCreator(Hf, Dm), Ia = /*@__PURE__*/ createNativeOpsCreator(Gf, gm), za = /*@__PURE__*/ createNativeOpsCreator(Vf, Tm), Yo = /*@__PURE__*/ createNativeOpsCreator(_f, Mm), Lo = /*@__PURE__*/ createNativeOpsCreator(Jf, ym), ra = /*@__PURE__*/ createNativeOpsCreator(Kf, Nm), ia = /*@__PURE__*/ createNativeOpsCreator(Qf, vm), ca = /*@__PURE__*/ createNativeOpsCreator(Xf, Pm), da = /*@__PURE__*/ createNativeOpsCreator(em, Fm), ua = /*@__PURE__*/ createNativeOpsCreator(nm, Em), la = /*@__PURE__*/ createNativeOpsCreator(tm, Om), $a = /*@__PURE__*/ createNativeOpsCreator(om, Sm), ko = /*@__PURE__*/ createNativeOpsCreator(rm, wm), v = /*@__PURE__*/ createNativeOpsCreator(im, Bm), Ym = {
  era: toStringViaPrimitive,
  eraYear: Za,
  year: Za,
  month: toPositiveInteger,
  monthCode(e) {
    const n = toStringViaPrimitive(e);
    return parseMonthCode(n), n;
  },
  day: toPositiveInteger
}, Cm = /*@__PURE__*/ Su(p, Za), km = /*@__PURE__*/ Su(O, Ba), Zm = {
  offset(e) {
    const n = toStringViaPrimitive(e);
    return parseOffsetNano(n), n;
  }
}, Rm = /*@__PURE__*/ Object.assign({}, Ym, Cm, km, Zm), zm = /*@__PURE__*/ gt(remapProps, p, w), Ga = /*@__PURE__*/ gt(remapProps, w, p), Am = "numeric", Um = [ "timeZoneName" ], jm = {
  month: Am,
  day: Am
}, qm = {
  year: Am,
  month: Am
}, Lm = /*@__PURE__*/ Object.assign({}, qm, {
  day: Am
}), Wm = {
  hour: Am,
  minute: Am,
  second: Am
}, xm = /*@__PURE__*/ Object.assign({}, Lm, Wm), $m = /*@__PURE__*/ Object.assign({}, xm, {
  timeZoneName: "short"
}), Hm = /*@__PURE__*/ Object.keys(qm), Gm = /*@__PURE__*/ Object.keys(jm), Vm = /*@__PURE__*/ Object.keys(Lm), _m = /*@__PURE__*/ Object.keys(Wm), Jm = [ "dateStyle" ], Km = /*@__PURE__*/ Hm.concat(Jm), Qm = /*@__PURE__*/ Gm.concat(Jm), Xm = /*@__PURE__*/ Vm.concat(Jm, [ "weekday" ]), ep = /*@__PURE__*/ _m.concat([ "dayPeriod", "timeStyle", "fractionalSecondDigits" ]), np = /*@__PURE__*/ Xm.concat(ep), tp = /*@__PURE__*/ Um.concat(ep), op = /*@__PURE__*/ Um.concat(Xm), rp = /*@__PURE__*/ Um.concat([ "day", "weekday" ], ep), ip = /*@__PURE__*/ Um.concat([ "year", "weekday" ], ep), ap = /*@__PURE__*/ createOptionsTransformer(np, xm), sp = /*@__PURE__*/ createOptionsTransformer(np, $m), cp = /*@__PURE__*/ createOptionsTransformer(np, xm, Um), up = /*@__PURE__*/ createOptionsTransformer(Xm, Lm, tp), lp = /*@__PURE__*/ createOptionsTransformer(ep, Wm, op), fp = /*@__PURE__*/ createOptionsTransformer(Km, qm, rp), dp = /*@__PURE__*/ createOptionsTransformer(Qm, jm, ip), mp = {}, pp = /*@__PURE__*/ computeNonBuggyIsoResolve(), Q = [ ap, I ], ot = [ sp, I, 0, (e, n) => {
  const t = e.timeZone;
  if (n && n.timeZone !== t) {
    throw new RangeError(pu);
  }
  return t;
} ], U = [ cp, isoToEpochMilli ], X = [ up, isoToEpochMilli ], tt = [ lp, e => isoTimeFieldsToNano(e) / Ke ], et = [ fp, isoToEpochMilli, pp ], nt = [ dp, isoToEpochMilli, pp ];

export { A as DurationBranding, Re as InstantBranding, G as PlainDateBranding, x as PlainDateTimeBranding, qt as PlainMonthDayBranding, ft as PlainTimeBranding, Qt as PlainYearMonthBranding, en as RawDateTimeFormat, _ as ZonedDateTimeBranding, Y as absDuration, so as addBigNanos, E as addDurations, lo as alignZonedEpoch, Oa as bigNanoToExactDays, La as bigNanoToNumber, gt as bindArgs, Ja as buildZonedIsoFields, io as checkEpochNanoInBounds, To as checkIsoDateInBounds, Do as checkIsoDateTimeInBounds, ba as clampEntity, Ra as clearIsoFields, pa as compareBigNanos, H as compareDurations, He as compareInstants, te as compareIsoDateFields, Yt as compareIsoDateTimeFields, Dt as compareIsoTimeFields, Be as compareZonedDateTimes, ho as computeDayFloor, ja as computeEpochNanoFrac, Ha as computeIsoDayOfWeek, fo as computeIsoDaysInWeek, Te as computeZonedHoursInDay, be as computeZonedStartOfDay, j as constructDurationSlots, qe as constructInstantSlots, ue as constructPlainDateSlots, Mt as constructPlainDateTimeSlots, kt as constructPlainMonthDaySlots, ut as constructPlainTimeSlots, Kt as constructPlainYearMonthSlots, ye as constructZonedDateTimeSlots, pe as createDurationSlots, an as createFormatForPrep, K as createFormatPrepper, t as createGetterDescriptors, xe as createInstantSlots, r as createNameDescriptors, xa as createNativeConvertOps, mo as createNativeDateModOps, Aa as createNativeDateRefineOps, la as createNativeDayOfYearOps, za as createNativeDayOps, da as createNativeDaysInMonthOps, ua as createNativeDaysInYearOps, Ia as createNativeDiffOps, ia as createNativeInLeapYearOps, Wo as createNativeMonthDayModOps, ko as createNativeMonthDayParseOps, _a as createNativeMonthDayRefineOps, ca as createNativeMonthsInYearOps, Wa as createNativeMoveOps, ra as createNativePartOps, v as createNativeStandardOps, $a as createNativeWeekOps, Lo as createNativeYearMonthDiffOps, Fo as createNativeYearMonthModOps, Yo as createNativeYearMonthMoveOps, Va as createNativeYearMonthRefineOps, W as createPlainDateSlots, jt as createPlainDateTimeSlots, St as createPlainTimeSlots, n as createPropDescriptors, o as createStringTagDescriptors, Xe as createZonedDateTimeSlots, X as dateConfig, U as dateTimeConfig, va as diffBigNanos, Ee as diffInstants, It as diffPlainDateTimes, oe as diffPlainDates, it as diffPlainTimes, _t as diffPlainYearMonth, we as diffZonedDateTimes, O as durationFieldNamesAsc, N as durationWithFields, na as epochMicroToInstant, ze as epochMilliToInstant, Pa as epochMilliToIso, $e as epochNanoToInstant, So as epochNanoToIso, ea as epochSecToInstant, fa as extractEpochNano, C as forbiddenValueOf, k as formatDurationIso, ke as formatInstantIso, sa as formatMonthCode, Se as formatOffsetNano, ce as formatPlainDateIso, Ft as formatPlainDateTimeIso, Jt as formatPlainMonthDayIso, ct as formatPlainTimeIso, Ht as formatPlainYearMonthIso, Fe as formatZonedDateTimeIso, ha as getCommonCalendarId, ga as getCommonTimeZoneId, Ue as getCurrentEpochNano, tn as getCurrentIsoDateTime, Qe as getCurrentTimeZoneId, y as getDurationBlank, aa as getEpochMicro, I as getEpochMilli, b as getEpochNano, ta as getEpochSec, $o as getSingleInstantFor, Io as identity, Q as instantConfig, Je as instantToZonedDateTime, Ve as instantsEqual, i as invalidBag, c as invalidCalendar, a as invalidCallingContext, rn as invalidFormatType, F as invalidTimeZone, s as isObjectLike, l as isoCalendarId, Ca as isoDateFieldNamesAlpha, At as isoTimeFieldDefaults, w as isoTimeFieldNamesAsc, Ga as isoTimeFieldsToCal, ma as isoToEpochNano, P as mapPropNames, e as mapProps, on as memoize, ln as mismatchingFormatTypes, nt as monthDayConfig, Ta as moveBigNano, Ua as moveByDays, ka as moveDateTime, Ye as moveInstant, ne as movePlainDate, wt as movePlainDateTime, at as movePlainTime, Gt as movePlainYearMonth, Na as moveToDayOfMonthUnsafe, Oe as moveZonedDateTime, Fa as moveZonedEpochs, no as nanoInHour, ro as nanoInMicro, Ke as nanoInMilli, ao as nanoInMinute, oo as nanoInSec, go as nanoInUtcDay, wa as nativeYearMonthAdd, B as negateDuration, Ge as numberToBigNano, f as parseCalendarId, R as parseDuration, We as parseInstant, me as parsePlainDate, Bt as parsePlainDateTime, xt as parsePlainMonthDay, ht as parsePlainTime, Xt as parsePlainYearMonth, $ as parseRelativeToSlots, M as parseTimeZoneId, Ne as parseZonedDateTime, yo as plainDateTimeToPlainMonthDay, po as plainDateTimeToPlainYearMonth, Ct as plainDateTimeToZonedDateTime, Pt as plainDateTimeWithFields, Mo as plainDateTimeWithPlainDate, pt as plainDateTimeWithPlainTime, vt as plainDateTimesEqual, ie as plainDateToPlainDateTime, se as plainDateToPlainMonthDay, le as plainDateToPlainYearMonth, ae as plainDateToZonedDateTime, ee as plainDateWithFields, re as plainDatesEqual, Vt as plainMonthDayToPlainDate, Et as plainMonthDayWithFields, Lt as plainMonthDaysEqual, Oo as plainTimeToPlainDateTime, vo as plainTimeToZonedDateTime, rt as plainTimeWithFields, st as plainTimesEqual, $t as plainYearMonthToPlainDate, Wt as plainYearMonthWithFields, zt as plainYearMonthsEqual, nn as pluckProps, Sa as prepareZonedEpochDiff, L as queryNativeTimeZone, Zt as refineCalendarId, Ze as refineDirectionOptions, q as refineDurationBag, z as refineMaybeZonedDateTimeBag, dt as refineOverflowOptions, de as refinePlainDateBag, Nt as refinePlainDateTimeBag, Rt as refinePlainMonthDayBag, Tt as refinePlainTimeBag, Ut as refinePlainYearMonthBag, Me as refineTimeZoneId, Ma as refineUnitDiffOptions, co as refineUnitRoundOptions, Ae as refineZonedDateTimeBag, je as refineZonedFieldOptions, D as requireBoolean, T as requireInteger, S as requireIntegerOrUndefined, _e as requireNumberIsInteger, oa as requireObjectLike, h as requirePositiveInteger, g as requirePositiveIntegerOrUndefined, d as requireString, m as requireStringOrUndefined, u as resolveCalendarId, Z as resolveTimeZoneId, Ya as roundBigNanoByInc, Da as roundByInc, V as roundDuration, Le as roundInstant, bt as roundPlainDateTime, lt as roundPlainTime, Ea as roundWithMode, Ie as roundZonedDateTime, uo as roundZonedEpochToInterval, Ot as slotsWithCalendarId, ge as slotsWithTimeZoneId, tt as timeConfig, p as timeFieldNamesAsc, Za as toInteger, Ba as toStrictInteger, J as totalDuration, ya as totalRelativeDuration, qa as unsupportedWeekNumbers, et as yearMonthConfig, ot as zonedConfig, Ce as zonedDateTimeToInstant, fe as zonedDateTimeToPlainDate, yt as zonedDateTimeToPlainDateTime, Xa as zonedDateTimeToPlainMonthDay, mt as zonedDateTimeToPlainTime, Qa as zonedDateTimeToPlainYearMonth, De as zonedDateTimeWithFields, Ka as zonedDateTimeWithPlainDate, Pe as zonedDateTimeWithPlainTime, ve as zonedDateTimesEqual, he as zonedEpochSlotsToIso };
