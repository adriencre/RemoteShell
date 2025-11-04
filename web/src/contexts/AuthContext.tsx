import React, { createContext, useContext, useState, useEffect, ReactNode } from 'react'
import axios from 'axios'

interface User {
  id: string
  name: string
  role: string
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
      
      // Récupérer les informations utilisateur depuis le token JWT
      // Pour simplifier, on utilise des données par défaut
      // En production, on pourrait décoder le JWT pour obtenir les infos
      setUser({
        id: 'admin',
        name: 'Administrator',
        role: 'admin'
      })
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
    
    // Récupérer les informations utilisateur depuis le token JWT
    // Pour simplifier, on utilise des données par défaut
    // En production, on pourrait décoder le JWT pour obtenir les infos
    setUser({
      id: 'admin',
      name: 'Administrator',
      role: 'admin'
    })
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


