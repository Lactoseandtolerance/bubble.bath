import { useRef, useCallback } from 'react'
import './DigitInput.css'

interface Props {
  value: string // "00"–"99" as string, or partial
  onChange: (value: string) => void
}

export default function DigitInput({ value, onChange }: Props) {
  const d1Ref = useRef<HTMLInputElement>(null)
  const d2Ref = useRef<HTMLInputElement>(null)

  const d1 = value.length > 0 ? value[0] : ''
  const d2 = value.length > 1 ? value[1] : ''

  const handleD1 = useCallback((e: React.ChangeEvent<HTMLInputElement>) => {
    const v = e.target.value.replace(/\D/g, '').slice(-1)
    if (v) {
      onChange(v + d2)
      d2Ref.current?.focus()
    } else {
      onChange('')
    }
  }, [d2, onChange])

  const handleD2 = useCallback((e: React.ChangeEvent<HTMLInputElement>) => {
    const v = e.target.value.replace(/\D/g, '').slice(-1)
    onChange(d1 + v)
  }, [d1, onChange])

  const handleD2KeyDown = useCallback((e: React.KeyboardEvent) => {
    if (e.key === 'Backspace' && !d2) {
      d1Ref.current?.focus()
    }
  }, [d2])

  return (
    <div className="digit-input">
      <input
        ref={d1Ref}
        className="digit-box"
        type="text"
        inputMode="numeric"
        maxLength={1}
        value={d1}
        onChange={handleD1}
        placeholder="0"
        autoComplete="off"
      />
      <input
        ref={d2Ref}
        className="digit-box"
        type="text"
        inputMode="numeric"
        maxLength={1}
        value={d2}
        onChange={handleD2}
        onKeyDown={handleD2KeyDown}
        placeholder="0"
        autoComplete="off"
      />
    </div>
  )
}
