/**
 * Ramona mark.
 *
 * Two nested arcs sheltering a point — an abstract "embrace" that reads as
 * care without literal imagery. Drawn on a 32px grid with `currentColor` so it
 * works on any surface and in either theme.
 */
export function Logo({ className, size = 28 }: { className?: string; size?: number }) {
  return (
    <svg
      width={size}
      height={size}
      viewBox="0 0 32 32"
      fill="none"
      className={className}
      role="img"
      aria-label="رمونا"
    >
      <path
        d="M4.5 18.5C4.5 11.6 10.1 6 17 6c5.6 0 10.5 3.7 10.5 9.2"
        stroke="currentColor"
        strokeWidth="2.6"
        strokeLinecap="round"
      />
      <path
        d="M9.5 25.5c0-4.4 3.1-7.8 7-7.8 3.4 0 6.2 2.5 6.9 5.8"
        stroke="currentColor"
        strokeWidth="2.6"
        strokeLinecap="round"
        opacity="0.55"
      />
      <circle cx="16.6" cy="23.4" r="2.6" fill="currentColor" />
    </svg>
  );
}

/** Wordmark used on the desktop stage and in the profile header. */
export function Wordmark({ className }: { className?: string }) {
  return (
    <span className={className}>
      <span className="font-extrabold tracking-tight">رمونا</span>
    </span>
  );
}
