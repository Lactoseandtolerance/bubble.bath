/** Convert HSV (H:0–360, S:0–100, V:0–100) to RGB (0–255 each). */
export function hsvToRgb(h: number, s: number, v: number): [number, number, number] {
  const s01 = s / 100
  const v01 = v / 100
  const c = v01 * s01
  const x = c * (1 - Math.abs(((h / 60) % 2) - 1))
  const m = v01 - c

  let r = 0, g = 0, b = 0
  if (h < 60)       { r = c; g = x }
  else if (h < 120) { r = x; g = c }
  else if (h < 180) { g = c; b = x }
  else if (h < 240) { g = x; b = c }
  else if (h < 300) { r = x; b = c }
  else              { r = c; b = x }

  return [
    Math.round((r + m) * 255),
    Math.round((g + m) * 255),
    Math.round((b + m) * 255),
  ]
}

/** Convert RGB (0–255 each) to hex string like "#ff0000". */
export function rgbToHex(r: number, g: number, b: number): string {
  return '#' + [r, g, b].map(c => c.toString(16).padStart(2, '0')).join('')
}

/** Convert HSV directly to hex string. */
export function hsvToHex(h: number, s: number, v: number): string {
  const [r, g, b] = hsvToRgb(h, s, v)
  return rgbToHex(r, g, b)
}

/** HSV Euclidean distance with circular hue — mirrors Go backend Distance(). */
export function hsvDistance(h1: number, s1: number, v1: number, h2: number, s2: number, v2: number): number {
  let hd = Math.abs(h1 - h2)
  if (hd > 180) hd = 360 - hd
  hd *= 100 / 180
  const sd = s1 - s2
  const vd = v1 - v2
  return Math.sqrt(hd * hd + sd * sd + vd * vd)
}
