import { z } from "zod";
const XUsePkceValues = ["SHA-256", "plain", "no"];
const XusePkceSchema = z.object({
  /**
   * Use x-usePkce to enable Proof Key for Code Exchange (PKCE) for the Oauth2 authorization code flow.
   */
  "x-usePkce": z.enum(XUsePkceValues).optional().default("no")
});
export {
  XUsePkceValues,
  XusePkceSchema
};
//# sourceMappingURL=x-use-pkce.js.map
