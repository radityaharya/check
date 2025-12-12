import { useStore } from '@tanstack/react-form';
import { useFieldContext, useFormContext } from '@/hooks/form-context';
import { Input } from '@/components/ui/input';
import { Textarea } from '@/components/ui/textarea';
import { Button } from '@/components/ui/button';
import { Label } from '@/components/ui/label';
import { Spinner } from '@/components/ui/spinner';
import { cn } from '@/lib/utils';
import type { Tag } from '@/types';

function FieldError({ errors }: { errors: Array<string | { message: string }> }) {
  if (!errors.length) return null;
  return (
    <div className="text-terminal-red text-[10px] mt-1">
      {errors.map((error, i) => (
        <div key={i}>{typeof error === 'string' ? error : error.message}</div>
      ))}
    </div>
  );
}

interface TextFieldProps {
  label: string;
  placeholder?: string;
  type?: 'text' | 'url' | 'password';
  required?: boolean;
  className?: string;
}

export function TextField({
  label,
  placeholder,
  type = 'text',
  required,
  className,
}: TextFieldProps) {
  const field = useFieldContext<string>();
  const errors = useStore(field.store, (state) => state.meta.errors);

  return (
    <div className={className}>
      <Label className="block text-[10px] uppercase text-terminal-muted tracking-widest mb-2">
        {label}
        {required && <span className="text-terminal-red ml-1">*</span>}
      </Label>
      <Input
        type={type}
        value={field.state.value ?? ''}
        placeholder={placeholder}
        onBlur={field.handleBlur}
        onChange={(e) => field.handleChange(e.target.value)}
        className="w-full bg-terminal-surface border-terminal-border text-terminal-text"
      />
      {field.state.meta.isTouched && <FieldError errors={errors} />}
    </div>
  );
}

interface NumberFieldProps {
  label: string;
  min?: number;
  max?: number;
  required?: boolean;
  className?: string;
}

export function NumberField({
  label,
  min,
  max,
  required,
  className,
}: NumberFieldProps) {
  const field = useFieldContext<number>();
  const errors = useStore(field.store, (state) => state.meta.errors);

  return (
    <div className={className}>
      <Label className="block text-[10px] uppercase text-terminal-muted tracking-widest mb-2">
        {label}
        {required && <span className="text-terminal-red ml-1">*</span>}
      </Label>
      <Input
        type="number"
        value={field.state.value ?? 0}
        min={min}
        max={max}
        onBlur={field.handleBlur}
        onChange={(e) => field.handleChange(Number(e.target.value))}
        className="w-full bg-terminal-surface border-terminal-border text-terminal-text"
      />
      {field.state.meta.isTouched && <FieldError errors={errors} />}
    </div>
  );
}

interface SelectFieldProps {
  label: string;
  options: { value: string; label: string }[];
  required?: boolean;
  className?: string;
  placeholder?: string;
}

export function SelectField({
  label,
  options,
  required,
  className,
  placeholder,
}: SelectFieldProps) {
  const field = useFieldContext<string>();
  const errors = useStore(field.store, (state) => state.meta.errors);

  return (
    <div className={className}>
      <Label className="block text-[10px] uppercase text-terminal-muted tracking-widest mb-2">
        {label}
        {required && <span className="text-terminal-red ml-1">*</span>}
      </Label>
      <select
        value={field.state.value ?? ''}
        onBlur={field.handleBlur}
        onChange={(e) => field.handleChange(e.target.value)}
        className="w-full bg-terminal-surface border border-terminal-border text-terminal-text px-4 py-2 rounded focus:border-terminal-green outline-none transition font-mono"
      >
        {placeholder && <option value="">{placeholder}</option>}
        {options.map((opt) => (
          <option key={opt.value} value={opt.value}>
            {opt.label}
          </option>
        ))}
      </select>
      {field.state.meta.isTouched && <FieldError errors={errors} />}
    </div>
  );
}

interface TextAreaFieldProps {
  label: string;
  placeholder?: string;
  rows?: number;
  required?: boolean;
  className?: string;
}

export function TextAreaField({
  label,
  placeholder,
  rows = 3,
  required,
  className,
}: TextAreaFieldProps) {
  const field = useFieldContext<string>();
  const errors = useStore(field.store, (state) => state.meta.errors);

  return (
    <div className={className}>
      <Label className="block text-[10px] uppercase text-terminal-muted tracking-widest mb-2">
        {label}
        {required && <span className="text-terminal-red ml-1">*</span>}
      </Label>
      <Textarea
        value={field.state.value ?? ''}
        placeholder={placeholder}
        rows={rows}
        onBlur={field.handleBlur}
        onChange={(e) => field.handleChange(e.target.value)}
        className="w-full bg-terminal-surface border-terminal-border text-terminal-text font-mono text-sm"
      />
      {field.state.meta.isTouched && <FieldError errors={errors} />}
    </div>
  );
}

interface CheckboxFieldProps {
  label: string;
  className?: string;
}

export function CheckboxField({ label, className }: CheckboxFieldProps) {
  const field = useFieldContext<boolean>();

  return (
    <div className={cn('flex items-center gap-3', className)}>
      <input
        type="checkbox"
        checked={field.state.value ?? false}
        onChange={(e) => field.handleChange(e.target.checked)}
        id={label}
        className="w-4 h-4 rounded bg-terminal-bg border-terminal-border text-terminal-green focus:ring-terminal-green"
      />
      <label htmlFor={label} className="text-sm cursor-pointer">
        {label}
      </label>
    </div>
  );
}

interface TagsFieldProps {
  label: string;
  tags: Tag[];
  className?: string;
}

export function TagsField({ label, tags, className }: TagsFieldProps) {
  const field = useFieldContext<number[]>();

  const toggleTag = (tagId: number) => {
    const currentTags = field.state.value ?? [];
    if (currentTags.includes(tagId)) {
      field.handleChange(currentTags.filter((id) => id !== tagId));
    } else {
      field.handleChange([...currentTags, tagId]);
    }
  };

  return (
    <div className={className}>
      <Label className="block text-[10px] uppercase text-terminal-muted tracking-widest mb-2">
        {label}
      </Label>
      <div className="flex flex-wrap gap-2 p-3 bg-terminal-surface border border-terminal-border rounded min-h-[50px]">
        {tags.map((tag) => {
          const selected = (field.state.value ?? []).includes(tag.id);
          return (
            <button
              key={tag.id}
              type="button"
              onClick={() => toggleTag(tag.id)}
              className={cn(
                'text-[10px] px-2 py-1 rounded transition',
                !selected && 'opacity-40 hover:opacity-70'
              )}
              style={{
                background: `${tag.color}30`,
                color: tag.color,
                border: `1px solid ${selected ? tag.color : 'transparent'}`,
              }}
            >
              {tag.name}
            </button>
          );
        })}
        {tags.length === 0 && (
          <span className="text-terminal-muted text-xs">No tags created</span>
        )}
      </div>
    </div>
  );
}

interface SubmitButtonProps {
  label: string;
  className?: string;
}

export function SubmitButton({ label, className }: SubmitButtonProps) {
  const form = useFormContext();
  return (
    <form.Subscribe selector={(state) => state.isSubmitting}>
      {(isSubmitting) => (
        <Button
          type="submit"
          disabled={isSubmitting}
          className={cn(
            'bg-terminal-green text-terminal-bg font-bold hover:opacity-90',
            className
          )}
        >
          {isSubmitting ? <Spinner size="sm" /> : label}
        </Button>
      )}
    </form.Subscribe>
  );
}
