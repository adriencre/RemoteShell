import React, { useState, useEffect } from 'react'
import { Link } from 'react-router-dom'
import axios from 'axios'
import { 
  Server, 
  Printer, 
  Terminal,
  RefreshCw,
  AlertCircle,
  CheckCircle,
  Search,
  Eye,
  Edit,
  ChevronDown,
  ChevronRight,
  Building2,
  Store,
  X,
  Save
} from 'lucide-react'

interface Agent {
  id: string
  name: string
  last_seen: string
  active: boolean
  printers: number
  franchise: string
  category: string
  system_info?: {
    hostname: string
    os: string
    arch: string
    uptime: number
    memory_total: number
    memory_used: number
    cpu_cores?: number
  }
}

interface GroupedAgents {
  [franchise: string]: {
    [category: string]: Agent[]
  }
}

const Agents: React.FC = () => {
  const [agents, setAgents] = useState<Agent[]>([])
  const [groupedAgents, setGroupedAgents] = useState<GroupedAgents>({})
  const [isLoading, setIsLoading] = useState(true)
  const [error, setError] = useState('')
  const [lastUpdate, setLastUpdate] = useState<Date>(new Date())
  const [searchTerm, setSearchTerm] = useState('')
  const [expandedFranchises, setExpandedFranchises] = useState<Set<string>>(new Set())
  const [editingAgent, setEditingAgent] = useState<string | null>(null)
  const [editFranchise, setEditFranchise] = useState('')
  const [editCategory, setEditCategory] = useState('')
  const [showUnassigned, setShowUnassigned] = useState(true)

  const fetchAgents = async () => {
    try {
      const response = await axios.get('/api/agents')
      const agentsList = response.data.agents || []
      setAgents(agentsList)
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
    const interval = setInterval(fetchAgents, 30000)
    return () => clearInterval(interval)
  }, [])

  useEffect(() => {
    groupAgents()
  }, [agents, searchTerm])

  const groupAgents = () => {
    let filtered = agents

    // Filtre par recherche
    if (searchTerm) {
      const term = searchTerm.toLowerCase()
      filtered = filtered.filter(a => 
        a.name.toLowerCase().includes(term) ||
        a.id.toLowerCase().includes(term) ||
        a.franchise?.toLowerCase().includes(term) ||
        a.category?.toLowerCase().includes(term) ||
        a.system_info?.hostname?.toLowerCase().includes(term)
      )
    }

    // Grouper par franchise puis par catégorie
    const grouped: GroupedAgents = {}
    const unassigned: Agent[] = []

    filtered.forEach(agent => {
      const franchise = agent.franchise || 'Non assigné'
      const category = agent.category || 'Non assigné'

      if (!agent.franchise && !agent.category) {
        unassigned.push(agent)
      } else {
        if (!grouped[franchise]) {
          grouped[franchise] = {}
        }
        if (!grouped[franchise][category]) {
          grouped[franchise][category] = []
        }
        grouped[franchise][category].push(agent)
      }
    })

    // Ajouter les non assignés si nécessaire
    if (unassigned.length > 0 && (showUnassigned || !searchTerm)) {
      if (!grouped['Non assigné']) {
        grouped['Non assigné'] = {}
      }
      if (!grouped['Non assigné']['Non assigné']) {
        grouped['Non assigné']['Non assigné'] = []
      }
      grouped['Non assigné']['Non assigné'] = unassigned
    }

    setGroupedAgents(grouped)
  }

  const toggleFranchise = (franchise: string) => {
    const newExpanded = new Set(expandedFranchises)
    if (newExpanded.has(franchise)) {
      newExpanded.delete(franchise)
    } else {
      newExpanded.add(franchise)
    }
    setExpandedFranchises(newExpanded)
  }

  const startEdit = (agent: Agent) => {
    setEditingAgent(agent.id)
    setEditFranchise(agent.franchise || '')
    setEditCategory(agent.category || '')
  }

  const cancelEdit = () => {
    setEditingAgent(null)
    setEditFranchise('')
    setEditCategory('')
  }

  const saveEdit = async (agentId: string) => {
    try {
      await axios.put(`/api/agents/${agentId}/metadata`, {
        franchise: editFranchise || '',
        category: editCategory || ''
      })
      await fetchAgents()
      setEditingAgent(null)
    } catch (err) {
      console.error('Erreur lors de la mise à jour:', err)
      alert('Erreur lors de la mise à jour des métadonnées')
    }
  }

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

  const getTimeAgo = (dateString: string) => {
    const date = new Date(dateString)
    const now = new Date()
    const diff = Math.floor((now.getTime() - date.getTime()) / 1000)
    if (diff < 60) return 'À l\'instant'
    if (diff < 3600) return `Il y a ${Math.floor(diff / 60)} min`
    if (diff < 86400) return `Il y a ${Math.floor(diff / 3600)} h`
    return `Il y a ${Math.floor(diff / 86400)} j`
  }

  // Obtenir les franchises triées
  const franchises = Object.keys(groupedAgents).sort()

  // Compter les agents par franchise
  const getFranchiseCount = (franchise: string) => {
    const categories = groupedAgents[franchise] || {}
    return Object.values(categories).reduce((sum, agents) => sum + agents.length, 0)
  }

  if (isLoading && agents.length === 0) {
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
          <h1 className="text-2xl font-bold text-gray-900">Agents par Franchise</h1>
          <p className="text-gray-600">
            {franchises.length} franchise{franchises.length !== 1 ? 's' : ''} • {agents.length} agent{agents.length !== 1 ? 's' : ''}
            {lastUpdate && ` • Dernière mise à jour: ${lastUpdate.toLocaleTimeString()}`}
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

      {/* Search */}
      <div className="card p-4">
        <div className="relative">
          <Search className="absolute left-3 top-1/2 transform -translate-y-1/2 h-5 w-5 text-gray-400" />
          <input
            type="text"
            placeholder="Rechercher un agent, une franchise ou une catégorie..."
            value={searchTerm}
            onChange={(e) => setSearchTerm(e.target.value)}
            className="input pl-10 w-full"
          />
          {searchTerm && (
            <button
              onClick={() => setSearchTerm('')}
              className="absolute right-3 top-1/2 transform -translate-y-1/2 text-gray-400 hover:text-gray-600"
            >
              <X className="h-4 w-4" />
            </button>
          )}
        </div>
        <div className="mt-3 flex items-center">
          <input
            type="checkbox"
            id="showUnassigned"
            checked={showUnassigned}
            onChange={(e) => setShowUnassigned(e.target.checked)}
            className="mr-2"
          />
          <label htmlFor="showUnassigned" className="text-sm text-gray-600">
            Afficher les agents non assignés
          </label>
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

      {/* Agents grouped by Franchise/Category */}
      {franchises.length === 0 ? (
        <div className="card p-12 text-center">
          <Server className="mx-auto h-12 w-12 text-gray-400" />
          <h3 className="mt-2 text-sm font-medium text-gray-900">Aucun agent</h3>
          <p className="mt-1 text-sm text-gray-500">
            {agents.length === 0
              ? 'Aucun serveur d\'impression n\'est actuellement connecté.'
              : 'Essayez de modifier vos critères de recherche.'}
          </p>
        </div>
      ) : (
        <div className="space-y-4">
          {franchises.map((franchise) => {
            const isExpanded = expandedFranchises.has(franchise)
            const categories = Object.keys(groupedAgents[franchise] || {}).sort()
            const franchiseCount = getFranchiseCount(franchise)

            return (
              <div key={franchise} className="card">
                {/* Franchise Header */}
                <div
                  className="px-6 py-4 border-b border-gray-200 cursor-pointer hover:bg-gray-50 transition-colors"
                  onClick={() => toggleFranchise(franchise)}
                >
                  <div className="flex items-center justify-between">
                    <div className="flex items-center space-x-3">
                      {isExpanded ? (
                        <ChevronDown className="h-5 w-5 text-gray-400" />
                      ) : (
                        <ChevronRight className="h-5 w-5 text-gray-400" />
                      )}
                      <Building2 className="h-5 w-5 text-primary-600" />
                      <div>
                        <h3 className="text-lg font-semibold text-gray-900">
                          {franchise}
                        </h3>
                        <p className="text-sm text-gray-500">
                          {franchiseCount} agent{franchiseCount !== 1 ? 's' : ''} • {categories.length} catégorie{categories.length !== 1 ? 's' : ''}
                        </p>
                      </div>
                    </div>
                  </div>
                </div>

                {/* Categories */}
                {isExpanded && (
                  <div className="divide-y divide-gray-200">
                    {categories.map((category) => {
                      const categoryAgents = groupedAgents[franchise][category] || []
                      const activeCount = categoryAgents.filter(a => a.active).length

                      return (
                        <div key={category} className="px-6 py-4">
                          {/* Category Header */}
                          <div className="flex items-center justify-between mb-3">
                            <div className="flex items-center space-x-2">
                              <Store className="h-4 w-4 text-blue-600" />
                              <h4 className="text-md font-medium text-gray-900">
                                {category}
                              </h4>
                              <span className="text-sm text-gray-500">
                                ({categoryAgents.length} agent{categoryAgents.length !== 1 ? 's' : ''})
                              </span>
                              {activeCount > 0 && (
                                <span className="text-xs text-green-600">
                                  {activeCount} actif{activeCount !== 1 ? 's' : ''}
                                </span>
                              )}
                            </div>
                          </div>

                          {/* Agents in Category */}
                          <div className="space-y-2 ml-6">
                            {categoryAgents.map((agent) => {
                              const StatusIcon = getStatusIcon(agent.active)
                              const isEditing = editingAgent === agent.id

                              return (
                                <div
                                  key={agent.id}
                                  className="flex items-center justify-between p-3 bg-gray-50 rounded-md hover:bg-gray-100 transition-colors"
                                >
                                  <div className="flex items-center space-x-3 flex-1 min-w-0">
                                    <StatusIcon className={`h-5 w-5 flex-shrink-0 ${getStatusColor(agent.active)}`} />
                                    
                                    {isEditing ? (
                                      <div className="flex items-center space-x-2 flex-1">
                                        <input
                                          type="text"
                                          placeholder="Franchise"
                                          value={editFranchise}
                                          onChange={(e) => setEditFranchise(e.target.value)}
                                          className="input flex-1 text-sm"
                                        />
                                        <input
                                          type="text"
                                          placeholder="Catégorie"
                                          value={editCategory}
                                          onChange={(e) => setEditCategory(e.target.value)}
                                          className="input flex-1 text-sm"
                                        />
                                        <button
                                          onClick={() => saveEdit(agent.id)}
                                          className="btn btn-primary btn-sm"
                                        >
                                          <Save className="h-4 w-4" />
                                        </button>
                                        <button
                                          onClick={cancelEdit}
                                          className="btn btn-secondary btn-sm"
                                        >
                                          <X className="h-4 w-4" />
                                        </button>
                                      </div>
                                    ) : (
                                      <>
                                        <div className="flex-1 min-w-0">
                                          <div className="flex items-center space-x-2">
                                            <Link
                                              to={`/agent/${agent.id}`}
                                              className="font-medium text-gray-900 hover:text-primary-600 truncate"
                                            >
                                              {agent.name}
                                            </Link>
                                            <span className={`inline-flex items-center px-2 py-0.5 rounded-full text-xs font-medium ${
                                              agent.active 
                                                ? 'bg-green-100 text-green-800' 
                                                : 'bg-red-100 text-red-800'
                                            }`}>
                                              {agent.active ? 'Actif' : 'Inactif'}
                                            </span>
                                          </div>
                                          <div className="text-xs text-gray-500 mt-1">
                                            ID: {agent.id} • {agent.system_info?.hostname || 'N/A'} • {getTimeAgo(agent.last_seen)}
                                            {agent.system_info && ` • Uptime: ${formatUptime(agent.system_info.uptime)}`}
                                          </div>
                                        </div>
                                      </>
                                    )}
                                  </div>

                                  {!isEditing && (
                                    <div className="flex items-center space-x-2 ml-4 flex-shrink-0">
                                      <button
                                        onClick={() => startEdit(agent)}
                                        className="btn btn-secondary btn-sm"
                                        title="Modifier la franchise/catégorie"
                                      >
                                        <Edit className="h-4 w-4" />
                                      </button>
                                      <Link
                                        to={`/agent/${agent.id}`}
                                        className="btn btn-secondary btn-sm"
                                        title="Voir les détails"
                                      >
                                        <Eye className="h-4 w-4" />
                                      </Link>
                                      <Link
                                        to={`/agent/${agent.id}/terminal`}
                                        className="btn btn-sm btn-secondary"
                                        title="Terminal"
                                      >
                                        <Terminal className="h-4 w-4" />
                                      </Link>
                                      <Link
                                        to={`/agent/${agent.id}/printers`}
                                        className="btn btn-sm btn-secondary"
                                        title="Imprimantes"
                                      >
                                        <Printer className="h-4 w-4" />
                                      </Link>
                                    </div>
                                  )}
                                </div>
                              )
                            })}
                          </div>
                        </div>
                      )
                    })}
                  </div>
                )}
              </div>
            )
          })}
        </div>
      )}
    </div>
  )
}

export default Agents
