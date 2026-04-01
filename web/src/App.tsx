import { Routes, Route, Navigate } from 'react-router-dom'
import SignupPage from './pages/SignupPage'

export default function App() {
  return (
    <Routes>
      <Route path="/signup" element={<SignupPage />} />
      <Route path="/login" element={<div style={{ color: '#e2e8f0', padding: 40 }}>Login — coming next task</div>} />
      <Route path="*" element={<Navigate to="/login" replace />} />
    </Routes>
  )
}
