import { z } from "zod";
const XScalarStabilityValues = {
  Deprecated: "deprecated",
  Experimental: "experimental",
  Stable: "stable"
};
const XScalarStabilitySchema = z.object({
  "x-scalar-stability": z.enum(Object.values(XScalarStabilityValues)).optional().catch(void 0)
});
export {
  XScalarStabilitySchema,
  XScalarStabilityValues
};
//# sourceMappingURL=x-scalar-stability.js.map
