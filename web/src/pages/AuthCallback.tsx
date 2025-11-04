import React, { useEffect } from 'react'
import { useNavigate, useSearchParams } from 'react-router-dom'
import { useAuth } from '../contexts/AuthContext'

const AuthCallback: React.FC = () => {
  const [searchParams] = useSearchParams()
  const navigate = useNavigate()
  const { setTokenFromOAuth2 } = useAuth()

  useEffect(() => {
    const token = searchParams.get('token')
    if (token) {
      // Sauvegarder le token et rediriger
      setTokenFromOAuth2(token)
      navigate('/dashboard', { replace: true })
    } else {
      // Pas de token, rediriger vers login
      navigate('/login', { replace: true })
    }
  }, [searchParams, navigate, setTokenFromOAuth2])

  return (
    <div className="min-h-screen flex items-center justify-center bg-gray-50">
      <div className="text-center">
        <div className="animate-spin rounded-full h-12 w-12 border-b-2 border-primary-600 mx-auto"></div>
        <p className="mt-4 text-gray-600">Authentification en cours...</p>
      </div>
    </div>
  )
}

export default AuthCallback


