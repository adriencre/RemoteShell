import React, { useState, useEffect } from 'react'
import { useParams, Link } from 'react-router-dom'
import axios from 'axios'
import { 
  FolderOpen,
  File,
  Upload,
  Download,
  Trash2,
  Plus,
  RefreshCw,
  Home,
  ArrowLeft,
  AlertCircle
} from 'lucide-react'

interface FileItem {
  name: string
  path: string
  size: number
  isDir: boolean
  modified: string
  mode: string
}

const FileManager: React.FC = () => {
  const { id } = useParams<{ id: string }>()
  const [currentPath, setCurrentPath] = useState('/')
  const [files, setFiles] = useState<FileItem[]>([])
  const [isLoading, setIsLoading] = useState(false)
  const [error, setError] = useState('')
  const [showUpload, setShowUpload] = useState(false)
  const [uploadFile, setUploadFile] = useState<File | null>(null)
  const [isUploading, setIsUploading] = useState(false)

  useEffect(() => {
    loadFiles(currentPath)
  }, [currentPath, id])

  const loadFiles = async (path: string) => {
    setIsLoading(true)
    setError('')
    
    try {
      const response = await axios.get(`/api/agents/${id}/files?path=${encodeURIComponent(path)}`)
      
      // Convertir les données de l'API en format FileItem
      const apiFiles = response.data.files || []
      const convertedFiles: FileItem[] = apiFiles.map((file: any) => ({
        name: file.path.split('/').pop() || file.path,
        path: file.path,
        size: file.size || 0,
        isDir: file.is_dir || false,
        modified: file.modified || new Date().toISOString(),
        mode: formatFileMode(file.mode || 0)
      }))
      
      setFiles(convertedFiles)
    } catch (err: any) {
      setError(err.response?.data?.error || 'Erreur lors du chargement des fichiers')
    } finally {
      setIsLoading(false)
    }
  }

  const navigateToPath = (path: string) => {
    setCurrentPath(path)
  }

  const goUp = () => {
    const parentPath = currentPath.split('/').slice(0, -2).join('/') + '/'
    if (parentPath !== '/') {
      navigateToPath(parentPath)
    }
  }

  const formatFileSize = (bytes: number) => {
    if (bytes === 0) return '-'
    const sizes = ['B', 'KB', 'MB', 'GB']
    const i = Math.floor(Math.log(bytes) / Math.log(1024))
    return Math.round(bytes / Math.pow(1024, i) * 100) / 100 + ' ' + sizes[i]
  }

  const formatDate = (dateString: string) => {
    return new Date(dateString).toLocaleString()
  }

  const formatFileMode = (mode: number) => {
    // Convertir les permissions Unix en format lisible
    const type = (mode & 0xF000) === 0x4000 ? 'd' : '-'
    const owner = [
      (mode & 0x0100) ? 'r' : '-',
      (mode & 0x0080) ? 'w' : '-',
      (mode & 0x0040) ? 'x' : '-'
    ].join('')
    const group = [
      (mode & 0x0020) ? 'r' : '-',
      (mode & 0x0010) ? 'w' : '-',
      (mode & 0x0008) ? 'x' : '-'
    ].join('')
    const other = [
      (mode & 0x0004) ? 'r' : '-',
      (mode & 0x0002) ? 'w' : '-',
      (mode & 0x0001) ? 'x' : '-'
    ].join('')
    
    return type + owner + group + other
  }

  const handleFileClick = (file: FileItem) => {
    if (file.isDir) {
      navigateToPath(file.path + '/')
    } else {
      // Télécharger le fichier
      downloadFile(file.path)
    }
  }

  const downloadFile = async (filePath: string) => {
    try {
      const response = await axios.get(`/api/agents/${id}/files/download?path=${encodeURIComponent(filePath)}`, {
        responseType: 'blob'
      })
      
      const url = window.URL.createObjectURL(new Blob([response.data]))
      const link = document.createElement('a')
      link.href = url
      link.setAttribute('download', filePath.split('/').pop() || 'file')
      document.body.appendChild(link)
      link.click()
      link.remove()
      window.URL.revokeObjectURL(url)
    } catch (err: any) {
      setError(err.response?.data?.error || 'Erreur lors du téléchargement')
    }
  }

  const deleteFile = async (filePath: string) => {
    if (!confirm('Êtes-vous sûr de vouloir supprimer ce fichier ?')) return
    
    try {
      await axios.delete(`/api/agents/${id}/files?path=${encodeURIComponent(filePath)}`)
      loadFiles(currentPath)
    } catch (err: any) {
      setError(err.response?.data?.error || 'Erreur lors de la suppression')
    }
  }

  const createDirectory = async () => {
    const name = prompt('Nom du nouveau dossier :')
    if (!name) return
    
    try {
      await axios.post(`/api/agents/${id}/files/dir`, {
        path: currentPath + name
      })
      loadFiles(currentPath)
    } catch (err: any) {
      setError(err.response?.data?.error || 'Erreur lors de la création du dossier')
    }
  }

  const handleUpload = async () => {
    if (!uploadFile) return
    
    setIsUploading(true)
    setError('') // Réinitialiser l'erreur
    const formData = new FormData()
    formData.append('file', uploadFile)
    formData.append('path', currentPath + uploadFile.name)
    
    try {
      const response = await axios.post(`/api/agents/${id}/files/upload`, formData, {
        headers: {
          'Content-Type': 'multipart/form-data'
        }
      })
      console.log('Upload réussi:', response.data)
      setUploadFile(null)
      setShowUpload(false)
      // Attendre un peu avant de rafraîchir pour s'assurer que le fichier est écrit
      setTimeout(() => {
        loadFiles(currentPath)
      }, 500)
    } catch (err: any) {
      console.error('Erreur upload:', err)
      const errorMsg = err.response?.data?.error || err.message || 'Erreur lors de l\'upload'
      setError(errorMsg)
      // Ne pas masquer le formulaire d'upload en cas d'erreur
    } finally {
      setIsUploading(false)
    }
  }

  const getFileIcon = (file: FileItem) => {
    if (file.isDir) {
      return <FolderOpen className="h-5 w-5 text-blue-500" />
    }
    return <File className="h-5 w-5 text-gray-500" />
  }

  return (
    <div className="space-y-6">
      {/* Header */}
      <div className="flex items-center justify-between">
        <div>
          <h1 className="text-2xl font-bold text-gray-900">Gestionnaire de fichiers - Agent {id}</h1>
          <p className="text-gray-600">Explorateur de fichiers</p>
        </div>
        <div className="flex items-center space-x-3">
          <button
            onClick={() => loadFiles(currentPath)}
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

      {/* Navigation */}
      <div className="card p-4">
        <div className="flex items-center space-x-2">
          <button
            onClick={() => navigateToPath('/')}
            className="btn btn-sm btn-secondary"
          >
            <Home className="h-4 w-4" />
          </button>
          <button
            onClick={goUp}
            className="btn btn-sm btn-secondary"
            disabled={currentPath === '/'}
          >
            <ArrowLeft className="h-4 w-4" />
          </button>
          <div className="flex-1">
            <input
              type="text"
              value={currentPath}
              onChange={(e) => setCurrentPath(e.target.value)}
              className="w-full px-3 py-1 border border-gray-300 rounded text-sm font-mono"
              placeholder="/chemin/vers/dossier"
            />
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

      {/* Actions */}
      <div className="card p-4">
        <div className="flex items-center space-x-3">
          <button
            onClick={createDirectory}
            className="btn btn-primary btn-sm"
          >
            <Plus className="h-4 w-4 mr-2" />
            Nouveau dossier
          </button>
          <button
            onClick={() => setShowUpload(!showUpload)}
            className="btn btn-secondary btn-sm"
          >
            <Upload className="h-4 w-4 mr-2" />
            Upload
          </button>
        </div>

        {/* Upload Form */}
        {showUpload && (
          <div className="mt-4 p-4 border border-gray-200 rounded-lg">
            <div className="flex items-center space-x-3">
              <input
                type="file"
                onChange={(e) => setUploadFile(e.target.files?.[0] || null)}
                className="flex-1"
              />
              <button
                onClick={handleUpload}
                disabled={!uploadFile || isUploading}
                className="btn btn-primary btn-sm"
              >
                {isUploading ? 'Upload...' : 'Upload'}
              </button>
            </div>
          </div>
        )}
      </div>

      {/* File List */}
      <div className="card">
        <div className="px-6 py-4 border-b border-gray-200">
          <h2 className="text-lg font-medium text-gray-900">
            Fichiers dans {currentPath}
          </h2>
        </div>
        
        {isLoading ? (
          <div className="p-6 text-center">
            <RefreshCw className="mx-auto h-8 w-8 text-gray-400 animate-spin" />
            <p className="mt-2 text-gray-500">Chargement...</p>
          </div>
        ) : files.length === 0 ? (
          <div className="p-6 text-center">
            <FolderOpen className="mx-auto h-12 w-12 text-gray-400" />
            <h3 className="mt-2 text-sm font-medium text-gray-900">Dossier vide</h3>
            <p className="mt-1 text-sm text-gray-500">
              Ce dossier ne contient aucun fichier.
            </p>
          </div>
        ) : (
          <div className="divide-y divide-gray-200">
            {files.map((file) => (
              <div
                key={file.path}
                className="px-6 py-4 hover:bg-gray-50 cursor-pointer"
                onClick={() => handleFileClick(file)}
              >
                <div className="flex items-center justify-between">
                  <div className="flex items-center space-x-3">
                    {getFileIcon(file)}
                    <div>
                      <p className="text-sm font-medium text-gray-900">{file.name}</p>
                      <p className="text-sm text-gray-500">
                        {file.mode} • {formatFileSize(file.size)} • {formatDate(file.modified)}
                      </p>
                    </div>
                  </div>
                  
                  <div className="flex items-center space-x-2">
                    {!file.isDir && (
                      <button
                        onClick={(e) => {
                          e.stopPropagation()
                          downloadFile(file.path)
                        }}
                        className="btn btn-sm btn-secondary"
                        title="Télécharger"
                      >
                        <Download className="h-4 w-4" />
                      </button>
                    )}
                    <button
                      onClick={(e) => {
                        e.stopPropagation()
                        deleteFile(file.path)
                      }}
                      className="btn btn-sm btn-danger"
                      title="Supprimer"
                    >
                      <Trash2 className="h-4 w-4" />
                    </button>
                  </div>
                </div>
              </div>
            ))}
          </div>
        )}
      </div>

      {/* Tips */}
      <div className="card p-4">
        <h3 className="text-sm font-medium text-gray-900 mb-2">💡 Conseils</h3>
        <ul className="text-sm text-gray-600 space-y-1">
          <li>• Cliquez sur un dossier pour l'ouvrir</li>
          <li>• Cliquez sur un fichier pour le télécharger</li>
          <li>• Utilisez les boutons d'action pour télécharger ou supprimer</li>
          <li>• Vous pouvez naviguer en modifiant le chemin dans la barre d'adresse</li>
        </ul>
      </div>
    </div>
  )
}

export default FileManager

