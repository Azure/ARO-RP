"use strict";

function createFormatCache() {
  const queryFormatFactory = internal.memoize((options => {
    const map = new Map;
    return (forcedTimeZoneId, locales, transformOptions) => {
      const key = [].concat(forcedTimeZoneId || [], locales || []).join();
      let format = map.get(key);
      return format || (format = internal.createFormatForPrep(forcedTimeZoneId, locales, options, transformOptions, 0), 
      map.set(key, format)), format;
    };
  }), WeakMap);
  return (forcedTimeZoneId, locales, options, transformOptions) => queryFormatFactory(options)(forcedTimeZoneId, locales, transformOptions);
}

function getCalendarId(slots) {
  return slots.calendar;
}

function getCalendarIdFromBag(bag) {
  return extractCalendarIdFromBag(bag) || internal.isoCalendarId;
}

function extractCalendarIdFromBag(bag) {
  const {calendar: calendar} = bag;
  if (void 0 !== calendar) {
    return internal.refineCalendarId(calendar);
  }
}

function computeDateFields(slots) {
  const calendarOps = internal.createNativePartOps(slots.calendar), [year, month, day] = calendarOps.dateParts(slots), [era, eraYear] = calendarOps.eraParts(slots), [monthCodeNumber, isLeapMonth] = calendarOps.monthCodeParts(year, month);
  return {
    era: era,
    eraYear: eraYear,
    year: year,
    monthCode: internal.formatMonthCode(monthCodeNumber, isLeapMonth),
    month: month,
    day: day
  };
}

function computeInLeapYear(slots) {
  return internal.createNativeInLeapYearOps(slots.calendar).inLeapYear(slots);
}

function computeMonthsInYear(slots) {
  return internal.createNativeMonthsInYearOps(slots.calendar).monthsInYear(slots);
}

function computeDaysInMonth(slots) {
  return internal.createNativeDaysInMonthOps(slots.calendar).daysInMonth(slots);
}

function computeDaysInYear(slots) {
  return internal.createNativeDaysInYearOps(slots.calendar).daysInYear(slots);
}

function computeDayOfYear(slots) {
  return internal.createNativeDayOfYearOps(slots.calendar).dayOfYear(slots);
}

function computeWeekOfYear(slots) {
  return internal.createNativeWeekOps(slots.calendar).weekOfYear(slots);
}

function computeYearOfWeek(slots) {
  return internal.createNativeWeekOps(slots.calendar).yearOfWeek(slots);
}

function diffZonedLargeUnits(unit, record0, record1, options) {
  const timeZoneId = internal.getCommonTimeZoneId(record0.timeZone, record1.timeZone), timeZoneOps = internal.queryNativeTimeZone(timeZoneId), calendarId = internal.getCommonCalendarId(record0.calendar, record1.calendar), calendarOps = internal.createNativeDiffOps(calendarId);
  return diffDateUnits(internal.extractEpochNano, internal.bindArgs(internal.prepareZonedEpochDiff, timeZoneOps), internal.bindArgs(internal.moveZonedEpochs, timeZoneOps), ((f0, f1) => calendarOps.dateUntil(f0, f1, unit)), unit, calendarOps, record0, record1, options);
}

function diffPlainLargeUnits(unit, record0, record1, options) {
  const calendarId = internal.getCommonCalendarId(record0.calendar, record1.calendar), calendarOps = internal.createNativeDiffOps(calendarId);
  return diffDateUnits(internal.isoToEpochNano, identityMarkersToIsoFields, internal.moveDateTime, ((f0, f1) => calendarOps.dateUntil(f0, f1, unit)), unit, calendarOps, record0, record1, options);
}

function identityMarkersToIsoFields(m0, m1) {
  return [ m0, m1 ];
}

function diffDateUnits(markerToEpochNano, markersToIsoFields, moveMarker, diffIsoFields, unit, calendarOps, marker0, marker1, options) {
  const [roundingInc, roundingMode] = internal.refineUnitDiffOptions(unit, options), startEpochNano = markerToEpochNano(marker0), endEpochNano = markerToEpochNano(marker1), sign = internal.compareBigNanos(endEpochNano, startEpochNano);
  if (!sign) {
    return 0;
  }
  const [isoFields0, isoFields1] = markersToIsoFields(marker0, marker1, sign), durationFields = diffIsoFields(isoFields0, isoFields1);
  let res = internal.totalRelativeDuration(durationFields, endEpochNano, unit, calendarOps, marker0, markerToEpochNano, moveMarker);
  return roundingInc && (res = internal.roundByInc(res, roundingInc, roundingMode)), 
  res;
}

function diffZonedDayLikeUnits(unit, daysInUnit, record0, record1, options) {
  const [roundingInc, roundingMode] = internal.refineUnitDiffOptions(unit, options), timeZoneId = internal.getCommonTimeZoneId(record0.timeZone, record1.timeZone), timeZoneOps = internal.queryNativeTimeZone(timeZoneId), sign = internal.compareBigNanos(record1.epochNanoseconds, record0.epochNanoseconds), [isoFields0, isoFields1, remainderNano] = internal.prepareZonedEpochDiff(timeZoneOps, record0, record1, sign), nanoDiff = internal.moveBigNano(internal.diffBigNanos(internal.isoToEpochNano(isoFields0), internal.isoToEpochNano(isoFields1)), remainderNano);
  let res = internal.bigNanoToExactDays(nanoDiff) / daysInUnit;
  return roundingInc && (res = internal.roundByInc(res, roundingInc, roundingMode)), 
  res;
}

function diffPlainDayLikeUnit(markerToEpochNano, unit, daysInUnit, record0, record1, options) {
  const [roundingInc, roundingMode] = internal.refineUnitDiffOptions(unit, options), nanoDiff = internal.diffBigNanos(markerToEpochNano(record0), markerToEpochNano(record1));
  let res = internal.bigNanoToExactDays(nanoDiff) / daysInUnit;
  return roundingInc && (res = internal.roundByInc(res, roundingInc, roundingMode)), 
  res;
}

function diffTimeUnit(markerToEpochNano, unit, nanoInUnit, record0, record1, options) {
  const [roundingInc, roundingMode] = internal.refineUnitDiffOptions(unit, options);
  let nanoDiff = internal.diffBigNanos(markerToEpochNano(record0), markerToEpochNano(record1));
  return roundingInc && (nanoDiff = internal.roundBigNanoByInc(nanoDiff, nanoInUnit * roundingInc, roundingMode)), 
  internal.bigNanoToNumber(nanoDiff, nanoInUnit, !roundingInc);
}

function reversedMove(f) {
  return (slots, units, options) => f(slots, -units, options);
}

function moveByYears(slots, years, options) {
  const overflow = internal.refineOverflowOptions(options);
  if (!years) {
    return slots;
  }
  const calendarOps = internal.createNativeMoveOps(slots.calendar), isoFields = internal.epochMilliToIso(internal.nativeYearMonthAdd(calendarOps, slots, internal.toStrictInteger(years), 0, overflow)), isoDateFields = internal.pluckProps(internal.isoDateFieldNamesAlpha, isoFields);
  return {
    ...slots,
    ...isoDateFields
  };
}

function moveByMonths(slots, months, options) {
  const overflow = internal.refineOverflowOptions(options);
  if (!months) {
    return slots;
  }
  const calendarOps = internal.createNativeMoveOps(slots.calendar), isoFields = internal.epochMilliToIso(internal.nativeYearMonthAdd(calendarOps, slots, 0, internal.toStrictInteger(months), overflow)), isoDateFields = internal.pluckProps(internal.isoDateFieldNamesAlpha, isoFields);
  return {
    ...slots,
    ...isoDateFields
  };
}

function moveByIsoWeeks(slots, weeks) {
  return internal.moveByDays(slots, 7 * internal.toStrictInteger(weeks));
}

function moveByDaysStrict(slots, weeks) {
  return internal.moveByDays(slots, internal.toStrictInteger(weeks));
}

function moveToDayOfYear(slots, dayOfYear, options) {
  const overflow = internal.refineOverflowOptions(options), {calendar: calendar} = slots, daysInYear = internal.createNativeDaysInYearOps(calendar).daysInYear(slots), normDayOfYear = internal.clampEntity(dayOfMonthLabel, internal.toInteger(dayOfYear, dayOfMonthLabel), 1, daysInYear, overflow), currentDayOfYear = internal.createNativeDayOfYearOps(calendar).dayOfYear(slots);
  return internal.moveByDays(slots, normDayOfYear - currentDayOfYear);
}

function moveToDayOfMonth(slots, day, options) {
  const overflow = internal.refineOverflowOptions(options), {calendar: calendar} = slots, daysInMonth = internal.createNativeDaysInMonthOps(calendar).daysInMonth(slots), normDayOfMonth = internal.clampEntity(dayLabel, internal.toInteger(day, dayLabel), 1, daysInMonth, overflow);
  return internal.moveToDayOfMonthUnsafe(internal.createNativeDayOps(calendar), slots, normDayOfMonth);
}

function moveToDayOfWeek(slots, dayOfWeek, options) {
  const overflow = internal.refineOverflowOptions(options), normDayOfWeek = internal.clampEntity(dayOfWeekLabel, internal.toInteger(dayOfWeek, dayOfWeekLabel), 1, 7, overflow);
  return internal.moveByDays(slots, normDayOfWeek - internal.computeIsoDayOfWeek(slots));
}

function slotsWithWeekOfYear(slots, weekOfYear, options) {
  const overflow = internal.refineOverflowOptions(options), calendarOps = internal.createNativeWeekOps(slots.calendar), [currentWeekOfYear, , weeksInYear] = calendarOps.weekParts(slots);
  if (void 0 === currentWeekOfYear) {
    throw new RangeError(internal.unsupportedWeekNumbers);
  }
  return moveByIsoWeeks(slots, internal.clampEntity(weekOfYearLabel, internal.toInteger(weekOfYear, weekOfYearLabel), 1, weeksInYear, overflow) - currentWeekOfYear);
}

function computeYearFloor(slots, calendarOps = internal.createNativeConvertOps(slots.calendar)) {
  const [year0] = calendarOps.dateParts(slots);
  return {
    ...internal.epochMilliToIso(calendarOps.epochMilli(year0)),
    year: year0
  };
}

function computeMonthFloor(slots, calendarOps = internal.createNativeConvertOps(slots.calendar)) {
  const [year0, month0] = calendarOps.dateParts(slots);
  return {
    ...internal.epochMilliToIso(calendarOps.epochMilli(year0, month0)),
    year: year0,
    month: month0
  };
}

function computeIsoWeekFloor(slots) {
  const dayOfWeek = internal.computeIsoDayOfWeek(slots);
  return {
    ...internal.moveByDays(slots, 1 - dayOfWeek),
    ...internal.isoTimeFieldDefaults
  };
}

function computeYearCeil(slots) {
  return computeYearInterval(slots)[1];
}

function computeMonthCeil(slots) {
  return computeMonthInterval(slots)[1];
}

function computeIsoWeekCeil(slots) {
  return computeIsoWeekInterval(slots)[1];
}

function computeYearInterval(slots) {
  const calendarOps = internal.createNativeConvertOps(slots.calendar), isoFields0 = computeYearFloor(slots), year1 = isoFields0.year + 1;
  return [ isoFields0, internal.epochMilliToIso(calendarOps.epochMilli(year1)) ];
}

function computeMonthInterval(slots) {
  const calendarOps = internal.createNativeConvertOps(slots.calendar), isoFields0 = computeMonthFloor(slots, calendarOps), [year1, month1] = calendarOps.monthAdd(isoFields0.year, isoFields0.month, 1);
  return [ isoFields0, internal.epochMilliToIso(calendarOps.epochMilli(year1, month1)) ];
}

function computeIsoWeekInterval(slots) {
  const isoFields0 = computeIsoWeekFloor(slots);
  return [ isoFields0, moveByIsoWeeks(isoFields0, 1) ];
}

function roundDateTimeToInterval(computeInterval, slots, roundingMode) {
  const [isoFields0, isoFields1] = computeInterval(slots), epochNano0 = internal.isoToEpochNano(isoFields0), epochNano1 = internal.isoToEpochNano(isoFields1), epochNano = internal.isoToEpochNano(slots), frac = internal.computeEpochNanoFrac(epochNano, epochNano0, epochNano1), isoFieldsRounded = internal.roundWithMode(frac, roundingMode) ? isoFields1 : isoFields0;
  return {
    ...slots,
    ...isoFieldsRounded
  };
}

function offsetNanoseconds(record) {
  return internal.zonedEpochSlotsToIso(record, internal.queryNativeTimeZone).offsetNanoseconds;
}

function adaptDateFunc(dateFunc) {
  return record => dateFunc(internal.zonedEpochSlotsToIso(record, internal.queryNativeTimeZone));
}

function moveByTimeUnit$1(nanoInUnit, record, units) {
  const epochNano1 = internal.addBigNanos(record.epochNanoseconds, internal.numberToBigNano(internal.toStrictInteger(units), nanoInUnit));
  return {
    ...record,
    epochNanoseconds: internal.checkEpochNanoInBounds(epochNano1)
  };
}

function rountToInterval(unit, computeInterval, record, options) {
  const [, roundingMode] = internal.refineUnitRoundOptions(unit, options), timeZoneOps = internal.queryNativeTimeZone(record.timeZone), epochNano1 = internal.roundZonedEpochToInterval(computeInterval, timeZoneOps, record, roundingMode);
  return {
    ...record,
    epochNanoseconds: internal.checkEpochNanoInBounds(epochNano1)
  };
}

function aligned$2(computeAlignment, nanoDelta = 0) {
  return record => {
    const timeZoneOps = internal.queryNativeTimeZone(record.timeZone), epochNano1 = internal.moveBigNano(internal.alignZonedEpoch(computeAlignment, timeZoneOps, record), nanoDelta);
    return {
      ...record,
      epochNanoseconds: internal.checkEpochNanoInBounds(epochNano1)
    };
  };
}

function zonedTransform(transformIso) {
  return (record, ...args) => {
    const timeZoneOps = internal.queryNativeTimeZone(record.timeZone), isoSlots = internal.zonedEpochSlotsToIso(record, timeZoneOps), transformedIsoSlots = transformIso(isoSlots, ...args), epochNano1 = internal.getSingleInstantFor(timeZoneOps, transformedIsoSlots);
    return {
      ...record,
      epochNanoseconds: internal.checkEpochNanoInBounds(epochNano1)
    };
  };
}

function addYears$1(record, years, options) {
  return internal.checkIsoDateTimeInBounds(moveByYears(record, years, options));
}

function addMonths$1(record, months, options) {
  return internal.checkIsoDateTimeInBounds(moveByMonths(record, months, options));
}

function addWeeks$1(record, weeks) {
  return internal.checkIsoDateTimeInBounds(moveByIsoWeeks(record, weeks));
}

function addDays$1(record, days) {
  return internal.checkIsoDateTimeInBounds(moveByDaysStrict(record, days));
}

function moveByTimeUnit(nanoInUnit, record, units) {
  const epochNano0 = internal.isoToEpochNano(record), epochNano1 = internal.addBigNanos(epochNano0, internal.numberToBigNano(internal.toStrictInteger(units), nanoInUnit));
  return internal.checkIsoDateTimeInBounds({
    ...record,
    ...internal.epochNanoToIso(epochNano1, 0)
  });
}

function roundToInterval$1(unit, computeInterval, record, options) {
  const [, roundingMode] = internal.refineUnitRoundOptions(unit, options);
  return internal.createPlainDateTimeSlots(internal.checkIsoDateTimeInBounds(roundDateTimeToInterval(computeInterval, record, roundingMode)));
}

function aligned$1(computeAlignment, nanoDelta = 0) {
  return record0 => {
    let isoFields = computeAlignment(record0);
    return nanoDelta && (isoFields = internal.epochNanoToIso(internal.isoToEpochNano(isoFields), nanoDelta)), 
    internal.createPlainDateTimeSlots(internal.checkIsoDateTimeInBounds({
      ...record0,
      ...isoFields
    }));
  };
}

function addYears(record, years, options) {
  return internal.checkIsoDateInBounds(moveByYears(record, years, options));
}

function addMonths(record, months, options) {
  return internal.checkIsoDateInBounds(moveByMonths(record, months, options));
}

function addWeeks(record, weeks) {
  return internal.createPlainDateSlots(internal.checkIsoDateInBounds(moveByIsoWeeks(record, weeks)));
}

function addDays(record, days) {
  return internal.createPlainDateSlots(internal.checkIsoDateInBounds(moveByDaysStrict(record, days)));
}

function roundToInterval(unit, computeInterval, record0, options) {
  const [, roundingMode] = internal.refineUnitRoundOptions(unit, options), isoFields = roundDateTimeToInterval(computeInterval, record0, roundingMode);
  return internal.createPlainDateSlots(internal.checkIsoDateInBounds({
    ...record0,
    ...isoFields
  }));
}

function aligned(computeAlignment, dayDelta = 0) {
  return record0 => {
    const isoFields = internal.moveByDays(computeAlignment(record0), dayDelta);
    return internal.createPlainDateSlots(internal.checkIsoDateInBounds({
      ...record0,
      ...isoFields
    }));
  };
}

var internal = require("./internal.cjs");

const create$7 = internal.constructInstantSlots, fromEpochSeconds = internal.epochSecToInstant, fromEpochMilliseconds = internal.epochMilliToInstant, fromEpochMicroseconds = internal.epochMicroToInstant, fromEpochNanoseconds = internal.epochNanoToInstant, fromString$7 = internal.parseInstant, epochSeconds$1 = internal.getEpochSec, epochMilliseconds$1 = internal.getEpochMilli, epochMicroseconds$1 = internal.getEpochMicro, epochNanoseconds$1 = internal.getEpochNano, add$6 = internal.bindArgs(internal.moveInstant, 0), subtract$6 = internal.bindArgs(internal.moveInstant, 1), until$5 = internal.bindArgs(internal.diffInstants, 0), since$5 = internal.bindArgs(internal.diffInstants, 1), round$4 = internal.roundInstant, equals$6 = internal.instantsEqual, compare$6 = internal.compareInstants, prepFormat$6 = internal.createFormatPrepper(internal.instantConfig, createFormatCache()), toString$7 = internal.bindArgs(internal.formatInstantIso, internal.refineTimeZoneId, internal.queryNativeTimeZone), diffZonedYears = internal.bindArgs(diffZonedLargeUnits, 9), diffZonedMonths = internal.bindArgs(diffZonedLargeUnits, 8), diffZonedWeeks = internal.bindArgs(diffZonedDayLikeUnits, 7, 7), diffZonedDays = internal.bindArgs(diffZonedDayLikeUnits, 6, 1), diffZonedTimeUnits = internal.bindArgs(diffTimeUnit, internal.extractEpochNano), diffPlainYears = internal.bindArgs(diffPlainLargeUnits, 9), diffPlainMonths = internal.bindArgs(diffPlainLargeUnits, 8), diffPlainWeeks = internal.bindArgs(diffPlainDayLikeUnit, internal.isoToEpochNano, 7, 7), diffPlainDays = internal.bindArgs(diffPlainDayLikeUnit, internal.isoToEpochNano, 6, 1), diffPlainTimeUnits = internal.bindArgs(diffTimeUnit, internal.isoToEpochNano), dayOfMonthLabel = "dayOfMonth", dayLabel = "day", dayOfWeekLabel = "dayOfWeek", weekOfYearLabel = "weekOfYear", computeHourFloor = internal.bindArgs(internal.clearIsoFields, 5), computeMinuteFloor = internal.bindArgs(internal.clearIsoFields, 4), computeSecFloor = internal.bindArgs(internal.clearIsoFields, 3), computeMilliFloor = internal.bindArgs(internal.clearIsoFields, 2), computeMicroFloor = internal.bindArgs(internal.clearIsoFields, 1), create$6 = internal.bindArgs(internal.constructZonedDateTimeSlots, internal.refineCalendarId, internal.refineTimeZoneId), fromString$6 = internal.parseZonedDateTime, getFields$5 = internal.memoize((record => {
  const isoFields = internal.zonedEpochSlotsToIso(record, internal.queryNativeTimeZone), offsetString = internal.formatOffsetNano(isoFields.offsetNanoseconds);
  return {
    ...computeDateFields(isoFields),
    ...internal.isoTimeFieldsToCal(isoFields),
    offset: offsetString
  };
}), WeakMap), getISOFields$5 = internal.bindArgs(internal.buildZonedIsoFields, internal.queryNativeTimeZone), calendarId$4 = getCalendarId, epochSeconds = internal.getEpochSec, epochMilliseconds = internal.getEpochMilli, epochMicroseconds = internal.getEpochMicro, epochNanoseconds = internal.getEpochNano, dayOfWeek$2 = adaptDateFunc(internal.computeIsoDayOfWeek), daysInWeek$2 = adaptDateFunc(internal.computeIsoDaysInWeek), weekOfYear$2 = adaptDateFunc(computeWeekOfYear), yearOfWeek$2 = adaptDateFunc(computeYearOfWeek), dayOfYear$2 = adaptDateFunc(computeDayOfYear), daysInMonth$3 = adaptDateFunc(computeDaysInMonth), daysInYear$3 = adaptDateFunc(computeDaysInYear), monthsInYear$3 = adaptDateFunc(computeMonthsInYear), inLeapYear$3 = adaptDateFunc(computeInLeapYear), hoursInDay = internal.bindArgs(internal.computeZonedHoursInDay, internal.queryNativeTimeZone), withPlainDate$1 = internal.bindArgs(internal.zonedDateTimeWithPlainDate, internal.queryNativeTimeZone), withPlainTime$1 = internal.bindArgs(internal.zonedDateTimeWithPlainTime, internal.queryNativeTimeZone), add$5 = internal.bindArgs(internal.moveZonedDateTime, internal.createNativeMoveOps, internal.queryNativeTimeZone, 0), subtract$5 = internal.bindArgs(internal.moveZonedDateTime, internal.createNativeMoveOps, internal.queryNativeTimeZone, 1), until$4 = internal.bindArgs(internal.diffZonedDateTimes, internal.createNativeDiffOps, internal.queryNativeTimeZone, 0), since$4 = internal.bindArgs(internal.diffZonedDateTimes, internal.createNativeDiffOps, internal.queryNativeTimeZone, 1), round$3 = internal.bindArgs(internal.roundZonedDateTime, internal.queryNativeTimeZone), startOfDay$1 = internal.bindArgs(internal.computeZonedStartOfDay, internal.queryNativeTimeZone), equals$5 = internal.zonedDateTimesEqual, compare$5 = internal.compareZonedDateTimes, toInstant = internal.zonedDateTimeToInstant, toPlainDateTime$2 = internal.bindArgs(internal.zonedDateTimeToPlainDateTime, internal.queryNativeTimeZone), toPlainDate$3 = internal.bindArgs(internal.zonedDateTimeToPlainDate, internal.queryNativeTimeZone), toPlainTime$1 = internal.bindArgs(internal.zonedDateTimeToPlainTime, internal.queryNativeTimeZone), prepFormat$5 = internal.createFormatPrepper(internal.zonedConfig, createFormatCache()), toString$6 = internal.bindArgs(internal.formatZonedDateTimeIso, internal.queryNativeTimeZone), withDayOfYear$2 = zonedTransform(moveToDayOfYear), withDayOfMonth$2 = zonedTransform(moveToDayOfMonth), withDayOfWeek$2 = zonedTransform(moveToDayOfWeek), withWeekOfYear$2 = zonedTransform(slotsWithWeekOfYear), addYears$2 = zonedTransform(moveByYears), addMonths$2 = zonedTransform(moveByMonths), addWeeks$2 = zonedTransform(moveByIsoWeeks), addDays$2 = zonedTransform(moveByDaysStrict), addHours$1 = internal.bindArgs(moveByTimeUnit$1, internal.nanoInHour), addMinutes$1 = internal.bindArgs(moveByTimeUnit$1, internal.nanoInMinute), addSeconds$1 = internal.bindArgs(moveByTimeUnit$1, internal.nanoInSec), addMilliseconds$1 = internal.bindArgs(moveByTimeUnit$1, internal.nanoInMilli), addMicroseconds$1 = internal.bindArgs(moveByTimeUnit$1, internal.nanoInMicro), addNanoseconds$1 = internal.bindArgs(moveByTimeUnit$1, 1), subtractYears$2 = reversedMove(addYears$2), subtractMonths$2 = reversedMove(addMonths$2), subtractWeeks$2 = reversedMove(addWeeks$2), subtractDays$2 = reversedMove(addDays$2), subtractHours$1 = reversedMove(addHours$1), subtractMinutes$1 = reversedMove(addMinutes$1), subtractSeconds$1 = reversedMove(addSeconds$1), subtractMilliseconds$1 = reversedMove(addMilliseconds$1), subtractMicroseconds$1 = reversedMove(addMicroseconds$1), subtractNanoseconds$1 = reversedMove(addNanoseconds$1), roundToYear$2 = internal.bindArgs(rountToInterval, 9, computeYearInterval), roundToMonth$2 = internal.bindArgs(rountToInterval, 8, computeMonthInterval), roundToWeek$2 = internal.bindArgs(rountToInterval, 7, computeIsoWeekInterval), startOfYear$2 = aligned$2(computeYearFloor), startOfMonth$2 = aligned$2(computeMonthFloor), startOfWeek$2 = aligned$2(computeIsoWeekFloor), startOfHour$1 = aligned$2(computeHourFloor), startOfMinute$1 = aligned$2(computeMinuteFloor), startOfSecond$1 = aligned$2(computeSecFloor), startOfMillisecond$1 = aligned$2(computeMilliFloor), startOfMicrosecond$1 = aligned$2(computeMicroFloor), endOfYear$2 = aligned$2(computeYearCeil, -1), endOfMonth$2 = aligned$2(computeMonthCeil, -1), endOfWeek$2 = aligned$2(computeIsoWeekCeil, -1), endOfDay$1 = aligned$2(internal.computeDayFloor, internal.nanoInUtcDay - 1), endOfHour$1 = aligned$2(computeHourFloor, internal.nanoInHour - 1), endOfMinute$1 = aligned$2(computeMinuteFloor, internal.nanoInMinute - 1), endOfSecond$1 = aligned$2(computeSecFloor, internal.nanoInSec - 1), endOfMillisecond$1 = aligned$2(computeMilliFloor, internal.nanoInMilli - 1), endOfMicrosecond$1 = aligned$2(computeMicroFloor, internal.nanoInMicro - 1), diffYears$2 = diffZonedYears, diffMonths$2 = diffZonedMonths, diffWeeks$2 = diffZonedWeeks, diffDays$2 = diffZonedDays, diffHours$1 = internal.bindArgs(diffZonedTimeUnits, 5, internal.nanoInHour), diffMinutes$1 = internal.bindArgs(diffZonedTimeUnits, 4, internal.nanoInMinute), diffSeconds$1 = internal.bindArgs(diffZonedTimeUnits, 3, internal.nanoInSec), diffMilliseconds$1 = internal.bindArgs(diffZonedTimeUnits, 2, internal.nanoInMilli), diffMicroseconds$1 = internal.bindArgs(diffZonedTimeUnits, 1, internal.nanoInMicro), diffNanoseconds$1 = internal.bindArgs(diffZonedTimeUnits, 0, 1), create$5 = internal.bindArgs(internal.constructPlainDateTimeSlots, internal.refineCalendarId), fromString$5 = internal.parsePlainDateTime, getFields$4 = internal.memoize((record => ({
  ...computeDateFields(record),
  ...internal.isoTimeFieldsToCal(record)
})), WeakMap), getISOFields$4 = internal.identity, calendarId$3 = getCalendarId, dayOfWeek$1 = internal.computeIsoDayOfWeek, daysInWeek$1 = internal.computeIsoDaysInWeek, weekOfYear$1 = computeWeekOfYear, yearOfWeek$1 = computeYearOfWeek, dayOfYear$1 = computeDayOfYear, daysInMonth$2 = computeDaysInMonth, daysInYear$2 = computeDaysInYear, monthsInYear$2 = computeMonthsInYear, inLeapYear$2 = computeInLeapYear, withPlainDate = internal.plainDateTimeWithPlainDate, withPlainTime = internal.plainDateTimeWithPlainTime, add$4 = internal.bindArgs(internal.movePlainDateTime, internal.createNativeMoveOps, 0), subtract$4 = internal.bindArgs(internal.movePlainDateTime, internal.createNativeMoveOps, 1), until$3 = internal.bindArgs(internal.diffPlainDateTimes, internal.createNativeDiffOps, 0), since$3 = internal.bindArgs(internal.diffPlainDateTimes, internal.createNativeDiffOps, 1), round$2 = internal.roundPlainDateTime, equals$4 = internal.plainDateTimesEqual, compare$4 = internal.compareIsoDateTimeFields, toZonedDateTime$2 = internal.bindArgs(internal.plainDateTimeToZonedDateTime, internal.queryNativeTimeZone), toPlainDate$2 = internal.createPlainDateSlots, toPlainTime = internal.createPlainTimeSlots, prepFormat$4 = internal.createFormatPrepper(internal.dateTimeConfig, createFormatCache()), toString$5 = internal.formatPlainDateTimeIso, addHours = internal.bindArgs(moveByTimeUnit, internal.nanoInHour), addMinutes = internal.bindArgs(moveByTimeUnit, internal.nanoInMinute), addSeconds = internal.bindArgs(moveByTimeUnit, internal.nanoInSec), addMilliseconds = internal.bindArgs(moveByTimeUnit, internal.nanoInMilli), addMicroseconds = internal.bindArgs(moveByTimeUnit, internal.nanoInMicro), addNanoseconds = internal.bindArgs(moveByTimeUnit, 1), subtractYears$1 = reversedMove(addYears$1), subtractMonths$1 = reversedMove(addMonths$1), subtractWeeks$1 = reversedMove(addWeeks$1), subtractDays$1 = reversedMove(addDays$1), subtractHours = reversedMove(addHours), subtractMinutes = reversedMove(addMinutes), subtractSeconds = reversedMove(addSeconds), subtractMilliseconds = reversedMove(addMilliseconds), subtractMicroseconds = reversedMove(addMicroseconds), subtractNanoseconds = reversedMove(addNanoseconds), roundToYear$1 = internal.bindArgs(roundToInterval$1, 9, computeYearInterval), roundToMonth$1 = internal.bindArgs(roundToInterval$1, 8, computeMonthInterval), roundToWeek$1 = internal.bindArgs(roundToInterval$1, 7, computeIsoWeekInterval), startOfYear$1 = aligned$1(computeYearFloor), startOfMonth$1 = aligned$1(computeMonthFloor), startOfWeek$1 = aligned$1(computeIsoWeekFloor), startOfDay = aligned$1(internal.computeDayFloor), startOfHour = aligned$1(computeHourFloor), startOfMinute = aligned$1(computeMinuteFloor), startOfSecond = aligned$1(computeSecFloor), startOfMillisecond = aligned$1(computeMilliFloor), startOfMicrosecond = aligned$1(computeMicroFloor), endOfYear$1 = aligned$1(computeYearCeil, -1), endOfMonth$1 = aligned$1(computeMonthCeil, -1), endOfWeek$1 = aligned$1(computeIsoWeekCeil, -1), endOfDay = aligned$1(internal.computeDayFloor, internal.nanoInUtcDay - 1), endOfHour = aligned$1(computeHourFloor, internal.nanoInHour - 1), endOfMinute = aligned$1(computeMinuteFloor, internal.nanoInMinute - 1), endOfSecond = aligned$1(computeSecFloor, internal.nanoInSec - 1), endOfMillisecond = aligned$1(computeMilliFloor, internal.nanoInMilli - 1), endOfMicrosecond = aligned$1(computeMicroFloor, internal.nanoInMicro - 1), diffYears$1 = diffPlainYears, diffMonths$1 = diffPlainMonths, diffWeeks$1 = diffPlainWeeks, diffDays$1 = diffPlainDays, diffHours = internal.bindArgs(diffPlainTimeUnits, 5, internal.nanoInHour), diffMinutes = internal.bindArgs(diffPlainTimeUnits, 4, internal.nanoInMinute), diffSeconds = internal.bindArgs(diffPlainTimeUnits, 3, internal.nanoInSec), diffMilliseconds = internal.bindArgs(diffPlainTimeUnits, 2, internal.nanoInMilli), diffMicroseconds = internal.bindArgs(diffPlainTimeUnits, 1, internal.nanoInMicro), diffNanoseconds = internal.bindArgs(diffPlainTimeUnits, 0, 1), create$4 = internal.bindArgs(internal.constructPlainDateSlots, internal.refineCalendarId), fromString$4 = internal.parsePlainDate, getFields$3 = internal.memoize(computeDateFields, WeakMap), getISOFields$3 = internal.identity, calendarId$2 = getCalendarId, dayOfWeek = internal.computeIsoDayOfWeek, daysInWeek = internal.computeIsoDaysInWeek, weekOfYear = computeWeekOfYear, yearOfWeek = computeYearOfWeek, dayOfYear = computeDayOfYear, daysInMonth$1 = computeDaysInMonth, daysInYear$1 = computeDaysInYear, monthsInYear$1 = computeMonthsInYear, inLeapYear$1 = computeInLeapYear, add$3 = internal.bindArgs(internal.movePlainDate, internal.createNativeMoveOps, 0), subtract$3 = internal.bindArgs(internal.movePlainDate, internal.createNativeMoveOps, 1), until$2 = internal.bindArgs(internal.diffPlainDates, internal.createNativeDiffOps, 0), since$2 = internal.bindArgs(internal.diffPlainDates, internal.createNativeDiffOps, 1), equals$3 = internal.plainDatesEqual, compare$3 = internal.compareIsoDateFields, toPlainDateTime$1 = internal.plainDateToPlainDateTime, prepFormat$3 = internal.createFormatPrepper(internal.dateConfig, createFormatCache()), toString$4 = internal.formatPlainDateIso, subtractYears = reversedMove(addYears), subtractMonths = reversedMove(addMonths), subtractWeeks = reversedMove(addWeeks), subtractDays = reversedMove(addDays), roundToYear = internal.bindArgs(roundToInterval, 9, computeYearInterval), roundToMonth = internal.bindArgs(roundToInterval, 8, computeMonthInterval), roundToWeek = internal.bindArgs(roundToInterval, 7, computeIsoWeekInterval), startOfYear = aligned(computeYearFloor), startOfMonth = aligned(computeMonthFloor), startOfWeek = aligned(computeIsoWeekFloor), endOfYear = aligned(computeYearCeil, -1), endOfMonth = aligned(computeMonthCeil, -1), endOfWeek = aligned(computeIsoWeekCeil, -1), diffYears = diffPlainYears, diffMonths = diffPlainMonths, diffWeeks = diffPlainWeeks, diffDays = diffPlainDays, create$3 = internal.constructPlainTimeSlots, fromFields$3 = internal.refinePlainTimeBag, fromString$3 = internal.parsePlainTime, getFields$2 = internal.memoize(internal.isoTimeFieldsToCal, WeakMap), getISOFields$2 = internal.identity, add$2 = internal.bindArgs(internal.movePlainTime, 0), subtract$2 = internal.bindArgs(internal.movePlainTime, 1), until$1 = internal.bindArgs(internal.diffPlainTimes, 0), since$1 = internal.bindArgs(internal.diffPlainTimes, 1), round$1 = internal.roundPlainTime, equals$2 = internal.plainTimesEqual, compare$2 = internal.compareIsoTimeFields, toZonedDateTime = internal.bindArgs(internal.plainTimeToZonedDateTime, internal.refineTimeZoneId, internal.identity, internal.queryNativeTimeZone), toPlainDateTime = internal.plainTimeToPlainDateTime, prepFormat$2 = internal.createFormatPrepper(internal.timeConfig, createFormatCache()), toString$3 = internal.formatPlainTimeIso, create$2 = internal.bindArgs(internal.constructPlainYearMonthSlots, internal.refineCalendarId), fromString$2 = internal.bindArgs(internal.parsePlainYearMonth, internal.createNativeDayOps), getFields$1 = internal.memoize((slots => {
  const calendarOps = internal.createNativePartOps(slots.calendar), [year, month] = calendarOps.dateParts(slots), [era, eraYear] = calendarOps.eraParts(slots), [monthCodeNumber, isLeapMonth] = calendarOps.monthCodeParts(year, month);
  return {
    era: era,
    eraYear: eraYear,
    year: year,
    monthCode: internal.formatMonthCode(monthCodeNumber, isLeapMonth),
    month: month
  };
}), WeakMap), getISOFields$1 = internal.identity, calendarId$1 = getCalendarId, daysInMonth = computeDaysInMonth, daysInYear = computeDaysInYear, monthsInYear = computeMonthsInYear, inLeapYear = computeInLeapYear, add$1 = internal.bindArgs(internal.movePlainYearMonth, internal.createNativeYearMonthMoveOps, 0), subtract$1 = internal.bindArgs(internal.movePlainYearMonth, internal.createNativeYearMonthMoveOps, 1), until = internal.bindArgs(internal.diffPlainYearMonth, internal.createNativeYearMonthDiffOps, 0), since = internal.bindArgs(internal.diffPlainYearMonth, internal.createNativeYearMonthDiffOps, 1), equals$1 = internal.plainYearMonthsEqual, compare$1 = internal.compareIsoDateFields, prepFormat$1 = internal.createFormatPrepper(internal.yearMonthConfig, createFormatCache()), toString$2 = internal.formatPlainYearMonthIso, create$1 = internal.bindArgs(internal.constructPlainMonthDaySlots, internal.refineCalendarId), fromString$1 = internal.bindArgs(internal.parsePlainMonthDay, internal.createNativeMonthDayParseOps), getFields = internal.memoize((slots => {
  const calendarOps = internal.createNativePartOps(slots.calendar), [year, month, day] = calendarOps.dateParts(slots), [monthCodeNumber, isLeapMonth] = calendarOps.monthCodeParts(year, month);
  return {
    monthCode: internal.formatMonthCode(monthCodeNumber, isLeapMonth),
    month: month,
    day: day
  };
}), WeakMap), getISOFields = internal.identity, calendarId = getCalendarId, equals = internal.plainMonthDaysEqual, prepFormat = internal.createFormatPrepper(internal.monthDayConfig, createFormatCache()), toString$1 = internal.formatPlainMonthDayIso, create = internal.constructDurationSlots, fromFields = internal.refineDurationBag, fromString = internal.parseDuration, blank = internal.getDurationBlank, withFields = internal.durationWithFields, negated = internal.negateDuration, abs = internal.absDuration, add = internal.bindArgs(internal.addDurations, internal.identity, internal.createNativeDiffOps, internal.queryNativeTimeZone, 0), subtract = internal.bindArgs(internal.addDurations, internal.identity, internal.createNativeDiffOps, internal.queryNativeTimeZone, 1), round = internal.bindArgs(internal.roundDuration, internal.identity, internal.createNativeDiffOps, internal.queryNativeTimeZone), total = internal.bindArgs(internal.totalDuration, internal.identity, internal.createNativeDiffOps, internal.queryNativeTimeZone), compare = internal.bindArgs(internal.compareDurations, internal.identity, internal.createNativeDiffOps, internal.queryNativeTimeZone), toString = internal.formatDurationIso, timeZoneId = internal.getCurrentTimeZoneId;

exports.abs = abs, exports.add = add$6, exports.add$1 = add$5, exports.add$2 = add$4, 
exports.add$3 = add$3, exports.add$4 = add$2, exports.add$5 = add$1, exports.add$6 = add, 
exports.addDays = addDays$2, exports.addDays$1 = addDays$1, exports.addDays$2 = addDays, 
exports.addHours = addHours$1, exports.addHours$1 = addHours, exports.addMicroseconds = addMicroseconds$1, 
exports.addMicroseconds$1 = addMicroseconds, exports.addMilliseconds = addMilliseconds$1, 
exports.addMilliseconds$1 = addMilliseconds, exports.addMinutes = addMinutes$1, 
exports.addMinutes$1 = addMinutes, exports.addMonths = addMonths$2, exports.addMonths$1 = addMonths$1, 
exports.addMonths$2 = addMonths, exports.addNanoseconds = addNanoseconds$1, exports.addNanoseconds$1 = addNanoseconds, 
exports.addSeconds = addSeconds$1, exports.addSeconds$1 = addSeconds, exports.addWeeks = addWeeks$2, 
exports.addWeeks$1 = addWeeks$1, exports.addWeeks$2 = addWeeks, exports.addYears = addYears$2, 
exports.addYears$1 = addYears$1, exports.addYears$2 = addYears, exports.blank = blank, 
exports.calendarId = calendarId$4, exports.calendarId$1 = calendarId$3, exports.calendarId$2 = calendarId$2, 
exports.calendarId$3 = calendarId$1, exports.calendarId$4 = calendarId, exports.compare = compare$6, 
exports.compare$1 = compare$5, exports.compare$2 = compare$4, exports.compare$3 = compare$3, 
exports.compare$4 = compare$2, exports.compare$5 = compare$1, exports.compare$6 = compare, 
exports.create = create$7, exports.create$1 = create$6, exports.create$2 = create$5, 
exports.create$3 = create$4, exports.create$4 = create$3, exports.create$5 = create$2, 
exports.create$6 = create$1, exports.create$7 = create, exports.dayOfWeek = dayOfWeek$2, 
exports.dayOfWeek$1 = dayOfWeek$1, exports.dayOfWeek$2 = dayOfWeek, exports.dayOfYear = dayOfYear$2, 
exports.dayOfYear$1 = dayOfYear$1, exports.dayOfYear$2 = dayOfYear, exports.daysInMonth = daysInMonth$3, 
exports.daysInMonth$1 = daysInMonth$2, exports.daysInMonth$2 = daysInMonth$1, exports.daysInMonth$3 = daysInMonth, 
exports.daysInWeek = daysInWeek$2, exports.daysInWeek$1 = daysInWeek$1, exports.daysInWeek$2 = daysInWeek, 
exports.daysInYear = daysInYear$3, exports.daysInYear$1 = daysInYear$2, exports.daysInYear$2 = daysInYear$1, 
exports.daysInYear$3 = daysInYear, exports.diffDays = diffDays$2, exports.diffDays$1 = diffDays$1, 
exports.diffDays$2 = diffDays, exports.diffHours = diffHours$1, exports.diffHours$1 = diffHours, 
exports.diffMicroseconds = diffMicroseconds$1, exports.diffMicroseconds$1 = diffMicroseconds, 
exports.diffMilliseconds = diffMilliseconds$1, exports.diffMilliseconds$1 = diffMilliseconds, 
exports.diffMinutes = diffMinutes$1, exports.diffMinutes$1 = diffMinutes, exports.diffMonths = diffMonths$2, 
exports.diffMonths$1 = diffMonths$1, exports.diffMonths$2 = diffMonths, exports.diffNanoseconds = diffNanoseconds$1, 
exports.diffNanoseconds$1 = diffNanoseconds, exports.diffSeconds = diffSeconds$1, 
exports.diffSeconds$1 = diffSeconds, exports.diffWeeks = diffWeeks$2, exports.diffWeeks$1 = diffWeeks$1, 
exports.diffWeeks$2 = diffWeeks, exports.diffYears = diffYears$2, exports.diffYears$1 = diffYears$1, 
exports.diffYears$2 = diffYears, exports.endOfDay = endOfDay$1, exports.endOfDay$1 = endOfDay, 
exports.endOfHour = endOfHour$1, exports.endOfHour$1 = endOfHour, exports.endOfMicrosecond = endOfMicrosecond$1, 
exports.endOfMicrosecond$1 = endOfMicrosecond, exports.endOfMillisecond = endOfMillisecond$1, 
exports.endOfMillisecond$1 = endOfMillisecond, exports.endOfMinute = endOfMinute$1, 
exports.endOfMinute$1 = endOfMinute, exports.endOfMonth = endOfMonth$2, exports.endOfMonth$1 = endOfMonth$1, 
exports.endOfMonth$2 = endOfMonth, exports.endOfSecond = endOfSecond$1, exports.endOfSecond$1 = endOfSecond, 
exports.endOfWeek = endOfWeek$2, exports.endOfWeek$1 = endOfWeek$1, exports.endOfWeek$2 = endOfWeek, 
exports.endOfYear = endOfYear$2, exports.endOfYear$1 = endOfYear$1, exports.endOfYear$2 = endOfYear, 
exports.epochMicroseconds = epochMicroseconds$1, exports.epochMicroseconds$1 = epochMicroseconds, 
exports.epochMilliseconds = epochMilliseconds$1, exports.epochMilliseconds$1 = epochMilliseconds, 
exports.epochNanoseconds = epochNanoseconds$1, exports.epochNanoseconds$1 = epochNanoseconds, 
exports.epochSeconds = epochSeconds$1, exports.epochSeconds$1 = epochSeconds, exports.equals = equals$6, 
exports.equals$1 = equals$5, exports.equals$2 = equals$4, exports.equals$3 = equals$3, 
exports.equals$4 = equals$2, exports.equals$5 = equals$1, exports.equals$6 = equals, 
exports.fromEpochMicroseconds = fromEpochMicroseconds, exports.fromEpochMilliseconds = fromEpochMilliseconds, 
exports.fromEpochNanoseconds = fromEpochNanoseconds, exports.fromEpochSeconds = fromEpochSeconds, 
exports.fromFields = (fields, options) => {
  const calendarId = getCalendarIdFromBag(fields);
  return internal.refineZonedDateTimeBag(internal.refineTimeZoneId, internal.queryNativeTimeZone, internal.createNativeDateRefineOps(calendarId), calendarId, fields, options);
}, exports.fromFields$1 = (fields, options) => internal.refinePlainDateTimeBag(internal.createNativeDateRefineOps(getCalendarIdFromBag(fields)), fields, options), 
exports.fromFields$2 = (fields, options) => internal.refinePlainDateBag(internal.createNativeDateRefineOps(getCalendarIdFromBag(fields)), fields, options), 
exports.fromFields$3 = fromFields$3, exports.fromFields$4 = (fields, options) => internal.refinePlainYearMonthBag(internal.createNativeYearMonthRefineOps(getCalendarIdFromBag(fields)), fields, options), 
exports.fromFields$5 = (fields, options) => {
  const calendarMaybe = extractCalendarIdFromBag(fields), calendar = calendarMaybe || internal.isoCalendarId;
  return internal.refinePlainMonthDayBag(internal.createNativeMonthDayRefineOps(calendar), !calendarMaybe, fields, options);
}, exports.fromFields$6 = fromFields, exports.fromString = fromString$7, exports.fromString$1 = fromString$6, 
exports.fromString$2 = fromString$5, exports.fromString$3 = fromString$4, exports.fromString$4 = fromString$3, 
exports.fromString$5 = fromString$2, exports.fromString$6 = fromString$1, exports.fromString$7 = fromString, 
exports.getFields = getFields$5, exports.getFields$1 = getFields$4, exports.getFields$2 = getFields$3, 
exports.getFields$3 = getFields$2, exports.getFields$4 = getFields$1, exports.getFields$5 = getFields, 
exports.getISOFields = getISOFields$5, exports.getISOFields$1 = getISOFields$4, 
exports.getISOFields$2 = getISOFields$3, exports.getISOFields$3 = getISOFields$2, 
exports.getISOFields$4 = getISOFields$1, exports.getISOFields$5 = getISOFields, 
exports.hoursInDay = hoursInDay, exports.inLeapYear = inLeapYear$3, exports.inLeapYear$1 = inLeapYear$2, 
exports.inLeapYear$2 = inLeapYear$1, exports.inLeapYear$3 = inLeapYear, exports.instant = () => internal.createInstantSlots(internal.getCurrentEpochNano()), 
exports.isInstance = record => Boolean(record) && record.branding === internal.InstantBranding, 
exports.isInstance$1 = record => Boolean(record) && record.branding === internal.ZonedDateTimeBranding, 
exports.isInstance$2 = record => Boolean(record) && record.branding === internal.PlainDateTimeBranding, 
exports.isInstance$3 = record => Boolean(record) && record.branding === internal.PlainDateBranding, 
exports.isInstance$4 = record => Boolean(record) && record.branding === internal.PlainTimeBranding, 
exports.isInstance$5 = record => Boolean(record) && record.branding === internal.PlainYearMonthBranding, 
exports.isInstance$6 = record => Boolean(record) && record.branding === internal.PlainMonthDayBranding, 
exports.isInstance$7 = record => Boolean(record) && record.branding === internal.DurationBranding, 
exports.monthsInYear = monthsInYear$3, exports.monthsInYear$1 = monthsInYear$2, 
exports.monthsInYear$2 = monthsInYear$1, exports.monthsInYear$3 = monthsInYear, 
exports.negated = negated, exports.offset = record => internal.formatOffsetNano(offsetNanoseconds(record)), 
exports.offsetNanoseconds = offsetNanoseconds, exports.plainDate = (calendar, timeZone = internal.getCurrentTimeZoneId()) => internal.createPlainDateSlots(internal.getCurrentIsoDateTime(internal.queryNativeTimeZone(internal.refineTimeZoneId(timeZone))), internal.refineCalendarId(calendar)), 
exports.plainDateISO = (timeZone = internal.getCurrentTimeZoneId()) => internal.createPlainDateSlots(internal.getCurrentIsoDateTime(internal.queryNativeTimeZone(internal.refineTimeZoneId(timeZone))), internal.isoCalendarId), 
exports.plainDateTime = (calendar, timeZone = internal.getCurrentTimeZoneId()) => internal.createPlainDateTimeSlots(internal.getCurrentIsoDateTime(internal.queryNativeTimeZone(internal.refineTimeZoneId(timeZone))), internal.refineCalendarId(calendar)), 
exports.plainDateTimeISO = (timeZone = internal.getCurrentTimeZoneId()) => internal.createPlainDateTimeSlots(internal.getCurrentIsoDateTime(internal.queryNativeTimeZone(internal.refineTimeZoneId(timeZone))), internal.isoCalendarId), 
exports.plainTimeISO = (timeZone = internal.getCurrentTimeZoneId()) => internal.createPlainTimeSlots(internal.getCurrentIsoDateTime(internal.queryNativeTimeZone(internal.refineTimeZoneId(timeZone)))), 
exports.rangeToLocaleString = (record0, record1, locales, options) => {
  const [format, epochMilli0, epochMilli1] = prepFormat$6(locales, options, record0, record1);
  return format.formatRange(epochMilli0, epochMilli1);
}, exports.rangeToLocaleString$1 = (record0, record1, locales, options) => {
  const [format, epochMilli0, epochMilli1] = prepFormat$5(locales, options, record0, record1);
  return format.formatRange(epochMilli0, epochMilli1);
}, exports.rangeToLocaleString$2 = (record0, record1, locales, options) => {
  const [format, epochMilli0, epochMilli1] = prepFormat$4(locales, options, record0, record1);
  return format.formatRange(epochMilli0, epochMilli1);
}, exports.rangeToLocaleString$3 = (record0, record1, locales, options) => {
  const [format, epochMilli0, epochMilli1] = prepFormat$3(locales, options, record0, record1);
  return format.formatRange(epochMilli0, epochMilli1);
}, exports.rangeToLocaleString$4 = (record0, record1, locales, options) => {
  const [format, epochMilli0, epochMilli1] = prepFormat$2(locales, options, record0, record1);
  return format.formatRange(epochMilli0, epochMilli1);
}, exports.rangeToLocaleString$5 = (record0, record1, locales, options) => {
  const [format, epochMilli0, epochMilli1] = prepFormat$1(locales, options, record0, record1);
  return format.formatRange(epochMilli0, epochMilli1);
}, exports.rangeToLocaleString$6 = (record0, record1, locales, options) => {
  const [format, epochMilli0, epochMilli1] = prepFormat(locales, options, record0, record1);
  return format.formatRange(epochMilli0, epochMilli1);
}, exports.rangeToLocaleStringParts = (record0, record1, locales, options) => {
  const [format, epochMilli0, epochMilli1] = prepFormat$6(locales, options, record0, record1);
  return format.formatRangeToParts(epochMilli0, epochMilli1);
}, exports.rangeToLocaleStringParts$1 = (record0, record1, locales, options) => {
  const [format, epochMilli0, epochMilli1] = prepFormat$5(locales, options, record0, record1);
  return format.formatRangeToParts(epochMilli0, epochMilli1);
}, exports.rangeToLocaleStringParts$2 = (record0, record1, locales, options) => {
  const [format, epochMilli0, epochMilli1] = prepFormat$4(locales, options, record0, record1);
  return format.formatRangeToParts(epochMilli0, epochMilli1);
}, exports.rangeToLocaleStringParts$3 = (record0, record1, locales, options) => {
  const [format, epochMilli0, epochMilli1] = prepFormat$3(locales, options, record0, record1);
  return format.formatRangeToParts(epochMilli0, epochMilli1);
}, exports.rangeToLocaleStringParts$4 = (record0, record1, locales, options) => {
  const [format, epochMilli0, epochMilli1] = prepFormat$2(locales, options, record0, record1);
  return format.formatRangeToParts(epochMilli0, epochMilli1);
}, exports.rangeToLocaleStringParts$5 = (record0, record1, locales, options) => {
  const [format, epochMilli0, epochMilli1] = prepFormat$1(locales, options, record0, record1);
  return format.formatRangeToParts(epochMilli0, epochMilli1);
}, exports.rangeToLocaleStringParts$6 = (record0, record1, locales, options) => {
  const [format, epochMilli0, epochMilli1] = prepFormat(locales, options, record0, record1);
  return format.formatRangeToParts(epochMilli0, epochMilli1);
}, exports.round = round$4, exports.round$1 = round$3, exports.round$2 = round$2, 
exports.round$3 = round$1, exports.round$4 = round, exports.roundToMonth = roundToMonth$2, 
exports.roundToMonth$1 = roundToMonth$1, exports.roundToMonth$2 = roundToMonth, 
exports.roundToWeek = roundToWeek$2, exports.roundToWeek$1 = roundToWeek$1, exports.roundToWeek$2 = roundToWeek, 
exports.roundToYear = roundToYear$2, exports.roundToYear$1 = roundToYear$1, exports.roundToYear$2 = roundToYear, 
exports.since = since$5, exports.since$1 = since$4, exports.since$2 = since$3, exports.since$3 = since$2, 
exports.since$4 = since$1, exports.since$5 = since, exports.startOfDay = startOfDay$1, 
exports.startOfDay$1 = startOfDay, exports.startOfHour = startOfHour$1, exports.startOfHour$1 = startOfHour, 
exports.startOfMicrosecond = startOfMicrosecond$1, exports.startOfMicrosecond$1 = startOfMicrosecond, 
exports.startOfMillisecond = startOfMillisecond$1, exports.startOfMillisecond$1 = startOfMillisecond, 
exports.startOfMinute = startOfMinute$1, exports.startOfMinute$1 = startOfMinute, 
exports.startOfMonth = startOfMonth$2, exports.startOfMonth$1 = startOfMonth$1, 
exports.startOfMonth$2 = startOfMonth, exports.startOfSecond = startOfSecond$1, 
exports.startOfSecond$1 = startOfSecond, exports.startOfWeek = startOfWeek$2, exports.startOfWeek$1 = startOfWeek$1, 
exports.startOfWeek$2 = startOfWeek, exports.startOfYear = startOfYear$2, exports.startOfYear$1 = startOfYear$1, 
exports.startOfYear$2 = startOfYear, exports.subtract = subtract$6, exports.subtract$1 = subtract$5, 
exports.subtract$2 = subtract$4, exports.subtract$3 = subtract$3, exports.subtract$4 = subtract$2, 
exports.subtract$5 = subtract$1, exports.subtract$6 = subtract, exports.subtractDays = subtractDays$2, 
exports.subtractDays$1 = subtractDays$1, exports.subtractDays$2 = subtractDays, 
exports.subtractHours = subtractHours$1, exports.subtractHours$1 = subtractHours, 
exports.subtractMicroseconds = subtractMicroseconds$1, exports.subtractMicroseconds$1 = subtractMicroseconds, 
exports.subtractMilliseconds = subtractMilliseconds$1, exports.subtractMilliseconds$1 = subtractMilliseconds, 
exports.subtractMinutes = subtractMinutes$1, exports.subtractMinutes$1 = subtractMinutes, 
exports.subtractMonths = subtractMonths$2, exports.subtractMonths$1 = subtractMonths$1, 
exports.subtractMonths$2 = subtractMonths, exports.subtractNanoseconds = subtractNanoseconds$1, 
exports.subtractNanoseconds$1 = subtractNanoseconds, exports.subtractSeconds = subtractSeconds$1, 
exports.subtractSeconds$1 = subtractSeconds, exports.subtractWeeks = subtractWeeks$2, 
exports.subtractWeeks$1 = subtractWeeks$1, exports.subtractWeeks$2 = subtractWeeks, 
exports.subtractYears = subtractYears$2, exports.subtractYears$1 = subtractYears$1, 
exports.subtractYears$2 = subtractYears, exports.timeZoneId = record => record.timeZone, 
exports.timeZoneId$1 = timeZoneId, exports.toInstant = toInstant, exports.toLocaleString = (record, locales, options) => {
  const [format, epochMilli] = prepFormat$6(locales, options, record);
  return format.format(epochMilli);
}, exports.toLocaleString$1 = (record, locales, options) => {
  const [format, epochMilli] = prepFormat$5(locales, options, record);
  return format.format(epochMilli);
}, exports.toLocaleString$2 = (record, locales, options) => {
  const [format, epochMilli] = prepFormat$4(locales, options, record);
  return format.format(epochMilli);
}, exports.toLocaleString$3 = (record, locales, options) => {
  const [format, epochMilli] = prepFormat$3(locales, options, record);
  return format.format(epochMilli);
}, exports.toLocaleString$4 = (record, locales, options) => {
  const [format, epochMilli] = prepFormat$2(locales, options, record);
  return format.format(epochMilli);
}, exports.toLocaleString$5 = (record, locales, options) => {
  const [format, epochMilli] = prepFormat$1(locales, options, record);
  return format.format(epochMilli);
}, exports.toLocaleString$6 = (record, locales, options) => {
  const [format, epochMilli] = prepFormat(locales, options, record);
  return format.format(epochMilli);
}, exports.toLocaleString$7 = (record, locales, options) => Intl.DurationFormat ? new Intl.DurationFormat(locales, options).format(record) : internal.formatDurationIso(record), 
exports.toLocaleStringParts = (record, locales, options) => {
  const [format, epochMilli] = prepFormat$6(locales, options, record);
  return format.formatToParts(epochMilli);
}, exports.toLocaleStringParts$1 = (record, locales, options) => {
  const [format, epochMilli] = prepFormat$5(locales, options, record);
  return format.formatToParts(epochMilli);
}, exports.toLocaleStringParts$2 = (record, locales, options) => {
  const [format, epochMilli] = prepFormat$4(locales, options, record);
  return format.formatToParts(epochMilli);
}, exports.toLocaleStringParts$3 = (record, locales, options) => {
  const [format, epochMilli] = prepFormat$3(locales, options, record);
  return format.formatToParts(epochMilli);
}, exports.toLocaleStringParts$4 = (record, locales, options) => {
  const [format, epochMilli] = prepFormat$2(locales, options, record);
  return format.formatToParts(epochMilli);
}, exports.toLocaleStringParts$5 = (record, locales, options) => {
  const [format, epochMilli] = prepFormat$1(locales, options, record);
  return format.formatToParts(epochMilli);
}, exports.toLocaleStringParts$6 = (record, locales, options) => {
  const [format, epochMilli] = prepFormat(locales, options, record);
  return format.formatToParts(epochMilli);
}, exports.toPlainDate = toPlainDate$3, exports.toPlainDate$1 = toPlainDate$2, exports.toPlainDate$2 = (record, fields) => internal.plainYearMonthToPlainDate(internal.createNativeDateModOps, record, getFields$1(record), fields), 
exports.toPlainDate$3 = (record, fields) => internal.plainMonthDayToPlainDate(internal.createNativeDateModOps, record, getFields(record), fields), 
exports.toPlainDateTime = toPlainDateTime$2, exports.toPlainDateTime$1 = toPlainDateTime$1, 
exports.toPlainDateTime$2 = toPlainDateTime, exports.toPlainMonthDay = record => internal.zonedDateTimeToPlainMonthDay(internal.createNativeMonthDayRefineOps, record, getFields$5(record)), 
exports.toPlainMonthDay$1 = record => internal.plainDateTimeToPlainMonthDay(internal.createNativeMonthDayRefineOps, record, getFields$4(record)), 
exports.toPlainMonthDay$2 = record => internal.plainDateToPlainMonthDay(internal.createNativeMonthDayRefineOps, record, getFields$3(record)), 
exports.toPlainTime = toPlainTime$1, exports.toPlainTime$1 = toPlainTime, exports.toPlainYearMonth = record => internal.zonedDateTimeToPlainYearMonth(internal.createNativeYearMonthRefineOps, record, getFields$5(record)), 
exports.toPlainYearMonth$1 = record => internal.plainDateTimeToPlainYearMonth(internal.createNativeYearMonthRefineOps, record, getFields$4(record)), 
exports.toPlainYearMonth$2 = record => internal.plainDateToPlainYearMonth(internal.createNativeYearMonthRefineOps, record, getFields$3(record)), 
exports.toString = toString$7, exports.toString$1 = toString$6, exports.toString$2 = toString$5, 
exports.toString$3 = toString$4, exports.toString$4 = toString$3, exports.toString$5 = toString$2, 
exports.toString$6 = toString$1, exports.toString$7 = toString, exports.toZonedDateTime = (record, options) => {
  const refinedObj = internal.requireObjectLike(options);
  return internal.instantToZonedDateTime(record, internal.refineTimeZoneId(refinedObj.timeZone), internal.refineCalendarId(refinedObj.calendar));
}, exports.toZonedDateTime$1 = toZonedDateTime$2, exports.toZonedDateTime$2 = (record, options) => {
  const optionsObj = "string" == typeof options ? {
    timeZone: options
  } : options;
  return internal.plainDateToZonedDateTime(internal.refineTimeZoneId, internal.identity, internal.queryNativeTimeZone, record, optionsObj);
}, exports.toZonedDateTime$3 = toZonedDateTime, exports.toZonedDateTimeISO = (record, timeZone) => internal.instantToZonedDateTime(record, internal.refineTimeZoneId(timeZone)), 
exports.total = total, exports.until = until$5, exports.until$1 = until$4, exports.until$2 = until$3, 
exports.until$3 = until$2, exports.until$4 = until$1, exports.until$5 = until, exports.weekOfYear = weekOfYear$2, 
exports.weekOfYear$1 = weekOfYear$1, exports.weekOfYear$2 = weekOfYear, exports.withCalendar = (record, calendar) => internal.slotsWithCalendarId(record, internal.refineCalendarId(calendar)), 
exports.withCalendar$1 = (record, calendar) => internal.slotsWithCalendarId(record, internal.refineCalendarId(calendar)), 
exports.withCalendar$2 = (record, calendar) => internal.slotsWithCalendarId(record, internal.refineCalendarId(calendar)), 
exports.withDayOfMonth = withDayOfMonth$2, exports.withDayOfMonth$1 = (record, dayOfMonth, options) => internal.checkIsoDateTimeInBounds(moveToDayOfMonth(record, dayOfMonth, options)), 
exports.withDayOfMonth$2 = (record, dayOfMonth, options) => internal.createPlainDateSlots(internal.checkIsoDateInBounds(moveToDayOfMonth(record, dayOfMonth, options))), 
exports.withDayOfWeek = withDayOfWeek$2, exports.withDayOfWeek$1 = (record, dayOfWeek, options) => internal.checkIsoDateTimeInBounds(moveToDayOfWeek(record, dayOfWeek, options)), 
exports.withDayOfWeek$2 = (record, dayOfWeek, options) => internal.createPlainDateSlots(internal.checkIsoDateInBounds(moveToDayOfWeek(record, dayOfWeek, options))), 
exports.withDayOfYear = withDayOfYear$2, exports.withDayOfYear$1 = (record, dayOfYear, options) => internal.checkIsoDateTimeInBounds(moveToDayOfYear(record, dayOfYear, options)), 
exports.withDayOfYear$2 = (record, dayOfYear, options) => internal.createPlainDateSlots(internal.checkIsoDateInBounds(moveToDayOfYear(record, dayOfYear, options))), 
exports.withFields = (record, fields, options) => internal.zonedDateTimeWithFields(internal.createNativeDateModOps, internal.queryNativeTimeZone, record, fields, options), 
exports.withFields$1 = (record, fields, options) => internal.plainDateTimeWithFields(internal.createNativeDateModOps, record, fields, options), 
exports.withFields$2 = (record, fields, options) => internal.plainDateWithFields(internal.createNativeDateModOps, record, fields, options), 
exports.withFields$3 = (record, fields, options) => internal.plainTimeWithFields(getFields$2(record), fields, options), 
exports.withFields$4 = (record, fields, options) => internal.plainYearMonthWithFields(internal.createNativeYearMonthModOps, record, fields, options), 
exports.withFields$5 = (record, fields, options) => internal.plainMonthDayWithFields(internal.createNativeMonthDayModOps, record, fields, options), 
exports.withFields$6 = withFields, exports.withPlainDate = withPlainDate$1, exports.withPlainDate$1 = withPlainDate, 
exports.withPlainTime = withPlainTime$1, exports.withPlainTime$1 = withPlainTime, 
exports.withTimeZone = (record, timeZone) => internal.slotsWithTimeZoneId(record, internal.refineTimeZoneId(timeZone)), 
exports.withWeekOfYear = withWeekOfYear$2, exports.withWeekOfYear$1 = (record, weekOfYear, options) => internal.checkIsoDateTimeInBounds(slotsWithWeekOfYear(record, weekOfYear, options)), 
exports.withWeekOfYear$2 = (record, weekOfYear, options) => internal.createPlainDateSlots(internal.checkIsoDateInBounds(slotsWithWeekOfYear(record, weekOfYear, options))), 
exports.yearOfWeek = yearOfWeek$2, exports.yearOfWeek$1 = yearOfWeek$1, exports.yearOfWeek$2 = yearOfWeek, 
exports.zonedDateTime = (calendar, timeZone = internal.getCurrentTimeZoneId()) => internal.createZonedDateTimeSlots(internal.getCurrentEpochNano(), internal.refineTimeZoneId(timeZone), internal.refineCalendarId(calendar)), 
exports.zonedDateTimeISO = (timeZone = internal.getCurrentTimeZoneId()) => internal.createZonedDateTimeSlots(internal.getCurrentEpochNano(), internal.refineTimeZoneId(timeZone), internal.isoCalendarId);
