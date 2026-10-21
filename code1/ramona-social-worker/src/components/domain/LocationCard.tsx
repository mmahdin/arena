import { Building2, Compass, Info, MapPin, Navigation } from "lucide-react";
import type { Address } from "@/types";
import { useToast } from "@/components/ui/Toast";
import "./LocationCard.css";

/**
 * Address + a lightweight, self-contained map illustration. The map is a
 * decorative placeholder: the production app will swap this canvas for the
 * mapping SDK without changing the surrounding layout.
 */
export function LocationCard({ address, title }: { address: Address; title?: string }) {
  const toast = useToast();

  return (
    <section className="location-card" aria-label="موقعیت بازدید">
      <div className="location-map" aria-hidden="true">
        <svg viewBox="0 0 400 132" preserveAspectRatio="none" role="presentation">
          <rect width="400" height="132" fill="#eef0ea" />
          <g stroke="#e2e5dd" strokeWidth="10" strokeLinecap="round">
            <path d="M-20 38 H420" />
            <path d="M-20 96 H420" />
            <path d="M96 -10 V150" />
            <path d="M268 -10 V150" />
          </g>
          <g fill="#e6e9e2">
            <rect x="16" y="8" width="56" height="20" rx="4" />
            <rect x="120" y="52" width="68" height="30" rx="4" />
            <rect x="212" y="10" width="42" height="18" rx="4" />
            <rect x="296" y="52" width="72" height="32" rx="4" />
            <rect x="30" y="106" width="52" height="18" rx="4" />
            <rect x="120" y="106" width="60" height="18" rx="4" />
            <rect x="300" y="104" width="70" height="20" rx="4" />
          </g>
          <g fill="#dfe4db">
            <circle cx="76" cy="68" r="9" />
            <circle cx="316" cy="32" r="7" />
            <circle cx="196" cy="112" r="6" />
          </g>
          <path
            d="M40 122 C 110 118, 150 92, 196 78 S 300 46, 372 34"
            fill="none"
            stroke="#17715c"
            strokeWidth="2.5"
            strokeDasharray="5 6"
            strokeLinecap="round"
            opacity="0.5"
          />
        </svg>
        <span className="location-map__pin">
          <span className="location-map__pulse" />
          <MapPin size={17} aria-hidden="true" />
        </span>
        <span className="location-map__badge">
          <Compass size={12} aria-hidden="true" />
          {address.district}
        </span>
      </div>

      <div className="location-card__body">
        <div className="location-card__address">
          <span className="location-card__icon" aria-hidden="true">
            <Building2 size={17} />
          </span>
          <div>
            {title ? <p className="location-card__title">{title}</p> : null}
            <p className="location-card__line">{address.line}</p>
            <p className="location-card__city">{address.city} · {address.district}</p>
          </div>
        </div>

        {address.accessNote ? (
          <p className="location-card__note">
            <Info size={14} aria-hidden="true" />
            <span>{address.accessNote}</span>
          </p>
        ) : null}

        <button
          type="button"
          className="location-card__route"
          onClick={() =>
            toast.show({
              message: "مسیریابی در نسخه نمایشی به نقشه متصل نیست.",
              tone: "default",
            })
          }
        >
          <Navigation size={15} aria-hidden="true" />
          نمایش مسیر تا این آدرس
        </button>
      </div>
    </section>
  );
}
