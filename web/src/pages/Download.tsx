import React from 'react'
import { Download as DownloadIcon } from 'lucide-react'

const Download: React.FC = () => {
  const serverUrl = window.location.origin

  const handleDownload = () => {
    window.location.href = `${serverUrl}/download/agent`
  }

  const handleInstallScript = () => {
    window.location.href = `${serverUrl}/download/install-agent.sh`
  }

  return (
    <div className="min-h-screen bg-gray-50 flex items-center justify-center px-4">
      <div className="max-w-md w-full bg-white rounded-lg shadow-lg p-8">
        <div className="text-center">
          <div className="mx-auto flex items-center justify-center h-16 w-16 rounded-full bg-blue-100 mb-4">
            <DownloadIcon className="h-8 w-8 text-blue-600" />
          </div>
          <h1 className="text-2xl font-bold text-gray-900 mb-2">
            Télécharger l'agent RemoteShell
          </h1>
          <p className="text-gray-600 mb-6">
            Téléchargez le client agent pour vous connecter au serveur RemoteShell
          </p>
          
          <div className="space-y-3">
            <button
              onClick={handleDownload}
              className="w-full bg-blue-600 hover:bg-blue-700 text-white font-semibold py-3 px-6 rounded-lg transition-colors duration-200 flex items-center justify-center gap-2"
            >
              <DownloadIcon className="h-5 w-5" />
              Télécharger l'agent
            </button>
            
            <button
              onClick={handleInstallScript}
              className="w-full bg-green-600 hover:bg-green-700 text-white font-semibold py-3 px-6 rounded-lg transition-colors duration-200 flex items-center justify-center gap-2"
            >
              <DownloadIcon className="h-5 w-5" />
              Télécharger le script d'installation
            </button>
          </div>

          <div className="mt-8 pt-6 border-t border-gray-200">
            <h3 className="text-sm font-semibold text-gray-900 mb-3">
              Instructions d'installation
            </h3>
            <div className="text-left text-sm text-gray-600 space-y-4">
              <div>
                <p className="font-semibold text-gray-900 mb-2">🚀 Installation automatique (recommandé):</p>
                <p className="mb-2">Exécutez cette commande sur le serveur d'impression :</p>
                <code className="block bg-gray-100 p-2 rounded text-xs break-all">
                  curl -sSL {serverUrl}/download/install-agent.sh | sudo bash
                </code>
                <p className="mt-2 text-xs text-gray-500">
                  Le script téléchargera automatiquement l'agent et l'installera en service systemd.
                </p>
              </div>
              
              <div>
                <p className="font-semibold text-gray-900 mb-2">📥 Installation manuelle:</p>
                <p><strong>Linux/Mac:</strong></p>
                <ol className="list-decimal list-inside space-y-1 ml-2">
                  <li>Téléchargez le fichier agent</li>
                  <li>Rendez-le exécutable: <code className="bg-gray-100 px-1 rounded">chmod +x rms-agent</code></li>
                  <li>Exécutez: <code className="bg-gray-100 px-1 rounded">./rms-agent --server SERVEUR:PORT --id ID --name "Nom" --token TOKEN</code></li>
                </ol>
              </div>
            </div>
          </div>
        </div>
      </div>
    </div>
  )
}

export default Download

