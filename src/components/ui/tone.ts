import type { Tone } from '@/types';

/**
 * Tone → class map.
 *
 * Colour intent is expressed once, here. A component receives a `Tone` and
 * never picks raw palette values, which is what makes light/dark themes and a
 * future re-brand a single-file change. All strings are literal so Tailwind's
 * scanner keeps them in the build.
 */

export interface ToneClasses {
  text: string;
  soft: string;
  border: string;
  solid: string;
  dot: string;
}

export const TONE: Record<Tone, ToneClasses> = {
  neutral: {
    text: 'text-ink-2',
    soft: 'bg-sunken',
    border: 'border-line',
    solid: 'bg-ink-3',
    dot: 'bg-ink-3',
  },
  brand: {
    text: 'text-brand-ink',
    soft: 'bg-brand-soft',
    border: 'border-brand-muted',
    solid: 'bg-brand',
    dot: 'bg-brand',
  },
  success: {
    text: 'text-success',
    soft: 'bg-success-soft',
    border: 'border-success/25',
    solid: 'bg-success',
    dot: 'bg-success',
  },
  warning: {
    text: 'text-warning',
    soft: 'bg-warning-soft',
    border: 'border-warning/30',
    solid: 'bg-warning',
    dot: 'bg-warning',
  },
  danger: {
    text: 'text-danger',
    soft: 'bg-danger-soft',
    border: 'border-danger/25',
    solid: 'bg-danger',
    dot: 'bg-danger',
  },
  info: {
    text: 'text-info',
    soft: 'bg-info-soft',
    border: 'border-info/25',
    solid: 'bg-info',
    dot: 'bg-info',
  },
  accent: {
    text: 'text-accent-ink',
    soft: 'bg-accent-soft',
    border: 'border-accent/25',
    solid: 'bg-accent',
    dot: 'bg-accent',
  },
};

/** Soft-tinted icon tile used at the start of list rows. */
export const toneTile = (tone: Tone) => `${TONE[tone].soft} ${TONE[tone].text}`;
