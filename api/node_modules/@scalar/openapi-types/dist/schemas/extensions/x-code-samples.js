import { z } from "zod";
const XCodeSampleSchema = z.object({
  lang: z.string().optional().catch(void 0),
  label: z.string().optional().catch(void 0),
  source: z.string()
});
const XCodeSamplesSchema = z.object({
  "x-codeSamples": XCodeSampleSchema.array().optional().catch(void 0),
  "x-code-samples": XCodeSampleSchema.array().optional().catch(void 0),
  "x-custom-examples": XCodeSampleSchema.array().optional().catch(void 0)
});
export {
  XCodeSampleSchema,
  XCodeSamplesSchema
};
//# sourceMappingURL=x-code-samples.js.map
