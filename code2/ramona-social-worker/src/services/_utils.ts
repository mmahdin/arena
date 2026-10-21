/**
 * Service utilities.
 *
 * `latency()` simulates a network round-trip so loading states are real and
 * visible during development. Set `SIMULATED_LATENCY` to 0 for instant data.
 */
export const SIMULATED_LATENCY = 220;

export function latency<T>(value: T, ms = SIMULATED_LATENCY): Promise<T> {
  if (ms <= 0) return Promise.resolve(value);
  return new Promise((resolve) => setTimeout(() => resolve(value), ms + Math.random() * 120));
}

export function nowIso(): string {
  const d = new Date();
  const pad = (n: number) => String(n).padStart(2, '0');
  return `${d.getFullYear()}-${pad(d.getMonth() + 1)}-${pad(d.getDate())}T${pad(d.getHours())}:${pad(
    d.getMinutes(),
  )}`;
}

export function uid(prefix = 'id'): string {
  return `${prefix}-${Math.random().toString(36).slice(2, 8)}`;
}
