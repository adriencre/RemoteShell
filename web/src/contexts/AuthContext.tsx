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

  useEffect(() => {
    // Vérifier si un token existe dans le localStorage
    const savedToken = localStorage.getItem('token')
    if (savedToken) {
      setToken(savedToken)
      // Configurer axios avec le token
      axios.defaults.headers.common['Authorization'] = `Bearer ${savedToken}`
      
      // Récupérer les informations utilisateur
      // Pour simplifier, on utilise des données par défaut
      setUser({
        id: 'admin',
        name: 'Administrator',
        role: 'admin'
      })
    }
    setIsLoading(false)
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

  const value = {
    user,
    token,
    login,
    logout,
    isLoading
  }

  return (
    <AuthContext.Provider value={value}>
      {children}
    </AuthContext.Provider>
  )
}


