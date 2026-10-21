import { cn } from "@/lib/cn";
import "./Switch.css";

export interface SwitchProps {
  checked: boolean;
  onChange: (checked: boolean) => void;
  label: string;
  description?: string;
  id?: string;
}

export function Switch({ checked, onChange, label, description, id }: SwitchProps) {
  return (
    <div className="switch-row">
      <div className="switch-row__text">
        <span className="switch-row__label" id={id ? `${id}-label` : undefined}>
          {label}
        </span>
        {description ? <span className="switch-row__desc">{description}</span> : null}
      </div>
      <button
        type="button"
        role="switch"
        aria-checked={checked}
        aria-labelledby={id ? `${id}-label` : undefined}
        aria-label={id ? label : undefined}
        className={cn("switch", checked && "is-on")}
        onClick={() => onChange(!checked)}
      >
        <span className="switch__thumb" aria-hidden="true" />
      </button>
    </div>
  );
}
