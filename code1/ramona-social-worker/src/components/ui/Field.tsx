import type {
  InputHTMLAttributes,
  ReactNode,
  SelectHTMLAttributes,
  TextareaHTMLAttributes,
} from "react";
import { ChevronDown } from "lucide-react";
import { cn } from "@/lib/cn";
import "./Field.css";

export interface FieldProps {
  label: string;
  htmlFor?: string;
  hint?: string;
  error?: string;
  required?: boolean;
  counter?: string;
  children: ReactNode;
  className?: string;
}

export function Field({
  label,
  htmlFor,
  hint,
  error,
  required,
  counter,
  children,
  className,
}: FieldProps) {
  return (
    <div className={cn("field", error && "field--error", className)}>
      <div className="field__head">
        <label className="field__label" htmlFor={htmlFor}>
          {label}
          {required ? <span className="field__required" aria-hidden="true">*</span> : null}
        </label>
        {counter ? <span className="field__counter num">{counter}</span> : null}
      </div>
      {children}
      {error ? (
        <p className="field__error">{error}</p>
      ) : hint ? (
        <p className="field__hint">{hint}</p>
      ) : null}
    </div>
  );
}

export function Input({ className, ...rest }: InputHTMLAttributes<HTMLInputElement>) {
  return <input className={cn("input", className)} {...rest} />;
}

export function Textarea({ className, ...rest }: TextareaHTMLAttributes<HTMLTextAreaElement>) {
  return <textarea className={cn("textarea", className)} {...rest} />;
}

export function Select({ className, children, ...rest }: SelectHTMLAttributes<HTMLSelectElement>) {
  return (
    <div className="select-wrap">
      <select className={cn("select", className)} {...rest}>
        {children}
      </select>
      <ChevronDown className="select-wrap__chevron" size={16} aria-hidden="true" />
    </div>
  );
}
