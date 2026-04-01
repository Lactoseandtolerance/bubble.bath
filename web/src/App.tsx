import { Routes, Route, Navigate } from 'react-router-dom'

export default function App() {
  return (
    <Routes>
      <Route path="/signup" element={<div>Signup — coming soon</div>} />
      <Route path="/login" element={<div>Login — coming soon</div>} />
      <Route path="*" element={<Navigate to="/login" replace />} />
    </Routes>
  )
}
