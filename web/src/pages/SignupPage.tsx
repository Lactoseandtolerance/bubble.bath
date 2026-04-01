import { useState } from 'react'
import { Link } from 'react-router-dom'
import ColorPicker, { type HSV } from '../components/ColorPicker'
import DigitInput from '../components/DigitInput'
import { hsvDistance, hsvToHex } from '../utils/color'
import { signup, ApiRequestError } from '../api/client'
import './SignupPage.css'

type Step = 'digit' | 'color' | 'confirm' | 'success'

const CONFIRM_TOLERANCE = 15

export default function SignupPage() {
  const [step, setStep] = useState<Step>('digit')
  const [digitCode, setDigitCode] = useState('')
  const [color, setColor] = useState<HSV>({ h: 180, s: 50, v: 80 })
  const [confirmColor, setConfirmColor] = useState<HSV>({ h: 0, s: 50, v: 80 })
  const [error, setError] = useState('')
  const [loading, setLoading] = useState(false)

  const handleDigitNext = () => {
    if (digitCode.length !== 2) {
      setError('Enter a 2-digit code')
      return
    }
    setError('')
    setStep('color')
  }

  const handleColorNext = () => {
    setError('')
    setConfirmColor({ h: 0, s: 50, v: 80 })
    setStep('confirm')
  }

  const handleConfirmSubmit = async () => {
    const dist = hsvDistance(
      color.h, color.s, color.v,
      confirmColor.h, confirmColor.s, confirmColor.v,
    )
    if (dist > CONFIRM_TOLERANCE) {
      setError(`Colors don't match (distance: ${dist.toFixed(1)}). Try picking your color again.`)
      return
    }

    setError('')
    setLoading(true)
    try {
      const tokens = await signup(parseInt(digitCode), color.h, color.s, color.v)
      localStorage.setItem('bb_access', tokens.access_token)
      localStorage.setItem('bb_refresh', tokens.refresh_token)
      setStep('success')
    } catch (e) {
      if (e instanceof ApiRequestError) {
        if (e.status === 409) {
          setError('This number + color combination is already taken. Try different inputs.')
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

  const stepIndex = ['digit', 'color', 'confirm', 'success'].indexOf(step)

  return (
    <div className="auth-page">
      <div className="auth-card">
        <h1 className="auth-title">Create Identity</h1>

        <div className="step-indicator">
          {[0, 1, 2].map((i) => (
            <div key={i} className={`step-dot ${stepIndex >= i ? 'active' : ''}`} />
          ))}
        </div>

        {step === 'digit' && (
          <div className="step-content">
            <p className="step-label">Choose your secret number</p>
            <DigitInput value={digitCode} onChange={setDigitCode} />
            <button className="btn-primary" onClick={handleDigitNext}>Next</button>
          </div>
        )}

        {step === 'color' && (
          <div className="step-content">
            <p className="step-label">Choose your secret color</p>
            <ColorPicker hsv={color} onChange={setColor} />
            <button className="btn-primary" onClick={handleColorNext}>Next</button>
            <button className="btn-secondary" onClick={() => setStep('digit')}>Back</button>
          </div>
        )}

        {step === 'confirm' && (
          <div className="step-content">
            <p className="step-label">Confirm — pick your color again from memory</p>
            <ColorPicker hsv={confirmColor} onChange={setConfirmColor} />
            <button
              className="btn-primary"
              onClick={handleConfirmSubmit}
              disabled={loading}
            >
              {loading ? 'Creating...' : 'Create Identity'}
            </button>
            <button className="btn-secondary" onClick={() => setStep('color')}>Back</button>
          </div>
        )}

        {step === 'success' && (
          <div className="step-content">
            <div
              className="success-swatch"
              style={{ backgroundColor: hsvToHex(color.h, color.s, color.v) }}
            />
            <p className="step-label success-text">Identity created!</p>
            <p className="step-hint">
              Remember your number (<strong>{digitCode}</strong>) and color.
            </p>
            <Link to="/login" className="btn-primary" style={{ textAlign: 'center' }}>
              Go to Login
            </Link>
          </div>
        )}

        {error && <p className="auth-error">{error}</p>}

        {step !== 'success' && (
          <p className="auth-link">
            Already have an identity? <Link to="/login">Log in</Link>
          </p>
        )}
      </div>
    </div>
  )
}
