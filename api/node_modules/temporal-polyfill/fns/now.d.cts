import { getCurrentTimeZoneId } from '../chunks/internal.js';
import { Record } from './instant.js';
import { Record as Record$3 } from './plaindate.js';
import { Record as Record$2 } from './plaindatetime.js';
import { Record as Record$4 } from './plaintime.js';
import { Record as Record$1 } from './zoneddatetime.js';





declare const timeZoneId: typeof getCurrentTimeZoneId;
declare function instant(): Record;
declare function zonedDateTime(calendar: string, timeZone?: string): Record$1;
declare function zonedDateTimeISO(timeZone?: string): Record$1;
declare function plainDateTime(calendar: string, timeZone?: string): Record$2;
declare function plainDateTimeISO(timeZone?: string): Record$2;
declare function plainDate(calendar: string, timeZone?: string): Record$3;
declare function plainDateISO(timeZone?: string): Record$3;
declare function plainTimeISO(timeZone?: string): Record$4;

export { instant, plainDate, plainDateISO, plainDateTime, plainDateTimeISO, plainTimeISO, timeZoneId, zonedDateTime, zonedDateTimeISO };
