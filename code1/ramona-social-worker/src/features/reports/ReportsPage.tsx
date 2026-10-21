import { Navigate } from "react-router-dom";

/**
 * Reports are part of the work inbox. The list lives in the "reports" segment
 * of the Tasks screen so the bottom navigation stays at five destinations.
 */
export function ReportsPage() {
  return <Navigate to="/tasks?view=reports" replace />;
}
