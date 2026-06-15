export default class AdditionalPropValidationError extends BaseValidationError {
    constructor(...args: any[]);
    name: string;
    getError(): {
        message: string;
        path: string;
    };
}
import BaseValidationError from './base.js';
//# sourceMappingURL=additional-prop.d.ts.map