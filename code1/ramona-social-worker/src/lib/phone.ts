const FA = "۰۱۲۳۴۵۶۷۸۹";

/** Persian digits → ASCII digits, stripping spaces/dashes. */
export function normalizeDigits(value: string): string {
  return value
    .replace(/[۰-۹]/g, (d) => String(FA.indexOf(d)))
    .replace(/[^\d+]/g, "");
}

export function telHref(phone: string): string {
  return `tel:${normalizeDigits(phone)}`;
}
