import { Operation, SwaggerExample } from "../swagger/swaggerTypes";

//#region Common

type TransformRaw<T, Additional = {}, OptionalKey extends keyof T = never> = {
  [P in OptionalKey]?: T[P];
} & {
  [P in Exclude<keyof T, OptionalKey | keyof Additional>]-?: Exclude<T[P], undefined>;
} & Additional;

export type VarType = Variable["type"];

export type VarValue = boolean | number | string | VarValue[] | { [key: string]: VarValue };

export type Variable =
  | BoolVariable
  | IntVariable
  | StringVariable
  | SecureStringVariable
  | ArrayVariable
  | ObjectVariable
  | SecureObjectVariable;

export interface StringVariable {
  type: "string";
  value?: string;
  prefix?: string;
}

export interface SecureStringVariable {
  type: "secureString";
  value?: string;
  prefix?: string;
}

export interface BoolVariable {
  type: "bool";
  value?: boolean;
}

export interface IntVariable {
  type: "int";
  value?: number;
}

export interface ObjectVariable {
  type: "object";
  value?: { [key: string]: VarValue };
  patches?: JsonPatchOp[];
}

export interface SecureObjectVariable {
  type: "secureObject";
  value?: { [key: string]: VarValue };
  patches?: JsonPatchOp[];
}

export interface ArrayVariable {
  type: "array";
  value?: VarValue[];
  patches?: JsonPatchOp[];
}

export interface RawVariableScope {
  variables?: {
    [variableName: string]: string | Variable;
  };
}

export interface VariableScope {
  variables: {
    [variableName: string]: Variable;
  };
  requiredVariables: string[];
  secretVariables: string[];
}

export interface OutputVariables {
  [variableName: string]: {
    type?: VarType;
    fromRequest: string;
    fromResponse: string;
  };
}

export interface RawNoneAuthentication {
  type: "None";
}

export type NoneAuthentication = TransformRaw<RawNoneAuthentication>;

export interface RawAADTokenAuthentication {
  type: "AADToken";
  scope?: string;
}

export type AADTokenAuthentication = TransformRaw<RawAADTokenAuthentication>;

export interface RawAzureKeyAuthentication {
  type: "AzureKey";
  key: string;
  name?: string;
  in?: "header" | "query";
}

export type AzureKeyAuthentication = TransformRaw<RawAzureKeyAuthentication>;

export type RawAuthentication =
  | RawNoneAuthentication
  | RawAADTokenAuthentication
  | RawAzureKeyAuthentication;

export type Authentication = NoneAuthentication | AADTokenAuthentication | AzureKeyAuthentication;

export interface ReadmeTag {
  name: string;
  filePath: string;
  tag?: string;
}

//#endregion

//#region Step Base

type RawStepBase = RawVariableScope & {
  step?: string;
  description?: string;
  outputVariables?: OutputVariables;
};

type StepBase = VariableScope & {
  isPrepareStep?: boolean;
  isCleanUpStep?: boolean;
};

export type Step = StepRestCall | StepArmTemplate | StepRoleAssignment;
export type RawStep =
  | RawStepOperation
  | RawStepExample
  | RawStepArmTemplate
  | RawStepArmScript
  | RawStepRoleAssignment;

//#endregion

//#region Step RestCall

export type RawStepExample = RawStepBase & {
  operationId?: string;
  exampleFile: string;
  requestUpdate?: JsonPatchOp[];
  responseUpdate?: JsonPatchOp[];
  authentication?: RawAuthentication;
};

export type RawStepOperation = RawStepBase & {
  operationId: string;
  readmeTag?: string;
  parameters?: { [parameterName: string]: VarValue };
  responses?: StepResponseAssertion;
  authentication?: RawAuthentication;
};

export type StepRestCallExample = StepBase & {};

export type StepRestCall = StepBase & {
  type: "restCall";
  step: string;
  description?: string;
  operationId: string;
  operation?: Operation;
  exampleFile?: string;
  parameters: SwaggerExample["parameters"];
  responses: SwaggerExample["responses"];
  responseAssertion?: StepResponseAssertion;
  outputVariables?: OutputVariables;
  externalReference?: boolean;
  isManagementPlane?: boolean;
  authentication?: Authentication;
  _resolvedParameters?: SwaggerExample["parameters"];
};

export type StepResponseAssertion = {
  [statusCode: string]:
    | {
        headers?: { [headerName: string]: string };
        body?: any;
      }
    | JsonPatchOpTest[];
};

//#endregion

//#region ARM Steps
export type RawStepArmScript = RawStepBase & {
  armDeploymentScript: string;
  arguments?: string;
  environmentVariables?: Array<{
    name: string;
    value: string;
  }>;
};

export type ArmTemplateVariableType =
  | "string"
  | "securestring"
  | "int"
  | "bool"
  | "object"
  | "secureObject"
  | "array";

export type RawStepArmTemplate = RawStepBase & {
  armTemplate: string;
};

export type StepArmTemplate = TransformRaw<
  RawStepArmTemplate,
  StepBase & {
    type: "armTemplateDeployment";
    armTemplatePayload: ArmTemplate;
  },
  "description"
>;

export interface ArmResource {
  name: string;
  apiVersion: string;
  type: string;
  location?: string;
  properties?: object;
}

export type ArmDeploymentScriptResource = ArmResource & {
  type: "Microsoft.Resources/deploymentScripts";
  kind: "AzurePowerShell" | "AzureCLI";
  identity?: {
    type: "UserAssigned";
    userAssignedIdentities: {
      [name: string]: {};
    };
  };
  properties: {
    arguments?: string;
    azPowerShellVersion?: string;
    azCliVersion?: string;
    scriptContent: string;
    forceUpdateTag?: string;
    timeout?: string;
    cleanupPreference?: string;
    retentionInterval?: string;
    environmentVariables?: Array<{
      name: string;
      value?: string;
      secureValue?: string;
    }>;
  };
};

export interface ArmTemplate {
  $schema?: string;
  contentVersion?: string;
  parameters?: {
    [name: string]: {
      type: ArmTemplateVariableType;
      defaultValue?: any;
    };
  };
  outputs?: {
    [name: string]: {
      condition?: string;
      type: ArmTemplateVariableType;
    };
  };
  resources?: ArmResource[];
}

export type RawStepRoleAssignment = RawStepBase & {
  roleAssignment: RoleAssignment;
};

export type StepRoleAssignment = TransformRaw<
  RawStepRoleAssignment,
  StepBase & {
    type: "armRoleAssignment";
    authentication?: Authentication;
  },
  "description"
>;

export interface RoleAssignment {
  scope: string;
  roleDefinitionId?: string;
  roleName?: string;
  principalId: string;
  principalType?: "User" | "Group" | "ServicePrincipal" | "ForeignGroup" | "Device";
}

//#endregion

//#region JsonPatchOp

export interface JsonPatchOpAdd {
  add: string;
  value: any;
}

export interface JsonPatchOpRemove {
  remove: string;
  oldValue?: any;
}

export interface JsonPatchOpReplace {
  replace: string;
  value: any;
  oldValue?: any;
}

export interface JsonPatchOpCopy {
  copy: string;
  from: string;
}

export interface JsonPatchOpMove {
  move: string;
  from: string;
}

export interface JsonPatchOpTest {
  test: string;
  value?: any;
  expression?: string;
}

export type JsonPatchOp =
  | JsonPatchOpAdd
  | JsonPatchOpRemove
  | JsonPatchOpReplace
  | JsonPatchOpCopy
  | JsonPatchOpMove
  | JsonPatchOpTest;

//#endregion

//#region Scenario

export type RawScenario = RawVariableScope & {
  scenario?: string;
  description?: string;
  steps: RawStep[];
  authentication?: RawAuthentication;
};

export type Scenario = TransformRaw<
  RawScenario,
  {
    steps: Step[];
    authentication?: Authentication;
    _scenarioDef: ScenarioDefinition;
  } & VariableScope
>;

//#endregion

//#region ScenarioDefinitionFile

export type ArmScope = "ResourceGroup" | "Subscription" | "Tenant" | "None";

export type RawScenarioDefinition = RawVariableScope & {
  scope?: ArmScope | string;
  prepareSteps?: RawStep[];
  scenarios: RawScenario[];
  cleanUpSteps?: RawStep[];
  authentication?: RawAuthentication;
};

export type ScenarioDefinition = TransformRaw<
  RawScenarioDefinition,
  VariableScope & {
    name: string;
    prepareSteps: Step[];
    scenarios: Scenario[];
    cleanUpSteps: Step[];
    authentication?: Authentication;
    _filePath: string;
    _swaggerFilePaths: string[];
  }
>;
//#endregion

//#region Runner specific types
export interface NewmanReport {
  executions: NewmanExecution[];
  timings: any;
  variables: { [variableName: string]: Variable };
}

export interface SimpleItemMetadata {
  type: "simple";
  operationId: string;
  exampleName?: string;
  itemName: string;
  step: string;
}

export interface LroItemMetadata {
  type: "LRO";
  poller_item_name: string;
  operationId: string;
  exampleName?: string;
  itemName: string;
  step: string;
}

export interface DelayItemMetadata {
  type: "delay";
  lro_item_name: string;
}

export interface PollerItemMetadata {
  type: "poller";
  lro_item_name: string;
}

export interface FinalGetItemMetadata {
  type: "finalGet";
  lro_item_name: string;
  step: string;
}

export type ItemMetadata =
  | SimpleItemMetadata
  | LroItemMetadata
  | DelayItemMetadata
  | PollerItemMetadata
  | FinalGetItemMetadata;

export interface NewmanExecution {
  id: string;
  request: NewmanRequest;
  response: NewmanResponse;
  annotation?: ItemMetadata;
  assertions: NewmanAssertion[];
}

export interface NewmanAssertion {
  name: string;
  test: string;
  message: string;
  stack: string;
}
export interface NewmanRequest {
  url: string;
  method: string;
  headers: { [key: string]: any };
  body: string;
}

export interface NewmanResponse {
  statusCode: number;
  headers: { [key: string]: any };
  body: string;
  responseTime: number;
}

export interface TestResources {
  ["test-resources"]: Array<{ [key: string]: string }>;
}

//#endregion
