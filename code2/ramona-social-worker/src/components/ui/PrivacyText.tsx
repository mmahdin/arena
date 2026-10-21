import { useEffect, useState } from 'react';
import { cn } from '@/lib/cn';
import { useSettings } from '@/store/settings';

const MASK_LABEL = 'اطلاعات محرمانه';

/**
 * Confidential text.
 *
 * Social workers read this app on buses, in hallways and inside the homes of
 * the families they assess. When "field privacy" is on, identifiers render
 * blurred. Blur (rather than replacing the text with dots) keeps the layout
 * stable, so nothing jumps when the mode is toggled.
 *
 * Set `interactive` to false when the value already sits inside a link or
 * button — nested interactive elements are invalid, and in that case the value
 * is revealed by turning the mode off.
 */
export function PrivacyText({
  value,
  className,
  interactive = true,
  revealMs = 4000,
}: {
  value: string;
  className?: string;
  interactive?: boolean;
  revealMs?: number;
}) {
  const { fieldPrivacy } = useSettings();
  const [revealed, setRevealed] = useState(false);

  useEffect(() => {
    if (!revealed) return;
    const t = window.setTimeout(() => setRevealed(false), revealMs);
    return () => window.clearTimeout(t);
  }, [revealed, revealMs]);

  if (!fieldPrivacy) return <span className={className}>{value}</span>;

  if (!interactive) {
    return (
      <span className={className} aria-label={MASK_LABEL}>
        <span className="masked" aria-hidden>
          {value}
        </span>
      </span>
    );
  }

  return (
    <button
      type="button"
      onClick={(e) => {
        e.preventDefault();
        e.stopPropagation();
        setRevealed(true);
      }}
      aria-label={revealed ? value : `${MASK_LABEL} — برای نمایش لمس کنید`}
      className={cn('press rounded-xs text-start', className)}
    >
      <span className={cn(!revealed && 'masked')} aria-hidden={!revealed}>
        {value}
      </span>
    </button>
  );
}
