import { PlainYearMonthBranding, YearMonthFields, PlainYearMonthBag, YearMonthBag, IsoDateFields, OverflowOptions, DiffOptions, YearMonthUnitName, CalendarDisplayOptions, NumberSign, LocalesArg } from '../chunks/internal.js';
import { Record as Record$1 } from './duration.js';
import { Record as Record$2 } from './plaindate.js';








type Record = {
    /**
     * @deprecated Use the isInstance() function instead.
     */
    readonly branding: typeof PlainYearMonthBranding;
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
type Fields = YearMonthFields;
type FromFields = PlainYearMonthBag;
type WithFields = YearMonthBag;
type ISOFields = IsoDateFields;
type ToPlainDateFields = {
    day: number;
};
type AssignmentOptions = OverflowOptions;
type ArithmeticOptions = OverflowOptions;
type DifferenceOptions = DiffOptions<YearMonthUnitName>;
type ToStringOptions = CalendarDisplayOptions;
declare const create: (isoYear: number, isoMonth: number, calendar?: string, referenceIsoDay?: number) => Record;
declare function fromFields(fields: FromFields, options?: AssignmentOptions): Record;
declare const fromString: (s: string) => Record;
declare function isInstance(record: any): record is Record;
declare const getFields: (record: Record) => Fields;
declare const getISOFields: (record: Record) => ISOFields;
declare const calendarId: (record: Record) => string;
declare const daysInMonth: (record: Record) => number;
declare const daysInYear: (record: Record) => number;
declare const monthsInYear: (record: Record) => number;
declare const inLeapYear: (record: Record) => boolean;
declare function withFields(record: Record, fields: WithFields, options?: AssignmentOptions): Record;
declare const add: (plainYearMonthFields: Record, durationRecord: Record$1, options?: ArithmeticOptions) => Record;
declare const subtract: (plainYearMonthFields: Record, durationRecord: Record$1, options?: ArithmeticOptions) => Record;
declare const until: (record0: Record, record1: Record, options?: DifferenceOptions) => Record$1;
declare const since: (record0: Record, record1: Record, options?: DifferenceOptions) => Record$1;
declare const equals: (record0: Record, record1: Record) => boolean;
declare const compare: (record0: Record, record1: Record) => NumberSign;
declare function toPlainDate(record: Record, fields: ToPlainDateFields): Record$2;
declare function toLocaleString(record: Record, locales?: LocalesArg, options?: Intl.DateTimeFormatOptions): string;
declare function toLocaleStringParts(record: Record, locales?: LocalesArg, options?: Intl.DateTimeFormatOptions): Intl.DateTimeFormatPart[];
declare function rangeToLocaleString(record0: Record, record1: Record, locales?: LocalesArg, options?: Intl.DateTimeFormatOptions): string;
declare function rangeToLocaleStringParts(record0: Record, record1: Record, locales?: LocalesArg, options?: Intl.DateTimeFormatOptions): ReturnType<Intl.DateTimeFormat['formatRangeToParts']>;
declare const toString: (record: Record, options?: ToStringOptions) => string;

export { type ArithmeticOptions, type AssignmentOptions, type DifferenceOptions, type Fields, type FromFields, type ISOFields, type Record, type ToPlainDateFields, type ToStringOptions, type WithFields, add, calendarId, compare, create, daysInMonth, daysInYear, equals, fromFields, fromString, getFields, getISOFields, inLeapYear, isInstance, monthsInYear, rangeToLocaleString, rangeToLocaleStringParts, since, subtract, toLocaleString, toLocaleStringParts, toPlainDate, toString, until, withFields };
