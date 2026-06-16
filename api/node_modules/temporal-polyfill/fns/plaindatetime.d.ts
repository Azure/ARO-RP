import { PlainDateTimeBranding, DateTimeFields, PlainDateTimeBag, DateTimeBag, IsoDateTimeFields, OverflowOptions, DiffOptions, UnitName, RoundingOptions, DayTimeUnitName, EpochDisambigOptions, DateTimeDisplayOptions, NumberSign, LocalesArg, RoundingMathOptions, RoundingModeName, Marker } from '../chunks/internal.js';
import * as temporal_spec from 'temporal-spec';
import { Record as Record$3 } from './duration.js';
import { Record as Record$1 } from './plaindate.js';
import { Record as Record$6 } from './plainmonthday.js';
import { Record as Record$2 } from './plaintime.js';
import { Record as Record$5 } from './plainyearmonth.js';
import { Record as Record$4 } from './zoneddatetime.js';



type Record = {
    /**
     * @deprecated Use the isInstance() function instead.
     */
    branding: typeof PlainDateTimeBranding;
    /**
     * @deprecated Use the calendarId() function instead.
     */
    readonly calendar: string;
    /**
     * @deprecated Use the getISOFields() function instead.
     */
    readonly isoYear: number;
    /**
     * @deprecated Use the getISOFields() function instead.
     */
    readonly isoMonth: number;
    /**
     * @deprecated Use the getISOFields() function instead.
     */
    readonly isoDay: number;
    /**
     * @deprecated Use the getISOFields() function instead.
     */
    readonly isoHour: number;
    /**
     * @deprecated Use the getISOFields() function instead.
     */
    readonly isoMinute: number;
    /**
     * @deprecated Use the getISOFields() function instead.
     */
    readonly isoSecond: number;
    /**
     * @deprecated Use the getISOFields() function instead.
     */
    readonly isoMillisecond: number;
    /**
     * @deprecated Use the getISOFields() function instead.
     */
    readonly isoMicrosecond: number;
    /**
     * @deprecated Use the getISOFields() function instead.
     */
    readonly isoNanosecond: number;
};
type Fields = DateTimeFields;
type FromFields = PlainDateTimeBag;
type WithFields = DateTimeBag;
type ISOFields = IsoDateTimeFields;
type AssignmentOptions = OverflowOptions;
type ArithmeticOptions = OverflowOptions;
type DifferenceOptions = DiffOptions<UnitName>;
type RoundOptions = RoundingOptions<DayTimeUnitName>;
type ToZonedDateTimeOptions = EpochDisambigOptions;
type ToStringOptions = DateTimeDisplayOptions;
declare const create: (isoYear: number, isoMonth: number, isoDay: number, isoHour?: number, isoMinute?: number, isoSecond?: number, isoMillisecond?: number, isoMicrosecond?: number, isoNanosecond?: number, calendar?: string) => Record;
declare function fromFields(fields: FromFields, options?: AssignmentOptions): Record;
declare const fromString: (s: string) => Record;
declare function isInstance(record: any): record is Record;
declare const getFields: (key: Record) => DateTimeFields;
declare const getISOFields: (record: Record) => ISOFields;
declare const calendarId: (record: Record) => string;
declare const dayOfWeek: (record: Record) => number;
declare const daysInWeek: (record: Record) => number;
declare const weekOfYear: (record: Record) => number | undefined;
declare const yearOfWeek: (record: Record) => number | undefined;
declare const dayOfYear: (record: Record) => number;
declare const daysInMonth: (record: Record) => number;
declare const daysInYear: (record: Record) => number;
declare const monthsInYear: (record: Record) => number;
declare const inLeapYear: (record: Record) => boolean;
declare function withFields(record: Record, fields: WithFields, options?: AssignmentOptions): Record;
declare function withCalendar(record: Record, calendar: string): Record;
declare const withPlainDate: (plainDateTimeRecord: Record, plainDateRecord: Record$1) => Record;
declare const withPlainTime: (plainDateTimeRecord: Record, plainTimeRecord?: Record$2) => Record;
declare const add: (plainDateTimeRecord: Record, durationRecord: Record$3, options?: ArithmeticOptions) => Record;
declare const subtract: (plainDateTimeRecord: Record, durationRecord: Record$3, options?: ArithmeticOptions) => Record;
declare const until: (record0: Record, record1: Record, options?: DifferenceOptions) => Record$3;
declare const since: (record0: Record, record1: Record, options?: DifferenceOptions) => Record$3;
declare const round: (record: Record, options: DayTimeUnitName | RoundOptions) => Record;
declare const equals: (record0: Record, record1: Record) => boolean;
declare const compare: (record0: Record, record1: Record) => NumberSign;
declare const toZonedDateTime: (record: Record, timeZone: string, options?: ToZonedDateTimeOptions) => Record$4;
declare const toPlainDate: (record: Record) => Record$1;
declare const toPlainTime: (record: Record) => Record$2;
declare function toPlainYearMonth(record: Record): Record$5;
declare function toPlainMonthDay(record: Record): Record$6;
declare function toLocaleString(record: Record, locales?: LocalesArg, options?: Intl.DateTimeFormatOptions): string;
declare function toLocaleStringParts(record: Record, locales?: LocalesArg, options?: Intl.DateTimeFormatOptions): Intl.DateTimeFormatPart[];
declare function rangeToLocaleString(record0: Record, record1: Record, locales?: LocalesArg, options?: Intl.DateTimeFormatOptions): string;
declare function rangeToLocaleStringParts(record0: Record, record1: Record, locales?: LocalesArg, options?: Intl.DateTimeFormatOptions): ReturnType<Intl.DateTimeFormat['formatRangeToParts']>;
declare const toString: (record: Record, options?: ToStringOptions) => string;
declare function withDayOfYear(record: Record, dayOfYear: number, options?: OverflowOptions): Record;
declare function withDayOfMonth(record: Record, dayOfMonth: number, options?: OverflowOptions): Record;
declare function withDayOfWeek(record: Record, dayOfWeek: number, options?: OverflowOptions): Record;
declare function withWeekOfYear(record: Record, weekOfYear: number, options?: OverflowOptions): Record;
declare function addYears(record: Record, years: number, options?: OverflowOptions): Record;
declare function addMonths(record: Record, months: number, options?: OverflowOptions): Record;
declare function addWeeks(record: Record, weeks: number): Record;
declare function addDays(record: Record, days: number): Record;
declare const addHours: (record: Record, units: number) => Record;
declare const addMinutes: (record: Record, units: number) => Record;
declare const addSeconds: (record: Record, units: number) => Record;
declare const addMilliseconds: (record: Record, units: number) => Record;
declare const addMicroseconds: (record: Record, units: number) => Record;
declare const addNanoseconds: (record: Record, units: number) => Record;
declare const subtractYears: (slots: Record, units: number, options?: OverflowOptions | undefined) => Record;
declare const subtractMonths: (slots: Record, units: number, options?: OverflowOptions | undefined) => Record;
declare const subtractWeeks: (slots: Record, units: number, options?: OverflowOptions | undefined) => Record;
declare const subtractDays: (slots: Record, units: number, options?: OverflowOptions | undefined) => Record;
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
declare const startOfDay: (record: Record) => Record;
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
declare const diffYears: (record0: Record, record1: Record, options?: RoundingModeName | RoundingMathOptions) => number;
declare const diffMonths: (record0: Record, record1: Record, options?: RoundingModeName | RoundingMathOptions) => number;
declare const diffWeeks: (record0: Record, record1: Record, options?: RoundingModeName | RoundingMathOptions) => number;
declare const diffDays: (record0: Record, record1: Record, options?: RoundingModeName | RoundingMathOptions) => number;
declare const diffHours: (record0: Marker, record1: Marker, options?: temporal_spec.Temporal.RoundingMode | RoundingMathOptions | undefined) => number;
declare const diffMinutes: (record0: Marker, record1: Marker, options?: temporal_spec.Temporal.RoundingMode | RoundingMathOptions | undefined) => number;
declare const diffSeconds: (record0: Marker, record1: Marker, options?: temporal_spec.Temporal.RoundingMode | RoundingMathOptions | undefined) => number;
declare const diffMilliseconds: (record0: Marker, record1: Marker, options?: temporal_spec.Temporal.RoundingMode | RoundingMathOptions | undefined) => number;
declare const diffMicroseconds: (record0: Marker, record1: Marker, options?: temporal_spec.Temporal.RoundingMode | RoundingMathOptions | undefined) => number;
declare const diffNanoseconds: (record0: Marker, record1: Marker, options?: temporal_spec.Temporal.RoundingMode | RoundingMathOptions | undefined) => number;

export { type ArithmeticOptions, type AssignmentOptions, type DifferenceOptions, type Fields, type FromFields, type ISOFields, type Record, type RoundOptions, type ToStringOptions, type ToZonedDateTimeOptions, type WithFields, add, addDays, addHours, addMicroseconds, addMilliseconds, addMinutes, addMonths, addNanoseconds, addSeconds, addWeeks, addYears, calendarId, compare, create, dayOfWeek, dayOfYear, daysInMonth, daysInWeek, daysInYear, diffDays, diffHours, diffMicroseconds, diffMilliseconds, diffMinutes, diffMonths, diffNanoseconds, diffSeconds, diffWeeks, diffYears, endOfDay, endOfHour, endOfMicrosecond, endOfMillisecond, endOfMinute, endOfMonth, endOfSecond, endOfWeek, endOfYear, equals, fromFields, fromString, getFields, getISOFields, inLeapYear, isInstance, monthsInYear, rangeToLocaleString, rangeToLocaleStringParts, round, roundToMonth, roundToWeek, roundToYear, since, startOfDay, startOfHour, startOfMicrosecond, startOfMillisecond, startOfMinute, startOfMonth, startOfSecond, startOfWeek, startOfYear, subtract, subtractDays, subtractHours, subtractMicroseconds, subtractMilliseconds, subtractMinutes, subtractMonths, subtractNanoseconds, subtractSeconds, subtractWeeks, subtractYears, toLocaleString, toLocaleStringParts, toPlainDate, toPlainMonthDay, toPlainTime, toPlainYearMonth, toString, toZonedDateTime, until, weekOfYear, withCalendar, withDayOfMonth, withDayOfWeek, withDayOfYear, withFields, withPlainDate, withPlainTime, withWeekOfYear, yearOfWeek };
