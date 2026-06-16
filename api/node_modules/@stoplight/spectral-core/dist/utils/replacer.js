"use strict";
Object.defineProperty(exports, "__esModule", { value: true });
exports.Replacer = void 0;
const expr_eval_fork_1 = require("expr-eval-fork");
class Replacer {
    constructor(count) {
        this.regex = new RegExp(`#?${'{'.repeat(count)}([^}\n]+)${'}'.repeat(count)}`, 'g');
        this.functions = {};
    }
    addFunction(name, filter) {
        this.functions[name] = filter;
    }
    print(input, values) {
        const parser = new expr_eval_fork_1.Parser();
        const functions = parser.functions;
        functions.toUpperCase = (value) => {
            return String(value).toUpperCase();
        };
        functions.concat = (...args) => {
            return args.map(String).join('');
        };
        Object.entries(this.functions).forEach(([name, fn]) => {
            functions[name] = fn.bind(values);
        });
        const context = values;
        return input.replace(this.regex, (_substr, identifier, index) => {
            if (index < 0 || index >= input.length) {
                return '';
            }
            const shouldEvaluate = input[index] === '#';
            if (shouldEvaluate) {
                let expression = identifier.trim();
                if (/constructor|process|require|global|mainModule|fs|child_process/.test(expression)) {
                    console.error('Disallowed expression detected:', expression);
                    return '';
                }
                try {
                    expression = expression
                        .replace(/(\S+)\.toUpperCase\(\)/g, 'toUpperCase($1)')
                        .replace(/\s*\+\s*/g, ',')
                        .replace(/^(.+)$/, 'concat($1)');
                    return String(parser.evaluate(expression, context));
                }
                catch (error) {
                    console.error('Expression evaluation error:', error);
                    return '';
                }
            }
            if (!(identifier in values)) {
                return '';
            }
            return String(values[identifier]);
        });
    }
}
exports.Replacer = Replacer;
//# sourceMappingURL=replacer.js.map