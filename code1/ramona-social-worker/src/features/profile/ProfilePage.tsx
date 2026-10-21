import { useState } from "react";
import { useNavigate } from "react-router-dom";
import {
  Bell,
  Briefcase,
  CalendarClock,
  Check,
  ChevronLeft,
  ClipboardCheck,
  Copy,
  Info,
  Lock,
  LogOut,
  Mail,
  Palette,
  Pencil,
  Phone,
  ShieldCheck,
  UserRound,
} from "lucide-react";
import { useAppActions, useWorkspace, useNow } from "@/app/store";
import { formatDate, parse } from "@/lib/date";
import { Page, PageHeader } from "@/components/app/Page";
import { Avatar } from "@/components/ui/Avatar";
import { Badge } from "@/components/ui/Badge";
import { Button } from "@/components/ui/Button";
import { Sheet } from "@/components/ui/Sheet";
import { Field, Input } from "@/components/ui/Field";
import { Switch } from "@/components/ui/Switch";
import { useToast } from "@/components/ui/Toast";
import { PersonRow } from "@/components/domain/People";
import "./ProfilePage.css";

export function ProfilePage() {
  const data = useWorkspace();
  const now = useNow();
  const actions = useAppActions();
  const toast = useToast();
  const navigate = useNavigate();
  const profile = data.profile;

  const [editOpen, setEditOpen] = useState(false);
  const [phone, setPhone] = useState(profile.phone);
  const [email, setEmail] = useState(profile.email);
  const [notificationsOn, setNotificationsOn] = useState(true);
  const [remindersOn, setRemindersOn] = useState(true);

  const saveProfile = () => {
    actions.updateProfile({ phone: phone.trim(), email: email.trim() });
    setEditOpen(false);
    toast.show({ message: "اطلاعات تماس به‌روزرسانی شد.", tone: "success" });
  };

  const copy = async (value: string, label: string) => {
    try {
      await navigator.clipboard.writeText(value);
      toast.show({ message: `${label} کپی شد.`, tone: "success" });
    } catch {
      toast.show({ message: `${label}: ${value}`, tone: "default" });
    }
  };

  const stats = [
    { label: "پرونده فعال", value: profile.stats.activeCases },
    { label: "بازدید این ماه", value: profile.stats.visitsThisMonth },
    { label: "گزارش در جریان", value: profile.stats.openReports },
    { label: "نرخ تأیید گزارش", value: profile.stats.approvalRate, suffix: "٪" },
  ];

  return (
    <Page>
      <PageHeader eyebrow="حساب حرفه‌ای شما" title="پروفایل" />

      <section className="profile-card">
        <div className="profile-card__head">
          <Avatar name={profile.fullName} size="xl" />
          <div className="profile-card__identity">
            <h2 className="profile-card__name">{profile.fullName}</h2>
            <p className="profile-card__title">{profile.title}</p>
            <p className="profile-card__office">{profile.office}</p>
          </div>
          <button
            type="button"
            className="profile-card__edit"
            onClick={() => setEditOpen(true)}
            aria-label="ویرایش اطلاعات تماس"
          >
            <Pencil size={16} />
          </button>
        </div>

        <div className="profile-card__meta">
          <span>
            کد پرسنلی <span className="num">{profile.employeeCode}</span>
          </span>
          <span className="dot-sep" aria-hidden="true">·</span>
          <span>
            عضویت از <span className="num">{formatDate(parse(profile.joinedAt))}</span>
          </span>
        </div>

        <div className="profile-stats">
          {stats.map((stat) => (
            <div className="profile-stat" key={stat.label}>
              <span className="profile-stat__value num">
                {new Intl.NumberFormat("fa-IR").format(stat.value)}
                {stat.suffix ?? ""}
              </span>
              <span className="profile-stat__label">{stat.label}</span>
            </div>
          ))}
        </div>
      </section>

      <section className="panel">
        <h2 className="panel__title">
          <UserRound size={15} aria-hidden="true" />
          سرپرست تیم
        </h2>
        <PersonRow
          name={profile.supervisor.name}
          subtitle={profile.supervisor.title}
          badge={<Badge tone="brand">سرپرست</Badge>}
          tone="info"
        />
      </section>

      <section className="panel">
        <div className="panel__head">
          <h2 className="panel__title">
            <ShieldCheck size={15} aria-hidden="true" />
            دسترسی‌ها
          </h2>
          <Badge tone="neutral" variant="outline">بر اساس شرح وظایف</Badge>
        </div>
        <ul className="permissions">
          {profile.permissions.map((permission) => (
            <li
              key={permission.label}
              className={permission.granted ? "permission" : "permission is-locked"}
            >
              <span className="permission__icon" aria-hidden="true">
                {permission.granted ? <Check size={13} strokeWidth={3} /> : <Lock size={12} />}
              </span>
              <span className="permission__label">{permission.label}</span>
              <span className="permission__state">
                {permission.granted ? "مجاز" : "غیرفعال"}
              </span>
            </li>
          ))}
        </ul>
        <p className="profile-note">
          دسترسی به اطلاعات هویتی و پرونده‌ها بر اساس مسئولیت شما تعیین می‌شود. برای تغییر سطح
          دسترسی با سرپرست تیم هماهنگ کنید.
        </p>
      </section>

      <section className="panel">
        <h2 className="panel__title">
          <Phone size={15} aria-hidden="true" />
          اطلاعات تماس
        </h2>
        <div className="contact-list">
          <button type="button" className="contact-row" onClick={() => copy(profile.phone, "شماره تماس")}>
            <span className="contact-row__icon" aria-hidden="true">
              <Phone size={15} />
            </span>
            <span className="contact-row__text">
              <span className="contact-row__label">تلفن همراه</span>
              <span className="contact-row__value num">{profile.phone}</span>
            </span>
            <Copy size={14} className="contact-row__action" aria-hidden="true" />
          </button>
          <button type="button" className="contact-row" onClick={() => copy(profile.email, "ایمیل")}>
            <span className="contact-row__icon" aria-hidden="true">
              <Mail size={15} />
            </span>
            <span className="contact-row__text">
              <span className="contact-row__label">ایمیل سازمانی</span>
              <span className="contact-row__value">{profile.email}</span>
            </span>
            <Copy size={14} className="contact-row__action" aria-hidden="true" />
          </button>
          <div className="contact-row contact-row--static">
            <span className="contact-row__icon" aria-hidden="true">
              <Briefcase size={15} />
            </span>
            <span className="contact-row__text">
              <span className="contact-row__label">شماره پروانه مددکاری</span>
              <span className="contact-row__value num">{profile.licenseNumber}</span>
            </span>
          </div>
        </div>
      </section>

      <section className="panel">
        <h2 className="panel__title">
          <CalendarClock size={15} aria-hidden="true" />
          ساعات کاری
        </h2>
        <ul className="availability">
          {profile.availability.map((slot) => (
            <li key={slot.day} className="availability__row">
              <span>{slot.day}</span>
              <span className="availability__hours num">{slot.hours}</span>
            </li>
          ))}
        </ul>
      </section>

      <section className="panel">
        <h2 className="panel__title">
          <Bell size={15} aria-hidden="true" />
          تنظیمات
        </h2>
        <div className="profile-switches">
          <Switch
            id="notif"
            checked={notificationsOn}
            onChange={setNotificationsOn}
            label="اعلان‌های کارتابل"
            description="ارجاع جدید، تغییر زمان بازدید و بازگشت گزارش"
          />
          <Switch
            id="reminders"
            checked={remindersOn}
            onChange={setRemindersOn}
            label="یادآور بازدیدها"
            description="یک ساعت پیش از هر بازدید"
          />
        </div>
      </section>

      <section className="profile-links">
        <button type="button" className="profile-link" onClick={() => navigate("/design")}>
          <span className="profile-link__icon" aria-hidden="true">
            <Palette size={16} />
          </span>
          <span className="profile-link__text">
            <span className="profile-link__title">سیستم طراحی رامونا</span>
            <span className="profile-link__sub">راهنمای رنگ‌ها، تایپوگرافی و کامپوننت‌ها</span>
          </span>
          <ChevronLeft size={16} className="profile-link__chev" aria-hidden="true" />
        </button>
        <button
          type="button"
          className="profile-link"
          onClick={() =>
            toast.show({
              message: "رامونا، نسخه نمایشی ۰.۱ — داده‌های این نسخه آزمایشی است.",
              tone: "default",
            })
          }
        >
          <span className="profile-link__icon" aria-hidden="true">
            <Info size={16} />
          </span>
          <span className="profile-link__text">
            <span className="profile-link__title">درباره رامونا</span>
            <span className="profile-link__sub">نسخه ۰.۱.۰ · پلتفرم واگذاری و فرزندخواندگی</span>
          </span>
          <ChevronLeft size={16} className="profile-link__chev" aria-hidden="true" />
        </button>
      </section>

      <button
        type="button"
        className="profile-logout"
        onClick={() =>
          toast.show({ message: "خروج از حساب در نسخه نمایشی غیرفعال است.", tone: "default" })
        }
      >
        <LogOut size={16} aria-hidden="true" />
        خروج از حساب
      </button>

      <p className="profile-footer">
        رامونا — کارتابل مددکار اجتماعی · آخرین به‌روزرسانی داده‌ها:{" "}
        {formatDate(now)}
      </p>

      <Sheet
        open={editOpen}
        onClose={() => setEditOpen(false)}
        title="ویرایش اطلاعات تماس"
        description="تغییر شماره تماس و ایمیل نیازمند تأیید سرپرست است."
        footer={
          <>
            <Button variant="secondary" onClick={() => setEditOpen(false)}>
              انصراف
            </Button>
            <Button onClick={saveProfile} leadingIcon={<ClipboardCheck size={16} />}>
              ذخیره اطلاعات
            </Button>
          </>
        }
      >
        <div className="stack stack-4">
          <Field label="شماره تلفن همراه" htmlFor="profile-phone">
            <Input
              id="profile-phone"
              value={phone}
              onChange={(event) => setPhone(event.target.value)}
              inputMode="tel"
              dir="ltr"
              style={{ textAlign: "start" }}
            />
          </Field>
          <Field label="ایمیل سازمانی" htmlFor="profile-email">
            <Input
              id="profile-email"
              type="email"
              value={email}
              onChange={(event) => setEmail(event.target.value)}
              dir="ltr"
              style={{ textAlign: "start" }}
            />
          </Field>
        </div>
      </Sheet>
    </Page>
  );
}
