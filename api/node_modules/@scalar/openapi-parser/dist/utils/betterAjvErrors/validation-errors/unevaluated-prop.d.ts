export default class UnevaluatedPropValidationError extends BaseValidationError {
    constructor(...args: any[]);
    name: string;
    getError(): {
        message: string;
        path: string;
    };
}
import BaseValidationError from './base.js';
//# sourceMappingURL=unevaluated-prop.d.ts.map