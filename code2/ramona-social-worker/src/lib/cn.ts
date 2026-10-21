import { clsx, type ClassValue } from 'clsx';

/** Join conditional class names. The only styling helper you should need. */
export function cn(...inputs: ClassValue[]) {
  return clsx(inputs);
}
