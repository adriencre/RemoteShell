import { BrowserRouter as Router, Routes, Route, Navigate } from 'react-router-dom'
import { AuthProvider } from './contexts/AuthContext'
import { WebSocketProvider } from './contexts/WebSocketContext'
import Layout from './components/Layout'
import Login from './pages/Login'
import AuthCallback from './pages/AuthCallback'
import Dashboard from './pages/Dashboard'
import Agents from './pages/Agents'
import AgentDetail from './pages/AgentDetail'
import Terminal from './pages/Terminal'
import FileManager from './pages/FileManager'
import PrinterMonitor from './pages/PrinterMonitor'
import ServiceManager from './pages/ServiceManager'
import LogViewer from './pages/LogViewer'
import ProtectedRoute from './components/ProtectedRoute'

function App() {
  return (
    <AuthProvider>
      <WebSocketProvider>
        <Router>
          <div className="min-h-screen bg-gray-50">
            <Routes>
              <Route path="/login" element={<Login />} />
              <Route path="/auth/callback" element={<AuthCallback />} />
              <Route path="/" element={
                <ProtectedRoute>
                  <Layout />
                </ProtectedRoute>
              }>
                <Route index element={<Navigate to="/dashboard" replace />} />
                <Route path="dashboard" element={<Dashboard />} />
                <Route path="agents" element={<Agents />} />
                <Route path="agent/:id" element={<AgentDetail />} />
                <Route path="agent/:id/terminal" element={<Terminal />} />
                <Route path="agent/:id/files" element={<FileManager />} />
                <Route path="agent/:id/printers" element={<PrinterMonitor />} />
                <Route path="agent/:id/services" element={<ServiceManager />} />
                <Route path="agent/:id/logs" element={<LogViewer />} />
              </Route>
            </Routes>
          </div>
        </Router>
      </WebSocketProvider>
    </AuthProvider>
  )
}

export default App
