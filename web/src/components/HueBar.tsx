import { useRef, useEffect, useCallback } from 'react'
import { hsvToRgb } from '../utils/color'
import './HueBar.css'

interface Props {
  hue: number
  onChange: (hue: number) => void
}

const WIDTH = 360
const HEIGHT = 24

export default function HueBar({ hue, onChange }: Props) {
  const canvasRef = useRef<HTMLCanvasElement>(null)
  const dragging = useRef(false)

  const draw = useCallback(() => {
    const canvas = canvasRef.current
    if (!canvas) return
    const ctx = canvas.getContext('2d')
    if (!ctx) return

    // Draw hue spectrum
    for (let x = 0; x < WIDTH; x++) {
      const h = Math.round((x / WIDTH) * 360)
      const [r, g, b] = hsvToRgb(h, 100, 100)
      ctx.fillStyle = `rgb(${r},${g},${b})`
      ctx.fillRect(x, 0, 1, HEIGHT)
    }

    // Selector indicator
    const sx = (hue / 360) * WIDTH
    ctx.strokeStyle = '#fff'
    ctx.lineWidth = 2
    ctx.strokeRect(sx - 3, 1, 6, HEIGHT - 2)
    ctx.strokeStyle = '#000'
    ctx.lineWidth = 1
    ctx.strokeRect(sx - 4, 0, 8, HEIGHT)
  }, [hue])

  useEffect(() => { draw() }, [draw])

  const handlePointer = useCallback((e: React.PointerEvent) => {
    const canvas = canvasRef.current
    if (!canvas) return
    const rect = canvas.getBoundingClientRect()
    const x = Math.max(0, Math.min(e.clientX - rect.left, rect.width))
    onChange(Math.min(360, Math.max(0, Math.round((x / rect.width) * 360))))
  }, [onChange])

  return (
    <canvas
      ref={canvasRef}
      className="hue-bar"
      width={WIDTH}
      height={HEIGHT}
      onPointerDown={(e) => {
        dragging.current = true
        e.currentTarget.setPointerCapture(e.pointerId)
        handlePointer(e)
      }}
      onPointerMove={(e) => { if (dragging.current) handlePointer(e) }}
      onPointerUp={() => { dragging.current = false }}
    />
  )
}
