/**
 * Persian digit presentation helpers.
 *
 * Data stays numeric everywhere in the app; conversion to Persian digits
 * happens only at the presentation edge so a value can never be corrupted on
 * its way back to an API.
 */

const FA_DIGITS = ['۰', '۱', '۲', '۳', '۴', '۵', '۶', '۷', '۸', '۹'];

/** Convert Latin digits in any string/number to Persian digits. */
export function fa(input: string | number): string {
  return String(input).replace(/\d/g, (d) => FA_DIGITS[Number(d)]);
}

/** Convert Persian/Arabic digits back to Latin — use when parsing input. */
export function toLatinDigits(input: string): string {
  return input
    .replace(/[۰-۹]/g, (d) => String('۰۱۲۳۴۵۶۷۸۹'.indexOf(d)))
    .replace(/[٠-٩]/g, (d) => String('٠١٢٣٤٥٦٧٨٩'.indexOf(d)));
}

/** Thousands separator with Persian digits: 1240 → «۱٬۲۴۰» */
export function faNumber(n: number): string {
  return fa(n.toLocaleString('en-US'));
}

/** «۳ روز» / «۱ روز» */
export function faCount(n: number, unit: string): string {
  return `${fa(n)} ${unit}`;
}

/** Initials for avatars from a Persian full name: «نگار احمدی» → «ن‌ا» */
export function initials(name: string): string {
  const parts = name
    .trim()
    .split(/\s+/)
    .filter((w) => w !== 'خانواده')
    .filter(Boolean);
  if (parts.length === 0) return '؟';
  if (parts.length === 1) return parts[0].slice(0, 2);
  return `${parts[0][0]}\u200C${parts[parts.length - 1][0]}`;
}
