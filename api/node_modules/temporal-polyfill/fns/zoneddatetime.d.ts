import { ZonedDateTimeBranding, BigNano, ZonedDateTimeFields, ZonedDateTimeBag, DateTimeBag, ZonedIsoFields, ZonedFieldOptions, OverflowOptions, DiffOptions, UnitName, RoundingOptions, DayTimeUnitName, ZonedDateTimeDisplayOptions, NumberSign, LocalesArg, DateSlots, IsoDateFields, RoundingMathOptions, ZonedDateTimeSlots, Marker } from '../chunks/internal.js';
import * as temporal_spec from 'temporal-spec';
import { Record as Record$3 } from './duration.js';
import { Record as Record$4 } from './instant.js';
import { Record as Record$1 } from './plaindate.js';
import { Record as Record$5 } from './plaindatetime.js';
import { Record as Record$7 } from './plainmonthday.js';
import { Record as Record$2 } from './plaintime.js';
import { Record as Record$6 } from './plainyearmonth.js';


type Record = {
    /**
     * @deprecated Use the isInstance() function instead.
     */
    readonly branding: typeof ZonedDateTimeBranding;
    /**
     * @deprecated Use the calendarId() function instead.
     */
    readonly calendar: string;
    /**
     * @deprecated Use the timeZoneId() function instead.
     */
    readonly timeZone: string;
    /**
     * @deprecated Use the epochNanoseconds() function instead.
     */
    readonly epochNanoseconds: BigNano;
};
type Fields = ZonedDateTimeFields;
type FromFields = ZonedDateTimeBag;
type WithFields = DateTimeBag;
type ISOFields = ZonedIsoFields;
type AssignmentOptions = ZonedFieldOptions;
type ArithmeticOptions = OverflowOptions;
type DifferenceOptions = DiffOptions<UnitName>;
type RoundOptions = RoundingOptions<DayTimeUnitName>;
type ToStringOptions = ZonedDateTimeDisplayOptions;
declare const create: (epochNanoseconds: bigint, timeZone: string, calendar?: string) => Record;
declare function fromFields(fields: FromFields, options?: AssignmentOptions): Record;
declare const fromString: (s: string, options?: AssignmentOptions) => Record;
declare function isInstance(record: any): record is Record;
declare const getFields: (key: Record) => ZonedDateTimeFields;
declare const getISOFields: (record: Record) => ISOFields;
declare const calendarId: (record: Record) => string;
declare function timeZoneId(record: Record): string;
declare const epochSeconds: (record: Record) => number;
declare const epochMilliseconds: (record: Record) => number;
declare const epochMicroseconds: (record: Record) => bigint;
declare const epochNanoseconds: (record: Record) => bigint;
declare function offsetNanoseconds(record: Record): number;
declare function offset(record: Record): string;
declare const dayOfWeek: (record: Record) => number;
declare const daysInWeek: (record: Record) => number;
declare const weekOfYear: (record: Record) => number | undefined;
declare const yearOfWeek: (record: Record) => number | undefined;
declare const dayOfYear: (record: Record) => number;
declare const daysInMonth: (record: Record) => number;
declare const daysInYear: (record: Record) => number;
declare const monthsInYear: (record: Record) => number;
declare const inLeapYear: (record: Record) => boolean;
declare const hoursInDay: (record: Record) => number;
declare function withFields(record: Record, fields: WithFields, options?: AssignmentOptions): Record;
declare function withCalendar(record: Record, calendar: string): Record;
declare function withTimeZone(record: Record, timeZone: string): Record;
declare const withPlainDate: (zonedDateTimeRecord: Record, plainDateRecord: Record$1) => Record;
declare const withPlainTime: (zonedDateTimeRecord: Record, plainTimeRecord?: Record$2) => Record;
declare const add: (zonedDateTimeRecord: Record, durationRecord: Record$3, options?: ArithmeticOptions) => Record;
declare const subtract: (zonedDateTimeRecord: Record, durationRecord: Record$3, options?: ArithmeticOptions) => Record;
declare const until: (record0: Record, record1: Record, options?: DifferenceOptions) => Record$3;
declare const since: (record0: Record, record1: Record, options?: DifferenceOptions) => Record$3;
declare const round: (record: Record, options: DayTimeUnitName | RoundOptions) => Record;
declare const startOfDay: (record: Record) => Record;
declare const equals: (record0: Record, record1: Record) => boolean;
declare const compare: (record0: Record, record1: Record) => NumberSign;
declare const toInstant: (record: Record) => Record$4;
declare const toPlainDateTime: (record: Record) => Record$5;
declare const toPlainDate: (record: Record) => Record$1;
declare const toPlainTime: (record: Record) => Record$2;
declare function toPlainYearMonth(record: Record): Record$6;
declare function toPlainMonthDay(record: Record): Record$7;
declare function toLocaleString(record: Record, locales?: LocalesArg, options?: Intl.DateTimeFormatOptions): string;
declare function toLocaleStringParts(record: Record, locales?: LocalesArg, options?: Intl.DateTimeFormatOptions): Intl.DateTimeFormatPart[];
declare function rangeToLocaleString(record0: Record, record1: Record, locales?: LocalesArg, options?: Intl.DateTimeFormatOptions): string;
declare function rangeToLocaleStringParts(record0: Record, record1: Record, locales?: LocalesArg, options?: Intl.DateTimeFormatOptions): ReturnType<Intl.DateTimeFormat['formatRangeToParts']>;
declare const toString: (record: Record, options?: ToStringOptions) => string;
declare const withDayOfYear: <S extends DateSlots>(record: Record, dayOfYear: number, options?: OverflowOptions | undefined) => Record;
declare const withDayOfMonth: <S extends DateSlots>(record: Record, day: number, options?: OverflowOptions | undefined) => Record;
declare const withDayOfWeek: <S extends IsoDateFields>(record: Record, dayOfWeek: number, options?: OverflowOptions | undefined) => Record;
declare const withWeekOfYear: <S extends DateSlots>(record: Record, weekOfYear: number, options?: OverflowOptions | undefined) => Record;
declare const addYears: <S extends DateSlots>(record: Record, years: number, options?: OverflowOptions | undefined) => Record;
declare const addMonths: <S extends DateSlots>(record: Record, months: number, options?: OverflowOptions | undefined) => Record;
declare const addWeeks: <F extends IsoDateFields>(record: Record, weeks: number) => Record;
declare const addDays: <F extends IsoDateFields>(record: Record, weeks: number) => Record;
declare const addHours: (record: Record, units: number) => Record;
declare const addMinutes: (record: Record, units: number) => Record;
declare const addSeconds: (record: Record, units: number) => Record;
declare const addMilliseconds: (record: Record, units: number) => Record;
declare const addMicroseconds: (record: Record, units: number) => Record;
declare const addNanoseconds: (record: Record, units: number) => Record;
declare const subtractYears: <S extends DateSlots>(slots: Record, units: number, options?: OverflowOptions | undefined) => Record;
declare const subtractMonths: <S extends DateSlots>(slots: Record, units: number, options?: OverflowOptions | undefined) => Record;
declare const subtractWeeks: <F extends IsoDateFields>(slots: Record, units: number, options?: OverflowOptions | undefined) => Record;
declare const subtractDays: <F extends IsoDateFields>(slots: Record, units: number, options?: OverflowOptions | undefined) => Record;
declare const subtractHours: (slots: Record, units: number, options?: OverflowOptions | undefined) => Record;
declare const subtractMinutes: (slots: Record, units: number, options?: OverflowOptions | undefined) => Record;
declare const subtractSeconds: (slots: Record, units: number, options?: OverflowOptions | undefined) => Record;
declare const subtractMilliseconds: (slots: Record, units: number, options?: OverflowOptions | undefined) => Record;
declare const subtractMicroseconds: (slots: Record, units: number, options?: OverflowOptions | undefined) => Record;
declare const subtractNanoseconds: (slots: Record, units: number, options?: OverflowOptions | undefined) => Record;
declare const roundToYear: (record: Record, options?: temporal_spec.Temporal.RoundingMode | RoundingMathOptions | undefined) => Record;
declare const roundToMonth: (record: Record, options?: temporal_spec.Temporal.RoundingMode | RoundingMathOptions | undefined) => Record;
declare const roundToWeek: (record: Record, options?: temporal_spec.Temporal.RoundingMode | RoundingMathOptions | undefined) => Record;
declare const startOfYear: (record: Record) => Record;
declare const startOfMonth: (record: Record) => Record;
declare const startOfWeek: (record: Record) => Record;
declare const startOfHour: (record: Record) => Record;
declare const startOfMinute: (record: Record) => Record;
declare const startOfSecond: (record: Record) => Record;
declare const startOfMillisecond: (record: Record) => Record;
declare const startOfMicrosecond: (record: Record) => Record;
declare const endOfYear: (record: Record) => Record;
declare const endOfMonth: (record: Record) => Record;
declare const endOfWeek: (record: Record) => Record;
declare const endOfDay: (record: Record) => Record;
declare const endOfHour: (record: Record) => Record;
declare const endOfMinute: (record: Record) => Record;
declare const endOfSecond: (record: Record) => Record;
declare const endOfMillisecond: (record: Record) => Record;
declare const endOfMicrosecond: (record: Record) => Record;
declare const diffYears: (record0: ZonedDateTimeSlots, record1: ZonedDateTimeSlots, options?: temporal_spec.Temporal.RoundingMode | RoundingMathOptions | undefined) => number;
declare const diffMonths: (record0: ZonedDateTimeSlots, record1: ZonedDateTimeSlots, options?: temporal_spec.Temporal.RoundingMode | RoundingMathOptions | undefined) => number;
declare const diffWeeks: (record0: ZonedDateTimeSlots, record1: ZonedDateTimeSlots, options?: temporal_spec.Temporal.RoundingMode | RoundingMathOptions | undefined) => number;
declare const diffDays: (record0: ZonedDateTimeSlots, record1: ZonedDateTimeSlots, options?: temporal_spec.Temporal.RoundingMode | RoundingMathOptions | undefined) => number;
declare const diffHours: (record0: Marker, record1: Marker, options?: temporal_spec.Temporal.RoundingMode | RoundingMathOptions | undefined) => number;
declare const diffMinutes: (record0: Marker, record1: Marker, options?: temporal_spec.Temporal.RoundingMode | RoundingMathOptions | undefined) => number;
declare const diffSeconds: (record0: Marker, record1: Marker, options?: temporal_spec.Temporal.RoundingMode | RoundingMathOptions | undefined) => number;
declare const diffMilliseconds: (record0: Marker, record1: Marker, options?: temporal_spec.Temporal.RoundingMode | RoundingMathOptions | undefined) => number;
declare const diffMicroseconds: (record0: Marker, record1: Marker, options?: temporal_spec.Temporal.RoundingMode | RoundingMathOptions | undefined) => number;
declare const diffNanoseconds: (record0: Marker, record1: Marker, options?: temporal_spec.Temporal.RoundingMode | RoundingMathOptions | undefined) => number;

export { type ArithmeticOptions, type AssignmentOptions, type DifferenceOptions, type Fields, type FromFields, type ISOFields, type Record, type RoundOptions, type ToStringOptions, type WithFields, add, addDays, addHours, addMicroseconds, addMilliseconds, addMinutes, addMonths, addNanoseconds, addSeconds, addWeeks, addYears, calendarId, compare, create, dayOfWeek, dayOfYear, daysInMonth, daysInWeek, daysInYear, diffDays, diffHours, diffMicroseconds, diffMilliseconds, diffMinutes, diffMonths, diffNanoseconds, diffSeconds, diffWeeks, diffYears, endOfDay, endOfHour, endOfMicrosecond, endOfMillisecond, endOfMinute, endOfMonth, endOfSecond, endOfWeek, endOfYear, epochMicroseconds, epochMilliseconds, epochNanoseconds, epochSeconds, equals, fromFields, fromString, getFields, getISOFields, hoursInDay, inLeapYear, isInstance, monthsInYear, offset, offsetNanoseconds, rangeToLocaleString, rangeToLocaleStringParts, round, roundToMonth, roundToWeek, roundToYear, since, startOfDay, startOfHour, startOfMicrosecond, startOfMillisecond, startOfMinute, startOfMonth, startOfSecond, startOfWeek, startOfYear, subtract, subtractDays, subtractHours, subtractMicroseconds, subtractMilliseconds, subtractMinutes, subtractMonths, subtractNanoseconds, subtractSeconds, subtractWeeks, subtractYears, timeZoneId, toInstant, toLocaleString, toLocaleStringParts, toPlainDate, toPlainDateTime, toPlainMonthDay, toPlainTime, toPlainYearMonth, toString, until, weekOfYear, withCalendar, withDayOfMonth, withDayOfWeek, withDayOfYear, withFields, withPlainDate, withPlainTime, withTimeZone, withWeekOfYear, yearOfWeek };
