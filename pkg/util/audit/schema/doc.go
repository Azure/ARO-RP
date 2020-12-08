/*
Package schema defines a number of structs to represent the IFx audit schema as
defined in
https://genevamondocs.azurewebsites.net/collect/instrument/audit/onboarding.html

The audit schema is made up of 3 parts:

- Part A schema is common to all event types. See
  https://microsoft.sharepoint.com/teams/WAG/EngSys/Monitor/Shared%20Documents/Common%20Schema%20Documents/Common%20Schema/Cloud%20Services-%20Part-B%20-%20Audit%20Event%20Schema.docx
- Part B is the audit event schema. See
  https://microsoft.sharepoint.com/teams/WAG/EngSys/Monitor/Shared%20Documents/Common%20Schema%20Documents/Common%20Schema/Cloud%20Services-%20Part-B%20-%20Audit%20Event%20Schema.docx
- Part C for service-specifc details about the audit events that are required
  to make the audit event valuable which do not fit in with the standard schema.
*/
package schema
