import React, { useState, useEffect, useRef } from 'react'
import { useParams, Link } from 'react-router-dom'
import axios from 'axios'
import { 
  FileText,
  RefreshCw,
  Download,
  Trash2,
  AlertCircle,
  Search,
  Play,
  Pause
} from 'lucide-react'

interface LogSource {
  name: string
  type: string
  path?: string
  description?: string
}

interface LogEntry {
  timestamp: string
  level?: string
  source?: string
  message: string
  unit?: string
}

const LogViewer: React.FC = () => {
  const { id } = useParams<{ id: string }>()
  const [sources, setSources] = useState<LogSource[]>([])
  const [selectedSource, setSelectedSource] = useState<LogSource | null>(null)
  const [logs, setLogs] = useState<LogEntry[]>([])
  const [isLoading, setIsLoading] = useState(true)
  const [error, setError] = useState('')
  const [lines, setLines] = useState(100)
  const [autoRefresh, setAutoRefresh] = useState(false)
  const [searchTerm, setSearchTerm] = useState('')
  const [filterLevel, setFilterLevel] = useState<string>('all')
  const logsEndRef = useRef<HTMLDivElement>(null)

  useEffect(() => {
    loadSources()
  }, [id])

  useEffect(() => {
    if (selectedSource) {
      loadLogs()
    }
  }, [selectedSource, lines])

  useEffect(() => {
    if (autoRefresh && selectedSource) {
      const interval = setInterval(loadLogs, 5000)
      return () => clearInterval(interval)
    }
  }, [autoRefresh, selectedSource])

  useEffect(() => {
    if (autoRefresh) {
      scrollToBottom()
    }
  }, [logs, autoRefresh])

  const scrollToBottom = () => {
    logsEndRef.current?.scrollIntoView({ behavior: 'smooth' })
  }

  const loadSources = async () => {
    setIsLoading(true)
    setError('')
    
    try {
      const response = await axios.get(`/api/agents/${id}/logs`)
      
      // Utiliser les vraies données de l'API
      if (response.data && response.data.sources) {
        setSources(response.data.sources)
        if (response.data.sources.length > 0) {
          setSelectedSource(response.data.sources[0])
        }
      } else {
        setSources([])
      }
    } catch (err: any) {
      setError(err.response?.data?.error || 'Erreur lors du chargement des sources')
      setSources([])
    } finally {
      setIsLoading(false)
    }
  }

  const loadLogs = async () => {
    if (!selectedSource) return
    
    setError('')
    
    try {
      const params: any = {
        type: selectedSource.type,
        lines: lines
      }
      
      if (selectedSource.path) {
        params.path = selectedSource.path
      }
      
      const response = await axios.get(
        `/api/agents/${id}/logs/${encodeURIComponent(selectedSource.name)}`,
        { params }
      )
      
      // Utiliser les vraies données de l'API
      if (response.data && response.data.logs) {
        setLogs(response.data.logs)
      } else if (response.data && response.data.entries) {
        setLogs(response.data.entries)
      } else {
        setLogs([])
      }
    } catch (err: any) {
      setError(err.response?.data?.error || 'Erreur lors du chargement des logs')
      setLogs([])
    }
  }

  const downloadLogs = () => {
    const content = logs.map(entry => {
      const timestamp = entry.timestamp
      const level = entry.level ? `[${entry.level.toUpperCase()}]` : ''
      return `${timestamp} ${level} ${entry.message}`
    }).join('\n')
    
    const blob = new Blob([content], { type: 'text/plain' })
    const url = window.URL.createObjectURL(blob)
    const link = document.createElement('a')
    link.href = url
    link.download = `${selectedSource?.name || 'logs'}-${Date.now()}.log`
    document.body.appendChild(link)
    link.click()
    document.body.removeChild(link)
    window.URL.revokeObjectURL(url)
  }

  const clearLogs = () => {
    if (confirm('Êtes-vous sûr de vouloir effacer l\'affichage des logs ?')) {
      setLogs([])
    }
  }

  const getLevelColor = (level?: string) => {
    switch (level?.toLowerCase()) {
      case 'error':
        return 'text-red-600'
      case 'warning':
        return 'text-yellow-600'
      case 'info':
        return 'text-blue-600'
      case 'debug':
        return 'text-gray-600'
      default:
        return 'text-gray-900'
    }
  }

  const filteredLogs = logs.filter(entry => {
    const matchesLevel = filterLevel === 'all' || entry.level?.toLowerCase() === filterLevel
    const matchesSearch = !searchTerm || entry.message.toLowerCase().includes(searchTerm.toLowerCase())
    return matchesLevel && matchesSearch
  })

  return (
    <div className="space-y-6">
      {/* Header */}
      <div className="flex items-center justify-between">
        <div>
          <h1 className="text-2xl font-bold text-gray-900">Visualisation des logs - Agent {id}</h1>
          <p className="text-gray-600">Consulter les logs de l'agent et du système</p>
        </div>
        <div className="flex items-center space-x-3">
          <button
            onClick={() => setAutoRefresh(!autoRefresh)}
            className={`btn btn-sm ${autoRefresh ? 'btn-primary' : 'btn-secondary'}`}
          >
            {autoRefresh ? (
              <>
                <Pause className="h-4 w-4 mr-2" />
                Pause
              </>
            ) : (
              <>
                <Play className="h-4 w-4 mr-2" />
                Auto
              </>
            )}
          </button>
          <button
            onClick={loadLogs}
            className="btn btn-secondary btn-sm"
            disabled={!selectedSource}
          >
            <RefreshCw className="h-4 w-4 mr-2" />
            Actualiser
          </button>
          <Link to={`/agent/${id}`} className="btn btn-secondary">
            ← Retour
          </Link>
        </div>
      </div>

      {/* Error */}
      {error && (
        <div className="rounded-md bg-red-50 p-4">
          <div className="flex">
            <AlertCircle className="h-5 w-5 text-red-400" />
            <div className="ml-3">
              <h3 className="text-sm font-medium text-red-800">Erreur</h3>
              <div className="mt-2 text-sm text-red-700">{error}</div>
            </div>
          </div>
        </div>
      )}

      {/* Controls */}
      <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-4 gap-4">
        <div className="card p-4">
          <label className="block text-sm font-medium text-gray-700 mb-2">
            Source de logs
          </label>
          <select
            value={selectedSource?.name || ''}
            onChange={(e) => {
              const source = sources.find(s => s.name === e.target.value)
              setSelectedSource(source || null)
            }}
            className="w-full px-3 py-2 border border-gray-300 rounded-md focus:ring-2 focus:ring-primary-500 focus:border-transparent"
          >
            {sources.map(source => (
              <option key={source.name} value={source.name}>
                {source.name} ({source.type})
              </option>
            ))}
          </select>
        </div>

        <div className="card p-4">
          <label className="block text-sm font-medium text-gray-700 mb-2">
            Nombre de lignes
          </label>
          <select
            value={lines}
            onChange={(e) => setLines(Number(e.target.value))}
            className="w-full px-3 py-2 border border-gray-300 rounded-md focus:ring-2 focus:ring-primary-500 focus:border-transparent"
          >
            <option value={50}>50 lignes</option>
            <option value={100}>100 lignes</option>
            <option value={200}>200 lignes</option>
            <option value={500}>500 lignes</option>
            <option value={1000}>1000 lignes</option>
          </select>
        </div>

        <div className="card p-4">
          <label className="block text-sm font-medium text-gray-700 mb-2">
            Niveau
          </label>
          <select
            value={filterLevel}
            onChange={(e) => setFilterLevel(e.target.value)}
            className="w-full px-3 py-2 border border-gray-300 rounded-md focus:ring-2 focus:ring-primary-500 focus:border-transparent"
          >
            <option value="all">Tous</option>
            <option value="error">Erreurs</option>
            <option value="warning">Avertissements</option>
            <option value="info">Informations</option>
            <option value="debug">Debug</option>
          </select>
        </div>

        <div className="card p-4">
          <label className="block text-sm font-medium text-gray-700 mb-2">
            Actions
          </label>
          <div className="flex gap-2">
            <button
              onClick={downloadLogs}
              className="btn btn-sm btn-secondary flex-1"
              disabled={logs.length === 0}
            >
              <Download className="h-4 w-4" />
            </button>
            <button
              onClick={clearLogs}
              className="btn btn-sm btn-secondary flex-1"
              disabled={logs.length === 0}
            >
              <Trash2 className="h-4 w-4" />
            </button>
          </div>
        </div>
      </div>

      {/* Search */}
      <div className="card p-4">
        <div className="relative">
          <Search className="absolute left-3 top-1/2 transform -translate-y-1/2 h-5 w-5 text-gray-400" />
          <input
            type="text"
            placeholder="Rechercher dans les logs..."
            value={searchTerm}
            onChange={(e) => setSearchTerm(e.target.value)}
            className="w-full pl-10 pr-3 py-2 border border-gray-300 rounded-md focus:ring-2 focus:ring-primary-500 focus:border-transparent"
          />
        </div>
      </div>

      {/* Logs Display */}
      <div className="card">
        <div className="px-6 py-4 border-b border-gray-200 flex items-center justify-between">
          <h2 className="text-lg font-medium text-gray-900">
            Logs ({filteredLogs.length})
          </h2>
          {selectedSource && (
            <span className="text-sm text-gray-500">
              {selectedSource.description}
            </span>
          )}
        </div>
        
        <div className="bg-gray-900 text-gray-100 p-4 font-mono text-sm overflow-x-auto" style={{ maxHeight: '600px', overflowY: 'auto' }}>
          {isLoading ? (
            <div className="text-center py-8">
              <RefreshCw className="mx-auto h-8 w-8 text-gray-400 animate-spin" />
              <p className="mt-2 text-gray-400">Chargement des logs...</p>
            </div>
          ) : filteredLogs.length === 0 ? (
            <div className="text-center py-8">
              <FileText className="mx-auto h-12 w-12 text-gray-600" />
              <p className="mt-2 text-gray-400">Aucun log disponible</p>
            </div>
          ) : (
            <div className="space-y-1">
              {filteredLogs.map((entry, index) => (
                <div key={index} className="hover:bg-gray-800 px-2 py-1 rounded">
                  <span className="text-gray-500 mr-2">
                    {new Date(entry.timestamp).toLocaleString()}
                  </span>
                  {entry.level && (
                    <span className={`mr-2 ${getLevelColor(entry.level)}`}>
                      [{entry.level.toUpperCase()}]
                    </span>
                  )}
                  <span className="text-gray-300">{entry.message}</span>
                </div>
              ))}
              <div ref={logsEndRef} />
            </div>
          )}
        </div>
      </div>

      {/* Tips */}
      <div className="card p-4">
        <h3 className="text-sm font-medium text-gray-900 mb-2">💡 Conseils</h3>
        <ul className="text-sm text-gray-600 space-y-1">
          <li>• Sélectionnez une source de logs dans la liste déroulante</li>
          <li>• Utilisez les filtres pour affiner votre recherche</li>
          <li>• Le mode auto-rafraîchissement actualise les logs toutes les 5 secondes</li>
          <li>• Téléchargez les logs pour une analyse hors ligne</li>
        </ul>
      </div>
    </div>
  )
}

export default LogViewer

