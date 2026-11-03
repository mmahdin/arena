// Adapted from ramona-platform-ui-family/src/design-system/components/JalaliDateField.tsx.
// JSX/CSS port: segmented dates, date-only validation, month/year navigation and focus behavior.
import { useEffect, useId, useRef, useState } from "react"
import {
  CalendarDays,
  ChevronDown,
  ChevronLeft,
  ChevronRight,
  X,
} from "lucide-react"
import { toFa } from "./jalaliDate.js"
import {
  jalaliDateToGregorianDateOnly,
  normalizeDateDigits,
  normalizeJalaliDate,
  todayJalaliDate,
  jalaliDateError,
} from "./jalaliDate.js"
import {
  Dialog,
  DialogContent,
  DialogTitle,
  DialogDescription,
} from "./CalendarDialog.jsx"
const cn = (...values) => values.filter(Boolean).join(" ")
const months = [
  "فروردین",
  "اردیبهشت",
  "خرداد",
  "تیر",
  "مرداد",
  "شهریور",
  "مهر",
  "آبان",
  "آذر",
  "دی",
  "بهمن",
  "اسفند",
]
const weekdays = ["ش", "ی", "د", "س", "چ", "پ", "ج"]
function JalaliDateField({
  id,
  name,
  label,
  value = "",
  onChange,
  disabled,
  required,
  hint,
  error,
  minDate,
  maxDate,
  className,
}) {
  const generated = useId()
  const inputId = id ?? generated
  const [year = "", month = "", day = ""] =
    normalizeJalaliDate(value).split("/")
  const parts = { year, month, day }
  const refs = {
    year: useRef(null),
    month: useRef(null),
    day: useRef(null),
  }
  const [calendarOpen, setCalendarOpen] = useState(false)
  const [invalid, setInvalid] = useState(false)
  const today = todayJalaliDate()
  const normalizedMin =
    minDate && jalaliDateToGregorianDateOnly(minDate)
      ? normalizeJalaliDate(minDate)
      : void 0
  const normalizedMax =
    maxDate && jalaliDateToGregorianDateOnly(maxDate)
      ? normalizeJalaliDate(maxDate)
      : void 0
  const completeValue = jalaliDateToGregorianDateOnly(value)
    ? normalizeJalaliDate(value)
    : ""
  const validation = jalaliDateError(
    value,
    required,
    normalizedMin,
    normalizedMax,
  )
  const fullyTyped = /^\d{4}\/\d{2}\/\d{2}$/.test(normalizeJalaliDate(value))
  const fieldError = error || (invalid || fullyTyped ? validation : void 0)
  const [visible, setVisible] = useState(() => ({
    year: Number(today.slice(0, 4)),
    month: Number(today.slice(5, 7)),
  }))
  useEffect(() => {
    refs.year.current?.setCustomValidity(error || validation || "")
  }, [value, error, validation])
  useEffect(() => {
    if (disabled) setCalendarOpen(false)
  }, [disabled])
  const change = (part, raw, pad = false) => {
    let next = normalizeDateDigits(raw)
      .replace(/\D/g, "")
      .slice(0, part === "year" ? 4 : 2)
    if (pad && part !== "year" && next) next = next.padStart(2, "0")
    const updated = { ...parts, [part]: next }
    onChange(
      Object.values(updated).some(Boolean)
        ? `${updated.year}/${updated.month}/${updated.day}`
        : "",
    )
    setInvalid(false)
    if (!pad && next.length === (part === "year" ? 4 : 2)) {
      if (part === "year") refs.month.current?.focus()
      if (part === "month") refs.day.current?.focus()
    }
  }
  const paste = (event) => {
    const normalized = normalizeJalaliDate(event.clipboardData.getData("text"))
    const full = /^(\d{4})\/(\d{1,2})\/(\d{1,2})$/.exec(normalized)
    if (full) {
      event.preventDefault()
      onChange(
        `${full[1]}/${full[2].padStart(2, "0")}/${full[3].padStart(2, "0")}`,
      )
      setInvalid(false)
      requestAnimationFrame(() => refs.day.current?.focus())
    }
  }
  const openCalendar = () => {
    let initial = completeValue || today
    if (normalizedMin && initial < normalizedMin) initial = normalizedMin
    if (normalizedMax && initial > normalizedMax) initial = normalizedMax
    setVisible({
      year: Number(initial.slice(0, 4)),
      month: Number(initial.slice(5, 7)),
    })
    setCalendarOpen(true)
  }
  return (
    <div className={cn("jalali-date-field", className)} data-slot="field">
      <div className="jdf-field-heading">
        <span id={`${inputId}-label`} className="jdf-label">
          {label}
          {required && (
            <span aria-hidden="true" className="jdf-required">
              *
            </span>
          )}
        </span>
        <button
          id={`${inputId}-trigger`}
          type="button"
          disabled={disabled}
          aria-label={`انتخاب ${label} از تقویم`}
          aria-haspopup="dialog"
          aria-expanded={calendarOpen}
          aria-controls={calendarOpen ? `${inputId}-calendar` : void 0}
          onClick={openCalendar}
          className="jdf-trigger"
        >
          <CalendarDays className="jdf-icon-small" aria-hidden="true" />
          انتخاب از تقویم
        </button>
      </div>
      <div
        id={inputId}
        role="group"
        aria-labelledby={`${inputId}-label`}
        aria-invalid={Boolean(fieldError)}
        aria-describedby={hint || fieldError ? `${inputId}-message` : void 0}
        dir="ltr"
        className="jdf-segments"
      >
        {["year", "month", "day"].map((part, index) => (
          <div key={part} className="jdf-segment-wrap">
            {index > 0 && (
              <span aria-hidden="true" className="jdf-separator">
                /
              </span>
            )}
            <input
              ref={refs[part]}
              id={`${inputId}-${part}`}
              type="text"
              inputMode="numeric"
              autoComplete="off"
              aria-label={{ year: "سال", month: "ماه", day: "روز" }[part]}
              aria-invalid={Boolean(fieldError)}
              aria-describedby={
                hint || fieldError ? `${inputId}-message` : void 0
              }
              required={required}
              disabled={disabled}
              maxLength={part === "year" ? 4 : 2}
              placeholder={{ year: "سال", month: "ماه", day: "روز" }[part]}
              value={toFa(parts[part])}
              className={cn("jdf-segment", fieldError && "jdf-invalid")}
              onChange={(event) => change(part, event.target.value)}
              onBlur={(event) => change(part, event.currentTarget.value, true)}
              onPaste={paste}
              onInvalid={(event) => {
                event.preventDefault()
                setInvalid(true)
                const first = event.currentTarget.form?.querySelector(
                  "input:invalid,select:invalid,textarea:invalid",
                )
                if (!first || first === event.currentTarget) {
                  const missing =
                    ["year", "month", "day"].find((p) => !parts[p]) ?? "day"
                  refs[missing].current?.focus()
                }
              }}
            />
          </div>
        ))}
      </div>
      {name && (
        <input type="hidden" name={name} value={value} disabled={disabled} />
      )}
      {(fieldError || hint) && (
        <p
          id={`${inputId}-message`}
          role={fieldError ? "alert" : void 0}
          className={cn("jdf-message", fieldError ? "jdf-error" : "jdf-hint")}
        >
          {fieldError || hint}
        </p>
      )}
      {calendarOpen && !disabled && (
        <JalaliCalendarDialog
          id={`${inputId}-calendar`}
          label={label}
          visible={visible}
          onVisibleChange={setVisible}
          selectedDate={completeValue}
          today={today}
          minDate={normalizedMin}
          maxDate={normalizedMax}
          returnFocusId={`${inputId}-trigger`}
          onSelect={(date) => {
            onChange(date)
            setInvalid(false)
            setCalendarOpen(false)
          }}
          onClose={() => setCalendarOpen(false)}
        />
      )}
    </div>
  )
}
const MIN_JALALI_YEAR = 1
const MAX_JALALI_YEAR = 3177
const YEARS_PER_PAGE = 12
const yearPageStart = (year) =>
  Math.floor(year / YEARS_PER_PAGE) * YEARS_PER_PAGE
function monthLength(year, month) {
  for (let day = 31; day >= 28; day--) {
    if (
      jalaliDateToGregorianDateOnly(
        `${String(year).padStart(4, "0")}/${String(month).padStart(2, "0")}/${day}`,
      )
    )
      return day
  }
  return 29
}
function previousMonth(year, month) {
  return month === 1
    ? { year: year - 1, month: 12 }
    : { year, month: month - 1 }
}
function nextMonth(year, month) {
  return month === 12
    ? { year: year + 1, month: 1 }
    : { year, month: month + 1 }
}
function JalaliCalendarDialog({
  id,
  label,
  visible,
  onVisibleChange,
  selectedDate,
  today,
  minDate,
  maxDate,
  onSelect,
  onClose,
  returnFocusId,
}) {
  const closeButtonRef = useRef(null)
  const yearTriggerRef = useRef(null)
  const activeYearRef = useRef(null)
  const restoreYearTriggerFocus = useRef(false)
  const [view, setView] = useState("days")
  const [yearPage, setYearPage] = useState(() => yearPageStart(visible.year))
  const minYear = Math.max(
    MIN_JALALI_YEAR,
    minDate ? Number(minDate.slice(0, 4)) : MIN_JALALI_YEAR,
  )
  const maxYear = Math.min(
    MAX_JALALI_YEAR,
    maxDate ? Number(maxDate.slice(0, 4)) : MAX_JALALI_YEAR,
  )
  const years = Array.from({ length: YEARS_PER_PAGE }, (_, i) => yearPage + i)
  const preferredFocusYear = Math.max(minYear, Math.min(maxYear, visible.year))
  const canGoToPreviousYears = yearPage - 1 >= minYear
  const canGoToNextYears = yearPage + YEARS_PER_PAGE <= maxYear
  const previous = previousMonth(visible.year, visible.month)
  const following = nextMonth(visible.year, visible.month)
  const previousLastDate = `${String(previous.year).padStart(4, "0")}/${String(previous.month).padStart(2, "0")}/${monthLength(previous.year, previous.month)}`
  const nextFirstDate = `${String(following.year).padStart(4, "0")}/${String(following.month).padStart(2, "0")}/01`
  const canGoToPreviousMonth =
    previous.year >= minYear && (!minDate || previousLastDate >= minDate)
  const canGoToNextMonth =
    following.year <= maxYear && (!maxDate || nextFirstDate <= maxDate)
  const openYearSelection = () => {
    setYearPage(yearPageStart(preferredFocusYear))
    setView("years")
  }
  const returnToDays = () => {
    restoreYearTriggerFocus.current = true
    setView("days")
  }
  const selectYear = (year) => {
    let month = visible.month
    if (minDate && year === minYear)
      month = Math.max(month, Number(minDate.slice(5, 7)))
    if (maxDate && year === maxYear)
      month = Math.min(month, Number(maxDate.slice(5, 7)))
    onVisibleChange({ year, month })
    returnToDays()
  }
  useEffect(() => {
    if (view === "years") activeYearRef.current?.focus()
    if (view === "days" && restoreYearTriggerFocus.current) {
      yearTriggerRef.current?.focus()
      restoreYearTriggerFocus.current = false
    }
  }, [view])
  const monthStart = jalaliDateToGregorianDateOnly(
    `${String(visible.year).padStart(4, "0")}/${String(visible.month).padStart(2, "0")}/01`,
  )
  const startOffset = monthStart
    ? (/* @__PURE__ */ new Date(`${monthStart}T12:00:00Z`).getUTCDay() + 1) % 7
    : 0
  const dayCount = monthLength(visible.year, visible.month)
  const days = Array.from(
    { length: Math.ceil((startOffset + dayCount) / 7) * 7 },
    (_, i) => {
      const day = i - startOffset + 1
      return day > 0 && day <= dayCount ? day : null
    },
  )
  const todayAllowed =
    (!minDate || today >= minDate) && (!maxDate || today <= maxDate)
  const todayMonth = today.slice(0, 7)
  const todayMonthAllowed =
    (!minDate || todayMonth >= minDate.slice(0, 7)) &&
    (!maxDate || todayMonth <= maxDate.slice(0, 7))
  return (
    <Dialog
      open
      onOpenChange={(open) => {
        if (!open) onClose()
      }}
    >
      <DialogContent
        id={id}
        dir="rtl"
        aria-labelledby={`${id}-title`}
        aria-describedby={`${id}-description`}
        showCloseButton={false}
        className="jdf-dialog"
        onOpenAutoFocus={(event) => {
          event.preventDefault()
          closeButtonRef.current?.focus()
        }}
        onCloseAutoFocus={(event) => {
          event.preventDefault()
          document.getElementById(returnFocusId)?.focus()
        }}
      >
        <div className="jdf-dialog-heading">
          <div>
            <DialogTitle id={`${id}-title`} className="jdf-title">
              {view === "years" ? "انتخاب سال" : "انتخاب تاریخ"}
            </DialogTitle>
            <DialogDescription
              id={`${id}-description`}
              className="jdf-description"
            >
              {label}
            </DialogDescription>
          </div>
          <button
            ref={closeButtonRef}
            type="button"
            aria-label="بستن تقویم"
            onClick={onClose}
            className="jdf-icon-button"
          >
            <X className="jdf-icon" aria-hidden />
          </button>
        </div>
        {view === "days" ? (
          <>
            <div className="jdf-navigation">
              <button
                type="button"
                aria-label="ماه قبل"
                disabled={!canGoToPreviousMonth}
                onClick={() => onVisibleChange(previous)}
                className="jdf-icon-button"
              >
                <ChevronRight className="jdf-icon" aria-hidden />
              </button>
              <div className="jdf-month-heading" aria-live="polite">
                <span>{months[visible.month - 1]}</span>
                <button
                  ref={yearTriggerRef}
                  type="button"
                  aria-label={`انتخاب سال، ${toFa(String(visible.year))}`}
                  onClick={openYearSelection}
                  className="jdf-year-trigger"
                >
                  {toFa(String(visible.year))}
                  <ChevronDown className="jdf-icon-small" aria-hidden="true" />
                </button>
              </div>
              <button
                type="button"
                aria-label="ماه بعد"
                disabled={!canGoToNextMonth}
                onClick={() => onVisibleChange(following)}
                className="jdf-icon-button"
              >
                <ChevronLeft className="jdf-icon" aria-hidden />
              </button>
            </div>
            <div className="jdf-days">
              {weekdays.map((day, i) => (
                <span key={i} className="jdf-weekday">
                  {day}
                </span>
              ))}
              {days.map((day, i) => {
                if (day === null)
                  return <span key={`empty-${i}`} aria-hidden="true" />
                const date = `${String(visible.year).padStart(4, "0")}/${String(visible.month).padStart(2, "0")}/${String(day).padStart(2, "0")}`
                const unavailable = Boolean(
                  (minDate && date < minDate) || (maxDate && date > maxDate),
                )
                const selected = date === selectedDate
                const isToday = date === today
                return (
                  <button
                    key={date}
                    type="button"
                    disabled={unavailable}
                    aria-pressed={selected}
                    aria-label={`${toFa(String(day))} ${months[visible.month - 1]} ${toFa(String(visible.year))}`}
                    onClick={() => onSelect(date)}
                    className={cn(
                      "jdf-day",
                      selected ? "is-selected" : isToday ? "is-today" : "",
                    )}
                  >
                    {toFa(String(day))}
                  </button>
                )
              })}
            </div>
          </>
        ) : (
          <>
            <div className="jdf-navigation">
              <button
                type="button"
                aria-label="۱۲ سال قبل"
                disabled={!canGoToPreviousYears}
                onClick={() => setYearPage((page) => page - YEARS_PER_PAGE)}
                className="jdf-icon-button"
              >
                <ChevronRight className="jdf-icon" aria-hidden />
              </button>
              <span className="jdf-year-range" aria-live="polite">
                {toFa(String(yearPage))} تا{" "}
                {toFa(String(yearPage + YEARS_PER_PAGE - 1))}
              </span>
              <button
                type="button"
                aria-label="۱۲ سال بعد"
                disabled={!canGoToNextYears}
                onClick={() => setYearPage((page) => page + YEARS_PER_PAGE)}
                className="jdf-icon-button"
              >
                <ChevronLeft className="jdf-icon" aria-hidden />
              </button>
            </div>
            <div
              role="group"
              aria-label="انتخاب سال شمسی"
              className="jdf-years"
            >
              {years.map((year) => {
                const selected = year === visible.year
                const isTodayYear = year === Number(today.slice(0, 4))
                return (
                  <button
                    key={year}
                    ref={year === preferredFocusYear ? activeYearRef : void 0}
                    type="button"
                    disabled={year < minYear || year > maxYear}
                    aria-pressed={selected}
                    aria-label={`سال ${toFa(String(year))}`}
                    onClick={() => selectYear(year)}
                    className={cn(
                      "jdf-year",
                      selected ? "is-selected" : isTodayYear ? "is-today" : "",
                    )}
                  >
                    {toFa(String(year))}
                  </button>
                )
              })}
            </div>
          </>
        )}
        <div className="jdf-footer">
          {view === "years" && (
            <button
              type="button"
              className="jdf-footer-button"
              onClick={returnToDays}
            >
              بازگشت به تقویم
            </button>
          )}
          {todayMonthAllowed && (
            <button
              type="button"
              className="jdf-footer-button"
              onClick={() => {
                onVisibleChange({
                  year: Number(today.slice(0, 4)),
                  month: Number(today.slice(5, 7)),
                })
                if (view === "years") returnToDays()
              }}
            >
              برو به ماه جاری
            </button>
          )}
          {todayAllowed && (
            <button
              type="button"
              className="jdf-footer-button"
              onClick={() => onSelect(today)}
            >
              انتخاب امروز
            </button>
          )}
        </div>
      </DialogContent>
    </Dialog>
  )
}
export { JalaliDateField }
