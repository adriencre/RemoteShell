import React, { useState } from 'react'
import { Server } from 'lucide-react'

const Login: React.FC = () => {
  const [loading, setLoading] = useState(false)
  const [error, setError] = useState('')

  const handleAuthentikLogin = () => {
    setError('')
    setLoading(true)
    // Rediriger vers l'endpoint OAuth2
    window.location.href = '/api/auth/oauth2/login'
  }

  return (
    <div className="min-h-screen bg-gradient-to-br from-blue-50 to-indigo-100 flex items-center justify-center py-12 px-4 sm:px-6 lg:px-8">
      <div className="max-w-md w-full space-y-8 bg-white rounded-xl shadow-2xl p-8">
        <div>
          <div className="flex justify-center mb-6">
            <div className="h-16 w-16 flex items-center justify-center rounded-full bg-primary-100">
              <Server className="h-8 w-8 text-primary-600" />
            </div>
          </div>
          <h2 className="mt-6 text-center text-3xl font-extrabold text-gray-900">
            RemoteShell
          </h2>
          <p className="mt-2 text-center text-sm text-gray-600">
            Connectez-vous avec votre compte Authentik SSO
          </p>
        </div>

        {error && (
          <div className="bg-red-50 border border-red-200 text-red-700 px-4 py-3 rounded-lg text-sm">
            {error}
          </div>
        )}

        <div className="mt-8">
          <button
            type="button"
            onClick={handleAuthentikLogin}
            disabled={loading}
            className="w-full flex justify-center items-center py-3 px-4 border-2 border-indigo-600 rounded-lg text-base font-medium text-indigo-600 bg-white hover:bg-indigo-50 focus:outline-none focus:ring-2 focus:ring-offset-2 focus:ring-indigo-500 disabled:opacity-50 disabled:cursor-not-allowed transition-colors shadow-sm"
          >
            {loading ? (
              <div className="flex items-center">
                <div className="animate-spin rounded-full h-5 w-5 border-b-2 border-indigo-600 mr-3"></div>
                Connexion en cours...
              </div>
            ) : (
              <>
                <svg className="h-6 w-6 mr-3" viewBox="0 0 24 24" fill="currentColor">
                  <path d="M12 2C6.48 2 2 6.48 2 12s4.48 10 10 10 10-4.48 10-10S17.52 2 12 2zm-2 15l-5-5 1.41-1.41L10 14.17l7.59-7.59L19 8l-9 9z"/>
                </svg>
                Se connecter avec Authentik SSO
              </>
            )}
          </button>
        </div>

        <div className="mt-6 text-center">
          <p className="text-xs text-gray-500">
            Vous devez avoir un compte Authentik pour accéder à l'application
          </p>
        </div>
      </div>
    </div>
  )
}

export default Login


