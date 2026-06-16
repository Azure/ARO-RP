import { Temporal as Un, DateTimeFormat as Xn, toTemporalInstant as tj } from "./chunks/classApi.js";

import { createPropDescriptors as n } from "./chunks/internal.js";

Object.defineProperties(globalThis, n({
  Temporal: Un
})), Object.defineProperties(Intl, n({
  DateTimeFormat: Xn
})), Object.defineProperties(Date.prototype, n({
  toTemporalInstant: tj
}));
