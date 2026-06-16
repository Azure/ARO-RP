import { InstantBranding, BigNano, DiffOptions, TimeUnitName, RoundingOptions, InstantDisplayOptions, UnitName, NumberSign, LocalesArg } from '../chunks/internal.js';
import { Record as Record$1 } from './duration.js';
import { Record as Record$2 } from './zoneddatetime.js';








type Record = {
    /**
     * @deprecated Use the isInstance() function instead.
     */
    readonly branding: typeof InstantBranding;
    /**
     * @deprecated Use the epochNanoseconds() function instead.
     */
    readonly epochNanoseconds: BigNano;
};
type DifferenceOptions = DiffOptions<TimeUnitName>;
type RoundOptions = RoundingOptions<TimeUnitName>;
type ToStringOptions = InstantDisplayOptions;
type ToZonedDateTimeOptions = {
    timeZone: string;
    calendar: string;
};
declare const create: (epochNanoseconds: bigint) => Record;
declare const fromEpochSeconds: (epochSeconds: number) => Record;
declare const fromEpochMilliseconds: (epochMilliseconds: number) => Record;
declare const fromEpochMicroseconds: (epochMicroseconds: bigint) => Record;
declare const fromEpochNanoseconds: (epochNanoseconds: bigint) => Record;
declare const fromString: (s: string) => Record;
declare function isInstance(record: any): record is Record;
declare const epochSeconds: (record: Record) => number;
declare const epochMilliseconds: (record: Record) => number;
declare const epochMicroseconds: (record: Record) => bigint;
declare const epochNanoseconds: (record: Record) => bigint;
declare const add: (instantRecord: Record, durationRecord: Record$1) => Record;
declare const subtract: (instantRecord: Record, durationRecord: Record$1) => Record;
declare const until: (record0: Record, record1: Record, options?: DifferenceOptions) => Record$1;
declare const since: (record0: Record, record1: Record, options?: DifferenceOptions) => Record$1;
declare const round: (record: Record, options?: UnitName | RoundOptions) => Record;
declare const equals: (record0: Record, record1: Record) => boolean;
declare const compare: (record0: Record, record1: Record) => NumberSign;
declare function toZonedDateTime(record: Record, options: ToZonedDateTimeOptions): Record$2;
declare function toZonedDateTimeISO(record: Record, timeZone: string): Record$2;
declare function toLocaleString(record: Record, locales?: LocalesArg, options?: Intl.DateTimeFormatOptions): string;
declare function toLocaleStringParts(record: Record, locales?: LocalesArg, options?: Intl.DateTimeFormatOptions): Intl.DateTimeFormatPart[];
declare function rangeToLocaleString(record0: Record, record1: Record, locales?: LocalesArg, options?: Intl.DateTimeFormatOptions): string;
declare function rangeToLocaleStringParts(record0: Record, record1: Record, locales?: LocalesArg, options?: Intl.DateTimeFormatOptions): ReturnType<Intl.DateTimeFormat['formatRangeToParts']>;
declare const toString: (record: Record, options?: ToStringOptions) => string;

export { type DifferenceOptions, type Record, type RoundOptions, type ToStringOptions, type ToZonedDateTimeOptions, add, compare, create, epochMicroseconds, epochMilliseconds, epochNanoseconds, epochSeconds, equals, fromEpochMicroseconds, fromEpochMilliseconds, fromEpochNanoseconds, fromEpochSeconds, fromString, isInstance, rangeToLocaleString, rangeToLocaleStringParts, round, since, subtract, toLocaleString, toLocaleStringParts, toString, toZonedDateTime, toZonedDateTimeISO, until };
