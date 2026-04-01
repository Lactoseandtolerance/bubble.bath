import { hsvToHex } from '../utils/color'
import HueBar from './HueBar'
import SatValSquare from './SatValSquare'
import './ColorPicker.css'

export interface HSV {
  h: number
  s: number
  v: number
}

interface Props {
  hsv: HSV
  onChange: (hsv: HSV) => void
}

export default function ColorPicker({ hsv, onChange }: Props) {
  const hex = hsvToHex(hsv.h, hsv.s, hsv.v)

  return (
    <div className="color-picker">
      <SatValSquare
        hue={hsv.h}
        saturation={hsv.s}
        value={hsv.v}
        onChange={(s, v) => onChange({ ...hsv, s, v })}
      />
      <HueBar
        hue={hsv.h}
        onChange={(h) => onChange({ ...hsv, h })}
      />
      <div className="color-preview">
        <div className="color-swatch" style={{ backgroundColor: hex }} />
        <div className="color-info">
          <span className="color-hex">{hex}</span>
          <span className="color-hsv">H:{hsv.h} S:{hsv.s} V:{hsv.v}</span>
        </div>
      </div>
    </div>
  )
}
