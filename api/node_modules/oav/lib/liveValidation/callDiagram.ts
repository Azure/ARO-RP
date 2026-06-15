// /* eslint-disable no-lone-blocks */
// import { SwaggerLoader } from "../swagger/swaggerLoader";
// import { JsonLoader } from "../swagger/jsonLoader";
// import { SuppressionLoader } from "../swagger/suppressionLoader";
// import { SwaggerSpec } from "../swagger/swaggerTypes";
// import { applySpecTransformers } from "../transform/transformer";
// import { pathRegexTransformer } from "../transform/pathRegexTransformer";
// import { referenceFieldsTransformer } from "../transform/referenceFieldsTransformer";
// import { resolveNestedDefinitionTransformer } from "../transform/resolveNestedDefinitionTransformer";
// import { xmsPathsTransformer } from "../transform/xmsPathsTransformer";
// import { discriminatorTransformer } from "../transform/discriminatorTransformer";
// import { allOfTransformer } from "../transform/allOfTransformer";
// import { noAdditionalPropertiesTransformer } from "../transform/noAdditionalPropertiesTransformer";
// import { nullableTransformer } from "../transform/nullableTransformer";
// import { pureObjectTransformer } from "../transform/pureObjectTransformer";
// import {
//   AjvSchemaValidator,
//   ajvErrorToSchemaValidateIssue,
// } from "../swaggerValidator/ajvSchemaValidator";
// import { OperationSearcher } from "./operationSearcher";
// import { LiveValidatorLoader } from "./liveValidatorLoader";
// import { LiveValidator, RequestResponsePair } from "./liveValidator";
// import {
//   LiveRequest,
//   validateSwaggerLiveRequest,
//   schemaValidateIssueToLiveValidationIssue,
//   LiveResponse,
//   validateSwaggerLiveResponse,
// } from "./operationValidator";

// const opts = {};
// const ctx = {} as any;

// export const callDiagram = async () => {
//   const liveValidator = new LiveValidator();
//   const operationSearcher = new OperationSearcher(liveValidator.logging);
//   const jsonLoader = JsonLoader.create(opts);
//   const swaggerLoader = SwaggerLoader.create(opts);
//   const suppressionLoader = SuppressionLoader.create(opts);
//   const liveValidatorLoader = LiveValidatorLoader.create(opts);
//   const schemaValidator = new AjvSchemaValidator(jsonLoader);

//   // Initialize
//   await liveValidator.initialize();
//   {
//     const specPaths = await liveValidator.getSwaggerPaths();
//     const allSpecs = [];

//     for (const specPath of specPaths) {
//       const spec = await liveValidator.getSwaggerInitializer(liveValidatorLoader, specPath);
//       {
//         const spec = await liveValidatorLoader.load(specPath);
//         {
//           const spec = await swaggerLoader.load(specPath);
//           {
//             const spec = ((await jsonLoader.load(specPath)) as unknown) as SwaggerSpec;
//             await suppressionLoader.load(spec);
//           }

//           applySpecTransformers(spec, ctx);
//           {
//             const transformers = [
//               xmsPathsTransformer,
//               resolveNestedDefinitionTransformer,
//               referenceFieldsTransformer,
//               pathRegexTransformer,
//             ];
//             for (const transformer of transformers) {
//               transformer.transform(spec, ctx);
//             }
//           }
//         }

//         operationSearcher.addSpecToCache(spec);
//       }
//       allSpecs.push(spec!);
//     }

//     liveValidatorLoader.transformLoadedSpecs();
//     {
//       const transformers = [
//         discriminatorTransformer,
//         allOfTransformer,
//         noAdditionalPropertiesTransformer,
//         nullableTransformer,
//         pureObjectTransformer,
//       ];
//       for (const transformer of transformers) {
//         transformer.transform(ctx);
//       }
//     }

//     // eslint-disable-next-line @typescript-eslint/no-floating-promises
//     liveValidator.loadAllSpecValidatorInBackground(allSpecs);
//   }

//   // Validate request response
//   await liveValidator.validateLiveRequestResponse({} as RequestResponsePair);
//   {
//     await liveValidator.validateLiveRequest({} as LiveRequest);
//     {
//       const { info } = liveValidator.getOperationInfo({} as LiveRequest, "");
//       {
//         const operation = operationSearcher.search();
//       }

//       const requestIssues = await validateSwaggerLiveRequest({} as LiveRequest);
//       {
//         const validate = await liveValidatorLoader.getRequestValidator(
//           info.operationMatch!.operation
//         );
//         {
//           const schema = {
//             properties: {
//               headers: {},
//               query: {},
//               body: {},
//             },
//           };

//           const ajvValidator = schemaValidator.compile(schema);
//         }

//         const validateCtx = { isResponse: false };
//         const jsonSchemaErrors = validate(validateCtx, {});
//         {
//           const ajvErrors = ajvValidator(validateCtx, {});
//           jsonSchemaErrors = ajvErrors.map(ajvErrorToSchemaValidateIssue);
//         }
//         const liveValidationIssues = schemaValidateIssueToLiveValidationIssue(jsonSchemaErrors);
//       }
//     }

//     await liveValidator.validateLiveResponse({} as LiveResponse, {} as any);
//     {
//       const { info } = liveValidator.getOperationInfo({} as LiveRequest, "");

//       const responseIssues = await validateSwaggerLiveResponse({} as LiveResponse);
//       {
//         const validate = await liveValidatorLoader.getResponseValidator(
//           info.operationMatch!.operation.responses[200]
//         );
//         {
//           const schema = {
//             properties: {
//               headers: {},
//               body: {},
//             },
//           };

//           const ajvValidator = schemaValidator.compile(schema);
//         }

//         const validateCtx = { isResponse: true };
//         const jsonSchemaErrors = validate(validateCtx, {});
//         {
//           const ajvErrors = ajvValidator(validateCtx);
//           jsonSchemaErrors = ajvErrors.map(ajvErrorToSchemaValidateIssue);
//         }
//         const liveValidationIssues = schemaValidateIssueToLiveValidationIssue(jsonSchemaErrors);
//       }
//     }
//   }
// };
