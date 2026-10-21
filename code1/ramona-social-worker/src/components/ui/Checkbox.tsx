import { Check } from "lucide-react";
import { cn } from "@/lib/cn";
import "./Checkbox.css";

export interface CheckboxProps {
  checked: boolean;
  onChange: () => void;
  label: string;
  className?: string;
  disabled?: boolean;
}

export function Checkbox({ checked, onChange, label, className, disabled }: CheckboxProps) {
  return (
    <button
      type="button"
      role="checkbox"
      aria-checked={checked}
      aria-label={label}
      disabled={disabled}
      className={cn("checkbox", checked && "is-checked", className)}
      onClick={onChange}
    >
      <span className="checkbox__box" aria-hidden="true">
        <Check size={13} strokeWidth={3} />
      </span>
    </button>
  );
}
