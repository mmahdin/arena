import type { ReactNode } from "react";
import { cn } from "@/lib/cn";
import "./Avatar.css";

type Tone = "brand" | "info" | "accent" | "warning" | "neutral";

const tones: Tone[] = ["brand", "info", "accent", "warning", "neutral"];

/** Stable pseudo-random tone so the same person keeps the same avatar colour. */
function toneFor(seed: string): Tone {
  let hash = 0;
  for (let i = 0; i < seed.length; i++) hash = (hash * 31 + seed.charCodeAt(i)) % 997;
  return tones[hash % tones.length];
}

export function initials(name: string): string {
  const clean = name.replace(/^خانواده\s+/, "").trim();
  const parts = clean.split(/\s+/).filter(Boolean);
  if (parts.length === 0) return "؟";
  if (parts.length === 1) return parts[0].slice(0, 2);
  return `${parts[0][0]}${parts[1][0]}`;
}

export interface AvatarProps {
  name: string;
  size?: "xs" | "sm" | "md" | "lg" | "xl";
  shape?: "circle" | "squircle";
  tone?: Tone;
  icon?: ReactNode;
  className?: string;
  /** Decorative when the name is already rendered next to it. */
  decorative?: boolean;
}

export function Avatar({
  name,
  size = "md",
  shape = "circle",
  tone,
  icon,
  className,
  decorative = true,
}: AvatarProps) {
  const resolvedTone = tone ?? toneFor(name);
  return (
    <span
      className={cn("avatar", `avatar--${size}`, `avatar--${shape}`, `avatar--${resolvedTone}`, className)}
      {...(decorative ? { "aria-hidden": true } : { role: "img", "aria-label": name })}
    >
      {icon ?? <span className="avatar__text">{initials(name)}</span>}
    </span>
  );
}
