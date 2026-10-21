import { Search, X } from "lucide-react";
import { cn } from "@/lib/cn";
import "./SearchInput.css";

export interface SearchInputProps {
  value: string;
  onChange: (value: string) => void;
  placeholder?: string;
  ariaLabel: string;
  className?: string;
}

export function SearchInput({
  value,
  onChange,
  placeholder = "جست‌وجو…",
  ariaLabel,
  className,
}: SearchInputProps) {
  return (
    <div className={cn("search-field", className)}>
      <Search className="search-field__icon" size={18} aria-hidden="true" />
      <input
        type="search"
        className="search-field__input"
        value={value}
        onChange={(e) => onChange(e.target.value)}
        placeholder={placeholder}
        aria-label={ariaLabel}
        enterKeyHint="search"
      />
      {value ? (
        <button
          type="button"
          className="search-field__clear"
          onClick={() => onChange("")}
          aria-label="پاک کردن جست‌وجو"
        >
          <X size={15} aria-hidden="true" />
        </button>
      ) : null}
    </div>
  );
}
