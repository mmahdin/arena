import { cn } from '@/lib/cn';

/** Small spinner. Inherits `currentColor`. */
export function Loader({ className }: { className?: string }) {
  return (
    <svg
      className={cn('size-4 animate-spin', className)}
      viewBox="0 0 24 24"
      fill="none"
      aria-hidden="true"
    >
      <circle cx="12" cy="12" r="9" stroke="currentColor" strokeOpacity="0.25" strokeWidth="2.5" />
      <path
        d="M21 12a9 9 0 0 0-9-9"
        stroke="currentColor"
        strokeWidth="2.5"
        strokeLinecap="round"
      />
    </svg>
  );
}

/** Full-area loading placeholder used on first paint of a screen. */
export function ScreenLoader({ label = 'در حال بارگذاری' }: { label?: string }) {
  return (
    <div className="flex flex-col items-center justify-center gap-3 py-24 text-ink-3">
      <Loader className="size-6 text-brand" />
      <p className="text-sm">{label}</p>
    </div>
  );
}
