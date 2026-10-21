import type { AppNotification } from "@/types";
import { at } from "./reference";

export const notifications: AppNotification[] = [
  {
    id: "n-1",
    kind: "report_revision",
    title: "گزارش شما نیاز به اصلاح دارد",
    body: "گزارش پایش دوره ۳ آوا موسوی از سوی سرپرست تیم بازگردانده شد. لطفاً بخش سلامت و شرایط تحصیلی را تکمیل کنید.",
    createdAt: at(0, 7, 40),
    read: false,
    caseId: "case-mousavi",
    href: "/reports/report-ava",
  },
  {
    id: "n-2",
    kind: "schedule_change",
    title: "زمان بازدید خانواده رضایی تغییر کرد",
    body: "بازدید امروز از ساعت ۱۰:۳۰ به ۱۱:۴۵ منتقل شد. لطفاً برنامه خود را به‌روزرسانی کنید.",
    createdAt: at(0, 6, 50),
    read: false,
    caseId: "case-rezaei",
    href: "/visits/visit-rezaei-today",
  },
  {
    id: "n-3",
    kind: "assignment",
    title: "پرونده جدید به شما ارجاع شد",
    body: "پرونده خانواده شریفی (۱۴۰۵-۱۱۸۸) با اولویت ارزیابی اولیه به کارتابل شما اضافه شد.",
    createdAt: at(-1, 14, 20),
    read: false,
    caseId: "case-sharifi",
    href: "/cases/case-sharifi",
  },
  {
    id: "n-4",
    kind: "report_approved",
    title: "گزارش پایش یسنا احمدی تأیید شد",
    body: "گزارش شما با امتیاز ۴ از ۵ تأیید شد. نتیجه در تاریخچه پرونده ثبت شد.",
    createdAt: at(-1, 10, 5),
    read: true,
    caseId: "case-ahmadi",
    href: "/reports/report-yasna",
  },
  {
    id: "n-5",
    kind: "message",
    title: "پیام از دکتر نگار صادقی",
    body: "برای بازدید پیگیری خانواده احمدی، لطفاً گزارش مدرسه را هم دریافت کنید.",
    createdAt: at(-2, 16, 30),
    read: true,
    caseId: "case-ahmadi",
    href: "/cases/case-ahmadi",
  },
  {
    id: "n-6",
    kind: "system",
    title: "به‌روزرسانی سطح دسترسی",
    body: "دسترسی شما به پرونده‌های منطقه ۳ تمدید شد. این دسترسی تا پایان آذر معتبر است.",
    createdAt: at(-4, 9, 0),
    read: true,
  },
];

export function unreadCount(items: AppNotification[]): number {
  return items.filter((n) => !n.read).length;
}
