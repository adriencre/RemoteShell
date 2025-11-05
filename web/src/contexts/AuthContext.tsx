import React, { createContext, useContext, useState, useEffect, ReactNode } from 'react'
import axios from 'axios'

interface User {
  id: string
  name: string
  role: string
}

// Fonction pour décoder le JWT et extraire les informations utilisateur
function decodeJWT(token: string): User | null {
  try {
    // Un JWT est composé de 3 parties séparées par des points : header.payload.signature
    const parts = token.split('.')
    if (parts.length !== 3) {
      return null
    }

    // Décoder le payload (2ème partie)
    const payload = parts[1]
    // Ajouter le padding si nécessaire pour base64
    const paddedPayload = payload + '='.repeat((4 - (payload.length % 4)) % 4)
    const decodedPayload = atob(paddedPayload.replace(/-/g, '+').replace(/_/g, '/'))
    const claims = JSON.parse(decodedPayload)

    // Extraire les informations utilisateur depuis les claims
    const userId = claims.user_id || claims.agent_id || claims.sub || ''
    const userName = claims.user_name || claims.agent_name || claims.name || ''
    const role = claims.role || 'user'

    // Vérifier que le token n'est pas expiré
    if (claims.exp && claims.exp * 1000 < Date.now()) {
      return null
    }

    return {
      id: userId,
      name: userName || 'Utilisateur',
      role: role
    }
  } catch (error) {
    console.error('Erreur lors du décodage du JWT:', error)
    return null
  }
}

interface AuthContextType {
  user: User | null
  token: string | null
  login: (username: string, password: string) => Promise<void>
  logout: () => void
  isLoading: boolean
  setTokenFromOAuth2: (token: string) => void
  oauth2Enabled: boolean
}

const AuthContext = createContext<AuthContextType | undefined>(undefined)

export const useAuth = () => {
  const context = useContext(AuthContext)
  if (context === undefined) {
    throw new Error('useAuth must be used within an AuthProvider')
  }
  return context
}

interface AuthProviderProps {
  children: ReactNode
}

export const AuthProvider: React.FC<AuthProviderProps> = ({ children }) => {
  const [user, setUser] = useState<User | null>(null)
  const [token, setToken] = useState<string | null>(null)
  const [isLoading, setIsLoading] = useState(true)
  const [oauth2Enabled, setOAuth2Enabled] = useState(false)

  useEffect(() => {
    // Vérifier si OAuth2 est activé
    const checkOAuth2 = async () => {
      try {
        const response = await axios.get('/api/auth/oauth2/config')
        if (response.data.enabled) {
          setOAuth2Enabled(true)
        }
      } catch (error) {
        // OAuth2 n'est pas configuré, continuer avec le login classique
        setOAuth2Enabled(false)
      }
    }

    // Vérifier si un token existe dans le localStorage
    const savedToken = localStorage.getItem('token')
    if (savedToken) {
      setToken(savedToken)
      // Configurer axios avec le token
      axios.defaults.headers.common['Authorization'] = `Bearer ${savedToken}`
      
      // Décoder le JWT pour récupérer les vraies informations utilisateur
      const decodedUser = decodeJWT(savedToken)
      if (decodedUser) {
        setUser(decodedUser)
      } else {
        // Si le token est invalide, le supprimer
        localStorage.removeItem('token')
        delete axios.defaults.headers.common['Authorization']
      }
      setIsLoading(false)
    } else {
      // Pas de token, vérifier OAuth2
      checkOAuth2().finally(() => {
        setIsLoading(false)
      })
    }
  }, [])

  const login = async (username: string, password: string) => {
    try {
      const response = await axios.post('/api/auth/login', {
        username,
        password
      })

      const { token: newToken, user: userData } = response.data
      
      setToken(newToken)
      setUser(userData)
      
      // Sauvegarder le token
      localStorage.setItem('token', newToken)
      
      // Configurer axios avec le token
      axios.defaults.headers.common['Authorization'] = `Bearer ${newToken}`
    } catch (error) {
      throw new Error('Identifiants invalides')
    }
  }

  const logout = () => {
    setUser(null)
    setToken(null)
    localStorage.removeItem('token')
    delete axios.defaults.headers.common['Authorization']
  }

  const setTokenFromOAuth2 = (newToken: string) => {
    setToken(newToken)
    localStorage.setItem('token', newToken)
    axios.defaults.headers.common['Authorization'] = `Bearer ${newToken}`
    
    // Décoder le JWT pour récupérer les vraies informations utilisateur depuis Authentik
    const decodedUser = decodeJWT(newToken)
    if (decodedUser) {
      setUser(decodedUser)
    } else {
      // Fallback si le décodage échoue
      setUser({
        id: 'user',
        name: 'Utilisateur',
        role: 'user'
      })
    }
  }

  const value = {
    user,
    token,
    login,
    logout,
    isLoading,
    setTokenFromOAuth2,
    oauth2Enabled
  }

  return (
    <AuthContext.Provider value={value}>
      {children}
    </AuthContext.Provider>
  )
}


