import { useEffect, useRef, useState } from 'react';
import { Search, X } from 'lucide-react';
import { cn } from '@/lib/cn';

export interface SearchBarProps {
  value: string;
  onChange: (value: string) => void;
  placeholder?: string;
  autoFocus?: boolean;
  onSubmit?: () => void;
  className?: string;
}

/** Inline search field with a clear affordance and a debounce-friendly API. */
export function SearchBar({ value, onChange, placeholder = 'جست‌وجو', autoFocus, onSubmit, className }: SearchBarProps) {
  const [local, setLocal] = useState(value);
  const ref = useRef<HTMLInputElement>(null);

  useEffect(() => setLocal(value), [value]);

  return (
    <form
      role="search"
      onSubmit={(e) => {
        e.preventDefault();
        onSubmit?.();
      }}
      className={cn('relative', className)}
    >
      <Search className="pointer-events-none absolute inset-y-0 start-3.5 my-auto size-[18px] text-ink-3" aria-hidden />
      <input
        ref={ref}
        type="search"
        value={local}
        autoFocus={autoFocus}
        onChange={(e) => {
          setLocal(e.target.value);
          onChange(e.target.value);
        }}
        placeholder={placeholder}
        aria-label={placeholder}
        className="h-11 w-full rounded-md border border-line bg-surface ps-11 pe-10 text-base text-ink placeholder:text-ink-3 focus:border-brand focus:outline-none focus:ring-2 focus:ring-brand/20 [&::-webkit-search-cancel-button]:hidden"
      />
      {local.length > 0 && (
        <button
          type="button"
          aria-label="پاک کردن جست‌وجو"
          onClick={() => {
            setLocal('');
            onChange('');
            ref.current?.focus();
          }}
          className="press absolute inset-y-0 end-2 my-auto grid size-8 place-items-center rounded-sm text-ink-3 hover:bg-sunken"
        >
          <X className="size-4" />
        </button>
      )}
    </form>
  );
}
