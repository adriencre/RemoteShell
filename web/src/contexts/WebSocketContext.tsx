import React, { createContext, useContext, useEffect, useState, ReactNode, useCallback, useRef } from 'react'
import { useAuth } from './AuthContext'

interface WebSocketMessage {
  type: string
  id?: string
  data?: any
  timestamp?: string
  agent_id?: string
}

interface WebSocketContextType {
  isConnected: boolean
  sendMessage: (message: any) => void
  lastMessage: WebSocketMessage | null
  onMessage: (callback: (message: WebSocketMessage) => void) => void
  offMessage: (callback: (message: WebSocketMessage) => void) => void
}

const WebSocketContext = createContext<WebSocketContextType | undefined>(undefined)

export const useWebSocket = () => {
  const context = useContext(WebSocketContext)
  if (context === undefined) {
    throw new Error('useWebSocket must be used within a WebSocketProvider')
  }
  return context
}

interface WebSocketProviderProps {
  children: ReactNode
}

export const WebSocketProvider: React.FC<WebSocketProviderProps> = ({ children }) => {
  const [socket, setSocket] = useState<WebSocket | null>(null)
  const [isConnected, setIsConnected] = useState(false)
  const [lastMessage, setLastMessage] = useState<WebSocketMessage | null>(null)
  const messageCallbacksRef = useRef<Set<(message: WebSocketMessage) => void>>(new Set())
  const { token } = useAuth()

  useEffect(() => {
    if (token) {
      connectWebSocket()
    }

    return () => {
      if (socket) {
        socket.close()
      }
    }
  }, [token])

  const connectWebSocket = () => {
    // Utiliser le même host que la page web
    const protocol = window.location.protocol === 'https:' ? 'wss:' : 'ws:'
    const host = window.location.host
    const wsUrl = `${protocol}//${host}/ws`
    const newSocket = new WebSocket(wsUrl)

    newSocket.onopen = () => {
      setIsConnected(true)
      setSocket(newSocket)
      
      // Envoyer le token d'authentification
      if (token) {
        newSocket.send(JSON.stringify({
          type: 'auth',
          data: { token: token }
        }))
      }
    }

    newSocket.onmessage = (event) => {
      try {
        const message: WebSocketMessage = JSON.parse(event.data)
        setLastMessage(message)
        
        // Appeler tous les callbacks enregistrés (utiliser .current pour avoir la dernière version)
        messageCallbacksRef.current.forEach(callback => {
          try {
            callback(message)
          } catch (error) {
            console.error('Erreur dans le callback WebSocket:', error)
          }
        })
      } catch (error) {
        console.error('Erreur de parsing du message WebSocket:', error)
      }
    }

    newSocket.onclose = () => {
      setIsConnected(false)
      setSocket(null)
      
      // Tentative de reconnexion après 5 secondes
      setTimeout(() => {
        if (token) {
          connectWebSocket()
        }
      }, 5000)
    }

    newSocket.onerror = (error) => {
      console.error('Erreur WebSocket:', error)
    }
  }

  const sendMessage = (message: any) => {
    if (socket && isConnected) {
      socket.send(JSON.stringify(message))
    }
  }

  const onMessage = useCallback((callback: (message: WebSocketMessage) => void) => {
    messageCallbacksRef.current.add(callback)
  }, [])

  const offMessage = useCallback((callback: (message: WebSocketMessage) => void) => {
    messageCallbacksRef.current.delete(callback)
  }, [])

  const value = {
    isConnected,
    sendMessage,
    lastMessage,
    onMessage,
    offMessage
  }

  return (
    <WebSocketContext.Provider value={value}>
      {children}
    </WebSocketContext.Provider>
  )
}

