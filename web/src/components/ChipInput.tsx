import { useId, useState, type KeyboardEvent } from 'react';
import { addChips } from '../lib/chips';
import { CloseIcon } from './icons';

interface Props {
  label: string;
  hint: string;
  placeholder: string;
  value: string[];
  onChange: (next: string[]) => void;
}

/**
 * A text box that turns what you type into removable chips on Enter or a
 * comma. Backspace in an empty box removes the last chip; anything still
 * typed when the box loses focus becomes a chip too, so nothing is lost.
 */
export function ChipInput({ label, hint, placeholder, value, onChange }: Props) {
  const [text, setText] = useState('');
  const id = useId();

  const commit = () => {
    if (text.trim() === '') return;
    onChange(addChips(value, text));
    setText('');
  };

  const onKeyDown = (e: KeyboardEvent<HTMLInputElement>) => {
    if (e.key === 'Enter' && (e.ctrlKey || e.metaKey)) {
      // Ctrl/Cmd+Enter submits the form; keep what's typed in the request.
      commit();
    } else if (e.key === 'Enter' || e.key === ',') {
      if (text.trim() !== '') {
        e.preventDefault();
        commit();
      } else if (e.key === ',') {
        e.preventDefault();
      }
    } else if (e.key === 'Backspace' && text === '' && value.length > 0) {
      onChange(value.slice(0, -1));
    }
  };

  return (
    <div className="field">
      <label className="field-label" htmlFor={id}>
        {label}
      </label>
      <p className="field-hint" id={`${id}-hint`}>
        {hint}
      </p>
      <div className="chip-box">
        {value.length > 0 && (
          <ul className="chips" aria-label={`${label}: ${value.length} added`}>
            {value.map((chip) => (
              <li key={chip} className="chip">
                <span>{chip}</span>
                <button
                  type="button"
                  className="chip-remove"
                  aria-label={`Remove ${chip}`}
                  onClick={() => onChange(value.filter((c) => c !== chip))}
                >
                  <CloseIcon />
                </button>
              </li>
            ))}
          </ul>
        )}
        <input
          id={id}
          className="chip-input"
          value={text}
          placeholder={value.length === 0 ? placeholder : 'Add another…'}
          aria-describedby={`${id}-hint`}
          onChange={(e) => {
            const next = e.target.value;
            // Pasting "a, b, c" adds all three at once.
            if (/[,;\n]/.test(next) && next.trim().length > 1) {
              onChange(addChips(value, next));
              setText('');
            } else {
              setText(next);
            }
          }}
          onKeyDown={onKeyDown}
          onBlur={commit}
        />
      </div>
    </div>
  );
}
