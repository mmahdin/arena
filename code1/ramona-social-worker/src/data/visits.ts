import type { Visit } from "@/types";
import { at } from "./reference";
import { findCase } from "./cases";

/**
 * Visits are generated from a small factory so the mock set stays consistent
 * with the case records (address, guardians, children) without duplication.
 */
function visit(
  input: Omit<Visit, "address" | "contactName" | "contactPhone"> & {
    address?: Visit["address"];
    contactName?: string;
    contactPhone?: string;
  }
): Visit {
  const relatedCase = findCase(input.caseId);
  return {
    ...input,
    address: input.address ?? relatedCase?.address ?? {
      city: "تهران",
      district: "—",
      line: "—",
    },
    contactName:
      input.contactName ?? relatedCase?.guardians[0]?.fullName ?? "هماهنگی دفتر",
    contactPhone: input.contactPhone ?? relatedCase?.guardians[0]?.phone,
  };
}

export const visits: Visit[] = [
  /* ---------------------------- Today ---------------------------- */
  visit({
    id: "visit-mousavi-today",
    caseId: "case-mousavi",
    caseTitle: "خانواده موسوی",
    kind: "monitoring",
    status: "scheduled",
    startsAt: at(0, 9, 30),
    durationMin: 60,
    childIds: ["child-ava"],
    objective: "بررسی شرایط مراقبت و سلامت آوا و تکمیل چک‌لیست پایش دوره ۳",
    notes: "آوا روزهای زوج تا ساعت ۱۲ در مهدکودک است؛ هماهنگی با مادر انجام شده.",
    checklist: [
      { id: "c1", label: "مشاهده فضای اتاق و محل خواب کودک", done: false },
      { id: "c2", label: "گفت‌وگو با مراقب اصلی درباره تغذیه و خواب", done: false },
      { id: "c3", label: "بررسی وضعیت تحصیلی و حضور در مهدکودک", done: false },
      { id: "c4", label: "مشاهده تعامل عاطفی کودک با اعضای خانواده", done: false },
      { id: "c5", label: "ثبت عکس‌های مستند (با رضایت)", done: false },
    ],
  }),
  visit({
    id: "visit-rezaei-today",
    caseId: "case-rezaei",
    caseTitle: "خانواده رضایی",
    kind: "initial_assessment",
    status: "scheduled",
    startsAt: at(0, 11, 45),
    durationMin: 90,
    objective:
      "بازدید از منزل متقاضیان، ارزیابی شرایط مسکن و محیط زندگی و تکمیل فرم ارزیابی اولیه",
    notes: "همراه داشتن فرم ارزیابی چاپی و لیست مدارک ناقص.",
    checklist: [
      { id: "c1", label: "بررسی متراژ، نور و ایمنی منزل", done: false },
      { id: "c2", label: "مشاهده اتاق اختصاصی کودک", done: false },
      { id: "c3", label: "گفت‌وگو با زوج درباره انگیزه و انتظارات", done: false },
      { id: "c4", label: "بررسی وضعیت مالی و ثبات شغلی", done: false },
      { id: "c5", label: "تکمیل فرم ارزیابی اولیه", done: false },
    ],
  }),
  visit({
    id: "visit-ahmadi-today",
    caseId: "case-ahmadi",
    caseTitle: "خانواده احمدی",
    kind: "follow_up",
    status: "scheduled",
    startsAt: at(0, 16, 15),
    durationMin: 60,
    childIds: ["child-yasna"],
    mandatory: true,
    objective:
      "پیگیری غیبت‌های مدرسه یسنا و گفت‌وگو با مادر درباره برنامه آموزشی کودک",
    notes: "پرونده با اولویت بالا؛ نتیجه باید حداکثر تا فردا ثبت شود.",
    checklist: [
      { id: "c1", label: "گفت‌وگو با مادر درباره علت غیبت‌ها", done: false },
      { id: "c2", label: "بررسی دفتر ارتباط مدرسه", done: false },
      { id: "c3", label: "مشاهده وضعیت تغذیه و بهداشت کودک", done: false },
      { id: "c4", label: "تعیین اقدام پیگیری بعدی", done: false },
    ],
  }),

  /* --------------------------- Tomorrow --------------------------- */
  visit({
    id: "visit-karimi-tomorrow",
    caseId: "case-karimi",
    caseTitle: "خانواده کریمی",
    kind: "introduction",
    status: "scheduled",
    startsAt: at(1, 10, 0),
    durationMin: 60,
    childIds: ["child-danial"],
    objective: "جلسه آشنایی سوم؛ مشاهده تعامل دانیال با خانواده و ثبت مشاهدات",
    checklist: [
      { id: "c1", label: "مشاهده تعامل کودک با والدین", done: false },
      { id: "c2", label: "گفت‌وگو با کودک در فضای امن", done: false },
      { id: "c3", label: "ثبت نکات آماده‌سازی خانواده", done: false },
    ],
  }),
  visit({
    id: "visit-sharifi-tomorrow",
    caseId: "case-sharifi",
    caseTitle: "خانواده شریفی",
    kind: "initial_assessment",
    status: "scheduled",
    startsAt: at(1, 14, 30),
    durationMin: 90,
    objective: "نخستین بازدید از منزل متقاضیان و تکمیل ارزیابی شرایط زندگی",
    checklist: [
      { id: "c1", label: "بازدید از تمام فضاهای منزل", done: false },
      { id: "c2", label: "گفت‌وگو با زوج به‌صورت جداگانه", done: false },
      { id: "c3", label: "تکمیل چک‌لیست ایمنی کودک", done: false },
    ],
  }),

  /* -------------------------- This week -------------------------- */
  visit({
    id: "visit-mousavi-next",
    caseId: "case-mousavi",
    caseTitle: "خانواده موسوی",
    kind: "monitoring",
    status: "scheduled",
    startsAt: at(3, 9, 30),
    durationMin: 60,
    childIds: ["child-ava"],
    objective: "پایش دوره ۴ — بررسی روند سازگاری و آمادگی تحصیلی",
    checklist: [],
  }),
  visit({
    id: "visit-ahmadi-next",
    caseId: "case-ahmadi",
    caseTitle: "خانواده احمدی",
    kind: "follow_up",
    status: "scheduled",
    startsAt: at(4, 16, 0),
    durationMin: 45,
    childIds: ["child-yasna"],
    objective: "بازدید پیگیری نتیجه اقدامات آموزشی یسنا",
    checklist: [],
  }),
  visit({
    id: "visit-karimi-committee",
    caseId: "case-karimi",
    caseTitle: "خانواده کریمی",
    kind: "office_meeting",
    status: "scheduled",
    startsAt: at(5, 10, 0),
    durationMin: 60,
    objective: "جلسه کمیته معرفی و ارائه گزارش آمادگی خانواده",
    address: {
      city: "تهران",
      district: "مرکز رامونا — خیابان ولیعصر",
      line: "خیابان ولیعصر، نبش کوچه شهید بهرامی، پلاک ۲۴۰، طبقه ۳ — سالن جلسات الف",
    },
    contactName: "دبیرخانه کمیته",
    contactPhone: "۰۲۱ ۸۸۷۴ ۲۲۱۰",
    checklist: [],
  }),
  visit({
    id: "visit-rezaei-review",
    caseId: "case-rezaei",
    caseTitle: "خانواده رضایی",
    kind: "office_meeting",
    status: "scheduled",
    startsAt: at(7, 9, 0),
    durationMin: 45,
    objective: "جلسه بررسی پرونده با سرپرست تیم",
    address: {
      city: "تهران",
      district: "مرکز رامونا — خیابان ولیعصر",
      line: "خیابان ولیعصر، نبض کوچه شهید بهرامی، پلاک ۲۴۰، طبقه ۳ — اتاق ۳۱۲",
    },
    checklist: [],
  }),
  visit({
    id: "visit-nikpour-next",
    caseId: "case-nikpour",
    caseTitle: "خانواده نیک‌پور",
    kind: "monitoring",
    status: "scheduled",
    startsAt: at(12, 10, 0),
    durationMin: 60,
    childIds: ["child-mahsa"],
    objective: "پایش دوره ۶ — ارزیابی نهایی قبل از جمع‌بندی پرونده",
    checklist: [],
  }),

  /* ----------------------------- Past ----------------------------- */
  visit({
    id: "visit-nikpour-1d",
    caseId: "case-nikpour",
    caseTitle: "خانواده نیک‌پور",
    kind: "monitoring",
    status: "completed",
    startsAt: at(-1, 10, 0),
    durationMin: 55,
    childIds: ["child-mahsa"],
    objective: "پایش دوره ۵ — بررسی شرایط رشد و پیوند عاطفی",
    reportId: "report-mahsa",
    notes: "بازدید در فضای آرام انجام شد؛ همه شاخص‌ها در محدوده مطلوب.",
    checklist: [
      { id: "c1", label: "مشاهده اتاق کودک", done: true },
      { id: "c2", label: "گفت‌وگو با مادر", done: true },
      { id: "c3", label: "ثبت مشاهدات", done: true },
    ],
  }),
  visit({
    id: "visit-mousavi-2d",
    caseId: "case-mousavi",
    caseTitle: "خانواده موسوی",
    kind: "monitoring",
    status: "completed",
    startsAt: at(-2, 10, 0),
    durationMin: 60,
    childIds: ["child-ava"],
    objective: "پایش دوره ۳ — بررسی سلامت و شرایط تحصیلی آوا",
    reportId: "report-ava",
    notes: "بازدید انجام شد. گزارش برای تکمیل بخش سلامت بازگردانده شد.",
    checklist: [
      { id: "c1", label: "مشاهده اتاق کودک", done: true },
      { id: "c2", label: "گفت‌وگو با مراقب اصلی", done: true },
      { id: "c3", label: "بررسی مدارک مدرسه", done: true },
    ],
  }),
  visit({
    id: "visit-rezaei-3d",
    caseId: "case-rezaei",
    caseTitle: "خانواده رضایی",
    kind: "initial_assessment",
    status: "needs_followup",
    startsAt: at(-3, 9, 30),
    durationMin: 75,
    objective: "بازدید اولیه منزل و بررسی شرایط مسکن",
    reportId: "report-rezaei",
    notes: "مدارک درآمدی ناقص بود؛ نیاز به بازدید تکمیلی.",
    checklist: [
      { id: "c1", label: "بازدید از فضاهای منزل", done: true },
      { id: "c2", label: "بررسی مدارک مالی", done: false },
    ],
  }),
  visit({
    id: "visit-ahmadi-9d",
    caseId: "case-ahmadi",
    caseTitle: "خانواده احمدی",
    kind: "monitoring",
    status: "completed",
    startsAt: at(-9, 11, 0),
    durationMin: 50,
    childIds: ["child-yasna"],
    objective: "پایش دوره ۲ — ارزیابی شرایط نگهداری و آموزش",
    reportId: "report-yasna",
    checklist: [
      { id: "c1", label: "مشاهده منزل", done: true },
      { id: "c2", label: "گفت‌وگو با مادر", done: true },
    ],
  }),
  visit({
    id: "visit-karimi-12d",
    caseId: "case-karimi",
    caseTitle: "خانواده کریمی",
    kind: "introduction",
    status: "completed",
    startsAt: at(-12, 10, 0),
    durationMin: 60,
    childIds: ["child-danial"],
    objective: "جلسه آشنایی دوم در مرکز",
    checklist: [
      { id: "c1", label: "مشاهده تعامل", done: true },
      { id: "c2", label: "ثبت مشاهدات", done: true },
    ],
  }),
  visit({
    id: "visit-sharifi-canceled",
    caseId: "case-sharifi",
    caseTitle: "خانواده شریفی",
    kind: "initial_assessment",
    status: "canceled",
    startsAt: at(-4, 15, 0),
    durationMin: 90,
    objective: "بازدید اولیه منزل",
    notes: "به درخواست خانواده لغو شد؛ زمان جدید هماهنگ شده است.",
    checklist: [],
  }),
  visit({
    id: "visit-mousavi-14d",
    caseId: "case-mousavi",
    caseTitle: "خانواده موسوی",
    kind: "monitoring",
    status: "completed",
    startsAt: at(-14, 9, 30),
    durationMin: 60,
    childIds: ["child-ava"],
    objective: "پایش دوره ۲ — بررسی روند سازگاری",
    reportId: "report-ava-2",
    checklist: [],
  }),
];

export function findVisit(id: string | undefined): Visit | undefined {
  return visits.find((v) => v.id === id);
}
