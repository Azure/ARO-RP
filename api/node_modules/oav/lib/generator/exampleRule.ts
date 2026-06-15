import { CacheItem } from "./exampleCache";
export interface ExampleRule {
  exampleNamePostfix: string;
  ruleName: string | undefined;
}

export type RuleValidatorFunc = (context: RuleContext) => boolean | undefined;

interface RuleValidator {
  onParameter?: RuleValidatorFunc;
  onSchema?: RuleValidatorFunc;
  onResponseBody?: RuleValidatorFunc;
  onResponseHeader?: RuleValidatorFunc;
}

const shouldSkip = (cache: CacheItem | undefined, isRequest?: boolean) => {
  return (
    (isRequest && cache?.options?.isReadonly) ||
    (!isRequest && (cache?.options?.isXmsSecret || cache?.options?.isWriteOnly))
  );
};

interface RuleContext {
  schema?: any;
  propertyName?: string | undefined;
  schemaCache?: CacheItem;
  isRequest?: boolean;
  parentSchema?: any;
}
const exampleRuleValidators = new Map<string, RuleValidator>();
exampleRuleValidators.set("MinimumSet", {
  onParameter: (context: RuleContext) => {
    return context?.schema.required;
  },
  onSchema: (context: RuleContext) => {
    if (context?.propertyName) {
      return context?.schemaCache?.required?.includes(context?.propertyName);
    } else if (context.schemaCache && context?.isRequest !== undefined) {
      return !shouldSkip(context.schemaCache, context?.isRequest);
    }
    return true;
  },
});
exampleRuleValidators.set("MaximumSet", {
  onSchema: (context: RuleContext) => {
    if (context.schemaCache && context?.isRequest !== undefined) {
      return !shouldSkip(context.schemaCache, context?.isRequest);
    }
    return true;
  },
});

export function getRuleValidator(rule: ExampleRule | undefined): RuleValidator {
  const validators = exampleRuleValidators;
  if (rule?.ruleName && validators.has(rule.ruleName)) {
    return validators.get(rule.ruleName) || {};
  }
  return {};
}

export type RuleSet = ExampleRule[];
