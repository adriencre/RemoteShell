import React, { useState, useEffect, useCallback } from 'react'
import { useParams, Link } from 'react-router-dom'
import axios from 'axios'
import { 
  Server, 
  Clock, 
  Printer, 
  Terminal,
  FolderOpen,
  RefreshCw,
  AlertCircle,
  CheckCircle,
  Monitor,
  Cpu,
  MemoryStick,
  Settings,
  FileText
} from 'lucide-react'

interface Agent {
  id: string
  name: string
  last_seen: string
  active: boolean
  printers: any[] | null
  system_info?: {
    hostname: string
    os: string
    arch: string
    uptime: number
    memory_total: number
    memory_used: number
    cpu_cores: number
    disk_total: number
    disk_used: number
  } | null
}

const AgentDetail: React.FC = () => {
  const { id } = useParams<{ id: string }>()
  const [agent, setAgent] = useState<Agent | null>(null)
  const [isLoading, setIsLoading] = useState(true)
  const [error, setError] = useState('')

  // Debug: afficher l'ID récupéré
  React.useEffect(() => {
    console.log('AgentDetail: ID récupéré depuis useParams:', id)
  }, [id])

  const fetchAgent = useCallback(async () => {
    if (!id) {
      console.log('AgentDetail: Pas d\'ID dans les paramètres')
      setIsLoading(false)
      return
    }
    
    console.log('AgentDetail: Récupération de l\'agent avec l\'ID:', id)
    setIsLoading(true)
    
    try {
      const response = await axios.get(`/api/agents/${id}`)
      console.log('AgentDetail: Données reçues:', response.data)
      setAgent(response.data)
      setError('')
    } catch (err: any) {
      console.error('AgentDetail: Erreur lors de la récupération:', err)
      const errorMessage = err.response?.data?.error || err.message || 'Erreur lors du chargement des détails de l\'agent'
      setError(errorMessage)
      setAgent(null)
    } finally {
      setIsLoading(false)
    }
  }, [id])

  useEffect(() => {
    console.log('AgentDetail: useEffect déclenché avec l\'ID:', id)
    fetchAgent()
    
    // Actualiser toutes les 30 secondes
    const interval = setInterval(fetchAgent, 30000)
    return () => clearInterval(interval)
  }, [id, fetchAgent])

  const formatUptime = (seconds: number) => {
    const days = Math.floor(seconds / 86400)
    const hours = Math.floor((seconds % 86400) / 3600)
    const minutes = Math.floor((seconds % 3600) / 60)
    
    if (days > 0) return `${days}j ${hours}h ${minutes}m`
    if (hours > 0) return `${hours}h ${minutes}m`
    return `${minutes}m`
  }

  const formatBytes = (bytes: number) => {
    const sizes = ['B', 'KB', 'MB', 'GB', 'TB']
    if (bytes === 0) return '0 B'
    const i = Math.floor(Math.log(bytes) / Math.log(1024))
    return Math.round(bytes / Math.pow(1024, i) * 100) / 100 + ' ' + sizes[i]
  }

  const getStatusColor = (active: boolean) => {
    return active ? 'text-green-600' : 'text-red-600'
  }

  const getStatusIcon = (active: boolean) => {
    return active ? CheckCircle : AlertCircle
  }

  if (isLoading) {
    return (
      <div className="space-y-6">
        <div className="flex items-center justify-between">
          <div>
            <h1 className="text-2xl font-bold text-gray-900">Chargement...</h1>
            <p className="text-gray-600">ID: {id || 'Non défini'}</p>
          </div>
        </div>
        <div className="flex items-center justify-center h-64">
          <div className="animate-spin rounded-full h-12 w-12 border-b-2 border-primary-600"></div>
        </div>
      </div>
    )
  }

  if (error) {
    return (
      <div className="space-y-6">
        <div className="flex items-center justify-between">
          <div>
            <h1 className="text-2xl font-bold text-gray-900">Agent {id}</h1>
            <p className="text-gray-600">Détails de l'agent</p>
          </div>
          <Link to="/dashboard" className="btn btn-secondary">
            ← Retour au Dashboard
          </Link>
        </div>
        
        <div className="rounded-md bg-red-50 p-4">
          <div className="flex">
            <AlertCircle className="h-5 w-5 text-red-400" />
            <div className="ml-3">
              <h3 className="text-sm font-medium text-red-800">Erreur</h3>
              <div className="mt-2 text-sm text-red-700">{error}</div>
            </div>
          </div>
        </div>
      </div>
    )
  }

  if (!agent) {
    return (
      <div className="space-y-6">
        <div className="flex items-center justify-between">
          <div>
            <h1 className="text-2xl font-bold text-gray-900">Agent {id}</h1>
            <p className="text-gray-600">Détails de l'agent</p>
          </div>
          <Link to="/dashboard" className="btn btn-secondary">
            ← Retour au Dashboard
          </Link>
        </div>
        
        <div className="card p-6 text-center">
          <Server className="mx-auto h-12 w-12 text-gray-400" />
          <h3 className="mt-2 text-sm font-medium text-gray-900">Agent non trouvé</h3>
          <p className="mt-1 text-sm text-gray-500">
            L'agent avec l'ID "{id}" n'existe pas ou n'est pas connecté.
          </p>
        </div>
      </div>
    )
  }

  const StatusIcon = getStatusIcon(agent.active)

  return (
    <div className="space-y-6">
      {/* Header */}
      <div className="flex items-center justify-between">
        <div>
          <h1 className="text-2xl font-bold text-gray-900">{agent.name}</h1>
          <p className="text-gray-600">ID: {agent.id}</p>
        </div>
        <div className="flex items-center space-x-3">
          <button
            onClick={fetchAgent}
            className="btn btn-secondary btn-sm"
            disabled={isLoading}
          >
            <RefreshCw className={`h-4 w-4 mr-2 ${isLoading ? 'animate-spin' : ''}`} />
            Actualiser
          </button>
          <Link to="/dashboard" className="btn btn-secondary">
            ← Retour au Dashboard
          </Link>
        </div>
      </div>

      {/* Status Card */}
      <div className="card p-6">
        <div className="flex items-center space-x-4">
          <StatusIcon className={`h-8 w-8 ${getStatusColor(agent.active)}`} />
          <div>
            <h3 className="text-lg font-medium text-gray-900">Statut de l'agent</h3>
            <p className="text-sm text-gray-500">
              {agent.active ? 'Agent actif et connecté' : 'Agent inactif ou déconnecté'}
            </p>
          </div>
          <div className="ml-auto">
            <span className={`inline-flex items-center px-3 py-1 rounded-full text-sm font-medium ${
              agent.active 
                ? 'bg-green-100 text-green-800' 
                : 'bg-red-100 text-red-800'
            }`}>
              {agent.active ? 'Actif' : 'Inactif'}
            </span>
          </div>
        </div>
      </div>

      {/* System Information */}
      {agent.system_info && (
        <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-4 gap-6">
          <div className="card p-6">
            <div className="flex items-center">
              <div className="flex-shrink-0">
                <Monitor className="h-8 w-8 text-blue-600" />
              </div>
              <div className="ml-4">
                <p className="text-sm font-medium text-gray-500">Système</p>
                <p className="text-lg font-semibold text-gray-900">{agent.system_info.hostname}</p>
                <p className="text-sm text-gray-500">{agent.system_info.os} ({agent.system_info.arch})</p>
              </div>
            </div>
          </div>

          <div className="card p-6">
            <div className="flex items-center">
              <div className="flex-shrink-0">
                <Clock className="h-8 w-8 text-orange-600" />
              </div>
              <div className="ml-4">
                <p className="text-sm font-medium text-gray-500">Uptime</p>
                <p className="text-lg font-semibold text-gray-900">{formatUptime(agent.system_info.uptime)}</p>
                <p className="text-sm text-gray-500">Dernière connexion: {new Date(agent.last_seen).toLocaleString()}</p>
              </div>
            </div>
          </div>

          <div className="card p-6">
            <div className="flex items-center">
              <div className="flex-shrink-0">
                <Cpu className="h-8 w-8 text-purple-600" />
              </div>
              <div className="ml-4">
                <p className="text-sm font-medium text-gray-500">CPU</p>
                <p className="text-lg font-semibold text-gray-900">{agent.system_info.cpu_cores} cœurs</p>
                <p className="text-sm text-gray-500">Processeur</p>
              </div>
            </div>
          </div>

          <div className="card p-6">
            <div className="flex items-center">
              <div className="flex-shrink-0">
                <MemoryStick className="h-8 w-8 text-green-600" />
              </div>
              <div className="ml-4">
                <p className="text-sm font-medium text-gray-500">Mémoire</p>
                <p className="text-lg font-semibold text-gray-900">
                  {formatBytes(agent.system_info.memory_used)} / {formatBytes(agent.system_info.memory_total)}
                </p>
                <p className="text-sm text-gray-500">
                  {Math.round((agent.system_info.memory_used / agent.system_info.memory_total) * 100)}% utilisée
                </p>
              </div>
            </div>
          </div>
        </div>
      )}

      {/* Printers */}
      <div className="card">
        <div className="px-6 py-4 border-b border-gray-200">
          <h2 className="text-lg font-medium text-gray-900">Imprimantes ({agent.printers?.length || 0})</h2>
        </div>
        
        {!agent.printers || agent.printers.length === 0 ? (
          <div className="px-6 py-12 text-center">
            <Printer className="mx-auto h-12 w-12 text-gray-400" />
            <h3 className="mt-2 text-sm font-medium text-gray-900">Aucune imprimante</h3>
            <p className="mt-1 text-sm text-gray-500">
              Aucune imprimante n'est configurée sur cet agent.
            </p>
          </div>
        ) : (
          <div className="divide-y divide-gray-200">
            {agent.printers.map((printer, index) => (
              <div key={index} className="px-6 py-4">
                <div className="flex items-center justify-between">
                  <div className="flex items-center space-x-3">
                    <Printer className="h-5 w-5 text-gray-400" />
                    <div>
                      <p className="text-sm font-medium text-gray-900">{printer.name || `Imprimante ${index + 1}`}</p>
                      <p className="text-sm text-gray-500">{printer.status || 'Statut inconnu'}</p>
                    </div>
                  </div>
                </div>
              </div>
            ))}
          </div>
        )}
      </div>

      {/* Actions */}
      <div className="card p-6">
        <h3 className="text-lg font-medium text-gray-900 mb-4">Actions disponibles</h3>
        <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 gap-4">
          <Link
            to={`/agent/${agent.id}/terminal`}
            className="btn btn-primary w-full"
          >
            <Terminal className="h-4 w-4 mr-2" />
            Terminal
          </Link>
          <Link
            to={`/agent/${agent.id}/files`}
            className="btn btn-secondary w-full"
          >
            <FolderOpen className="h-4 w-4 mr-2" />
            Gestionnaire de fichiers
          </Link>
          <Link
            to={`/agent/${agent.id}/services`}
            className="btn btn-secondary w-full"
          >
            <Settings className="h-4 w-4 mr-2" />
            Gestion des services
          </Link>
          <Link
            to={`/agent/${agent.id}/logs`}
            className="btn btn-secondary w-full"
          >
            <FileText className="h-4 w-4 mr-2" />
            Visualisation des logs
          </Link>
          <Link
            to={`/agent/${agent.id}/printers`}
            className="btn btn-secondary w-full"
          >
            <Printer className="h-4 w-4 mr-2" />
            Gestion des imprimantes
          </Link>
        </div>
      </div>
    </div>
  )
}

export default AgentDetail
