import { cn } from '@/lib/cn';
import { fa } from '@/lib/digits';

/**
 * 1–5 rating scale.
 *
 * Professional assessment in this product is mostly ordinal judgement, so the
 * scale is the most-used input after text. It is rendered as five large,
 * finger-sized segments with anchored labels — never as tiny stars, which are
 * imprecise and hard to hit in the field.
 */
export interface ScaleInputProps {
  value: number;
  max?: number;
  onChange: (value: number) => void;
  lowLabel?: string;
  highLabel?: string;
  disabled?: boolean;
}

export function ScaleInput({ value, max = 5, onChange, lowLabel, highLabel, disabled }: ScaleInputProps) {
  return (
    <div>
      <div role="radiogroup" aria-label="مقیاس ارزیابی" className="flex gap-1.5">
        {Array.from({ length: max }, (_, i) => i + 1).map((n) => {
          const active = value >= n;
          return (
            <button
              key={n}
              type="button"
              role="radio"
              aria-checked={value === n}
              aria-label={`سطح ${fa(n)} از ${fa(max)}`}
              disabled={disabled}
              onClick={() => onChange(value === n ? 0 : n)}
              className={cn(
                'press h-9 flex-1 rounded-sm border text-2xs font-bold transition-colors',
                active
                  ? 'border-brand bg-brand text-white'
                  : 'border-line-strong bg-surface text-ink-3 hover:border-brand/50 hover:text-brand-ink',
                disabled && 'opacity-50',
              )}
            >
              {fa(n)}
            </button>
          );
        })}
      </div>
      {(lowLabel || highLabel) && (
        <div className="mt-1.5 flex items-center justify-between text-2xs text-ink-3">
          <span>{lowLabel}</span>
          <span>{highLabel}</span>
        </div>
      )}
    </div>
  );
}

/** Compact read-only scale for summary views. */
export function ScaleDots({ value, max = 5, tone = 'brand' }: { value: number; max?: number; tone?: 'brand' | 'success' | 'warning' | 'danger' }) {
  const fill = { brand: 'bg-brand', success: 'bg-success', warning: 'bg-warning', danger: 'bg-danger' }[tone];
  return (
    <span className="flex items-center gap-1" aria-label={`امتیاز ${fa(value)} از ${fa(max)}`}>
      {Array.from({ length: max }, (_, i) => (
        <span
          key={i}
          className={cn('h-1.5 w-3.5 rounded-pill', i < value ? fill : 'bg-line')}
        />
      ))}
    </span>
  );
}
