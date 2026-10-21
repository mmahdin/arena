import type { UserProfile } from '@/types';

export const currentUser: UserProfile = {
  id: 'u-1042',
  name: 'نگار احمدی',
  role: 'مددکار اجتماعی',
  organization: 'اداره کل بهزیستی استان تهران',
  unit: 'واحد میزبانی و فرزندخواندگی — شهرستان شمیرانات',
  employeeCode: 'SW-10482',
  licenseNumber: 'ش.م / ۱۲۴۵۷۸',
  phone: '۰۹۱۲ ۳۴۵ ۶۷۸۹',
  email: 'n.ahmadi@ramona.ir',
  supervisor: 'دکتر رضا کاویانی',
  coverageAreas: ['شمیرانات', 'تهران — منطقه ۱', 'تهران — منطقه ۱۲', 'لواسانات'],
  permissions: [
    'مشاهده پرونده‌های محوله',
    'ثبت بازدید و نتیجه آن',
    'تهیه و ارسال گزارش ارزیابی',
    'پایش کودکان جایگذاری‌شده',
    'درخواست هماهنگی مجدد',
  ],
  stats: { activeCases: 7, visitsThisMonth: 14, approvedReports: 9 },
};
