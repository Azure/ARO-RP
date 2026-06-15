import { z } from "zod";
const XEnumDescriptionsSchema = z.object({
  "x-enumDescriptions": z.record(z.string(), z.string()).catch({})
});
export {
  XEnumDescriptionsSchema
};
//# sourceMappingURL=x-enum-descriptions.js.map
