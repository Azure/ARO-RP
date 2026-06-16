import { z } from "zod";
const XScalarSecurityBody = z.object({
  "x-scalar-security-body": z.record(z.string(), z.string()).optional()
});
export {
  XScalarSecurityBody
};
//# sourceMappingURL=x-scalar-security-body.js.map
