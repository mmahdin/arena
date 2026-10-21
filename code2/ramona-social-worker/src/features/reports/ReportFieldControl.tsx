import type { ReportField } from '@/types';
import { ChecklistRow } from '@/components/ui/Checklist';
import { ScaleInput } from '@/components/ui/ScaleInput';
import { SelectInput, TextArea, TextInput } from '@/components/ui/Field';

/**
 * Generic field renderer.
 *
 * The editor knows nothing about "home assessment" or "monitoring" — it renders
 * whatever fields the report carries. New assessment forms are added as data
 * (see `src/data/mock/report-templates.ts`) and appear here automatically.
 */
export function ReportFieldControl({
  field,
  disabled,
  onChange,
}: {
  field: ReportField;
  disabled?: boolean;
  onChange: (value: string | number | string[]) => void;
}) {
  switch (field.kind) {
    case 'text':
      return field.multiline ? (
        <TextArea
          label={field.label}
          hint={field.hint}
          rows={4}
          disabled={disabled}
          value={field.value}
          onChange={(e) => onChange(e.target.value)}
          placeholder="توضیحات خود را بنویسید…"
        />
      ) : (
        <TextInput
          label={field.label}
          hint={field.hint}
          disabled={disabled}
          value={field.value}
          onChange={(e) => onChange(e.target.value)}
        />
      );

    case 'select':
      return (
        <SelectInput
          label={field.label}
          value={field.value}
          options={field.options}
          onChange={onChange}
          placeholder="انتخاب کنید…"
        />
      );

    case 'scale':
      return (
        <div className="space-y-2">
          <p className="text-sm font-semibold text-ink">{field.label}</p>
          <ScaleInput
            value={field.value}
            max={field.max}
            lowLabel={field.lowLabel}
            highLabel={field.highLabel}
            disabled={disabled}
            onChange={onChange}
          />
        </div>
      );

    case 'checklist':
      return (
        <div className="space-y-1">
          <p className="text-sm font-semibold text-ink">{field.label}</p>
          <div className="divide-y divide-line-soft">
            {field.options.map((option) => (
              <ChecklistRow
                key={option}
                label={option}
                disabled={disabled}
                checked={field.value.includes(option)}
                onChange={() =>
                  onChange(
                    field.value.includes(option)
                      ? field.value.filter((v) => v !== option)
                      : [...field.value, option],
                  )
                }
              />
            ))}
          </div>
        </div>
      );
  }
}
