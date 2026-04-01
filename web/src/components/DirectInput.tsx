import './DirectInput.css'

interface Props {
  hue: number
  saturation: number
  value: number
  onChange: (h: number, s: number, v: number) => void
}

function clamp(n: number, min: number, max: number): number {
  return Math.max(min, Math.min(max, isNaN(n) ? min : n))
}

export default function DirectInput({ hue, saturation, value, onChange }: Props) {
  return (
    <div className="direct-input">
      <label className="direct-field">
        <span className="direct-label">H</span>
        <input
          type="number"
          min={0}
          max={359}
          value={hue}
          onChange={(e) => onChange(clamp(+e.target.value, 0, 359), saturation, value)}
        />
        <span className="direct-range">0–359</span>
      </label>
      <label className="direct-field">
        <span className="direct-label">S</span>
        <input
          type="number"
          min={0}
          max={100}
          value={saturation}
          onChange={(e) => onChange(hue, clamp(+e.target.value, 0, 100), value)}
        />
        <span className="direct-range">0–100</span>
      </label>
      <label className="direct-field">
        <span className="direct-label">V</span>
        <input
          type="number"
          min={0}
          max={100}
          value={value}
          onChange={(e) => onChange(hue, saturation, clamp(+e.target.value, 0, 100))}
        />
        <span className="direct-range">0–100</span>
      </label>
    </div>
  )
}
