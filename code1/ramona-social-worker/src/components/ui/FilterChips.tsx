import { cn } from "@/lib/cn";
import "./FilterChips.css";

export interface FilterChipItem<T extends string> {
  value: T;
  label: string;
  count?: number;
}

export interface FilterChipsProps<T extends string> {
  items: FilterChipItem<T>[];
  value: T;
  onChange: (value: T) => void;
  ariaLabel: string;
  className?: string;
}

export function FilterChips<T extends string>({
  items,
  value,
  onChange,
  ariaLabel,
  className,
}: FilterChipsProps<T>) {
  return (
    <div className={cn("chips", "no-scrollbar", className)} role="group" aria-label={ariaLabel}>
      {items.map((item) => {
        const active = item.value === value;
        return (
          <button
            key={item.value}
            type="button"
            className={cn("chip", active && "is-active")}
            aria-pressed={active}
            onClick={() => onChange(item.value)}
          >
            {item.label}
            {typeof item.count === "number" ? (
              <span className="chip__count num">
                {new Intl.NumberFormat("fa-IR").format(item.count)}
              </span>
            ) : null}
          </button>
        );
      })}
    </div>
  );
}
