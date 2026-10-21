import type { Report } from "@/types";
import { at } from "./reference";

export const reports: Report[] = [
  {
    id: "report-ava",
    title: "گزارش پایش دوره ۳ — آوا موسوی",
    type: "monitoring",
    status: "needs_revision",
    caseId: "case-mousavi",
    caseTitle: "خانواده موسوی",
    visitId: "visit-mousavi-2d",
    authorName: "سارا محمدی",
    updatedAt: at(-1, 16, 40),
    submittedAt: at(-2, 19, 0),
    reviewedBy: "دکتر نگار صادقی",
    reviewNote:
      "با سلام، بخش سلامت و شرایط تحصیلی با جزئیات کافی تکمیل نشده است. لطفاً نتیجه پیگیری با مدرسه و وضعیت تغذیه کودک را دقیق‌تر ثبت کنید و سپس گزارش را دوباره ارسال نمایید.",
    progress: 65,
    sections: [
      {
        id: "housing",
        label: "شرایط مسکن و محیط زندگی",
        value:
          "منزل ۹۵ متری در طبقه سوم، دارای دو اتاق خواب است. اتاق آوا جدا و مجهز به میز تحریر و تخت مستقل است. نور طبیعی کافی و محیط امن برای کودک فراهم است.",
      },
      {
        id: "health",
        label: "وضعیت سلامت و تغذیه",
        value:
          "کودک از نظر ظاهری سالم، سرحال و دارای وزن متناسب با سن است. مادر برنامه سه وعده اصلی و دو میان‌وعده را شرح داد و حساسیت غذایی خاصی گزارش نشد.",
      },
      {
        id: "education",
        label: "شرایط تحصیلی",
        value:
          "آوا در مهدکودک دوره پیش‌دبستانی ثبت‌نام شده است. در پیگیری تلفنی با مربی، حضور منظم و مشارکت کودک در فعالیت‌های گروهی گزارش شد؛ نتیجه پیگیری با مدرسه نیز در پرونده ثبت شده است.",
      },
      {
        id: "attachment",
        label: "تعامل عاطفی و پیوند خانوادگی",
        value:
          "تعامل کودک با والدین گرم و طبیعی بود. کودک در طول بازدید احساس امنیت داشت و با مادر ارتباط کلامی خوبی برقرار کرد.",
      },
      {
        id: "conclusion",
        label: "جمع‌بندی و پیشنهاد",
        value:
          "روند سازگاری کودک مطلوب ارزیابی می‌شود. پیشنهاد می‌شود پایش دوره بعدی طبق برنامه انجام شود.",
      },
    ],
  },
  {
    id: "report-rezaei",
    title: "ارزیابی اجتماعی — خانواده رضایی",
    type: "social_assessment",
    status: "submitted",
    caseId: "case-rezaei",
    caseTitle: "خانواده رضایی",
    visitId: "visit-rezaei-3d",
    authorName: "سارا محمدی",
    updatedAt: at(-3, 18, 0),
    submittedAt: at(-3, 18, 0),
    progress: 100,
    sections: [
      {
        id: "housing",
        label: "شرایط مسکن و محیط زندگی",
        value:
          "آپارتمان ۱۲۰ متری با اتاق آماده برای کودک. ساختمان دارای آسانسور و فضای بازی مشترک است.",
      },
      {
        id: "economy",
        label: "وضعیت اقتصادی و اشتغال",
        value:
          "درآمد پایدار خانوار؛ مدارک مالی سال جاری ارائه شد. تعهدات بانکی در سطح متعارف است.",
      },
      {
        id: "motivation",
        label: "انگیزه و انتظارات متقاضیان",
        value:
          "زوج با انگیزه بالا و درک واقع‌بینانه از فرزندخواندگی. آمادگی برای همکاری با تیم مددکاری اعلام شده است.",
      },
      {
        id: "conclusion",
        label: "جمع‌بندی و پیشنهاد",
        value:
          "شرایط عمومی خانواده برای ادامه فرآیند مناسب ارزیابی می‌شود؛ نیاز به بازدید تکمیلی برای بررسی مدارک درآمدی.",
      },
    ],
  },
  {
    id: "report-danial",
    title: "گزارش جلسه معرفی — دانیال کریمی",
    type: "placement",
    status: "draft",
    caseId: "case-karimi",
    caseTitle: "خانواده کریمی",
    visitId: "visit-karimi-12d",
    authorName: "سارا محمدی",
    updatedAt: at(-1, 9, 20),
    progress: 45,
    sections: [
      {
        id: "session",
        label: "شرح جلسه آشنایی",
        value:
          "جلسه دوم آشنایی در مرکز رامونا برگزار شد. دانیال در ابتدا کمی محتاط بود و پس از حدود بیست دقیقه با خانواده وارد بازی شد.",
      },
      {
        id: "observation",
        label: "مشاهدات تعامل کودک و خانواده",
        value:
          "تعامل مادر با کودک گرم و همراه با احترام بود. کودک در پاسخ به تماس چشمی مادر واکنش مثبت نشان داد.",
      },
      {
        id: "plan",
        label: "اقدامات بعدی",
        value: "",
      },
    ],
  },
  {
    id: "report-yasna",
    title: "گزارش پایش دوره ۲ — یسنا احمدی",
    type: "monitoring",
    status: "approved",
    caseId: "case-ahmadi",
    caseTitle: "خانواده احمدی",
    visitId: "visit-ahmadi-9d",
    authorName: "سارا محمدی",
    updatedAt: at(-8, 14, 0),
    submittedAt: at(-9, 18, 0),
    reviewedBy: "دکتر نگار صادقی",
    rating: 4,
    progress: 100,
    sections: [
      {
        id: "housing",
        label: "شرایط مسکن و محیط زندگی",
        value:
          "منزل ۷۵ متری در یک طبقه با حیاط مشترک. فضای اختصاصی کودک محدود است اما شرایط بهداشتی قابل قبول است.",
      },
      {
        id: "health",
        label: "وضعیت سلامت و تغذیه",
        value: "کودک از نظر سلامت عمومی مناسب است؛ گزارش تغذیه نامنظم در برخی روزها ثبت شد.",
      },
      {
        id: "education",
        label: "شرایط تحصیلی",
        value:
          "یسنا در پایه دوم دبستان تحصیل می‌کند. در دو ماه گذشته چند غیبت غیرموجه ثبت شده است.",
      },
      {
        id: "conclusion",
        label: "جمع‌بندی و پیشنهاد",
        value:
          "ادامه پایش با تمرکز بر وضعیت آموزشی کودک و همکاری خانواده با مدرسه پیشنهاد می‌شود.",
      },
    ],
  },
  {
    id: "report-mahsa",
    title: "گزارش پایش دوره ۵ — مهسا نیک‌پور",
    type: "monitoring",
    status: "submitted",
    caseId: "case-nikpour",
    caseTitle: "خانواده نیک‌پور",
    visitId: "visit-nikpour-1d",
    authorName: "سارا محمدی",
    updatedAt: at(-1, 12, 0),
    submittedAt: at(-1, 12, 0),
    progress: 100,
    sections: [
      {
        id: "housing",
        label: "شرایط مسکن و محیط زندگی",
        value: "منزل ۱۴۰ متری، اتاق کودک مجزا و فضای بازی مناسب. محیط ایمن و مرتب.",
      },
      {
        id: "health",
        label: "وضعیت سلامت و تغذیه",
        value: "رشد کودک مطابق سن است؛ برنامه غذایی منظم و خواب کافی گزارش شد.",
      },
      {
        id: "attachment",
        label: "تعامل عاطفی و پیوند خانوادگی",
        value: "پیوند عاطفی قوی میان کودک و والدین مشاهده شد؛ کودک والدین را «مامان» و «بابا» صدا می‌زند.",
      },
      {
        id: "conclusion",
        label: "جمع‌بندی و پیشنهاد",
        value: "شرایط پایدار است. پیشنهاد جمع‌بندی پرونده پس از دوره ششم پایش.",
      },
    ],
  },
  {
    id: "report-sharifi",
    title: "ارزیابی اجتماعی — خانواده شریفی",
    type: "social_assessment",
    status: "draft",
    caseId: "case-sharifi",
    caseTitle: "خانواده شریفی",
    authorName: "سارا محمدی",
    updatedAt: at(-6, 12, 0),
    progress: 10,
    sections: [
      { id: "housing", label: "شرایط مسکن و محیط زندگی", value: "" },
      { id: "economy", label: "وضعیت اقتصادی و اشتغال", value: "" },
      { id: "motivation", label: "انگیزه و انتظارات متقاضیان", value: "" },
      { id: "conclusion", label: "جمع‌بندی و پیشنهاد", value: "" },
    ],
  },
  {
    id: "report-ava-2",
    title: "گزارش پایش دوره ۲ — آوا موسوی",
    type: "monitoring",
    status: "approved",
    caseId: "case-mousavi",
    caseTitle: "خانواده موسوی",
    visitId: "visit-mousavi-14d",
    authorName: "سارا محمدی",
    updatedAt: at(-13, 15, 30),
    submittedAt: at(-14, 18, 0),
    reviewedBy: "دکتر نگار صادقی",
    rating: 5,
    progress: 100,
    sections: [
      {
        id: "housing",
        label: "شرایط مسکن و محیط زندگی",
        value: "شرایط مسکن مناسب و پایدار؛ اتاق کودک مرتب و ایمن است.",
      },
      {
        id: "conclusion",
        label: "جمع‌بندی و پیشنهاد",
        value: "روند سازگاری کودک مطلوب است؛ ادامه پایش طبق برنامه پیشنهاد می‌شود.",
      },
    ],
  },
];

export function findReport(id: string | undefined): Report | undefined {
  return reports.find((r) => r.id === id);
}

export function reportsForCase(caseId: string): Report[] {
  return reports.filter((r) => r.caseId === caseId);
}
