import { describe, it, expect } from 'vitest'
import { hsvToRgb, rgbToHex, hsvToHex, hsvDistance } from './color'

describe('hsvToRgb', () => {
  it('converts pure red', () => {
    expect(hsvToRgb(0, 100, 100)).toEqual([255, 0, 0])
  })
  it('converts pure green', () => {
    expect(hsvToRgb(120, 100, 100)).toEqual([0, 255, 0])
  })
  it('converts pure blue', () => {
    expect(hsvToRgb(240, 100, 100)).toEqual([0, 0, 255])
  })
  it('converts white', () => {
    expect(hsvToRgb(0, 0, 100)).toEqual([255, 255, 255])
  })
  it('converts black', () => {
    expect(hsvToRgb(0, 0, 0)).toEqual([0, 0, 0])
  })
  it('converts 50% gray', () => {
    expect(hsvToRgb(0, 0, 50)).toEqual([128, 128, 128])
  })
})

describe('rgbToHex', () => {
  it('converts red', () => {
    expect(rgbToHex(255, 0, 0)).toBe('#ff0000')
  })
  it('converts black', () => {
    expect(rgbToHex(0, 0, 0)).toBe('#000000')
  })
  it('pads single digits', () => {
    expect(rgbToHex(1, 2, 3)).toBe('#010203')
  })
})

describe('hsvToHex', () => {
  it('converts red', () => {
    expect(hsvToHex(0, 100, 100)).toBe('#ff0000')
  })
})

describe('hsvDistance', () => {
  it('returns 0 for identical colors', () => {
    expect(hsvDistance(180, 50, 50, 180, 50, 50)).toBe(0)
  })
  it('handles circular hue (wrapping)', () => {
    const d1 = hsvDistance(5, 50, 50, 355, 50, 50)
    const d2 = hsvDistance(5, 50, 50, 15, 50, 50)
    expect(Math.abs(d1 - d2)).toBeLessThan(0.001)
  })
  it('computes max distance correctly', () => {
    const d = hsvDistance(0, 0, 0, 180, 100, 100)
    const expected = Math.sqrt(100 * 100 + 100 * 100 + 100 * 100)
    expect(Math.abs(d - expected)).toBeLessThan(0.001)
  })
})
