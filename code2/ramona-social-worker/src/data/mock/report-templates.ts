import type { ReportSection, ReportType } from '@/types';

/**
 * Report templates.
 *
 * Assessment forms are DATA, not screens: each report type declares its
 * sections and field types here, and one generic editor renders them. Adding a
 * new professional form later means adding an entry to `templateFor` — no new
 * component, no new route.
 */

export const monitoringSections = (): ReportSection[] => [
  {
    id: 'observations',
    title: 'مشاهدات بازدید',
    fields: [
      {
        id: 'obs-summary',
        kind: 'text',
        label: 'شرح مشاهدات',
        multiline: true,
        value:
          'بازدید در ساعت مقرر و با حضور مراقب اصلی انجام شد. کودک در بدو ورود محتاط بود اما پس از دقایقی در گفت‌وگوی خصوصی همکاری مناسبی داشت.',
        hint: 'مشاهدات عینی و قابل استناد بنویسید؛ از قضاوت کلی پرهیز کنید.',
      },
      {
        id: 'obs-done',
        kind: 'checklist',
        label: 'موارد بررسی‌شده در محل',
        options: ['تغذیه و بهداشت', 'فضای خواب کودک', 'وضعیت تحصیلی', 'تعامل با مراقب', 'سلامت جسمی'],
        value: ['تغذیه و بهداشت', 'وضعیت تحصیلی', 'تعامل با مراقب'],
      },
    ],
  },
  {
    id: 'living',
    title: 'شرایط سکونت و محیط منزل',
    fields: [
      { id: 'liv-clean', kind: 'scale', label: 'نظافت و بهداشت محیط', value: 3, max: 5, lowLabel: 'نامناسب', highLabel: 'بسیار مناسب' },
      { id: 'liv-safe', kind: 'scale', label: 'ایمنی محیط زندگی', value: 4, max: 5, lowLabel: 'پرخطر', highLabel: 'ایمن' },
      { id: 'liv-space', kind: 'scale', label: 'فضای اختصاصی کودک', value: 2, max: 5, lowLabel: 'ناموجود', highLabel: 'کامل' },
      {
        id: 'liv-tenure',
        kind: 'select',
        label: 'وضعیت سکونت',
        options: ['مالکیت شخصی', 'استیجاری', 'خانواده میزبان', 'سایر'],
        value: 'استیجاری',
      },
    ],
  },
  {
    id: 'child',
    title: 'وضعیت کودک و تعاملات خانوادگی',
    fields: [
      { id: 'ch-emotional', kind: 'scale', label: 'سلامت عاطفی کودک', value: 3, max: 5, lowLabel: 'نیازمند مداخله', highLabel: 'مطلوب' },
      { id: 'ch-bond', kind: 'scale', label: 'کیفیت تعامل با مراقبان', value: 3, max: 5, lowLabel: 'ضعیف', highLabel: 'قوی' },
      { id: 'ch-school', kind: 'scale', label: 'وضعیت تحصیلی و حضور در مدرسه', value: 2, max: 5, lowLabel: 'بحرانی', highLabel: 'مطلوب' },
      {
        id: 'ch-trend',
        kind: 'select',
        label: 'روند کلی نسبت به بازدید قبل',
        options: ['بهبود', 'بدون تغییر', 'نیازمند توجه', 'روند نزولی'],
        value: 'نیازمند توجه',
      },
    ],
  },
  {
    id: 'conclusion',
    title: 'جمع‌بندی و پیشنهاد مددکار',
    fields: [
      {
        id: 'con-rec',
        kind: 'select',
        label: 'پیشنهاد',
        options: [
          'ادامه وضعیت فعلی',
          'افزایش دفعات پایش',
          'ارجاع به مشاور',
          'برگزاری جلسه با خانواده',
          'ارجاع به کمیته تخصصی',
        ],
        value: 'ارجاع به مشاور',
      },
      {
        id: 'con-note',
        kind: 'text',
        label: 'توضیح جمع‌بندی',
        multiline: true,
        value:
          'با توجه به غیبت‌های پراکنده مدرسه و نشانه‌های اضطراب جدایی، ارجاع به مشاور مرکز و افزایش دفعات پایش به ماهی دو جلسه پیشنهاد می‌شود.',
      },
    ],
  },
];

export const assessmentSections = (): ReportSection[] => [
  {
    id: 'household',
    title: 'شناخت خانوار متقاضی',
    fields: [
      {
        id: 'hh-structure',
        kind: 'text',
        label: 'ترکیب و وضعیت خانوار',
        multiline: true,
        value: 'زوج بدون فرزند، هر دو شاغل، دارای شبکه حمایتی فامیلی در تهران.',
      },
      {
        id: 'hh-motivation',
        kind: 'scale',
        label: 'انگیزه و آمادگی روانی متقاضیان',
        value: 4,
        max: 5,
        lowLabel: 'نامشخص',
        highLabel: 'کاملاً آماده',
      },
    ],
  },
  {
    id: 'home',
    title: 'شرایط سکونت و محیط منزل',
    fields: [
      { id: 'hm-size', kind: 'scale', label: 'متناسب بودن فضای منزل', value: 0, max: 5, lowLabel: 'نامناسب', highLabel: 'بسیار مناسب' },
      { id: 'hm-safe', kind: 'scale', label: 'ایمنی و بهداشت محیط', value: 0, max: 5, lowLabel: 'پرخطر', highLabel: 'ایمن' },
      { id: 'hm-room', kind: 'scale', label: 'اتاق اختصاصی کودک', value: 0, max: 5, lowLabel: 'ناموجود', highLabel: 'کامل' },
    ],
  },
  {
    id: 'capacity',
    title: 'توانمندی اقتصادی و حمایتی',
    fields: [
      { id: 'cp-income', kind: 'scale', label: 'کفایت درآمد', value: 0, max: 5, lowLabel: 'ناکافی', highLabel: 'کافی و مستند' },
      { id: 'cp-support', kind: 'scale', label: 'شبکه حمایتی اطرافیان', value: 0, max: 5, lowLabel: 'ضعیف', highLabel: 'قوی' },
    ],
  },
  {
    id: 'verdict',
    title: 'نظر کارشناسی',
    fields: [
      {
        id: 'vd-rec',
        kind: 'select',
        label: 'نظر مددکار',
        options: ['در انتظار تکمیل', 'صلاحیت تأیید می‌شود', 'نیازمند تکمیل مستندات', 'صلاحیت تأیید نمی‌شود'],
        value: 'در انتظار تکمیل',
      },
      {
        id: 'vd-note',
        kind: 'text',
        label: 'توضیح',
        multiline: true,
        value: '',
        hint: 'تصمیم نهایی درباره صلاحیت بر عهده کمیته تخصصی و مراجع ذی‌صلاح است.',
      },
    ],
  },
];

const visitSections = (): ReportSection[] => [
  {
    id: 'visit-context',
    title: 'مشخصات بازدید',
    fields: [
      {
        id: 'vc-when',
        kind: 'text',
        label: 'زمان و شرایط بازدید',
        value: '',
        multiline: false,
        hint: 'ساعت ورود و خروج، حاضران در منزل.',
      },
      {
        id: 'vc-summary',
        kind: 'text',
        label: 'شرح بازدید',
        value: '',
        multiline: true,
        hint: 'فقط مشاهدات عینی؛ از نتیجه‌گیری کلی در این بخش پرهیز کنید.',
      },
    ],
  },
  {
    id: 'visit-findings',
    title: 'یافته‌ها',
    fields: [
      { id: 'vf-care', kind: 'scale', label: 'کیفیت مراقبت از کودک', value: 0, max: 5, lowLabel: 'نامطلوب', highLabel: 'مطلوب' },
      { id: 'vf-coop', kind: 'scale', label: 'همکاری خانواده', value: 0, max: 5, lowLabel: 'کم', highLabel: 'کامل' },
      {
        id: 'vf-issues',
        kind: 'checklist',
        label: 'موارد مشاهده‌شده',
        options: ['مشکل بهداشتی', 'غیبت تحصیلی', 'تنش خانوادگی', 'نیاز مالی', 'نیاز به مشاوره', 'مورد خاص'],
        value: [],
      },
    ],
  },
  {
    id: 'visit-next',
    title: 'اقدام بعدی',
    fields: [
      {
        id: 'vn-action',
        kind: 'select',
        label: 'اقدام پیشنهادی',
        options: ['بدون اقدام خاص', 'بازدید مجدد زودهنگام', 'ارجاع به مشاور', 'گزارش به سرپرست', 'ارجاع به کمیته تخصصی'],
        value: 'بدون اقدام خاص',
      },
      { id: 'vn-note', kind: 'text', label: 'توضیح', value: '', multiline: true },
    ],
  },
];

const TITLE: Record<ReportType, string> = {
  'home-assessment': 'گزارش ارزیابی شرایط خانواده',
  visit: 'گزارش بازدید',
  monitoring: 'گزارش پایش کودک',
  placement: 'گزارش جایگذاری',
  incident: 'گزارش مورد خاص',
};

export function templateFor(type: ReportType): { title: string; sections: ReportSection[] } {
  const sections =
    type === 'home-assessment'
      ? assessmentSections()
      : type === 'monitoring' || type === 'placement' || type === 'incident'
        ? monitoringSections()
        : visitSections();
  return { title: TITLE[type], sections };
}
