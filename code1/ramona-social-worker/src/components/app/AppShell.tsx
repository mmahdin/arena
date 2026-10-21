import type { ReactNode } from "react";
import { useLocation } from "react-router-dom";
import { cn } from "@/lib/cn";
import { PageSkeleton } from "@/components/ui/Skeleton";
import { useAppContext } from "@/app/store";
import { BottomNav, NAV_ROUTES } from "./BottomNav";
import "./AppShell.css";

export function AppShell({ children }: { children: ReactNode }) {
  const { status } = useAppContext();
  const location = useLocation();
  const showNav = NAV_ROUTES.includes(location.pathname);

  return (
    <div className="app-frame">
      <a className="skip-link" href="#main">
        رفتن به محتوای اصلی
      </a>
      <aside className="app-aside" aria-hidden="true">
        <div className="app-aside__brand">
          <span className="app-aside__mark">ر</span>
          <span className="app-aside__name">رامونا</span>
        </div>
        <p className="app-aside__product">کارتابل مددکار اجتماعی</p>
        <p className="app-aside__note">
          فضای حرفه‌ای مدیریت بازدیدها، پرونده‌ها و پایش کودکان در فرآیندهای فرزندخواندگی و
          واگذاری.
        </p>
        <div className="app-aside__rule" />
        <ul className="app-aside__points">
          <li>طراحی موبایل‌اول با محتوای فارسی</li>
          <li>دسترسی به اطلاعات بر اساس سطح مجوز</li>
          <li>داده‌های این نسخه آزمایشی است</li>
        </ul>
        <p className="app-aside__footer">نمایش دسکتاپ: قالب موبایل در وسط صفحه</p>
      </aside>

      <div className="app-column">
        <main
          id="main"
          className={cn("app-main", showNav && "app-main--with-nav")}
        >
          {status === "loading" ? <PageSkeleton /> : children}
        </main>
        {showNav && status === "ready" ? <BottomNav /> : null}
      </div>
    </div>
  );
}
