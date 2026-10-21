import { Check } from "lucide-react";
import type { CaseStage } from "@/types";
import { caseStageShort } from "@/lib/labels";
import { cn } from "@/lib/cn";

/** Stages shown on the rail — "closed" is communicated by the badge instead. */
const stages: CaseStage[] = ["assessment", "introduction", "placement", "monitoring"];

/**
 * Compact four-step progress rail for a case. Position is computed per-step so
 * the connecting line always aligns with the dots, independent of label length.
 */
export function CaseStageRail({ stage }: { stage: CaseStage }) {
  const currentIndex = Math.max(0, stages.indexOf(stage));
  const progress = (currentIndex / (stages.length - 1)) * 100;

  return (
    <div className="stage-rail" aria-label={`مرحله فعلی پرونده: ${caseStageShort[stage]}`}>
      <div className="stage-rail__track" aria-hidden="true">
        <span style={{ width: `${progress}%` }} />
      </div>
      <ol className="stage-rail__steps">
        {stages.map((item, index) => (
          <li
            key={item}
            className={cn(
              "stage-rail__step",
              index < currentIndex && "is-done",
              index === currentIndex && "is-current"
            )}
            aria-current={index === currentIndex ? "step" : undefined}
          >
            <span className="stage-rail__dot" aria-hidden="true">
              {index < currentIndex ? <Check size={10} strokeWidth={3.5} /> : null}
            </span>
            <span className="stage-rail__label">{caseStageShort[item]}</span>
          </li>
        ))}
      </ol>
    </div>
  );
}
