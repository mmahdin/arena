import { cn } from '@/lib/cn';
import { CASE_STAGE_ORDER, CASE_STAGE_SHORT } from '@/lib/labels';
import { caseStageIndex } from '@/lib/domain';
import type { CaseStage } from '@/types';

/**
 * Process stepper.
 *
 * Social workers rarely need the whole workflow — they need to know *where
 * this case is* and what comes next. A four-segment rail answers that faster
 * than any status label, and it stays honest about who owns each stage.
 */
export function Stepper({ stage, className }: { stage: CaseStage; className?: string }) {
  const current = caseStageIndex(stage);
  return (
    <ol className={cn('space-y-2', className)} aria-label="وضعیت پرونده در فرایند">
      <li className="flex gap-1.5">
        {CASE_STAGE_ORDER.map((s, i) => (
          <span
            key={s}
            aria-current={i === current ? 'step' : undefined}
            className={cn(
              'h-1.5 flex-1 rounded-pill transition-colors duration-slow',
              i < current && 'bg-brand-muted',
              i === current && 'bg-brand',
              i > current && 'bg-line',
            )}
          />
        ))}
      </li>
      <li className="flex gap-1.5">
        {CASE_STAGE_ORDER.map((s, i) => (
          <span
            key={s}
            className={cn(
              'flex-1 text-2xs leading-tight',
              i === current ? 'font-bold text-brand-ink' : i < current ? 'text-ink-3' : 'text-ink-3/70',
            )}
          >
            {CASE_STAGE_SHORT[s]}
          </span>
        ))}
      </li>
    </ol>
  );
}
