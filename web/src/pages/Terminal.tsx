import React, { useState, useEffect, useRef } from 'react'
import { useParams, Link } from 'react-router-dom'
import { useWebSocket } from '../contexts/WebSocketContext'
import { 
  Terminal as TerminalIcon,
  Send,
  Trash2,
  Copy,
  AlertCircle,
  CheckCircle
} from 'lucide-react'

interface CommandHistory {
  id: string
  command: string
  output: string
  error: string
  exitCode: number
  timestamp: Date
  duration: number
}

const Terminal: React.FC = () => {
  const { id } = useParams<{ id: string }>()
  const { sendMessage, onMessage, offMessage, isConnected } = useWebSocket()
  const [command, setCommand] = useState('')
  const [isExecuting, setIsExecuting] = useState(false)
  const [history, setHistory] = useState<CommandHistory[]>([])
  const terminalRef = useRef<HTMLDivElement>(null)
  const inputRef = useRef<HTMLInputElement>(null)

  // Auto-scroll vers le bas
  useEffect(() => {
    if (terminalRef.current) {
      terminalRef.current.scrollTop = terminalRef.current.scrollHeight
    }
  }, [history])

  // Focus sur l'input
  useEffect(() => {
    if (inputRef.current) {
      inputRef.current.focus()
    }
  }, [])

  // Écouter les messages WebSocket
  useEffect(() => {
    const handleWebSocketMessage = (message: any) => {
      if (message.type === 'command_result' && message.id) {
        // Mettre à jour l'historique avec les données reçues
        setHistory(prev => {
          const existingIndex = prev.findIndex(h => h.id === message.id)
          if (existingIndex !== -1) {
            const resultData = message.data
            const updated = [...prev]
            updated[existingIndex] = {
              ...prev[existingIndex],
              output: resultData.stdout || resultData.output || '',
              error: resultData.stderr || resultData.error || '',
              exitCode: resultData.exit_code || 0,
              duration: resultData.duration || 0
            }
            return updated
          }
          return prev
        })
        
        setIsExecuting(false)
      }
    }

    onMessage(handleWebSocketMessage)
    
    return () => {
      offMessage(handleWebSocketMessage)
    }
  }, [onMessage, offMessage])

  const executeCommand = async () => {
    if (!command.trim() || isExecuting || !isConnected) return

    const commandText = command.trim()
    const commandId = Date.now().toString()
    setIsExecuting(true)
    setCommand('')

    // Créer l'entrée d'historique en attente
    const newHistory: CommandHistory = {
      id: commandId,
      command: commandText,
      output: 'Envoi de la commande...',
      error: '',
      exitCode: 0,
      timestamp: new Date(),
      duration: 0
    }
    
    setHistory(prev => [...prev, newHistory])

    try {
      // Envoyer la commande via WebSocket
      sendMessage({
        type: 'command',
        id: commandId,
        agent_id: id,
        data: {
          command: commandText,
          working_dir: '.',
          timeout: 30
        }
      })

      // Mettre à jour l'historique pour indiquer que la commande est envoyée
      setHistory(prev => prev.map(h => 
        h.id === commandId 
          ? { ...h, output: 'Commande envoyée, en attente de la réponse...' }
          : h
      ))

    } catch (error: any) {
      const errorHistory: CommandHistory = {
        id: commandId,
        command: commandText,
        output: '',
        error: 'Erreur lors de l\'envoi de la commande',
        exitCode: 1,
        timestamp: new Date(),
        duration: 0
      }
      
      setHistory(prev => prev.map(h => h.id === commandId ? errorHistory : h))
    } finally {
      setIsExecuting(false)
    }
  }

  const handleKeyPress = (e: React.KeyboardEvent) => {
    if (e.key === 'Enter') {
      executeCommand()
    }
  }

  const clearHistory = () => {
    setHistory([])
  }

  const copyOutput = (output: string) => {
    navigator.clipboard.writeText(output)
  }

  const formatDuration = (ms: number) => {
    if (ms < 1000) return `${ms}ms`
    return `${(ms / 1000).toFixed(1)}s`
  }

  const formatTimestamp = (date: Date) => {
    return date.toLocaleTimeString()
  }

  return (
    <div className="space-y-6">
      {/* Header */}
      <div className="flex items-center justify-between">
        <div>
          <h1 className="text-2xl font-bold text-gray-900">Terminal - Agent {id}</h1>
          <p className="text-gray-600">Console interactive</p>
          <div className="flex items-center space-x-2 mt-1">
            <div className={`w-2 h-2 rounded-full ${isConnected ? 'bg-green-500' : 'bg-red-500'}`}></div>
            <span className="text-xs text-gray-500">
              {isConnected ? 'Connecté' : 'Déconnecté'}
            </span>
          </div>
        </div>
        <div className="flex items-center space-x-3">
          <button
            onClick={clearHistory}
            className="btn btn-secondary btn-sm"
            disabled={history.length === 0}
          >
            <Trash2 className="h-4 w-4 mr-2" />
            Effacer
          </button>
          <Link to={`/agent/${id}`} className="btn btn-secondary">
            ← Retour
          </Link>
        </div>
      </div>

      {/* Terminal */}
      <div className="card">
        <div className="px-6 py-4 border-b border-gray-200">
          <div className="flex items-center space-x-2">
            <TerminalIcon className="h-5 w-5 text-gray-500" />
            <h2 className="text-lg font-medium text-gray-900">Console</h2>
            <span className="text-sm text-gray-500">({history.length} commandes)</span>
          </div>
        </div>
        
        <div className="p-6">
          {/* Terminal Output */}
          <div 
            ref={terminalRef}
            className="bg-black text-green-400 font-mono text-sm p-4 rounded-lg h-96 overflow-y-auto mb-4"
          >
            {history.length === 0 ? (
              <div className="text-gray-500">
                <p>Terminal prêt. Tapez une commande ci-dessous.</p>
                <p className="mt-2">Exemples :</p>
                <p className="ml-4">• ls -la</p>
                <p className="ml-4">• pwd</p>
                <p className="ml-4">• whoami</p>
                <p className="ml-4">• ps aux</p>
              </div>
            ) : (
              history.map((item) => (
                <div key={item.id} className="mb-4">
                  {/* Command */}
                  <div className="flex items-center space-x-2 mb-2">
                    <span className="text-blue-400">$</span>
                    <span className="text-white">{item.command}</span>
                    <span className="text-gray-500 text-xs">
                      [{formatTimestamp(item.timestamp)} - {formatDuration(item.duration)}]
                    </span>
                    {item.exitCode === 0 ? (
                      <CheckCircle className="h-4 w-4 text-green-400" />
                    ) : (
                      <AlertCircle className="h-4 w-4 text-red-400" />
                    )}
                  </div>
                  
                  {/* Output */}
                  {item.output && (
                    <div className="ml-4 mb-2">
                      <pre className="whitespace-pre-wrap text-green-400">{item.output}</pre>
                      <button
                        onClick={() => copyOutput(item.output)}
                        className="text-xs text-gray-500 hover:text-gray-300 mt-1"
                      >
                        <Copy className="h-3 w-3 inline mr-1" />
                        Copier
                      </button>
                    </div>
                  )}
                  
                  {/* Error */}
                  {item.error && (
                    <div className="ml-4 mb-2">
                      <pre className="whitespace-pre-wrap text-red-400">{item.error}</pre>
                      <button
                        onClick={() => copyOutput(item.error)}
                        className="text-xs text-gray-500 hover:text-gray-300 mt-1"
                      >
                        <Copy className="h-3 w-3 inline mr-1" />
                        Copier
                      </button>
                    </div>
                  )}
                </div>
              ))
            )}
            
          </div>

          {/* Command Input */}
          <div className="flex space-x-2">
            <div className="flex-1 relative">
              <span className="absolute left-3 top-1/2 transform -translate-y-1/2 text-blue-400 font-mono">$</span>
              <input
                ref={inputRef}
                type="text"
                value={command}
                onChange={(e) => setCommand(e.target.value)}
                onKeyPress={handleKeyPress}
                placeholder="Tapez une commande..."
                className="w-full pl-8 pr-4 py-2 border border-gray-300 rounded-lg focus:ring-2 focus:ring-primary-500 focus:border-transparent font-mono"
                disabled={isExecuting}
              />
            </div>
            <button
              onClick={executeCommand}
              disabled={!command.trim() || isExecuting}
              className="btn btn-primary px-4"
            >
              <Send className="h-4 w-4" />
            </button>
          </div>
          
          {isExecuting && (
            <div className="mt-2 text-sm text-gray-500">
              Exécution en cours...
            </div>
          )}
        </div>
      </div>

      {/* Tips */}
      <div className="card p-4">
        <h3 className="text-sm font-medium text-gray-900 mb-2">💡 Conseils</h3>
        <ul className="text-sm text-gray-600 space-y-1">
          <li>• Utilisez <code className="bg-gray-100 px-1 rounded">ls -la</code> pour lister les fichiers</li>
          <li>• Utilisez <code className="bg-gray-100 px-1 rounded">pwd</code> pour voir le répertoire courant</li>
          <li>• Utilisez <code className="bg-gray-100 px-1 rounded">ps aux</code> pour voir les processus</li>
          <li>• Cliquez sur "Copier" pour copier la sortie d'une commande</li>
        </ul>
      </div>
    </div>
  )
}

export default Terminal

