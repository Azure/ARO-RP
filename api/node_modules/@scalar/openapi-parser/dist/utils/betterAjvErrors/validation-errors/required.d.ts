export default class RequiredValidationError extends BaseValidationError {
    constructor(...args: any[]);
    name: string;
    getError(): {
        message: string;
        path: string;
    };
}
import BaseValidationError from './base.js';
//# sourceMappingURL=required.d.ts.map