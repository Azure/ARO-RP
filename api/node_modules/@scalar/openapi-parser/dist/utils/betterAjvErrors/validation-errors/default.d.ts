export default class DefaultValidationError extends BaseValidationError {
    constructor(...args: any[]);
    name: string;
    getError(): {
        message: string;
        path: string;
    };
}
import BaseValidationError from './base.js';
//# sourceMappingURL=default.d.ts.map