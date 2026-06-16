import { PlainMonthDayBranding, MonthDayFields, PlainMonthDayBag, MonthDayBag, IsoDateFields, EraYearOrYear, OverflowOptions, CalendarDisplayOptions, LocalesArg } from '../chunks/internal.js';
import { Record as Record$1 } from './plaindate.js';









type Record = {
    /**
     * @deprecated Use the isInstance() function instead.
     */
    readonly branding: typeof PlainMonthDayBranding;
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
type Fields = MonthDayFields;
type FromFields = PlainMonthDayBag;
type WithFields = MonthDayBag;
type ISOFields = IsoDateFields;
type ToPlainDateFields = EraYearOrYear;
type AssignmentOptions = OverflowOptions;
type ToStringOptions = CalendarDisplayOptions;
declare const create: (isoMonth: number, isoDay: number, calendar?: string, referenceIsoYear?: number) => Record;
declare function fromFields(fields: FromFields, options?: AssignmentOptions): Record;
declare const fromString: (s: string) => Record;
declare function isInstance(record: any): record is Record;
declare const getFields: (record: Record) => Fields;
declare const getISOFields: (record: Record) => ISOFields;
declare const calendarId: (record: Record) => string;
declare function withFields(record: Record, fields: WithFields, options?: AssignmentOptions): Record;
declare const equals: (record0: Record, record1: Record) => boolean;
declare function toPlainDate(record: Record, fields: ToPlainDateFields): Record$1;
declare function toLocaleString(record: Record, locales?: LocalesArg, options?: Intl.DateTimeFormatOptions): string;
declare function toLocaleStringParts(record: Record, locales?: LocalesArg, options?: Intl.DateTimeFormatOptions): Intl.DateTimeFormatPart[];
declare function rangeToLocaleString(record0: Record, record1: Record, locales?: LocalesArg, options?: Intl.DateTimeFormatOptions): string;
declare function rangeToLocaleStringParts(record0: Record, record1: Record, locales?: LocalesArg, options?: Intl.DateTimeFormatOptions): ReturnType<Intl.DateTimeFormat['formatRangeToParts']>;
declare const toString: (record: Record, options?: ToStringOptions) => string;

export { type AssignmentOptions, type Fields, type FromFields, type ISOFields, type Record, type ToPlainDateFields, type ToStringOptions, type WithFields, calendarId, create, equals, fromFields, fromString, getFields, getISOFields, isInstance, rangeToLocaleString, rangeToLocaleStringParts, toLocaleString, toLocaleStringParts, toPlainDate, toString, withFields };
