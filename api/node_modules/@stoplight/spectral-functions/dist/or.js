"use strict";
Object.defineProperty(exports, "__esModule", { value: true });
const spectral_core_1 = require("@stoplight/spectral-core");
const optionSchemas_1 = require("./optionSchemas");
exports.default = (0, spectral_core_1.createRulesetFunction)({
    input: {
        type: 'object',
    },
    options: optionSchemas_1.optionSchemas.or,
}, function or(targetVal, { properties }) {
    if (properties.length < 2)
        return;
    const results = [];
    const intersection = Object.keys(targetVal).filter(value => -1 !== properties.indexOf(value));
    if (intersection.length == 0) {
        if (properties.length > 4) {
            const shortprops = properties.slice(0, 3);
            const count = String(properties.length - 3) + ' other properties must be defined';
            results.push({
                message: 'At least one of "' + shortprops.join('" or "') + '" or ' + count,
            });
        }
        else {
            results.push({
                message: 'At least one of "' + properties.join('" or "') + '" must be defined',
            });
        }
    }
    return results;
});
//# sourceMappingURL=or.js.map