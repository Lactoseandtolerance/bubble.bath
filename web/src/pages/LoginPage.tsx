import { useState } from 'react'
import { Link } from 'react-router-dom'
import ColorPicker, { type HSV } from '../components/ColorPicker'
import DirectInput from '../components/DirectInput'
import DigitInput from '../components/DigitInput'
import { loginPicker, loginDirect, ApiRequestError } from '../api/client'
import { hsvToHex } from '../utils/color'
import './LoginPage.css'

type Mode = 'picker' | 'direct'

export default function LoginPage() {
  const [mode, setMode] = useState<Mode>('picker')
  const [digitCode, setDigitCode] = useState('')
  const [hsv, setHsv] = useState<HSV>({ h: 180, s: 50, v: 80 })
  const [error, setError] = useState('')
  const [loading, setLoading] = useState(false)
  const [success, setSuccess] = useState(false)

  const handleSubmit = async () => {
    if (digitCode.length !== 2) {
      setError('Enter a 2-digit code')
      return
    }
    setError('')
    setLoading(true)
    try {
      const login = mode === 'picker' ? loginPicker : loginDirect
      const tokens = await login(parseInt(digitCode), hsv.h, hsv.s, hsv.v)
      localStorage.setItem('bb_access', tokens.access_token)
      localStorage.setItem('bb_refresh', tokens.refresh_token)
      setSuccess(true)
    } catch (e) {
      if (e instanceof ApiRequestError) {
        if (e.status === 401) {
          setError('No match — check your number and color')
        } else if (e.status === 429) {
          setError('Too many attempts — try again in a minute')
        } else {
          setError(e.message)
        }
      } else {
        setError('Something went wrong')
      }
    } finally {
      setLoading(false)
    }
  }

  if (success) {
    return (
      <div className="auth-page">
        <div className="auth-card">
          <div
            className="success-swatch"
            style={{ backgroundColor: hsvToHex(hsv.h, hsv.s, hsv.v) }}
          />
          <h1 className="auth-title">Welcome back</h1>
          <p className="step-label">You're authenticated.</p>
        </div>
      </div>
    )
  }

  return (
    <div className="auth-page">
      <div className="auth-card">
        <h1 className="auth-title">Log In</h1>

        <DigitInput value={digitCode} onChange={setDigitCode} />

        <div className="mode-toggle">
          <button
            className={`mode-btn ${mode === 'picker' ? 'active' : ''}`}
            onClick={() => setMode('picker')}
          >
            Color Picker
          </button>
          <button
            className={`mode-btn ${mode === 'direct' ? 'active' : ''}`}
            onClick={() => setMode('direct')}
          >
            Direct Input
          </button>
        </div>

        <div className="mode-content">
          {mode === 'picker' ? (
            <ColorPicker hsv={hsv} onChange={setHsv} />
          ) : (
            <DirectInput
              hue={hsv.h}
              saturation={hsv.s}
              value={hsv.v}
              onChange={(h, s, v) => setHsv({ h, s, v })}
            />
          )}
        </div>

        <button
          className="btn-primary"
          onClick={handleSubmit}
          disabled={loading}
        >
          {loading ? 'Authenticating...' : 'Log In'}
        </button>

        {error && <p className="auth-error">{error}</p>}

        <p className="auth-link">
          New here? <Link to="/signup">Create identity</Link>
        </p>
      </div>
    </div>
  )
}
