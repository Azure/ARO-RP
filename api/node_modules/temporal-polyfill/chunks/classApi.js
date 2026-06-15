function createSlotClass(i, l, s, c, u, f) {
  function Class(...t) {
    if (!(this instanceof Class)) {
      throw new TypeError(a);
    }
    {
      const e = l(...t);
      un(this, e), dbg(this, e, f);
    }
  }
  function bindMethod(t, e) {
    return Object.defineProperties((function(...e) {
      return t.call(this, getSpecificSlots(this), ...e);
    }), r(e));
  }
  function getSpecificSlots(t) {
    const e = cn(t);
    if (!e || e.branding !== i) {
      throw new TypeError(a);
    }
    return e;
  }
  return Object.defineProperties(Class.prototype, {
    ...t(e(bindMethod, s)),
    ...n(e(bindMethod, c)),
    ...o("Temporal." + i)
  }), Object.defineProperties(Class, {
    ...n(u),
    ...r(i)
  }), [ Class, t => {
    const e = Object.create(Class.prototype);
    return un(e, t), dbg(e, t, f), e;
  }, getSpecificSlots ];
}

function rejectInvalidBag(t) {
  if (cn(t) || void 0 !== t.calendar || void 0 !== t.timeZone) {
    throw new TypeError(i);
  }
  return t;
}

function dbg(t, e, n) {
  "dbg" === dbg.name && Object.defineProperty(t, "o", {
    value: n(e),
    writable: 0,
    enumerable: 0,
    configurable: 0
  });
}

function getCalendarIdFromBag(t) {
  return extractCalendarIdFromBag(t) || l;
}

function extractCalendarIdFromBag(t) {
  const {calendar: e} = t;
  if (void 0 !== e) {
    return refineCalendarArg(e);
  }
}

function refineCalendarArg(t) {
  if (s(t)) {
    const {calendar: e} = cn(t) || {};
    if (!e) {
      throw new TypeError(c(t));
    }
    return e;
  }
  return (t => u(f(d(t))))(t);
}

function createCalendarGetters(t) {
  const e = {};
  for (const n in t) {
    e[n] = t => {
      const {calendar: e} = t;
      return v(e)[n](t);
    };
  }
  return e;
}

function neverValueOf() {
  throw new TypeError(C);
}

function refineTimeZoneArg(t) {
  if (s(t)) {
    const {timeZone: e} = cn(t) || {};
    if (!e) {
      throw new TypeError(F(t));
    }
    return e;
  }
  return (t => Z(M(d(t))))(t);
}

function toDurationSlots(t) {
  if (s(t)) {
    const e = cn(t);
    return e && e.branding === A ? e : q(t);
  }
  return R(t);
}

function refinePublicRelativeTo(t) {
  if (void 0 !== t) {
    if (s(t)) {
      const e = cn(t) || {};
      switch (e.branding) {
       case _:
       case G:
        return e;

       case x:
        return W(e);
      }
      const n = getCalendarIdFromBag(t);
      return {
        ...z(refineTimeZoneArg, L, v(n), t),
        calendar: n
      };
    }
    return $(t);
  }
}

function toPlainTimeSlots(t, e) {
  if (s(t)) {
    const n = cn(t) || {};
    switch (n.branding) {
     case ft:
      return dt(e), n;

     case x:
      return dt(e), St(n);

     case _:
      return dt(e), mt(L, n);
    }
    return Tt(t, e);
  }
  const n = ht(t);
  return dt(e), n;
}

function optionalToPlainTimeFields(t) {
  return void 0 === t ? void 0 : toPlainTimeSlots(t);
}

function toPlainDateTimeSlots(t, e) {
  if (s(t)) {
    const n = cn(t) || {};
    switch (n.branding) {
     case x:
      return dt(e), n;

     case G:
      return dt(e), jt({
        ...n,
        ...At
      });

     case _:
      return dt(e), yt(L, n);
    }
    return Nt(v(getCalendarIdFromBag(t)), t, e);
  }
  const n = Bt(t);
  return dt(e), n;
}

function toPlainMonthDaySlots(t, e) {
  if (s(t)) {
    const n = cn(t);
    if (n && n.branding === qt) {
      return dt(e), n;
    }
    const o = extractCalendarIdFromBag(t);
    return Rt(v(o || l), !o, t, e);
  }
  const n = xt(v, t);
  return dt(e), n;
}

function toPlainYearMonthSlots(t, e) {
  if (s(t)) {
    const n = cn(t);
    return n && n.branding === Qt ? (dt(e), n) : Ut(v(getCalendarIdFromBag(t)), t, e);
  }
  const n = Xt(v, t);
  return dt(e), n;
}

function toPlainDateSlots(t, e) {
  if (s(t)) {
    const n = cn(t) || {};
    switch (n.branding) {
     case G:
      return dt(e), n;

     case x:
      return dt(e), W(n);

     case _:
      return dt(e), fe(L, n);
    }
    return de(v(getCalendarIdFromBag(t)), t, e);
  }
  const n = me(t);
  return dt(e), n;
}

function toZonedDateTimeSlots(t, e) {
  if (s(t)) {
    const n = cn(t);
    if (n && n.branding === _) {
      return je(e), n;
    }
    const o = getCalendarIdFromBag(t);
    return Ae(refineTimeZoneArg, L, v(o), o, t, e);
  }
  return Ne(t, e);
}

function adaptDateMethods(t) {
  return e((t => e => t(slotsToIso(e))), t);
}

function slotsToIso(t) {
  return he(t, L);
}

function toInstantSlots(t) {
  if (s(t)) {
    const e = cn(t);
    if (e) {
      switch (e.branding) {
       case Re:
        return e;

       case _:
        return xe(e.epochNanoseconds);
      }
    }
  }
  return We(t);
}

function toTemporalInstant() {
  const t = Date.prototype.valueOf.call(this);
  return Hn(xe(Ge(_e(t), Ke)));
}

function createDateTimeFormatClass() {
  function DateTimeFormatFunc(t, e) {
    return new DateTimeFormatNew(t, e);
  }
  function DateTimeFormatNew(t, e = Object.create(null)) {
    to.set(this, ((t, e) => {
      const n = new en(t, e), o = n.resolvedOptions(), r = o.locale, a = nn(Object.keys(e), o), i = on(createFormatPrepperForBranding), prepFormat = (t, ...e) => {
        if (t) {
          if (2 !== e.length) {
            throw new TypeError(ln);
          }
          for (const t of e) {
            if (void 0 === t) {
              throw new TypeError(ln);
            }
          }
        }
        t || void 0 !== e[0] || (e = []);
        const o = e.map((t => cn(t) || Number(t)));
        let l, s = 0;
        for (const t of o) {
          const e = "object" == typeof t ? t.branding : void 0;
          if (s++ && e !== l) {
            throw new TypeError(ln);
          }
          l = e;
        }
        return l ? i(l)(r, a, ...o) : [ n, ...o ];
      };
      return prepFormat.i = n, prepFormat;
    })(t, e));
  }
  const t = en.prototype, e = Object.getOwnPropertyDescriptors(t), n = Object.getOwnPropertyDescriptors(en);
  for (const t in e) {
    const n = e[t], o = t.startsWith("format") && createFormatMethod(t);
    "function" == typeof n.value ? n.value = "constructor" === t ? DateTimeFormatFunc : o || createProxiedMethod(t) : o && (n.get = function() {
      if (!to.has(this)) {
        throw new TypeError(a);
      }
      return (...t) => o.apply(this, t);
    }, Object.defineProperties(n.get, r(`get ${t}`)));
  }
  return n.prototype.value = DateTimeFormatNew.prototype = Object.create({}, e), Object.defineProperties(DateTimeFormatFunc, n), 
  DateTimeFormatFunc;
}

function createFormatMethod(t) {
  return Object.defineProperties((function(...e) {
    const n = to.get(this), [o, ...r] = n(t.includes("Range"), ...e);
    return o[t](...r);
  }), r(t));
}

function createProxiedMethod(t) {
  return Object.defineProperties((function(...e) {
    return to.get(this).i[t](...e);
  }), r(t));
}

function createFormatPrepperForBranding(t) {
  const e = vn[t];
  if (!e) {
    throw new TypeError(rn(t));
  }
  return K(e, on(an), 1);
}

import { createGetterDescriptors as t, mapProps as e, createPropDescriptors as n, createStringTagDescriptors as o, createNameDescriptors as r, invalidCallingContext as a, invalidBag as i, isoCalendarId as l, isObjectLike as s, invalidCalendar as c, resolveCalendarId as u, parseCalendarId as f, requireString as d, requireStringOrUndefined as m, requireIntegerOrUndefined as S, requireInteger as T, requirePositiveInteger as h, requireBoolean as D, requirePositiveIntegerOrUndefined as g, mapPropNames as P, durationFieldNamesAsc as O, timeFieldNamesAsc as p, isoTimeFieldNamesAsc as w, getEpochMilli as I, getEpochNano as b, createNativeStandardOps as v, forbiddenValueOf as C, invalidTimeZone as F, resolveTimeZoneId as Z, parseTimeZoneId as M, getDurationBlank as y, constructDurationSlots as j, DurationBranding as A, durationWithFields as N, negateDuration as B, absDuration as Y, addDurations as E, queryNativeTimeZone as L, roundDuration as V, totalDuration as J, formatDurationIso as k, refineDurationBag as q, parseDuration as R, PlainDateTimeBranding as x, createPlainDateSlots as W, PlainDateBranding as G, ZonedDateTimeBranding as _, refineMaybeZonedDateTimeBag as z, parseRelativeToSlots as $, compareDurations as H, createFormatPrepper as K, instantConfig as Q, dateTimeConfig as U, dateConfig as X, timeConfig as tt, yearMonthConfig as et, monthDayConfig as nt, zonedConfig as ot, plainTimeWithFields as rt, movePlainTime as at, diffPlainTimes as it, roundPlainTime as lt, plainTimesEqual as st, formatPlainTimeIso as ct, constructPlainTimeSlots as ut, PlainTimeBranding as ft, refineOverflowOptions as dt, zonedDateTimeToPlainTime as mt, createPlainTimeSlots as St, refinePlainTimeBag as Tt, parsePlainTime as ht, compareIsoTimeFields as Dt, bindArgs as gt, plainDateTimeWithFields as Pt, slotsWithCalendarId as Ot, plainDateTimeWithPlainTime as pt, movePlainDateTime as wt, diffPlainDateTimes as It, roundPlainDateTime as bt, plainDateTimesEqual as vt, plainDateTimeToZonedDateTime as Ct, formatPlainDateTimeIso as Ft, refineCalendarId as Zt, constructPlainDateTimeSlots as Mt, zonedDateTimeToPlainDateTime as yt, createPlainDateTimeSlots as jt, isoTimeFieldDefaults as At, refinePlainDateTimeBag as Nt, parsePlainDateTime as Bt, compareIsoDateTimeFields as Yt, plainMonthDayWithFields as Et, plainMonthDaysEqual as Lt, plainMonthDayToPlainDate as Vt, formatPlainMonthDayIso as Jt, constructPlainMonthDaySlots as kt, PlainMonthDayBranding as qt, refinePlainMonthDayBag as Rt, parsePlainMonthDay as xt, plainYearMonthWithFields as Wt, movePlainYearMonth as Gt, diffPlainYearMonth as _t, plainYearMonthsEqual as zt, plainYearMonthToPlainDate as $t, formatPlainYearMonthIso as Ht, constructPlainYearMonthSlots as Kt, PlainYearMonthBranding as Qt, refinePlainYearMonthBag as Ut, parsePlainYearMonth as Xt, compareIsoDateFields as te, plainDateWithFields as ee, movePlainDate as ne, diffPlainDates as oe, plainDatesEqual as re, plainDateToZonedDateTime as ae, plainDateToPlainDateTime as ie, plainDateToPlainYearMonth as le, plainDateToPlainMonthDay as se, formatPlainDateIso as ce, constructPlainDateSlots as ue, zonedDateTimeToPlainDate as fe, refinePlainDateBag as de, parsePlainDate as me, formatOffsetNano as Se, computeZonedHoursInDay as Te, zonedEpochSlotsToIso as he, zonedDateTimeWithFields as De, slotsWithTimeZoneId as ge, zonedDateTimeWithPlainTime as Pe, moveZonedDateTime as Oe, createDurationSlots as pe, diffZonedDateTimes as we, roundZonedDateTime as Ie, computeZonedStartOfDay as be, zonedDateTimesEqual as ve, zonedDateTimeToInstant as Ce, formatZonedDateTimeIso as Fe, refineDirectionOptions as Ze, refineTimeZoneId as Me, constructZonedDateTimeSlots as ye, refineZonedFieldOptions as je, refineZonedDateTimeBag as Ae, parseZonedDateTime as Ne, compareZonedDateTimes as Be, moveInstant as Ye, diffInstants as Ee, roundInstant as Le, instantsEqual as Ve, instantToZonedDateTime as Je, formatInstantIso as ke, constructInstantSlots as qe, InstantBranding as Re, createInstantSlots as xe, parseInstant as We, numberToBigNano as Ge, requireNumberIsInteger as _e, epochMilliToInstant as ze, epochNanoToInstant as $e, compareInstants as He, nanoInMilli as Ke, getCurrentTimeZoneId as Qe, getCurrentEpochNano as Ue, createZonedDateTimeSlots as Xe, getCurrentIsoDateTime as tn, RawDateTimeFormat as en, pluckProps as nn, memoize as on, invalidFormatType as rn, createFormatForPrep as an, mismatchingFormatTypes as ln } from "./internal.js";

const sn = /*@__PURE__*/ new WeakMap, cn = /*@__PURE__*/ sn.get.bind(sn), un = /*@__PURE__*/ sn.set.bind(sn), fn = {
  era: m,
  eraYear: S,
  year: T,
  month: h,
  daysInMonth: h,
  daysInYear: h,
  inLeapYear: D,
  monthsInYear: h
}, dn = {
  monthCode: d
}, mn = {
  day: h
}, Sn = {
  dayOfWeek: h,
  dayOfYear: h,
  weekOfYear: g,
  yearOfWeek: S,
  daysInWeek: h
}, Tn = /*@__PURE__*/ createCalendarGetters(/*@__PURE__*/ Object.assign({}, fn, dn, mn, Sn)), hn = /*@__PURE__*/ createCalendarGetters({
  ...fn,
  ...dn
}), Dn = /*@__PURE__*/ createCalendarGetters({
  ...dn,
  ...mn
}), gn = {
  calendarId: t => t.calendar
}, Pn = /*@__PURE__*/ P((t => e => e[t]), O.concat("sign")), On = /*@__PURE__*/ P(((t, e) => t => t[w[e]]), p), pn = {
  epochMilliseconds: I,
  epochNanoseconds: b
}, [wn, In, bn] = createSlotClass(A, j, {
  ...Pn,
  blank: y
}, {
  with: (t, e) => In(N(t, e)),
  negated: t => In(B(t)),
  abs: t => In(Y(t)),
  add: (t, e, n) => In(E(refinePublicRelativeTo, v, L, 0, t, toDurationSlots(e), n)),
  subtract: (t, e, n) => In(E(refinePublicRelativeTo, v, L, 1, t, toDurationSlots(e), n)),
  round: (t, e) => In(V(refinePublicRelativeTo, v, L, t, e)),
  total: (t, e) => J(refinePublicRelativeTo, v, L, t, e),
  toLocaleString(t, e, n) {
    return Intl.DurationFormat ? new Intl.DurationFormat(e, n).format(this) : k(t);
  },
  toString: k,
  toJSON: t => k(t),
  valueOf: neverValueOf
}, {
  from: t => In(toDurationSlots(t)),
  compare: (t, e, n) => H(refinePublicRelativeTo, v, L, toDurationSlots(t), toDurationSlots(e), n)
}, k), vn = {
  Instant: Q,
  PlainDateTime: U,
  PlainDate: X,
  PlainTime: tt,
  PlainYearMonth: et,
  PlainMonthDay: nt
}, Cn = /*@__PURE__*/ K(Q), Fn = /*@__PURE__*/ K(ot), Zn = /*@__PURE__*/ K(U), Mn = /*@__PURE__*/ K(X), yn = /*@__PURE__*/ K(tt), jn = /*@__PURE__*/ K(et), An = /*@__PURE__*/ K(nt), [Nn, Bn] = createSlotClass(ft, ut, On, {
  with(t, e, n) {
    return Bn(rt(this, rejectInvalidBag(e), n));
  },
  add: (t, e) => Bn(at(0, t, toDurationSlots(e))),
  subtract: (t, e) => Bn(at(1, t, toDurationSlots(e))),
  until: (t, e, n) => In(it(0, t, toPlainTimeSlots(e), n)),
  since: (t, e, n) => In(it(1, t, toPlainTimeSlots(e), n)),
  round: (t, e) => Bn(lt(t, e)),
  equals: (t, e) => st(t, toPlainTimeSlots(e)),
  toLocaleString(t, e, n) {
    const [o, r] = yn(e, n, t);
    return o.format(r);
  },
  toString: ct,
  toJSON: t => ct(t),
  valueOf: neverValueOf
}, {
  from: (t, e) => Bn(toPlainTimeSlots(t, e)),
  compare: (t, e) => Dt(toPlainTimeSlots(t), toPlainTimeSlots(e))
}, ct), [Yn, En] = createSlotClass(x, gt(Mt, Zt), {
  ...gn,
  ...Tn,
  ...On
}, {
  with: (t, e, n) => En(Pt(v, t, rejectInvalidBag(e), n)),
  withCalendar: (t, e) => En(Ot(t, refineCalendarArg(e))),
  withPlainTime: (t, e) => En(pt(t, optionalToPlainTimeFields(e))),
  add: (t, e, n) => En(wt(v, 0, t, toDurationSlots(e), n)),
  subtract: (t, e, n) => En(wt(v, 1, t, toDurationSlots(e), n)),
  until: (t, e, n) => In(It(v, 0, t, toPlainDateTimeSlots(e), n)),
  since: (t, e, n) => In(It(v, 1, t, toPlainDateTimeSlots(e), n)),
  round: (t, e) => En(bt(t, e)),
  equals: (t, e) => vt(t, toPlainDateTimeSlots(e)),
  toZonedDateTime: (t, e, n) => zn(Ct(L, t, refineTimeZoneArg(e), n)),
  toPlainDate: t => Wn(W(t)),
  toPlainTime: t => Bn(St(t)),
  toLocaleString(t, e, n) {
    const [o, r] = Zn(e, n, t);
    return o.format(r);
  },
  toString: Ft,
  toJSON: t => Ft(t),
  valueOf: neverValueOf
}, {
  from: (t, e) => En(toPlainDateTimeSlots(t, e)),
  compare: (t, e) => Yt(toPlainDateTimeSlots(t), toPlainDateTimeSlots(e))
}, Ft), [Ln, Vn, Jn] = createSlotClass(qt, gt(kt, Zt), {
  ...gn,
  ...Dn
}, {
  with: (t, e, n) => Vn(Et(v, t, rejectInvalidBag(e), n)),
  equals: (t, e) => Lt(t, toPlainMonthDaySlots(e)),
  toPlainDate(t, e) {
    return Wn(Vt(v, t, this, e));
  },
  toLocaleString(t, e, n) {
    const [o, r] = An(e, n, t);
    return o.format(r);
  },
  toString: Jt,
  toJSON: t => Jt(t),
  valueOf: neverValueOf
}, {
  from: (t, e) => Vn(toPlainMonthDaySlots(t, e))
}, Jt), [kn, qn, Rn] = createSlotClass(Qt, gt(Kt, Zt), {
  ...gn,
  ...hn
}, {
  with: (t, e, n) => qn(Wt(v, t, rejectInvalidBag(e), n)),
  add: (t, e, n) => qn(Gt(v, 0, t, toDurationSlots(e), n)),
  subtract: (t, e, n) => qn(Gt(v, 1, t, toDurationSlots(e), n)),
  until: (t, e, n) => In(_t(v, 0, t, toPlainYearMonthSlots(e), n)),
  since: (t, e, n) => In(_t(v, 1, t, toPlainYearMonthSlots(e), n)),
  equals: (t, e) => zt(t, toPlainYearMonthSlots(e)),
  toPlainDate(t, e) {
    return Wn($t(v, t, this, e));
  },
  toLocaleString(t, e, n) {
    const [o, r] = jn(e, n, t);
    return o.format(r);
  },
  toString: Ht,
  toJSON: t => Ht(t),
  valueOf: neverValueOf
}, {
  from: (t, e) => qn(toPlainYearMonthSlots(t, e)),
  compare: (t, e) => te(toPlainYearMonthSlots(t), toPlainYearMonthSlots(e))
}, Ht), [xn, Wn, Gn] = createSlotClass(G, gt(ue, Zt), {
  ...gn,
  ...Tn
}, {
  with: (t, e, n) => Wn(ee(v, t, rejectInvalidBag(e), n)),
  withCalendar: (t, e) => Wn(Ot(t, refineCalendarArg(e))),
  add: (t, e, n) => Wn(ne(v, 0, t, toDurationSlots(e), n)),
  subtract: (t, e, n) => Wn(ne(v, 1, t, toDurationSlots(e), n)),
  until: (t, e, n) => In(oe(v, 0, t, toPlainDateSlots(e), n)),
  since: (t, e, n) => In(oe(v, 1, t, toPlainDateSlots(e), n)),
  equals: (t, e) => re(t, toPlainDateSlots(e)),
  toZonedDateTime(t, e) {
    const n = s(e) ? e : {
      timeZone: e
    };
    return zn(ae(refineTimeZoneArg, toPlainTimeSlots, L, t, n));
  },
  toPlainDateTime: (t, e) => En(ie(t, optionalToPlainTimeFields(e))),
  toPlainYearMonth(t) {
    return qn(le(v, t, this));
  },
  toPlainMonthDay(t) {
    return Vn(se(v, t, this));
  },
  toLocaleString(t, e, n) {
    const [o, r] = Mn(e, n, t);
    return o.format(r);
  },
  toString: ce,
  toJSON: t => ce(t),
  valueOf: neverValueOf
}, {
  from: (t, e) => Wn(toPlainDateSlots(t, e)),
  compare: (t, e) => te(toPlainDateSlots(t), toPlainDateSlots(e))
}, ce), [_n, zn] = createSlotClass(_, gt(ye, Zt, Me), {
  ...pn,
  ...gn,
  ...adaptDateMethods(Tn),
  ...adaptDateMethods(On),
  offset: t => Se(slotsToIso(t).offsetNanoseconds),
  offsetNanoseconds: t => slotsToIso(t).offsetNanoseconds,
  timeZoneId: t => t.timeZone,
  hoursInDay: t => Te(L, t)
}, {
  with: (t, e, n) => zn(De(v, L, t, rejectInvalidBag(e), n)),
  withCalendar: (t, e) => zn(Ot(t, refineCalendarArg(e))),
  withTimeZone: (t, e) => zn(ge(t, refineTimeZoneArg(e))),
  withPlainTime: (t, e) => zn(Pe(L, t, optionalToPlainTimeFields(e))),
  add: (t, e, n) => zn(Oe(v, L, 0, t, toDurationSlots(e), n)),
  subtract: (t, e, n) => zn(Oe(v, L, 1, t, toDurationSlots(e), n)),
  until: (t, e, n) => In(pe(we(v, L, 0, t, toZonedDateTimeSlots(e), n))),
  since: (t, e, n) => In(pe(we(v, L, 1, t, toZonedDateTimeSlots(e), n))),
  round: (t, e) => zn(Ie(L, t, e)),
  startOfDay: t => zn(be(L, t)),
  equals: (t, e) => ve(t, toZonedDateTimeSlots(e)),
  toInstant: t => Hn(Ce(t)),
  toPlainDateTime: t => En(yt(L, t)),
  toPlainDate: t => Wn(fe(L, t)),
  toPlainTime: t => Bn(mt(L, t)),
  toLocaleString(t, e, n = {}) {
    const [o, r] = Fn(e, n, t);
    return o.format(r);
  },
  toString: (t, e) => Fe(L, t, e),
  toJSON: t => Fe(L, t),
  valueOf: neverValueOf,
  getTimeZoneTransition(t, e) {
    const {timeZone: n, epochNanoseconds: o} = t, r = Ze(e), a = L(n).l(o, r);
    return a ? zn({
      ...t,
      epochNanoseconds: a
    }) : null;
  }
}, {
  from: (t, e) => zn(toZonedDateTimeSlots(t, e)),
  compare: (t, e) => Be(toZonedDateTimeSlots(t), toZonedDateTimeSlots(e))
}, (t => Fe(L, t))), [$n, Hn, Kn] = createSlotClass(Re, qe, pn, {
  add: (t, e) => Hn(Ye(0, t, toDurationSlots(e))),
  subtract: (t, e) => Hn(Ye(1, t, toDurationSlots(e))),
  until: (t, e, n) => In(Ee(0, t, toInstantSlots(e), n)),
  since: (t, e, n) => In(Ee(1, t, toInstantSlots(e), n)),
  round: (t, e) => Hn(Le(t, e)),
  equals: (t, e) => Ve(t, toInstantSlots(e)),
  toZonedDateTimeISO: (t, e) => zn(Je(t, refineTimeZoneArg(e))),
  toLocaleString(t, e, n) {
    const [o, r] = Cn(e, n, t);
    return o.format(r);
  },
  toString: (t, e) => ke(refineTimeZoneArg, L, t, e),
  toJSON: t => ke(refineTimeZoneArg, L, t),
  valueOf: neverValueOf
}, {
  from: t => Hn(toInstantSlots(t)),
  fromEpochMilliseconds: t => Hn(ze(t)),
  fromEpochNanoseconds: t => Hn($e(t)),
  compare: (t, e) => He(toInstantSlots(t), toInstantSlots(e))
}, (t => ke(refineTimeZoneArg, L, t))), Qn = /*@__PURE__*/ Object.defineProperties({}, {
  ...o("Temporal.Now"),
  ...n({
    timeZoneId: () => Qe(),
    instant: () => Hn(xe(Ue())),
    zonedDateTimeISO: (t = Qe()) => zn(Xe(Ue(), refineTimeZoneArg(t), l)),
    plainDateTimeISO: (t = Qe()) => En(jt(tn(L(refineTimeZoneArg(t))), l)),
    plainDateISO: (t = Qe()) => Wn(W(tn(L(refineTimeZoneArg(t))), l)),
    plainTimeISO: (t = Qe()) => Bn(St(tn(L(refineTimeZoneArg(t)))))
  })
}), Un = /*@__PURE__*/ Object.defineProperties({}, {
  ...o("Temporal"),
  ...n({
    PlainYearMonth: kn,
    PlainMonthDay: Ln,
    PlainDate: xn,
    PlainTime: Nn,
    PlainDateTime: Yn,
    ZonedDateTime: _n,
    Instant: $n,
    Duration: wn,
    Now: Qn
  })
}), Xn = /*@__PURE__*/ createDateTimeFormatClass(), to = /*@__PURE__*/ new WeakMap, eo = /*@__PURE__*/ Object.defineProperties(Object.create(Intl), n({
  DateTimeFormat: Xn
}));

export { Xn as DateTimeFormat, eo as IntlExtended, Un as Temporal, toTemporalInstant };
