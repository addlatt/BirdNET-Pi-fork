import type { JSX } from 'preact';
import { useCallback } from 'preact/hooks';

// =============================================================================
// Common Styles
// =============================================================================

const inputBaseClass = `
  w-full py-2 px-3 border border-gray-300 dark:border-gray-600 rounded-lg
  bg-white dark:bg-gray-800 text-gray-900 dark:text-white
  placeholder-gray-500 dark:placeholder-gray-400
  focus:ring-2 focus:ring-primary-500 focus:border-transparent
  disabled:opacity-50 disabled:cursor-not-allowed
`;

const labelClass = `
  block text-sm font-medium text-gray-700 dark:text-gray-300 mb-1
`;

const helpTextClass = `
  text-xs text-gray-500 dark:text-gray-400 mt-1
`;

const errorClass = `
  text-xs text-red-500 dark:text-red-400 mt-1
`;

// =============================================================================
// TextInput
// =============================================================================

interface TextInputProps {
  label: string;
  value: string;
  onChange: (value: string) => void;
  placeholder?: string;
  helpText?: string;
  error?: string;
  disabled?: boolean;
  type?: 'text' | 'password' | 'email' | 'url';
  required?: boolean;
}

export function TextInput({
  label,
  value,
  onChange,
  placeholder,
  helpText,
  error,
  disabled = false,
  type = 'text',
  required = false,
}: TextInputProps): JSX.Element {
  const handleInput = useCallback(
    (e: Event) => {
      onChange((e.target as HTMLInputElement).value);
    },
    [onChange]
  );

  return (
    <div class="mb-4">
      <label class={labelClass}>
        {label}
        {required && <span class="text-red-500 ml-1">*</span>}
      </label>
      <input
        type={type}
        value={value}
        onInput={handleInput}
        placeholder={placeholder}
        disabled={disabled}
        class={`${inputBaseClass} ${error ? 'border-red-500' : ''}`}
      />
      {helpText && !error && <p class={helpTextClass}>{helpText}</p>}
      {error && <p class={errorClass}>{error}</p>}
    </div>
  );
}

// =============================================================================
// NumberInput
// =============================================================================

interface NumberInputProps {
  label: string;
  value: number;
  onChange: (value: number) => void;
  min?: number;
  max?: number;
  step?: number;
  helpText?: string;
  error?: string;
  disabled?: boolean;
  required?: boolean;
}

export function NumberInput({
  label,
  value,
  onChange,
  min,
  max,
  step = 1,
  helpText,
  error,
  disabled = false,
  required = false,
}: NumberInputProps): JSX.Element {
  const handleInput = useCallback(
    (e: Event) => {
      const newValue = parseFloat((e.target as HTMLInputElement).value);
      if (!isNaN(newValue)) {
        onChange(newValue);
      }
    },
    [onChange]
  );

  return (
    <div class="mb-4">
      <label class={labelClass}>
        {label}
        {required && <span class="text-red-500 ml-1">*</span>}
      </label>
      <input
        type="number"
        value={value}
        onInput={handleInput}
        min={min}
        max={max}
        step={step}
        disabled={disabled}
        class={`${inputBaseClass} ${error ? 'border-red-500' : ''}`}
      />
      {helpText && !error && <p class={helpTextClass}>{helpText}</p>}
      {error && <p class={errorClass}>{error}</p>}
    </div>
  );
}

// =============================================================================
// SliderInput
// =============================================================================

interface SliderInputProps {
  label: string;
  value: number;
  onChange: (value: number) => void;
  min: number;
  max: number;
  step?: number;
  helpText?: string;
  error?: string;
  disabled?: boolean;
  formatValue?: (value: number) => string;
}

export function SliderInput({
  label,
  value,
  onChange,
  min,
  max,
  step = 1,
  helpText,
  error,
  disabled = false,
  formatValue = (v) => String(v),
}: SliderInputProps): JSX.Element {
  const handleInput = useCallback(
    (e: Event) => {
      onChange(parseFloat((e.target as HTMLInputElement).value));
    },
    [onChange]
  );

  return (
    <div class="mb-4">
      <div class="flex justify-between items-center mb-1">
        <label class={labelClass.replace('mb-1', '')}>{label}</label>
        <span class="text-sm font-medium text-primary-600 dark:text-primary-400">
          {formatValue(value)}
        </span>
      </div>
      <input
        type="range"
        value={value}
        onInput={handleInput}
        min={min}
        max={max}
        step={step}
        disabled={disabled}
        class="w-full h-2 bg-gray-200 dark:bg-gray-700 rounded-lg appearance-none cursor-pointer
               disabled:opacity-50 disabled:cursor-not-allowed"
      />
      <div class="flex justify-between text-xs text-gray-500 dark:text-gray-400 mt-1">
        <span>{formatValue(min)}</span>
        <span>{formatValue(max)}</span>
      </div>
      {helpText && !error && <p class={helpTextClass}>{helpText}</p>}
      {error && <p class={errorClass}>{error}</p>}
    </div>
  );
}

// =============================================================================
// SelectInput
// =============================================================================

interface SelectOption {
  value: string | number;
  label: string;
}

interface SelectInputProps {
  label: string;
  value: string | number;
  onChange: (value: string) => void;
  options: SelectOption[];
  helpText?: string;
  error?: string;
  disabled?: boolean;
  required?: boolean;
}

export function SelectInput({
  label,
  value,
  onChange,
  options,
  helpText,
  error,
  disabled = false,
  required = false,
}: SelectInputProps): JSX.Element {
  const handleChange = useCallback(
    (e: Event) => {
      onChange((e.target as HTMLSelectElement).value);
    },
    [onChange]
  );

  return (
    <div class="mb-4">
      <label class={labelClass}>
        {label}
        {required && <span class="text-red-500 ml-1">*</span>}
      </label>
      <select
        value={value}
        onChange={handleChange}
        disabled={disabled}
        class={`${inputBaseClass} ${error ? 'border-red-500' : ''}`}
      >
        {options.map((opt) => (
          <option key={opt.value} value={opt.value}>
            {opt.label}
          </option>
        ))}
      </select>
      {helpText && !error && <p class={helpTextClass}>{helpText}</p>}
      {error && <p class={errorClass}>{error}</p>}
    </div>
  );
}

// =============================================================================
// TextAreaInput
// =============================================================================

interface TextAreaInputProps {
  label: string;
  value: string;
  onChange: (value: string) => void;
  placeholder?: string;
  rows?: number;
  helpText?: string;
  error?: string;
  disabled?: boolean;
}

export function TextAreaInput({
  label,
  value,
  onChange,
  placeholder,
  rows = 4,
  helpText,
  error,
  disabled = false,
}: TextAreaInputProps): JSX.Element {
  const handleInput = useCallback(
    (e: Event) => {
      onChange((e.target as HTMLTextAreaElement).value);
    },
    [onChange]
  );

  return (
    <div class="mb-4">
      <label class={labelClass}>{label}</label>
      <textarea
        value={value}
        onInput={handleInput}
        placeholder={placeholder}
        rows={rows}
        disabled={disabled}
        class={`${inputBaseClass} resize-y ${error ? 'border-red-500' : ''}`}
      />
      {helpText && !error && <p class={helpTextClass}>{helpText}</p>}
      {error && <p class={errorClass}>{error}</p>}
    </div>
  );
}

// =============================================================================
// ToggleInput
// =============================================================================

interface ToggleInputProps {
  label: string;
  value: boolean;
  onChange: (value: boolean) => void;
  helpText?: string;
  disabled?: boolean;
}

export function ToggleInput({
  label,
  value,
  onChange,
  helpText,
  disabled = false,
}: ToggleInputProps): JSX.Element {
  const handleChange = useCallback(() => {
    if (!disabled) {
      onChange(!value);
    }
  }, [value, onChange, disabled]);

  return (
    <div class="mb-4">
      <div class="flex items-center justify-between">
        <div>
          <label class={labelClass.replace('mb-1', '')}>{label}</label>
          {helpText && <p class={helpTextClass}>{helpText}</p>}
        </div>
        <button
          type="button"
          onClick={handleChange}
          disabled={disabled}
          class={`
            relative inline-flex h-6 w-11 flex-shrink-0 cursor-pointer rounded-full border-2 border-transparent
            transition-colors duration-200 ease-in-out focus:outline-none focus:ring-2 focus:ring-primary-500 focus:ring-offset-2
            ${value ? 'bg-primary-600' : 'bg-gray-200 dark:bg-gray-700'}
            ${disabled ? 'opacity-50 cursor-not-allowed' : ''}
          `}
          role="switch"
          aria-checked={value}
        >
          <span
            class={`
              pointer-events-none inline-block h-5 w-5 transform rounded-full bg-white shadow ring-0
              transition duration-200 ease-in-out
              ${value ? 'translate-x-5' : 'translate-x-0'}
            `}
          />
        </button>
      </div>
    </div>
  );
}

// =============================================================================
// CheckboxInput (for 0/1 integer flags)
// =============================================================================

interface CheckboxInputProps {
  label: string;
  value: number;
  onChange: (value: number) => void;
  helpText?: string;
  disabled?: boolean;
}

export function CheckboxInput({
  label,
  value,
  onChange,
  helpText,
  disabled = false,
}: CheckboxInputProps): JSX.Element {
  const handleChange = useCallback(
    (e: Event) => {
      onChange((e.target as HTMLInputElement).checked ? 1 : 0);
    },
    [onChange]
  );

  return (
    <div class="mb-4">
      <label class="flex items-start gap-3 cursor-pointer">
        <input
          type="checkbox"
          checked={value === 1}
          onChange={handleChange}
          disabled={disabled}
          class="mt-0.5 h-4 w-4 rounded border-gray-300 dark:border-gray-600
                 text-primary-600 focus:ring-primary-500
                 disabled:opacity-50 disabled:cursor-not-allowed"
        />
        <div>
          <span class="text-sm font-medium text-gray-700 dark:text-gray-300">{label}</span>
          {helpText && <p class={helpTextClass}>{helpText}</p>}
        </div>
      </label>
    </div>
  );
}

// =============================================================================
// FormSection
// =============================================================================

interface FormSectionProps {
  title: string;
  description?: string;
  children: JSX.Element | JSX.Element[];
}

export function FormSection({ title, description, children }: FormSectionProps): JSX.Element {
  return (
    <div class="bg-white dark:bg-gray-800 rounded-lg shadow-sm border border-gray-200 dark:border-gray-700 p-6 mb-6">
      <h3 class="text-lg font-semibold text-gray-900 dark:text-white mb-1">{title}</h3>
      {description && <p class="text-sm text-gray-500 dark:text-gray-400 mb-4">{description}</p>}
      <div class="space-y-4">{children}</div>
    </div>
  );
}

// =============================================================================
// SaveButton
// =============================================================================

interface SaveButtonProps {
  onClick: () => void;
  saving: boolean;
  disabled?: boolean;
  text?: string;
  savingText?: string;
}

export function SaveButton({
  onClick,
  saving,
  disabled = false,
  text = 'Save Settings',
  savingText = 'Saving...',
}: SaveButtonProps): JSX.Element {
  return (
    <button
      type="button"
      onClick={onClick}
      disabled={disabled || saving}
      class={`
        inline-flex items-center justify-center px-6 py-3 rounded-lg font-medium
        transition-colors duration-200
        ${
          disabled || saving
            ? 'bg-gray-300 dark:bg-gray-700 text-gray-500 dark:text-gray-400 cursor-not-allowed'
            : 'bg-primary-600 hover:bg-primary-700 text-white'
        }
      `}
    >
      {saving && (
        <svg
          class="animate-spin -ml-1 mr-2 h-4 w-4"
          fill="none"
          viewBox="0 0 24 24"
        >
          <circle
            class="opacity-25"
            cx="12"
            cy="12"
            r="10"
            stroke="currentColor"
            stroke-width="4"
          />
          <path
            class="opacity-75"
            fill="currentColor"
            d="M4 12a8 8 0 018-8V0C5.373 0 0 5.373 0 12h4zm2 5.291A7.962 7.962 0 014 12H0c0 3.042 1.135 5.824 3 7.938l3-2.647z"
          />
        </svg>
      )}
      {saving ? savingText : text}
    </button>
  );
}

// =============================================================================
// AlertMessage
// =============================================================================

interface AlertMessageProps {
  type: 'success' | 'error' | 'warning' | 'info';
  message: string;
  onDismiss?: () => void;
}

export function AlertMessage({ type, message, onDismiss }: AlertMessageProps): JSX.Element {
  const colors = {
    success: 'bg-green-50 dark:bg-green-900/30 border-green-400 text-green-700 dark:text-green-300',
    error: 'bg-red-50 dark:bg-red-900/30 border-red-400 text-red-700 dark:text-red-300',
    warning: 'bg-yellow-50 dark:bg-yellow-900/30 border-yellow-400 text-yellow-700 dark:text-yellow-300',
    info: 'bg-blue-50 dark:bg-blue-900/30 border-blue-400 text-blue-700 dark:text-blue-300',
  };

  return (
    <div class={`${colors[type]} border rounded-lg p-4 mb-4 flex items-start justify-between`}>
      <p class="text-sm">{message}</p>
      {onDismiss && (
        <button
          type="button"
          onClick={onDismiss}
          class="ml-4 text-current opacity-50 hover:opacity-100"
        >
          <svg class="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24">
            <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M6 18L18 6M6 6l12 12" />
          </svg>
        </button>
      )}
    </div>
  );
}
