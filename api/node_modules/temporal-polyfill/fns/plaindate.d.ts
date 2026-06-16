import * as temporal_spec from 'temporal-spec';
import { PlainDateBranding, DateFields, PlainDateBag, DateBag, IsoDateFields, OverflowOptions, DiffOptions, DateUnitName, CalendarDisplayOptions, NumberSign, LocalesArg, RoundingMathOptions, RoundingModeName } from '../chunks/internal.js';
import { Record as Record$2 } from './duration.js';
import { Record as Record$4 } from './plaindatetime.js';
import { Record as Record$6 } from './plainmonthday.js';
import { Record as Record$1 } from './plaintime.js';
import { Record as Record$5 } from './plainyearmonth.js';
import { Record as Record$3 } from './zoneddatetime.js';



type Record = {
    /**
     * @deprecated Use the isInstance() function instead.
     */
    readonly branding: typeof PlainDateBranding;
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
};
type Fields = DateFields;
type FromFields = PlainDateBag;
type WithFields = DateBag;
type ISOFields = IsoDateFields;
type AssignmentOptions = OverflowOptions;
type ArithmeticOptions = OverflowOptions;
type DifferenceOptions = DiffOptions<DateUnitName>;
type ToStringOptions = CalendarDisplayOptions;
type ToZonedDateTimeOptions = {
    timeZone: string;
    plainTime?: Record$1;
};
declare const create: (isoYear: number, isoMonth: number, isoDay: number, calendar?: string) => Record;
declare function fromFields(fields: FromFields, options?: AssignmentOptions): Record;
declare const fromString: (s: string) => Record;
declare function isInstance(record: any): record is Record;
declare const getFields: (record: Record) => Fields;
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
declare const add: (plainDateRecord: Record, durationRecord: Record$2, options?: ArithmeticOptions) => Record;
declare const subtract: (plainDateRecord: Record, durationRecord: Record$2, options?: ArithmeticOptions) => Record;
declare const until: (record0: Record, record1: Record, options?: DifferenceOptions) => Record$2;
declare const since: (record0: Record, record1: Record, options?: DifferenceOptions) => Record$2;
declare const equals: (record0: Record, record1: Record) => boolean;
declare const compare: (record0: Record, record1: Record) => NumberSign;
declare function toZonedDateTime(record: Record, options: string | ToZonedDateTimeOptions): Record$3;
declare const toPlainDateTime: (plainDateRecord: Record, plainTimeRecord?: Record$1) => Record$4;
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
declare const subtractYears: (slots: Record, units: number, options?: OverflowOptions | undefined) => Record;
declare const subtractMonths: (slots: Record, units: number, options?: OverflowOptions | undefined) => Record;
declare const subtractWeeks: (slots: Record, units: number, options?: OverflowOptions | undefined) => Record;
declare const subtractDays: (slots: Record, units: number, options?: OverflowOptions | undefined) => Record;
declare const roundToYear: (record0: Record, options?: temporal_spec.Temporal.RoundingMode | RoundingMathOptions | undefined) => Record;
declare const roundToMonth: (record0: Record, options?: temporal_spec.Temporal.RoundingMode | RoundingMathOptions | undefined) => Record;
declare const roundToWeek: (record0: Record, options?: temporal_spec.Temporal.RoundingMode | RoundingMathOptions | undefined) => Record;
declare const startOfYear: (record: Record) => Record;
declare const startOfMonth: (record: Record) => Record;
declare const startOfWeek: (record: Record) => Record;
declare const endOfYear: (record: Record) => Record;
declare const endOfMonth: (record: Record) => Record;
declare const endOfWeek: (record: Record) => Record;
declare const diffYears: (record0: Record, record1: Record, options?: RoundingModeName | RoundingMathOptions) => number;
declare const diffMonths: (record0: Record, record1: Record, options?: RoundingModeName | RoundingMathOptions) => number;
declare const diffWeeks: (record0: Record, record1: Record, options?: RoundingModeName | RoundingMathOptions) => number;
declare const diffDays: (record0: Record, record1: Record, options?: RoundingModeName | RoundingMathOptions) => number;

export { type ArithmeticOptions, type AssignmentOptions, type DifferenceOptions, type Fields, type FromFields, type ISOFields, type Record, type ToStringOptions, type ToZonedDateTimeOptions, type WithFields, add, addDays, addMonths, addWeeks, addYears, calendarId, compare, create, dayOfWeek, dayOfYear, daysInMonth, daysInWeek, daysInYear, diffDays, diffMonths, diffWeeks, diffYears, endOfMonth, endOfWeek, endOfYear, equals, fromFields, fromString, getFields, getISOFields, inLeapYear, isInstance, monthsInYear, rangeToLocaleString, rangeToLocaleStringParts, roundToMonth, roundToWeek, roundToYear, since, startOfMonth, startOfWeek, startOfYear, subtract, subtractDays, subtractMonths, subtractWeeks, subtractYears, toLocaleString, toLocaleStringParts, toPlainDateTime, toPlainMonthDay, toPlainYearMonth, toString, toZonedDateTime, until, weekOfYear, withCalendar, withDayOfMonth, withDayOfWeek, withDayOfYear, withFields, withWeekOfYear, yearOfWeek };
