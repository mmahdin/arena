/** Accept 11-digit Iranian phone-shaped local identifiers; no ownership proof or OTP. */
export function normalizeTestPhone(value) {
  const trimmed = value.trim();
  const normalized = [...trimmed].map((digit) => {
    const code = digit.codePointAt(0);
    if (code >= 0x06f0 && code <= 0x06f9) return String(code - 0x06f0);
    if (code >= 0x0660 && code <= 0x0669) return String(code - 0x0660);
    return digit;
  }).join('');
  return /^09[0-9]{9}$/.test(normalized) ? normalized : null;
}
