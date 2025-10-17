import React from 'react'
import { useParams } from 'react-router-dom'

const PrinterMonitor: React.FC = () => {
  const { id } = useParams<{ id: string }>()

  return (
    <div className="space-y-6">
      <div>
        <h1 className="text-2xl font-bold text-gray-900">Monitoring imprimantes - Agent {id}</h1>
        <p className="text-gray-600">Statut des imprimantes</p>
      </div>
      
      <div className="card p-6">
        <p className="text-gray-500">Monitoring imprimantes en cours de développement...</p>
      </div>
    </div>
  )
}

export default PrinterMonitor


