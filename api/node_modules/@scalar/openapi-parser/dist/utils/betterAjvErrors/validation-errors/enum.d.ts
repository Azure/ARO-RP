export default class EnumValidationError extends BaseValidationError {
    constructor(...args: any[]);
    name: string;
    getError(): {
        message: string;
        path: string;
    };
    findBestMatch(): any;
}
import BaseValidationError from './base.js';
//# sourceMappingURL=enum.d.ts.map