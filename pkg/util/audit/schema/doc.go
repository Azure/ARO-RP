/*
Package schema defines a number of structs to represent the IFx audit schema

The audit schema is made up of 3 parts:

- Part A schema is common to all event types
- Part B is the audit event schema
- Part C for service-specifc details about the audit events that are required
  to make the audit event valuable which do not fit in with the standard schema.
*/
package schema
