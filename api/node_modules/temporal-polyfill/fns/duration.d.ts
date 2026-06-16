import { DurationFields, DurationBranding, NumberSign, DurationBag, RelativeToOptions, DurationRoundingOptions, DurationTotalOptions, TimeDisplayOptions, UnitName, LocalesArg } from '../chunks/internal.js';
import { Record as Record$3 } from './plaindate.js';
import { Record as Record$2 } from './plaindatetime.js';
import { Record as Record$1 } from './zoneddatetime.js';







type Record = Readonly<DurationFields> & {
    /**
     * @deprecated Use the isInstance() function instead.
     */
    readonly branding: typeof DurationBranding;
    readonly sign: NumberSign;
};
type FromFields = DurationBag;
type WithFields = DurationBag;
type RelativeToRecord = Record$1 | Record$2 | Record$3;
type ArithmeticOptions = RelativeToOptions<RelativeToRecord>;
type RoundOptions = DurationRoundingOptions<RelativeToRecord>;
type TotalOptions = DurationTotalOptions<RelativeToRecord>;
type CompareOptions = RelativeToOptions<RelativeToRecord>;
type ToStringOptions = TimeDisplayOptions;
declare const create: (years?: number, months?: number, weeks?: number, days?: number, hours?: number, minutes?: number, seconds?: number, milliseconds?: number, microseconds?: number, nanoseconds?: number) => Record;
declare const fromFields: (fields: FromFields) => Record;
declare const fromString: (s: string) => Record;
declare function isInstance(record: any): record is Record;
declare const blank: (record: Record) => boolean;
declare const withFields: (record: Record, fields: WithFields) => Record;
declare const negated: (record: Record) => Record;
declare const abs: (record: Record) => Record;
declare const add: (record0: Record, record1: Record, options?: ArithmeticOptions) => Record;
declare const subtract: (record0: Record, record1: Record, options?: ArithmeticOptions) => Record;
declare const round: (record: Record, options?: RoundOptions) => Record;
declare const total: (record: Record, options?: UnitName | TotalOptions) => number;
declare const compare: (record0: Record, record1: Record, options?: CompareOptions) => NumberSign;
declare function toLocaleString(record: Record, locales?: LocalesArg, options?: Intl.DateTimeFormatOptions): string;
declare const toString: (record: Record, options?: ToStringOptions) => string;

export { type ArithmeticOptions, type CompareOptions, type FromFields, type Record, type RelativeToRecord, type RoundOptions, type ToStringOptions, type TotalOptions, type WithFields, abs, add, blank, compare, create, fromFields, fromString, isInstance, negated, round, subtract, toLocaleString, toString, total, withFields };
