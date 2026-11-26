import jalaali from "jalaali-js"
const toEn = (value) =>
  String(value).replace(/[۰-۹٠-٩]/g, (digit) =>
    String(digit.charCodeAt(0) - (digit >= "\u06F0" ? 1776 : 1632)),
  )
const toFa = (value) =>
  String(value).replace(
    /\d/g,
    (digit) =>
      "\u06F0\u06F1\u06F2\u06F3\u06F4\u06F5\u06F6\u06F7\u06F8\u06F9"[
        Number(digit)
      ],
  )
const normalizeDateDigits = toEn
function jalaliDateToGregorianDateOnly(value) {
  const match = /^(\d{4})\/(\d{2})\/(\d{2})$/.exec(
    toEn(value?.trim() ?? "").replace(/-/g, "/"),
  )
  if (!match) return ""
  const year = Number(match[1]),
    month = Number(match[2]),
    day = Number(match[3])
  if (year < 1 || year > 3177) return ""
  try {
    if (!jalaali.isValidJalaaliDate(year, month, day)) return ""
    const { gy, gm, gd } = jalaali.toGregorian(year, month, day)
    return `${String(gy).padStart(4, "0")}-${String(gm).padStart(2, "0")}-${String(gd).padStart(2, "0")}`
  } catch {
    return ""
  }
}
function todayJalaliDate() {
  return instantToJalaliDate(/* @__PURE__ */ new Date())
}
function instantToJalaliDate(value) {
  const parts = new Intl.DateTimeFormat("en-US-u-ca-persian", {
    timeZone: "Asia/Tehran",
    year: "numeric",
    month: "2-digit",
    day: "2-digit",
  }).formatToParts(typeof value === "string" ? new Date(value) : value)
  const part = (type) => parts.find((p) => p.type === type)?.value ?? ""
  return `${part("year").padStart(4, "0")}/${part("month")}/${part("day")}`
}
function normalizeJalaliDate(value) {
  let normalized = toEn(value.trim()).replace(/-/g, "/")
  if (/^\d{8}$/.test(normalized))
    normalized = `${normalized.slice(0, 4)}/${normalized.slice(4, 6)}/${normalized.slice(6)}`
  return normalized
}
function jalaliDateError(value, required = false, minDate, maxDate) {
  if (!value)
    return required
      ? "\u062A\u0627\u0631\u06CC\u062E \u0631\u0627 \u0648\u0627\u0631\u062F \u06A9\u0646\u06CC\u062F."
      : void 0
  const normalized = normalizeJalaliDate(value)
  if (!/^\d{4}\/\d{2}\/\d{2}$/.test(normalized))
    return "\u0633\u0627\u0644\u060C \u0645\u0627\u0647 \u0648 \u0631\u0648\u0632 \u0631\u0627 \u06A9\u0627\u0645\u0644 \u0648\u0627\u0631\u062F \u06A9\u0646\u06CC\u062F."
  if (!jalaliDateToGregorianDateOnly(normalized))
    return "\u062A\u0627\u0631\u06CC\u062E \u0634\u0645\u0633\u06CC \u0645\u0639\u062A\u0628\u0631 \u0646\u06CC\u0633\u062A."
  if (
    (minDate && normalized < normalizeJalaliDate(minDate)) ||
    (maxDate && normalized > normalizeJalaliDate(maxDate))
  )
    return "\u062A\u0627\u0631\u06CC\u062E \u062E\u0627\u0631\u062C \u0627\u0632 \u0628\u0627\u0632\u0647\u0654 \u0645\u062C\u0627\u0632 \u0627\u0633\u062A."
  return void 0
}
function gregorianDateOnlyToJalali(value) {
  if (!/^\d{4}-\d{2}-\d{2}$/.test(value || "")) return ""
  try {
    const [gy, gm, gd] = value.split("-").map(Number)
    const { jy, jm, jd } = jalaali.toJalaali(gy, gm, gd)
    const result =
      String(jy).padStart(4, "0") +
      "/" +
      String(jm).padStart(2, "0") +
      "/" +
      String(jd).padStart(2, "0")
    return jalaliDateToGregorianDateOnly(result) === value ? result : ""
  } catch {
    return ""
  }
}
export {
  gregorianDateOnlyToJalali,
  instantToJalaliDate,
  jalaliDateError,
  jalaliDateToGregorianDateOnly,
  normalizeDateDigits,
  normalizeJalaliDate,
  toEn,
  toFa,
  todayJalaliDate,
}
