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
  const [confirming, setConfirming] = useState(false)

  const handleSubmit = async () => {
    if (digitCode.length !== 2) {
      setError('Enter a 2-digit code')
      return
    }

    // In picker mode, show confirmation step first
    if (mode === 'picker' && !confirming) {
      setConfirming(true)
      setError('')
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
      setConfirming(false)
    }
  }

  const handlePickAgain = () => {
    setConfirming(false)
    setError('')
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

        <div className="mode-content">
          {mode === 'picker' ? (
            confirming ? (
              <div className="confirm-panel">
                <div className="confirm-heading">Confirm your color</div>
                <div className="confirm-row">
                  <div
                    className="confirm-swatch"
                    style={{ backgroundColor: hsvToHex(hsv.h, hsv.s, hsv.v) }}
                  />
                  <div className="confirm-values">
                    <span className="confirm-hsv">H: {hsv.h}  S: {hsv.s}  V: {hsv.v}</span>
                    <span className="confirm-tip">Tip: remember these for direct input</span>
                  </div>
                </div>
                <button
                  className="btn-primary"
                  onClick={handleSubmit}
                  disabled={loading}
                >
                  {loading ? 'Authenticating...' : 'Sign In'}
                </button>
                <button className="confirm-pick-again" onClick={handlePickAgain}>
                  ← Pick again
                </button>
              </div>
            ) : (
              <ColorPicker hsv={hsv} onChange={setHsv} />
            )
          ) : (
            <DirectInput
              hue={hsv.h}
              saturation={hsv.s}
              value={hsv.v}
              onChange={(h, s, v) => setHsv({ h, s, v })}
            />
          )}
        </div>

        {!confirming && (
          <button
            className="btn-primary"
            onClick={handleSubmit}
            disabled={loading}
          >
            {loading ? 'Authenticating...' : 'Log In'}
          </button>
        )}

        {error && <p className="auth-error">{error}</p>}

        <div className="mode-link-container">
          {mode === 'picker' && !confirming ? (
            <button className="mode-link" onClick={() => setMode('direct')}>
              Know your exact HSV? <strong>Use direct input →</strong>
            </button>
          ) : mode === 'direct' ? (
            <button className="mode-link" onClick={() => setMode('picker')}>
              Prefer the color picker? <strong>Switch to picker →</strong>
            </button>
          ) : null}
        </div>

        <p className="auth-link">
          New here? <Link to="/signup">Create identity</Link>
        </p>
      </div>
    </div>
  )
}
