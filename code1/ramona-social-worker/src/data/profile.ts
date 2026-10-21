import type { SocialWorkerProfile } from "@/types";
import { at } from "./reference";

export const profile: SocialWorkerProfile = {
  id: "sw-1",
  fullName: "سارا محمدی",
  title: "مددکار اجتماعی ارشد",
  employeeCode: "۴۲۱۸",
  office: "دفتر منطقه ۳ — تهران",
  email: "s.mohammadi@ramona.ir",
  phone: "۰۹۱۲ ۳۴۵ ۶۷۸۹",
  licenseNumber: "MSW-۱۴۰۲-۳۳۱۷",
  joinedAt: at(-1240, 9, 0),
  supervisor: { name: "دکتر نگار صادقی", title: "سرپرست تیم مددکاری" },
  stats: {
    activeCases: 6,
    visitsThisMonth: 14,
    openReports: 3,
    approvalRate: 92,
  },
  permissions: [
    { label: "مشاهده پرونده‌های ارجاع‌شده", granted: true },
    { label: "ثبت بازدید و گزارش", granted: true },
    { label: "دسترسی به اطلاعات هویتی کودکان", granted: true },
    { label: "تصمیم‌گیری درباره واگذاری", granted: false },
    { label: "مشاهده اسناد حقوقی پرونده", granted: false },
  ],
  availability: [
    { day: "شنبه تا چهارشنبه", hours: "۸:۰۰ تا ۱۶:۳۰" },
    { day: "پنج‌شنبه", hours: "۸:۰۰ تا ۱۲:۰۰" },
    { day: "جمعه", hours: "تعطیل" },
  ],
};
