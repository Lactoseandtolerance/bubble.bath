import { useRef, useEffect, useCallback } from 'react'
import { hsvToRgb } from '../utils/color'
import './SatValSquare.css'

interface Props {
  hue: number
  saturation: number
  value: number
  onChange: (s: number, v: number) => void
}

const SIZE = 256

export default function SatValSquare({ hue, saturation, value, onChange }: Props) {
  const canvasRef = useRef<HTMLCanvasElement>(null)
  const dragging = useRef(false)

  const draw = useCallback(() => {
    const canvas = canvasRef.current
    if (!canvas) return
    const ctx = canvas.getContext('2d')
    if (!ctx) return

    // Render the full S/V plane for the current hue
    const imageData = ctx.createImageData(SIZE, SIZE)
    for (let y = 0; y < SIZE; y++) {
      for (let x = 0; x < SIZE; x++) {
        const s = Math.round((x / (SIZE - 1)) * 100)
        const v = Math.round(((SIZE - 1 - y) / (SIZE - 1)) * 100)
        const [r, g, b] = hsvToRgb(hue, s, v)
        const i = (y * SIZE + x) * 4
        imageData.data[i] = r
        imageData.data[i + 1] = g
        imageData.data[i + 2] = b
        imageData.data[i + 3] = 255
      }
    }
    ctx.putImageData(imageData, 0, 0)

    // Crosshair selector
    const cx = (saturation / 100) * (SIZE - 1)
    const cy = ((100 - value) / 100) * (SIZE - 1)
    ctx.strokeStyle = value > 50 ? '#000' : '#fff'
    ctx.lineWidth = 2
    ctx.beginPath()
    ctx.arc(cx, cy, 7, 0, Math.PI * 2)
    ctx.stroke()
    ctx.strokeStyle = value > 50 ? '#fff' : '#000'
    ctx.lineWidth = 1
    ctx.beginPath()
    ctx.arc(cx, cy, 8, 0, Math.PI * 2)
    ctx.stroke()
  }, [hue, saturation, value])

  useEffect(() => { draw() }, [draw])

  const handlePointer = useCallback((e: React.PointerEvent) => {
    const canvas = canvasRef.current
    if (!canvas) return
    const rect = canvas.getBoundingClientRect()
    const x = Math.max(0, Math.min(e.clientX - rect.left, rect.width))
    const y = Math.max(0, Math.min(e.clientY - rect.top, rect.height))
    const s = Math.round((x / rect.width) * 100)
    const v = Math.round((1 - y / rect.height) * 100)
    onChange(
      Math.min(100, Math.max(0, s)),
      Math.min(100, Math.max(0, v)),
    )
  }, [onChange])

  return (
    <canvas
      ref={canvasRef}
      className="sat-val-square"
      width={SIZE}
      height={SIZE}
      role="slider"
      aria-label="Saturation and Value"
      aria-valuenow={saturation}
      aria-valuemin={0}
      aria-valuemax={100}
      tabIndex={0}
      onPointerDown={(e) => {
        dragging.current = true
        e.currentTarget.setPointerCapture(e.pointerId)
        handlePointer(e)
      }}
      onPointerMove={(e) => { if (dragging.current) handlePointer(e) }}
      onPointerUp={() => { dragging.current = false }}
      onPointerCancel={() => { dragging.current = false }}
    />
  )
}
