import { PlainTimeBranding, TimeFields, PlainTimeBag, TimeBag, IsoTimeFields, OverflowOptions, DiffOptions, TimeUnitName, RoundingOptions, TimeDisplayOptions, NumberSign, LocalesArg } from '../chunks/internal.js';
import { Record as Record$2 } from './duration.js';
import { Record as Record$1 } from './plaindate.js';
import { Record as Record$4 } from './plaindatetime.js';
import { Record as Record$3 } from './zoneddatetime.js';






type Record = {
    /**
     * @deprecated Use the isInstance() function instead.
     */
    branding: typeof PlainTimeBranding;
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
type Fields = TimeFields;
type FromFields = PlainTimeBag;
type WithFields = TimeBag;
type ISOFields = IsoTimeFields;
type AssignmentOptions = OverflowOptions;
type DifferenceOptions = DiffOptions<TimeUnitName>;
type RoundOptions = RoundingOptions<TimeUnitName>;
type ToStringOptions = TimeDisplayOptions;
type ToZonedDateTimeOptions = {
    timeZone: string;
    plainDate: Record$1;
};
declare const create: (isoHour?: number, isoMinute?: number, isoSecond?: number, isoMillisecond?: number, isoMicrosecond?: number, isoNanosecond?: number) => Record;
declare const fromFields: (fields: FromFields, options?: AssignmentOptions) => Record;
declare const fromString: (s: string) => Record;
declare function isInstance(record: any): record is Record;
declare const getFields: (record: Record) => Fields;
declare const getISOFields: (record: Record) => ISOFields;
declare function withFields(record: Record, fields: WithFields, options?: AssignmentOptions): Record;
declare const add: (plainTimeRecord: Record, durationRecord: Record$2) => Record;
declare const subtract: (plainTimeRecord: Record, durationRecord: Record$2) => Record;
declare const until: (record0: Record, record1: Record, options?: DifferenceOptions) => Record$2;
declare const since: (record0: Record, record1: Record, options?: DifferenceOptions) => Record$2;
declare const round: (record: Record, options: TimeUnitName | RoundOptions) => Record;
declare const equals: (record0: Record, record1: Record) => boolean;
declare const compare: (record0: Record, record1: Record) => NumberSign;
declare const toZonedDateTime: (plainTimeRecord: Record, options: ToZonedDateTimeOptions) => Record$3;
declare const toPlainDateTime: (plainTimeRecord: Record, plainDateRecord: Record$1) => Record$4;
declare function toLocaleString(record: Record, locales?: LocalesArg, options?: Intl.DateTimeFormatOptions): string;
declare function toLocaleStringParts(record: Record, locales?: LocalesArg, options?: Intl.DateTimeFormatOptions): Intl.DateTimeFormatPart[];
declare function rangeToLocaleString(record0: Record, record1: Record, locales?: LocalesArg, options?: Intl.DateTimeFormatOptions): string;
declare function rangeToLocaleStringParts(record0: Record, record1: Record, locales?: LocalesArg, options?: Intl.DateTimeFormatOptions): ReturnType<Intl.DateTimeFormat['formatRangeToParts']>;
declare const toString: (record: Record, options?: ToStringOptions) => string;

export { type AssignmentOptions, type DifferenceOptions, type Fields, type FromFields, type ISOFields, type Record, type RoundOptions, type ToStringOptions, type ToZonedDateTimeOptions, type WithFields, add, compare, create, equals, fromFields, fromString, getFields, getISOFields, isInstance, rangeToLocaleString, rangeToLocaleStringParts, round, since, subtract, toLocaleString, toLocaleStringParts, toPlainDateTime, toString, toZonedDateTime, until, withFields };
