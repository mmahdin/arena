import { cn } from "@/lib/cn";
import "./SegmentedControl.css";

export interface SegmentItem<T extends string> {
  value: T;
  label: string;
  count?: number;
}

export interface SegmentedControlProps<T extends string> {
  items: SegmentItem<T>[];
  value: T;
  onChange: (value: T) => void;
  ariaLabel: string;
  className?: string;
}

/**
 * In-place filter switcher (not navigation). Uses pressed-button semantics
 * rather than tabs so screen readers announce the active filter predictably.
 */
export function SegmentedControl<T extends string>({
  items,
  value,
  onChange,
  ariaLabel,
  className,
}: SegmentedControlProps<T>) {
  const activeIndex = Math.max(0, items.findIndex((i) => i.value === value));

  return (
    <div
      className={cn("segmented", className)}
      role="group"
      aria-label={ariaLabel}
      style={{ ["--seg-count" as string]: items.length, ["--seg-index" as string]: activeIndex }}
    >
      <span className="segmented__indicator" aria-hidden="true" />
      {items.map((item) => {
        const active = item.value === value;
        return (
          <button
            key={item.value}
            type="button"
            className={cn("segmented__item", active && "is-active")}
            aria-pressed={active}
            onClick={() => onChange(item.value)}
          >
            <span>{item.label}</span>
            {typeof item.count === "number" ? (
              <span className="segmented__count num">{new Intl.NumberFormat("fa-IR").format(item.count)}</span>
            ) : null}
          </button>
        );
      })}
    </div>
  );
}
