import { z } from "zod";
const XScalarCredentialsLocationSchema = z.object({
  "x-scalar-credentials-location": z.enum(["header", "body"]).optional()
});
export {
  XScalarCredentialsLocationSchema
};
//# sourceMappingURL=x-scalar-credentials-location.js.map
