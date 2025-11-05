import React, { useState, useEffect } from 'react'
import { Link } from 'react-router-dom'
import axios from 'axios'
import { 
  Server, 
  Activity, 
  Clock, 
  Printer, 
  Terminal,
  FolderOpen,
  RefreshCw,
  AlertCircle,
  CheckCircle,
  ArrowRight,
  Settings
} from 'lucide-react'

interface Agent {
  id: string
  name: string
  last_seen: string
  active: boolean
  printers: number
  system_info?: {
    hostname: string
    os: string
    arch: string
    uptime: number
    memory_total: number
    memory_used: number
  }
}

const Dashboard: React.FC = () => {
  const [agents, setAgents] = useState<Agent[]>([])
  const [isLoading, setIsLoading] = useState(true)
  const [error, setError] = useState('')
  const [lastUpdate, setLastUpdate] = useState<Date>(new Date())

  const fetchAgents = async () => {
    try {
      const response = await axios.get('/api/agents')
      setAgents(response.data.agents || [])
      setLastUpdate(new Date())
      setError('')
    } catch (err) {
      setError('Erreur lors du chargement des agents')
      console.error('Erreur:', err)
    } finally {
      setIsLoading(false)
    }
  }

  useEffect(() => {
    fetchAgents()
    
    // Actualiser toutes les 30 secondes
    const interval = setInterval(fetchAgents, 30000)
    return () => clearInterval(interval)
  }, [])

  const formatUptime = (seconds: number) => {
    const days = Math.floor(seconds / 86400)
    const hours = Math.floor((seconds % 86400) / 3600)
    const minutes = Math.floor((seconds % 3600) / 60)
    
    if (days > 0) return `${days}j ${hours}h`
    if (hours > 0) return `${hours}h ${minutes}m`
    return `${minutes}m`
  }


  const getStatusColor = (active: boolean) => {
    return active ? 'text-green-600' : 'text-red-600'
  }

  const getStatusIcon = (active: boolean) => {
    return active ? CheckCircle : AlertCircle
  }

  if (isLoading) {
    return (
      <div className="flex items-center justify-center h-64">
        <div className="animate-spin rounded-full h-12 w-12 border-b-2 border-primary-600"></div>
      </div>
    )
  }

  return (
    <div className="space-y-6">
      {/* Header */}
      <div className="flex items-center justify-between">
        <div>
          <h1 className="text-2xl font-bold text-gray-900">Dashboard</h1>
          <p className="text-gray-600">
            Dernière mise à jour: {lastUpdate.toLocaleTimeString()}
          </p>
        </div>
        <button
          onClick={fetchAgents}
          className="btn btn-secondary btn-sm"
          disabled={isLoading}
        >
          <RefreshCw className={`h-4 w-4 mr-2 ${isLoading ? 'animate-spin' : ''}`} />
          Actualiser
        </button>
      </div>

      {/* Stats */}
      <div className="grid grid-cols-1 md:grid-cols-4 gap-6">
        <div className="card p-6">
          <div className="flex items-center">
            <div className="flex-shrink-0">
              <Server className="h-8 w-8 text-primary-600" />
            </div>
            <div className="ml-4">
              <p className="text-sm font-medium text-gray-500">Agents</p>
              <p className="text-2xl font-semibold text-gray-900">{agents.length}</p>
            </div>
          </div>
        </div>

        <div className="card p-6">
          <div className="flex items-center">
            <div className="flex-shrink-0">
              <Activity className="h-8 w-8 text-green-600" />
            </div>
            <div className="ml-4">
              <p className="text-sm font-medium text-gray-500">Actifs</p>
              <p className="text-2xl font-semibold text-gray-900">
                {agents.filter(a => a.active).length}
              </p>
            </div>
          </div>
        </div>

        <div className="card p-6">
          <div className="flex items-center">
            <div className="flex-shrink-0">
              <Printer className="h-8 w-8 text-blue-600" />
            </div>
            <div className="ml-4">
              <p className="text-sm font-medium text-gray-500">Imprimantes</p>
              <p className="text-2xl font-semibold text-gray-900">
                {agents.reduce((sum, a) => sum + a.printers, 0)}
              </p>
            </div>
          </div>
        </div>

        <div className="card p-6">
          <div className="flex items-center">
            <div className="flex-shrink-0">
              <Clock className="h-8 w-8 text-orange-600" />
            </div>
            <div className="ml-4">
              <p className="text-sm font-medium text-gray-500">Inactifs</p>
              <p className="text-2xl font-semibold text-gray-900">
                {agents.filter(a => !a.active).length}
              </p>
            </div>
          </div>
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

      {/* Agents List */}
      <div className="card">
        <div className="px-6 py-4 border-b border-gray-200 flex items-center justify-between">
          <h2 className="text-lg font-medium text-gray-900">Serveurs d'impression</h2>
          {agents.length > 0 && (
            <Link
              to="/agents"
              className="btn btn-secondary btn-sm"
            >
              Voir tous les agents
              <ArrowRight className="h-4 w-4 ml-2" />
            </Link>
          )}
        </div>
        
        {agents.length === 0 ? (
          <div className="px-6 py-12 text-center">
            <Server className="mx-auto h-12 w-12 text-gray-400" />
            <h3 className="mt-2 text-sm font-medium text-gray-900">Aucun agent</h3>
            <p className="mt-1 text-sm text-gray-500">
              Aucun serveur d'impression n'est actuellement connecté.
            </p>
          </div>
        ) : (
          <div className="divide-y divide-gray-200">
            {agents.map((agent) => {
              const StatusIcon = getStatusIcon(agent.active)
              return (
                <div key={agent.id} className="px-6 py-4 hover:bg-gray-50">
                  <div className="flex items-center justify-between">
                    <div className="flex items-center space-x-4">
                      <div className="flex-shrink-0">
                        <StatusIcon className={`h-6 w-6 ${getStatusColor(agent.active)}`} />
                      </div>
                      <div className="flex-1 min-w-0">
                        <div className="flex items-center space-x-2">
                          <Link 
                            to={`/agent/${agent.id}`}
                            className="text-sm font-medium text-gray-900 truncate hover:text-primary-600 transition-colors"
                          >
                            {agent.name}
                          </Link>
                          <span className={`inline-flex items-center px-2.5 py-0.5 rounded-full text-xs font-medium ${
                            agent.active 
                              ? 'bg-green-100 text-green-800' 
                              : 'bg-red-100 text-red-800'
                          }`}>
                            {agent.active ? 'Actif' : 'Inactif'}
                          </span>
                        </div>
                        <div className="mt-1 flex items-center space-x-4 text-sm text-gray-500">
                          <span>ID: {agent.id}</span>
                          {agent.system_info && (
                            <>
                              <span>• {agent.system_info.hostname}</span>
                              <span>• {agent.system_info.os}</span>
                              <span>• {formatUptime(agent.system_info.uptime)}</span>
                            </>
                          )}
                        </div>
                      </div>
                    </div>
                    
                    <div className="flex items-center space-x-2">
                      {/* Imprimantes */}
                      <div className="flex items-center text-sm text-gray-500">
                        <Printer className="h-4 w-4 mr-1" />
                        {agent.printers}
                      </div>
                      
                      {/* Actions */}
                      <div className="flex items-center space-x-1">
                        <Link
                          to={`/agent/${agent.id}/terminal`}
                          className="btn btn-sm btn-secondary"
                          title="Terminal"
                        >
                          <Terminal className="h-4 w-4" />
                        </Link>
                        <Link
                          to={`/agent/${agent.id}/files`}
                          className="btn btn-sm btn-secondary"
                          title="Fichiers"
                        >
                          <FolderOpen className="h-4 w-4" />
                        </Link>
                        <Link
                          to={`/agent/${agent.id}/services`}
                          className="btn btn-sm btn-secondary"
                          title="Services"
                        >
                          <Settings className="h-4 w-4" />
                        </Link>
                      </div>
                    </div>
                  </div>
                </div>
              )
            })}
          </div>
        )}
      </div>
    </div>
  )
}

export default Dashboard
