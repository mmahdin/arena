/**
 * Mock data entry point.
 *
 * Only `src/services/api.ts` should import from this module. Everything else
 * consumes data through the app store, so swapping the mock layer for real
 * API calls touches a single boundary.
 */
export { cases, findCase } from "./cases";
export { visits, findVisit } from "./visits";
export { tasks } from "./tasks";
export { reports, findReport, reportsForCase } from "./reports";
export { notifications } from "./notifications";
export { profile } from "./profile";
export { NOW, at } from "./reference";
