import { z } from "zod";
const XScalarSdkInstallationSchema = z.object({
  "x-scalar-sdk-installation": z.object({
    lang: z.string(),
    source: z.string().optional().catch(void 0),
    description: z.string().optional().catch(void 0)
  }).array().optional().catch(void 0)
});
export {
  XScalarSdkInstallationSchema
};
//# sourceMappingURL=x-scalar-sdk-installation.js.map
