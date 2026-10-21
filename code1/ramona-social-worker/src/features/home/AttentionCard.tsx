import { Link } from "react-router-dom";
import { AlertCircle, ChevronLeft, ClipboardList, FileWarning, HeartPulse } from "lucide-react";
import type { AttentionItem } from "@/lib/selectors";
import { Badge } from "@/components/ui/Badge";
import { cn } from "@/lib/cn";
import "./AttentionCard.css";

const kindIcons = {
  report: FileWarning,
  task: ClipboardList,
  visit: AlertCircle,
  monitoring: HeartPulse,
};

export function AttentionCard({ items }: { items: AttentionItem[] }) {
  if (items.length === 0) return null;

  return (
    <section className="attention" aria-labelledby="attention-title">
      <div className="attention__head">
        <span className="attention__icon" aria-hidden="true">
          <AlertCircle size={18} />
        </span>
        <div className="attention__headtext">
          <h2 className="attention__title" id="attention-title">
            نیازمند اقدام
          </h2>
          <p className="attention__lead">مواردی که امروز به توجه شما نیاز دارند</p>
        </div>
        <Badge tone="danger" variant="soft">
          {new Intl.NumberFormat("fa-IR").format(items.length)} مورد
        </Badge>
      </div>

      <ul className="attention__list">
        {items.map((item) => {
          const Icon = kindIcons[item.kind];
          return (
            <li key={item.id}>
              <Link to={item.href} className="attention__item">
                <span className={cn("attention__item-icon", `is-${item.tone}`)} aria-hidden="true">
                  <Icon size={16} />
                </span>
                <span className="attention__item-text">
                  <span className="attention__item-title">{item.title}</span>
                  <span className="attention__item-meta">{item.meta}</span>
                </span>
                <ChevronLeft size={16} className="attention__chev" aria-hidden="true" />
              </Link>
            </li>
          );
        })}
      </ul>
    </section>
  );
}
