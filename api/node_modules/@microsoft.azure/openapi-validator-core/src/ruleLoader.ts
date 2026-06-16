import { IRuleSet } from "./types"
export interface IRuleLoader {
  getRuleSet: (rulesetPath?: string) => IRuleSet | Promise<IRuleSet>
}

export class BuiltInRuleLoader {
  getRuleSet(): IRuleSet {
    return {
      documentationUrl: "",
      rules: {
      }
    }
  }
}