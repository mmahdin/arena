import { cn } from "@/lib/cn";
import "./ProgressBar.css";

export interface ProgressBarProps {
  value: number;
  tone?: "brand" | "warning" | "danger" | "info" | "neutral";
  size?: "sm" | "md";
  label?: string;
  className?: string;
}

export function ProgressBar({ value, tone = "brand", size = "sm", label, className }: ProgressBarProps) {
  const clamped = Math.max(0, Math.min(100, value));
  return (
    <div
      className={cn("progress", `progress--${size}`, `progress--${tone}`, className)}
      role="progressbar"
      aria-valuenow={clamped}
      aria-valuemin={0}
      aria-valuemax={100}
      aria-label={label ?? "پیشرفت"}
    >
      <div className="progress__fill" style={{ width: `${clamped}%` }} />
    </div>
  );
}
