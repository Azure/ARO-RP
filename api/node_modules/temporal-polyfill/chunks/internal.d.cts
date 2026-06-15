import { Temporal } from 'temporal-spec';

type SubsecDigits = 0 | 1 | 2 | 3 | 4 | 5 | 6 | 7 | 8 | 9;

type NumberSign = -1 | 0 | 1;

type BigNano = [days: number, timeNano: number];

type LocalesArg = string | string[];

interface DurationDateFields {
    days: number;
    weeks: number;
    months: number;
    years: number;
}
interface DurationTimeFields {
    nanoseconds: number;
    microseconds: number;
    milliseconds: number;
    seconds: number;
    minutes: number;
    hours: number;
}
type DurationFields = DurationDateFields & DurationTimeFields;
type DurationYearMonthFieldName = 'years' | 'months';
type DurationDateFieldName = DurationYearMonthFieldName | 'weeks' | 'days';
type DurationTimeFieldName = 'hours' | 'minutes' | 'seconds' | 'milliseconds' | 'microseconds' | 'nanoseconds';
type DurationDayTimeFieldName = 'day' | DurationTimeFieldName;
type DurationFieldName = DurationDateFieldName | DurationTimeFieldName;

type StrictYearMonthUnitName = 'year' | 'month';
type StrictDateUnitName = StrictYearMonthUnitName | 'week' | 'day';
type StrictTimeUnitName = 'hour' | 'minute' | 'second' | 'millisecond' | 'microsecond' | 'nanosecond';
type StrictDayTimeUnitName = 'day' | StrictTimeUnitName;
type StrictUnitName = StrictDateUnitName | StrictTimeUnitName;
type YearMonthUnitName = StrictYearMonthUnitName | DurationYearMonthFieldName;
type DateUnitName = StrictDateUnitName | DurationDateFieldName;
type TimeUnitName = StrictTimeUnitName | DurationTimeFieldName;
type DayTimeUnitName = StrictDayTimeUnitName | DurationDayTimeFieldName;
type UnitName = StrictUnitName | DurationFieldName;

type ZonedFieldOptions = OverflowOptions & EpochDisambigOptions & OffsetDisambigOptions;
type RoundingMathOptions = RoundingIncOptions & RoundingModeOptions;
type DiffOptions<UN extends UnitName> = LargestUnitOptions<UN> & SmallestUnitOptions<UN> & RoundingMathOptions;
type RoundingOptions<UN extends DayTimeUnitName> = Required<SmallestUnitOptions<UN>> & RoundingMathOptions;
type DurationRoundingOptions<RA> = Required<SmallestUnitOptions<UnitName>> & LargestUnitOptions<UnitName> & RoundingMathOptions & RelativeToOptions<RA>;
type TimeDisplayOptions = SmallestUnitOptions<TimeUnitName> & RoundingModeOptions & SubsecDigitsOptions;
type ZonedDateTimeDisplayOptions = CalendarDisplayOptions & TimeZoneDisplayOptions & OffsetDisplayOptions & TimeDisplayOptions;
type RelativeToOptions<RA> = {
    relativeTo?: RA;
};
type DurationTotalOptions<RA> = TotalUnitOptions & RelativeToOptions<RA>;
type DateTimeDisplayOptions = CalendarDisplayOptions & TimeDisplayOptions;
interface SmallestUnitOptions<UN extends UnitName> {
    smallestUnit?: UN;
}
interface LargestUnitOptions<UN extends UnitName> {
    largestUnit?: UN;
}
interface TotalUnitOptions {
    unit: UnitName;
}
type InstantDisplayOptions = {
    timeZone?: string;
} & TimeDisplayOptions;
interface OverflowOptions {
    overflow?: Temporal.AssignmentOptions['overflow'];
}
interface EpochDisambigOptions {
    disambiguation?: Temporal.ToInstantOptions['disambiguation'];
}
interface OffsetDisambigOptions {
    offset?: Temporal.OffsetDisambiguationOptions['offset'];
}
interface CalendarDisplayOptions {
    calendarName?: Temporal.ShowCalendarOption['calendarName'];
}
interface TimeZoneDisplayOptions {
    timeZoneName?: Temporal.ZonedDateTimeToStringOptions['timeZoneName'];
}
interface OffsetDisplayOptions {
    offset?: Temporal.ZonedDateTimeToStringOptions['offset'];
}
type RoundingModeName = Temporal.DifferenceOptions<any>['roundingMode'];
interface RoundingModeOptions {
    roundingMode?: RoundingModeName;
}
interface RoundingIncOptions {
    roundingIncrement?: Temporal.DifferenceOptions<any>['roundingIncrement'];
}
interface SubsecDigitsOptions {
    fractionalSecondDigits?: SubsecDigits;
}

interface IsoDateFields {
    isoDay: number;
    isoMonth: number;
    isoYear: number;
}
interface IsoTimeFields {
    isoNanosecond: number;
    isoMicrosecond: number;
    isoMillisecond: number;
    isoSecond: number;
    isoMinute: number;
    isoHour: number;
}
type IsoDateTimeFields = IsoDateFields & IsoTimeFields;

declare const PlainYearMonthBranding: "PlainYearMonth";
declare const PlainMonthDayBranding: "PlainMonthDay";
declare const PlainDateBranding: "PlainDate";
declare const PlainDateTimeBranding: "PlainDateTime";
declare const PlainTimeBranding: "PlainTime";
declare const ZonedDateTimeBranding: "ZonedDateTime";
declare const InstantBranding: "Instant";
declare const DurationBranding: "Duration";
type EpochSlots = {
    epochNanoseconds: BigNano;
};
type EpochAndZoneSlots = EpochSlots & {
    timeZone: string;
};
type ZonedEpochSlots = EpochAndZoneSlots & {
    calendar: string;
};
type DateSlots = IsoDateFields & {
    calendar: string;
};
type ZonedDateTimeSlots = ZonedEpochSlots & {
    branding: typeof ZonedDateTimeBranding;
};

interface EraYearFields {
    era: string;
    eraYear: number;
}
type YearFields = Partial<EraYearFields> & {
    year: number;
};
interface MonthFields {
    monthCode: string;
    month: number;
}
interface DayFields {
    day: number;
}
type YearMonthFields = YearFields & MonthFields;
type DateFields = YearMonthFields & DayFields;
type MonthDayFields = MonthFields & DayFields;
interface TimeFields {
    hour: number;
    microsecond: number;
    millisecond: number;
    minute: number;
    nanosecond: number;
    second: number;
}
type DateTimeFields = DateFields & TimeFields;
type YearMonthBag = Partial<YearMonthFields>;
type DateBag = Partial<DateFields>;
type MonthDayBag = Partial<MonthDayFields>;
type DurationBag = Partial<DurationFields>;
type TimeBag = Partial<TimeFields>;
type DateTimeBag = DateBag & TimeBag;
type EraYearOrYear = EraYearFields | {
    year: number;
};

type ZonedIsoFields = IsoDateTimeFields & {
    calendar: string;
    timeZone: string;
    offset: string;
};
type ZonedDateTimeFields = DateTimeFields & {
    offset: string;
};

type Marker = IsoDateFields | IsoDateTimeFields | ZonedEpochSlots;

type PlainDateBag = DateBag & {
    calendar?: string;
};
type PlainDateTimeBag = DateBag & TimeBag & {
    calendar?: string;
};
type ZonedDateTimeBag = PlainDateTimeBag & {
    timeZone: string;
    offset?: string;
};
type PlainTimeBag = TimeBag;
type PlainYearMonthBag = YearMonthBag & {
    calendar?: string;
};
type PlainMonthDayBag = MonthDayBag & {
    calendar?: string;
};

declare function getCurrentTimeZoneId(): string;

export { type BigNano, type CalendarDisplayOptions, type DateBag, type DateFields, type DateSlots, type DateTimeBag, type DateTimeDisplayOptions, type DateTimeFields, type DateUnitName, type DayTimeUnitName, type DiffOptions, type DurationBag, DurationBranding, type DurationFields, type DurationRoundingOptions, type DurationTotalOptions, type EpochDisambigOptions, type EraYearOrYear, InstantBranding, type InstantDisplayOptions, type IsoDateFields, type IsoDateTimeFields, type IsoTimeFields, type LocalesArg, type Marker, type MonthDayBag, type MonthDayFields, type NumberSign, type OverflowOptions, type PlainDateBag, PlainDateBranding, type PlainDateTimeBag, PlainDateTimeBranding, type PlainMonthDayBag, PlainMonthDayBranding, type PlainTimeBag, PlainTimeBranding, type PlainYearMonthBag, PlainYearMonthBranding, type RelativeToOptions, type RoundingMathOptions, type RoundingModeName, type RoundingOptions, type TimeBag, type TimeDisplayOptions, type TimeFields, type TimeUnitName, type UnitName, type YearMonthBag, type YearMonthFields, type YearMonthUnitName, type ZonedDateTimeBag, ZonedDateTimeBranding, type ZonedDateTimeDisplayOptions, type ZonedDateTimeFields, type ZonedDateTimeSlots, type ZonedFieldOptions, type ZonedIsoFields, getCurrentTimeZoneId };
