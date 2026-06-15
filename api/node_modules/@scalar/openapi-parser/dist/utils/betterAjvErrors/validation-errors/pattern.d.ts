export default class PatternValidationError extends BaseValidationError {
    constructor(...args: any[]);
    name: string;
    getError(): {
        message: string;
        path: string;
    };
}
import BaseValidationError from './base.js';
//# sourceMappingURL=pattern.d.ts.map