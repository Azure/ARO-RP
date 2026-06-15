import { objectEntries } from "../object/object-entries.js";
const SECRET_FIELD_MAPPINGS = {
  clientSecret: "x-scalar-secret-client-secret",
  password: "x-scalar-secret-password",
  token: "x-scalar-secret-token",
  username: "x-scalar-secret-username",
  value: "x-scalar-secret-token",
  "x-scalar-client-id": "x-scalar-secret-client-id",
  "x-scalar-redirect-uri": "x-scalar-secret-redirect-uri"
};
const extractConfigSecrets = (input) => objectEntries(SECRET_FIELD_MAPPINGS).reduce((result, [field, secretField]) => {
  const value = input[field];
  if (value && typeof value === "string") {
    result[secretField] = value;
  }
  return result;
}, {});
const SECRETS_SET = new Set(
  objectEntries(SECRET_FIELD_MAPPINGS).flatMap(([oldSecret, newSecret]) => [oldSecret, newSecret])
);
const removeSecretFields = (input) => objectEntries(input).reduce((result, [key, value]) => {
  if (!SECRETS_SET.has(key)) {
    result[key] = value;
  }
  return result;
}, {});
export {
  extractConfigSecrets,
  removeSecretFields
};
//# sourceMappingURL=extract-config-secrets.js.map
