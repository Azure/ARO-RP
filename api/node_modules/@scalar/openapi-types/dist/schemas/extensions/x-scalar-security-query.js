import { z } from "zod";
const XScalarSecurityQuery = z.object({
  "x-scalar-security-query": z.record(z.string(), z.string()).optional()
});
export {
  XScalarSecurityQuery
};
//# sourceMappingURL=x-scalar-security-query.js.map
