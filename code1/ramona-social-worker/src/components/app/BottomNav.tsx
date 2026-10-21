import { NavLink, useLocation } from "react-router-dom";
import { CalendarCheck2, FolderClosed, House, ListChecks, UserRound } from "lucide-react";
import { cn } from "@/lib/cn";
import { useWorkspace } from "@/app/store";
import { overdueTasks, tasksDueToday } from "@/lib/selectors";
import "./BottomNav.css";

const items = [
  { to: "/", label: "خانه", icon: House, match: (path: string) => path === "/" },
  { to: "/visits", label: "بازدیدها", icon: CalendarCheck2, match: (path: string) => path.startsWith("/visits") },
  {
    to: "/tasks",
    label: "کارها",
    icon: ListChecks,
    match: (path: string) => path.startsWith("/tasks") || path.startsWith("/reports"),
  },
  { to: "/cases", label: "پرونده‌ها", icon: FolderClosed, match: (path: string) => path.startsWith("/cases") },
  { to: "/profile", label: "پروفایل", icon: UserRound, match: (path: string) => path.startsWith("/profile") },
];

export const NAV_ROUTES = ["/", "/visits", "/tasks", "/cases", "/profile"];

export function BottomNav() {
  const { pathname } = useLocation();
  const data = useWorkspace();

  const openCount = overdueTasks(data).length + tasksDueToday(data).filter(
    (t) => t.priority === "urgent" || t.priority === "high"
  ).length;

  return (
    <nav className="bottom-nav" aria-label="پیمایش اصلی">
      <ul className="bottom-nav__list">
        {items.map((item) => {
          const active = item.match(pathname);
          const Icon = item.icon;
          const showBadge = item.to === "/tasks" && openCount > 0;
          return (
            <li key={item.to}>
              <NavLink
                to={item.to}
                className={cn("nav-item", active && "is-active")}
                aria-current={active ? "page" : undefined}
                aria-label={showBadge ? `${item.label}، ${openCount} مورد نیازمند اقدام` : item.label}
              >
                <span className="nav-item__icon">
                  <Icon size={22} strokeWidth={active ? 2.2 : 1.8} aria-hidden="true" />
                  {showBadge ? <span className="nav-item__dot" aria-hidden="true" /> : null}
                </span>
                <span className="nav-item__label">{item.label}</span>
              </NavLink>
            </li>
          );
        })}
      </ul>
    </nav>
  );
}
