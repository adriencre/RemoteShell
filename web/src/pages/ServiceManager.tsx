import React, { useState, useEffect } from 'react'
import { useParams, Link } from 'react-router-dom'
import axios from 'axios'
import { 
  Server,
  Play,
  Square,
  RefreshCw,
  CheckCircle,
  XCircle,
  AlertCircle,
  Loader,
  Settings,
  Container
} from 'lucide-react'

interface ServiceInfo {
  name: string
  type: string
  status: string
  state: string
  description?: string
  enabled?: boolean
  container_id?: string
  image?: string
}

const ServiceManager: React.FC = () => {
  const { id } = useParams<{ id: string }>()
  const [services, setServices] = useState<ServiceInfo[]>([])
  const [isLoading, setIsLoading] = useState(true)
  const [error, setError] = useState('')
  const [actionLoading, setActionLoading] = useState<string | null>(null)
  const [filter, setFilter] = useState<'all' | 'systemd' | 'docker'>('all')
  const [searchTerm, setSearchTerm] = useState('')

  useEffect(() => {
    loadServices()
    // Rafraîchir toutes les 30 secondes
    const interval = setInterval(loadServices, 30000)
    return () => clearInterval(interval)
  }, [id])

  const loadServices = async () => {
    setIsLoading(true)
    setError('')
    
    try {
      const response = await axios.get(`/api/agents/${id}/services`)
      
      // Utiliser les vraies données de l'API
      if (response.data && response.data.services) {
        setServices(response.data.services)
      } else {
        setServices([])
      }
    } catch (err: any) {
      setError(err.response?.data?.error || 'Erreur lors du chargement des services')
      setServices([])
    } finally {
      setIsLoading(false)
    }
  }

  const executeAction = async (service: ServiceInfo, action: string) => {
    const actionKey = `${service.name}-${action}`
    setActionLoading(actionKey)
    setError('')
    
    try {
      await axios.post(
        `/api/agents/${id}/services/${encodeURIComponent(service.name)}/${action}`,
        null,
        { params: { type: service.type } }
      )
      
      // Attendre un peu puis recharger
      setTimeout(loadServices, 1000)
    } catch (err: any) {
      setError(err.response?.data?.error || `Erreur lors de l'action ${action}`)
    } finally {
      setActionLoading(null)
    }
  }

  const getStateIcon = (state: string) => {
    switch (state.toLowerCase()) {
      case 'active':
      case 'running':
        return <CheckCircle className="h-5 w-5 text-green-500" />
      case 'inactive':
      case 'stopped':
        return <XCircle className="h-5 w-5 text-gray-400" />
      case 'failed':
        return <AlertCircle className="h-5 w-5 text-red-500" />
      default:
        return <AlertCircle className="h-5 w-5 text-yellow-500" />
    }
  }

  const getStateColor = (state: string) => {
    switch (state.toLowerCase()) {
      case 'active':
      case 'running':
        return 'bg-green-100 text-green-800'
      case 'inactive':
      case 'stopped':
        return 'bg-gray-100 text-gray-800'
      case 'failed':
        return 'bg-red-100 text-red-800'
      default:
        return 'bg-yellow-100 text-yellow-800'
    }
  }

  const getTypeIcon = (type: string) => {
    return type === 'docker' ? (
      <Container className="h-5 w-5 text-blue-500" />
    ) : (
      <Settings className="h-5 w-5 text-purple-500" />
    )
  }

  const filteredServices = services.filter(service => {
    const matchesFilter = filter === 'all' || service.type === filter
    const matchesSearch = service.name.toLowerCase().includes(searchTerm.toLowerCase()) ||
                         service.description?.toLowerCase().includes(searchTerm.toLowerCase())
    return matchesFilter && matchesSearch
  })

  return (
    <div className="space-y-6">
      {/* Header */}
      <div className="flex items-center justify-between">
        <div>
          <h1 className="text-2xl font-bold text-gray-900">Gestion des services - Agent {id}</h1>
          <p className="text-gray-600">Contrôle des services systemd et Docker</p>
        </div>
        <div className="flex items-center space-x-3">
          <button
            onClick={loadServices}
            className="btn btn-secondary btn-sm"
            disabled={isLoading}
          >
            <RefreshCw className={`h-4 w-4 mr-2 ${isLoading ? 'animate-spin' : ''}`} />
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

      {/* Filters */}
      <div className="card p-4">
        <div className="flex flex-col sm:flex-row gap-4">
          <div className="flex-1">
            <input
              type="text"
              placeholder="Rechercher un service..."
              value={searchTerm}
              onChange={(e) => setSearchTerm(e.target.value)}
              className="w-full px-3 py-2 border border-gray-300 rounded-md focus:ring-2 focus:ring-primary-500 focus:border-transparent"
            />
          </div>
          <div className="flex gap-2">
            <button
              onClick={() => setFilter('all')}
              className={`px-4 py-2 rounded-md ${
                filter === 'all'
                  ? 'bg-primary-600 text-white'
                  : 'bg-gray-100 text-gray-700 hover:bg-gray-200'
              }`}
            >
              Tous
            </button>
            <button
              onClick={() => setFilter('systemd')}
              className={`px-4 py-2 rounded-md ${
                filter === 'systemd'
                  ? 'bg-primary-600 text-white'
                  : 'bg-gray-100 text-gray-700 hover:bg-gray-200'
              }`}
            >
              Systemd
            </button>
            <button
              onClick={() => setFilter('docker')}
              className={`px-4 py-2 rounded-md ${
                filter === 'docker'
                  ? 'bg-primary-600 text-white'
                  : 'bg-gray-100 text-gray-700 hover:bg-gray-200'
              }`}
            >
              Docker
            </button>
          </div>
        </div>
      </div>

      {/* Services List */}
      <div className="card">
        <div className="px-6 py-4 border-b border-gray-200">
          <h2 className="text-lg font-medium text-gray-900">
            Services ({filteredServices.length})
          </h2>
        </div>
        
        {isLoading ? (
          <div className="p-6 text-center">
            <RefreshCw className="mx-auto h-8 w-8 text-gray-400 animate-spin" />
            <p className="mt-2 text-gray-500">Chargement des services...</p>
          </div>
        ) : filteredServices.length === 0 ? (
          <div className="p-6 text-center">
            <Server className="mx-auto h-12 w-12 text-gray-400" />
            <h3 className="mt-2 text-sm font-medium text-gray-900">Aucun service trouvé</h3>
            <p className="mt-1 text-sm text-gray-500">
              Aucun service ne correspond à vos critères de recherche.
            </p>
          </div>
        ) : (
          <div className="divide-y divide-gray-200">
            {filteredServices.map((service) => {
              const isActive = service.state === 'active' || service.status === 'running'
              
              return (
                <div
                  key={`${service.type}-${service.name}`}
                  className="px-6 py-4 hover:bg-gray-50"
                >
                  <div className="flex items-center justify-between">
                    <div className="flex items-center space-x-4 flex-1">
                      {getTypeIcon(service.type)}
                      {getStateIcon(service.state)}
                      <div className="flex-1 min-w-0">
                        <div className="flex items-center gap-2">
                          <p className="text-sm font-medium text-gray-900 truncate">
                            {service.name}
                          </p>
                          <span className={`inline-flex items-center px-2.5 py-0.5 rounded-full text-xs font-medium ${getStateColor(service.state)}`}>
                            {service.state}
                          </span>
                          {service.enabled && service.type === 'systemd' && (
                            <span className="inline-flex items-center px-2.5 py-0.5 rounded-full text-xs font-medium bg-blue-100 text-blue-800">
                              Activé
                            </span>
                          )}
                        </div>
                        <p className="text-sm text-gray-500 mt-1">
                          {service.description || 'Aucune description'}
                        </p>
                        {service.type === 'docker' && service.image && (
                          <p className="text-xs text-gray-400 mt-1">
                            Image: {service.image}
                          </p>
                        )}
                      </div>
                    </div>
                    
                    <div className="flex items-center space-x-2 ml-4">
                      {isActive ? (
                        <>
                          <button
                            onClick={() => executeAction(service, 'restart')}
                            disabled={actionLoading === `${service.name}-restart`}
                            className="btn btn-sm btn-secondary"
                            title="Redémarrer"
                          >
                            {actionLoading === `${service.name}-restart` ? (
                              <Loader className="h-4 w-4 animate-spin" />
                            ) : (
                              <RefreshCw className="h-4 w-4" />
                            )}
                          </button>
                          <button
                            onClick={() => executeAction(service, 'stop')}
                            disabled={actionLoading === `${service.name}-stop`}
                            className="btn btn-sm btn-danger"
                            title="Arrêter"
                          >
                            {actionLoading === `${service.name}-stop` ? (
                              <Loader className="h-4 w-4 animate-spin" />
                            ) : (
                              <Square className="h-4 w-4" />
                            )}
                          </button>
                        </>
                      ) : (
                        <button
                          onClick={() => executeAction(service, 'start')}
                          disabled={actionLoading === `${service.name}-start`}
                          className="btn btn-sm btn-primary"
                          title="Démarrer"
                        >
                          {actionLoading === `${service.name}-start` ? (
                            <Loader className="h-4 w-4 animate-spin" />
                          ) : (
                            <Play className="h-4 w-4" />
                          )}
                        </button>
                      )}
                    </div>
                  </div>
                </div>
              )
            })}
          </div>
        )}
      </div>

      {/* Tips */}
      <div className="card p-4">
        <h3 className="text-sm font-medium text-gray-900 mb-2">💡 Conseils</h3>
        <ul className="text-sm text-gray-600 space-y-1">
          <li>• Utilisez les filtres pour afficher uniquement les services systemd ou Docker</li>
          <li>• Les services actifs peuvent être arrêtés ou redémarrés</li>
          <li>• Les services inactifs peuvent être démarrés</li>
          <li>• Les modifications prennent effet immédiatement</li>
        </ul>
      </div>
    </div>
  )
}

export default ServiceManager

