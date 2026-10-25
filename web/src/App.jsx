import React, { useState, useEffect, useCallback } from 'react'
import Navbar from './components/Navbar.jsx'
import LandingView from './components/LandingView.jsx'
import TimelineView from './components/TimelineView.jsx'
import TimeTravelView from './components/TimeTravelView.jsx'
import ManualChangesView from './components/ManualChangesView.jsx'
import IncidentsView from './components/IncidentsView.jsx'
import DiffModal from './components/DiffModal.jsx'
import ResourceDetailModal from './components/ResourceDetailModal.jsx'

export default function App() {
  const [activeTab, setActiveTab] = useState('landing')
  const [stats, setStats] = useState(null)
  const [revisions, setRevisions] = useState([])
  const [manualChanges, setManualChanges] = useState([])
  const [incidents, setIncidents] = useState([])
  const [namespaces, setNamespaces] = useState([])
  const [sseConnected, setSseConnected] = useState(false)

  // Modals state
  const [activeDiffRevision, setActiveDiffRevision] = useState(null)
  const [activeResourceSpec, setActiveResourceSpec] = useState(null)

  // Time travel target state for navigation
  const [timeTravelTarget, setTimeTravelTarget] = useState({
    timestamp: null,
    namespace: 'production'
  })

  // Fetch initial data
  const loadData = useCallback(async () => {
    try {
      const [statusRes, revRes, manRes, incRes, nsRes] = await Promise.all([
        fetch('/api/status'),
        fetch('/api/revisions?limit=100'),
        fetch('/api/manual-changes?limit=50'),
        fetch('/api/incidents'),
        fetch('/api/namespaces'),
      ])

      if (statusRes.ok) setStats(await statusRes.json())
      if (revRes.ok) setRevisions(await revRes.json())
      if (manRes.ok) setManualChanges(await manRes.json())
      if (incRes.ok) setIncidents(await incRes.json())
      if (nsRes.ok) setNamespaces(await nsRes.json())
    } catch (err) {
      console.error('Failed to load initial data:', err)
    }
  }, [])

  useEffect(() => {
    loadData()
  }, [loadData])

  // Setup Server-Sent Events (SSE)
  useEffect(() => {
    let es
    try {
      es = new EventSource('/api/stream')
      es.onopen = () => setSseConnected(true)
      es.onerror = () => setSseConnected(false)

      es.addEventListener('REVISION_ADDED', (e) => {
        try {
          const rev = JSON.parse(e.data)
          setRevisions((prev) => [rev, ...prev.slice(0, 199)])
          if (rev.actor_type === 'Human') {
            setManualChanges((prev) => [rev, ...prev])
          }
          setStats((prev) => prev ? { ...prev, total_revisions: prev.total_revisions + 1 } : prev)
        } catch (err) {
          console.error('Error parsing SSE revision:', err)
        }
      })

      es.addEventListener('INCIDENT_ALERT', (e) => {
        try {
          const inc = JSON.parse(e.data)
          setIncidents((prev) => [inc, ...prev])
          setStats((prev) => prev ? { ...prev, unresolved_incidents: prev.unresolved_incidents + 1 } : prev)
        } catch (err) {
          console.error('Error parsing SSE incident:', err)
        }
      })
    } catch (err) {
      console.error('Failed to initialize EventSource:', err)
    }

    return () => {
      if (es) es.close()
    }
  }, [])

  // Navigation helper to jump directly to Time Travel
  const handleTimeTravelTo = (timestamp, ns) => {
    setTimeTravelTarget({
      timestamp,
      namespace: ns || 'production'
    })
    setActiveTab('timetravel')
  }

  // Inspect revision YAML
  const handleInspectRevision = async (rev) => {
    try {
      const res = await fetch(`/api/revisions/${rev.id}`)
      if (res.ok) {
        const json = await res.json()
        setActiveResourceSpec({
          ...rev,
          yaml: json.yaml,
        })
      } else {
        setActiveResourceSpec(rev)
      }
    } catch {
      setActiveResourceSpec(rev)
    }
  }

  return (
    <div className="min-h-screen bg-[#09090b] text-zinc-100 flex flex-col font-sans">
      {/* Top Navigation */}
      <Navbar
        activeTab={activeTab}
        setActiveTab={setActiveTab}
        stats={stats}
        sseConnected={sseConnected}
        incidentCount={incidents.length}
        manualCount={manualChanges.length}
        onRefresh={loadData}
      />

      {/* Main Content Area */}
      <main className="flex-1 max-w-7xl w-full mx-auto p-2.5 sm:p-4 md:p-6">
        {activeTab === 'landing' && (
          <LandingView
            stats={stats}
            onNavigate={(tab) => setActiveTab(tab)}
          />
        )}

        {activeTab === 'timeline' && (
          <TimelineView
            revisions={revisions}
            namespaces={namespaces}
            onInspectRevision={handleInspectRevision}
            onViewDiff={(rev) => setActiveDiffRevision(rev)}
            onTimeTravelTo={handleTimeTravelTo}
          />
        )}

        {activeTab === 'timetravel' && (
          <TimeTravelView
            namespaces={namespaces}
            initialTimestamp={timeTravelTarget.timestamp}
            initialNamespace={timeTravelTarget.namespace}
            onInspectSpec={(res) => setActiveResourceSpec(res)}
            onViewDiff={(res) => setActiveDiffRevision(res)}
          />
        )}

        {activeTab === 'manual' && (
          <ManualChangesView
            manualChanges={manualChanges}
            onViewDiff={(ch) => setActiveDiffRevision(ch)}
            onInspectRevision={handleInspectRevision}
          />
        )}

        {activeTab === 'incidents' && (
          <IncidentsView
            incidents={incidents}
            onViewDiff={(rev) => setActiveDiffRevision(rev)}
            onInspectRevision={handleInspectRevision}
            onTimeTravelTo={handleTimeTravelTo}
          />
        )}
      </main>

      {/* Footer */}
      <footer className="border-t border-zinc-900 bg-[#09090b] py-3.5 px-4 text-center font-mono text-[11px] text-zinc-500">
        Cluster Time Machine • In-Cluster Kubernetes Audit Recorder & Point-in-Time RCA Reconstructor
      </footer>

      {/* Modals */}
      {activeDiffRevision && (
        <DiffModal
          revision={activeDiffRevision}
          onClose={() => setActiveDiffRevision(null)}
        />
      )}

      {activeResourceSpec && (
        <ResourceDetailModal
          resource={activeResourceSpec}
          onClose={() => setActiveResourceSpec(null)}
        />
      )}
    </div>
  )
}
