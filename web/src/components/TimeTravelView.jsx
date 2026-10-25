import React, { useState, useEffect } from 'react'
import { 
  Rewind, 
  Calendar, 
  Clock, 
  Layers, 
  Search, 
  Copy, 
  Check, 
  User, 
  GitBranch, 
  Cpu, 
  FileCode,
  Sliders,
  Table as TableIcon,
  LayoutGrid
} from 'lucide-react'

export default function TimeTravelView({ 
  namespaces, 
  initialTimestamp, 
  initialNamespace,
  onInspectSpec, 
  onViewDiff 
}) {
  const [sliderOffsetMinutes, setSliderOffsetMinutes] = useState(0)
  const [targetTimeStr, setTargetTimeStr] = useState('')
  const [selectedNamespace, setSelectedNamespace] = useState(initialNamespace || 'ALL')
  const [data, setData] = useState({ timestamp: null, total: 0, resources: [] })
  const [loading, setLoading] = useState(false)
  const [searchTerm, setSearchTerm] = useState('')
  const [copied, setCopied] = useState(false)
  const [viewMode, setViewMode] = useState('table') // 'table' or 'cards'

  // Initialize timestamp
  useEffect(() => {
    if (initialTimestamp) {
      setTargetTimeStr(new Date(initialTimestamp).toISOString())
      fetchHistoricalState(new Date(initialTimestamp).toISOString(), initialNamespace)
    } else {
      const now = new Date()
      setTargetTimeStr(now.toISOString())
      fetchHistoricalState(now.toISOString(), selectedNamespace)
    }
  }, [initialTimestamp])

  // Fetch reconstructed cluster snapshot from API
  const fetchHistoricalState = async (timestamp, ns) => {
    setLoading(true)
    try {
      const queryNs = ns && ns !== 'ALL' ? `&namespace=${encodeURIComponent(ns)}` : ''
      const res = await fetch(`/api/timetravel?t=${encodeURIComponent(timestamp)}${queryNs}`)
      if (res.ok) {
        const json = await res.json()
        setData(json)
      } else {
        setData({ timestamp, total: 0, resources: [] })
      }
    } catch (err) {
      console.error('Failed to reconstruct cluster state:', err)
      setData({ timestamp, total: 0, resources: [] })
    } finally {
      setLoading(false)
    }
  }

  // Handle slider scrub (offset in minutes from NOW)
  const handleSliderChange = (minutesOffset) => {
    setSliderOffsetMinutes(minutesOffset)
    const targetDate = new Date(Date.now() + minutesOffset * 60 * 1000)
    const iso = targetDate.toISOString()
    setTargetTimeStr(iso)
    fetchHistoricalState(iso, selectedNamespace)
  }

  // Handle direct manual date submission
  const handleCustomTimeSubmit = (e) => {
    e.preventDefault()
    if (!targetTimeStr) return
    fetchHistoricalState(targetTimeStr, selectedNamespace)
  }

  // Filter resources by name / kind / manager
  const filteredResources = (data.resources || []).filter((r) => {
    if (searchTerm) {
      const q = searchTerm.toLowerCase()
      const matchName = r.name?.toLowerCase().includes(q)
      const matchKind = r.kind?.toLowerCase().includes(q)
      const matchNs = r.namespace?.toLowerCase().includes(q)
      const matchMgr = r.manager_name?.toLowerCase().includes(q)
      if (!matchName && !matchKind && !matchNs && !matchMgr) return false
    }
    return true
  })

  // Export full reconstructed snapshot manifest
  const copySnapshot = () => {
    if (!data.resources || data.resources.length === 0) return
    const combinedYAML = data.resources
      .map(r => r.raw_yaml || `# ${r.kind}/${r.name}`)
      .join('\n---\n')
    navigator.clipboard.writeText(combinedYAML)
    setCopied(true)
    setTimeout(() => setCopied(false), 2000)
  }

  return (
    <div className="space-y-4 font-mono text-xs">
      {/* Time Travel Controls Card */}
      <div className="bg-[#121215] border border-zinc-800 p-3 sm:p-4 rounded space-y-3 sm:space-y-4">
        <div className="flex flex-col sm:flex-row sm:items-center justify-between gap-2.5 border-b border-zinc-800/80 pb-3">
          <div className="flex items-center space-x-2">
            <div className="p-1.5 rounded bg-orange-950/60 border border-orange-800 text-orange-400 shrink-0">
              <Rewind className="w-4 h-4" />
            </div>
            <div>
              <h2 className="text-xs sm:text-sm font-bold text-zinc-100 uppercase tracking-wide">
                Point-in-Time Cluster Reconstruction
              </h2>
              <p className="text-[10px] sm:text-[11px] text-zinc-500">
                Reconstruct any resource or entire namespace at historical timestamp T
              </p>
            </div>
          </div>

          {/* Quick presets */}
          <div className="flex flex-wrap items-center gap-1.5">
            <span className="text-[10px] text-zinc-500 uppercase mr-1">Jump:</span>
            {[
              { label: 'NOW', val: 0 },
              { label: '-5m', val: -5 },
              { label: '-15m', val: -15 },
              { label: '-30m', val: -30 },
              { label: '-1h', val: -60 },
              { label: '-2h', val: -120 },
            ].map(p => (
              <button
                key={p.label}
                onClick={() => handleSliderChange(p.val)}
                className={`px-2 py-0.5 sm:py-1 rounded text-[10px] sm:text-[11px] transition ${
                  sliderOffsetMinutes === p.val
                    ? 'bg-orange-950 text-orange-300 border border-orange-700 font-bold'
                    : 'bg-[#09090b] text-zinc-400 hover:text-zinc-200 border border-zinc-800'
                }`}
              >
                {p.label}
              </button>
            ))}
          </div>
        </div>

        {/* Timeline Range Slider */}
        <div className="space-y-1.5 pt-1">
          <div className="flex items-center justify-between text-[10px] sm:text-[11px] text-zinc-400">
            <span>2h ago (-120m)</span>
            <span className="text-orange-400 font-bold">
              {sliderOffsetMinutes === 0 ? 'Cluster HEAD (NOW)' : `${Math.abs(sliderOffsetMinutes)} minutes ago`}
            </span>
            <span>Current (0m)</span>
          </div>
          <input
            type="range"
            min="-120"
            max="0"
            step="1"
            value={sliderOffsetMinutes}
            onChange={(e) => handleSliderChange(parseInt(e.target.value, 10))}
            className="w-full accent-orange-500 bg-zinc-800 h-1.5 rounded cursor-pointer"
          />
        </div>

        {/* Timestamp & Namespace Form */}
        <form onSubmit={handleCustomTimeSubmit} className="flex flex-wrap items-center gap-2.5 pt-1">
          <div className="flex items-center space-x-1.5 w-full sm:w-auto">
            <Calendar className="w-3.5 h-3.5 text-zinc-500 shrink-0" />
            <span className="text-[10px] sm:text-[11px] text-zinc-400 whitespace-nowrap">Target (T):</span>
            <input
              type="text"
              value={targetTimeStr}
              onChange={(e) => setTargetTimeStr(e.target.value)}
              placeholder="2026-10-22T17:00:00Z"
              className="bg-[#09090b] border border-zinc-800 rounded px-2.5 py-1 text-zinc-200 flex-1 sm:w-60 focus:outline-none focus:border-orange-500 text-xs"
            />
          </div>

          <div className="flex items-center space-x-1.5">
            <span className="text-[10px] sm:text-[11px] text-zinc-400">NS:</span>
            <select
              value={selectedNamespace}
              onChange={(e) => setSelectedNamespace(e.target.value)}
              className="bg-[#09090b] border border-zinc-800 rounded px-2 py-1 text-zinc-200 focus:outline-none focus:border-zinc-700 text-xs"
            >
              <option value="ALL">All Namespaces</option>
              {namespaces.map(ns => (
                <option key={ns} value={ns}>{ns}</option>
              ))}
            </select>
          </div>

          <button
            type="submit"
            disabled={loading}
            className="px-3 py-1 rounded bg-zinc-800 hover:bg-zinc-700 text-orange-400 border border-zinc-700 font-semibold transition text-xs"
          >
            {loading ? 'Reconstructing...' : 'Reconstruct'}
          </button>

          {data.resources?.length > 0 && (
            <button
              type="button"
              onClick={copySnapshot}
              className="ml-auto px-2.5 py-1 rounded bg-zinc-900 hover:bg-zinc-800 text-zinc-300 border border-zinc-800 flex items-center space-x-1.5 transition text-xs"
              title="Copy multi-resource YAML manifest at timestamp T"
            >
              {copied ? <Check className="w-3.5 h-3.5 text-emerald-400" /> : <Copy className="w-3.5 h-3.5" />}
              <span>{copied ? 'Copied' : 'Export Snapshot'}</span>
            </button>
          )}
        </form>
      </div>

      {/* Reconstructed Resources Table Card */}
      <div className="bg-[#121215] border border-zinc-800 rounded overflow-hidden">
        <div className="p-2.5 sm:p-3 border-b border-zinc-800 bg-[#0c0c0e] flex flex-col sm:flex-row sm:items-center justify-between gap-2.5">
          <div className="flex flex-wrap items-center gap-2">
            <span className="text-zinc-400 text-xs font-semibold">
              ACTIVE AT T:
            </span>
            <span className="px-2 py-0.5 rounded bg-orange-950 text-orange-300 border border-orange-800 text-[10px] sm:text-[11px] font-bold">
              {data.total || 0} resources
            </span>
            <span className="text-[10px] sm:text-[11px] text-zinc-500 truncate max-w-xs">
              ({data.timestamp ? new Date(data.timestamp).toLocaleTimeString() : ''})
            </span>
          </div>

          <div className="flex items-center gap-2">
            <div className="relative flex-1 sm:w-56">
              <Search className="w-3.5 h-3.5 text-zinc-500 absolute left-2 top-1/2 -translate-y-1/2" />
              <input
                type="text"
                value={searchTerm}
                onChange={(e) => setSearchTerm(e.target.value)}
                placeholder="Filter resources..."
                className="w-full bg-[#09090b] border border-zinc-800 rounded pl-7 pr-2 py-1 text-xs text-zinc-300 placeholder-zinc-600 focus:outline-none focus:border-zinc-700"
              />
            </div>

            {/* View Mode Switcher */}
            <div className="flex items-center bg-[#09090b] border border-zinc-800 rounded p-0.5 shrink-0">
              <button
                onClick={() => setViewMode('table')}
                className={`p-1 sm:px-2 sm:py-0.5 rounded text-[11px] flex items-center space-x-1 transition ${
                  viewMode === 'table'
                    ? 'bg-zinc-800 text-orange-400 font-semibold shadow-none'
                    : 'text-zinc-500 hover:text-zinc-300'
                }`}
                title="Table View"
              >
                <TableIcon className="w-3.5 h-3.5" />
                <span className="hidden sm:inline">Table</span>
              </button>
              <button
                onClick={() => setViewMode('cards')}
                className={`p-1 sm:px-2 sm:py-0.5 rounded text-[11px] flex items-center space-x-1 transition ${
                  viewMode === 'cards'
                    ? 'bg-zinc-800 text-orange-400 font-semibold shadow-none'
                    : 'text-zinc-500 hover:text-zinc-300'
                }`}
                title="Cards View"
              >
                <LayoutGrid className="w-3.5 h-3.5" />
                <span className="hidden sm:inline">Cards</span>
              </button>
            </div>
          </div>
        </div>

        {filteredResources.length === 0 ? (
          <div className="py-12 text-center text-zinc-500">
            {loading ? 'Reconstructing historical state...' : 'No active resources found at this timestamp.'}
          </div>
        ) : viewMode === 'cards' ? (
          /* Responsive Cards View */
          <div className="divide-y divide-zinc-900 font-mono text-xs">
            {filteredResources.map((res) => {
              const isHuman = res.actor_type === 'Human'
              const isGitOps = res.actor_type === 'GitOps'

              return (
                <div key={res.resource_id} className="p-3 sm:p-4 hover:bg-zinc-800/40 transition flex flex-col gap-2">
                  <div className="flex flex-wrap items-center justify-between gap-1.5">
                    <div className="flex flex-wrap items-center gap-1.5">
                      <span className="font-bold text-zinc-100 text-sm">
                        {res.kind} <span className="text-zinc-500">/</span> <span className="text-orange-300 break-all">{res.name}</span>
                      </span>
                      <span className="px-1.5 py-0.5 rounded bg-zinc-900 border border-zinc-800 text-[10px] text-zinc-400">
                        ns: {res.namespace}
                      </span>
                    </div>

                    <button
                      onClick={() => onInspectSpec({
                        name: res.name,
                        kind: res.kind,
                        namespace: res.namespace,
                        timestamp: res.timestamp,
                        manager_name: res.manager_name,
                        actor_type: res.actor_type,
                        raw_json: res.raw_json,
                        yaml: res.raw_yaml,
                      })}
                      className="px-2.5 py-1 rounded bg-zinc-900 hover:bg-zinc-800 text-orange-400 border border-orange-900/60 hover:border-orange-600 text-xs font-semibold transition shrink-0 ml-auto sm:ml-0"
                    >
                      View YAML
                    </button>
                  </div>

                  <div className="flex flex-wrap items-center gap-2 text-[11px] text-zinc-400">
                    {/* Actor Badge */}
                    {isHuman ? (
                      <span className="inline-flex items-center space-x-1 px-1.5 py-0.5 rounded text-[10px] bg-amber-950/60 text-amber-300 border border-amber-800">
                        <User className="w-2.5 h-2.5" />
                        <span>Human</span>
                      </span>
                    ) : isGitOps ? (
                      <span className="inline-flex items-center space-x-1 px-1.5 py-0.5 rounded text-[10px] bg-purple-950/60 text-purple-300 border border-purple-800">
                        <GitBranch className="w-2.5 h-2.5" />
                        <span>GitOps</span>
                      </span>
                    ) : (
                      <span className="inline-flex items-center space-x-1 px-1.5 py-0.5 rounded text-[10px] bg-zinc-900 text-zinc-400 border border-zinc-800">
                        <Cpu className="w-2.5 h-2.5" />
                        <span>Controller</span>
                      </span>
                    )}

                    <span>Manager: <span className="text-zinc-300">{res.manager_name}</span></span>
                    <span>•</span>
                    <span className="text-zinc-500">Effective: {new Date(res.timestamp).toLocaleTimeString([], { hour: '2-digit', minute: '2-digit', second: '2-digit', hour12: false })}</span>
                  </div>
                </div>
              )
            })}
          </div>
        ) : (
          /* Table View with horizontal scroll and sticky Inspect */
          <div className="overflow-x-auto w-full">
            <table className="w-full text-left font-mono text-xs border-collapse min-w-[760px]">
              <thead>
                <tr className="border-b border-zinc-800 bg-[#0c0c0e] text-zinc-500 text-[10px] uppercase tracking-wider">
                  <th className="py-2.5 px-3 w-32">Namespace</th>
                  <th className="py-2.5 px-3 w-28">Kind</th>
                  <th className="py-2.5 px-3 min-w-[150px]">Resource Name</th>
                  <th className="py-2.5 px-3 w-24">Last Actor</th>
                  <th className="py-2.5 px-3 w-28">Manager</th>
                  <th className="py-2.5 px-3 w-36">Effective Timestamp</th>
                  <th className="py-2.5 px-3 text-right w-28 sticky right-0 bg-[#0c0c0e] shadow-[-6px_0_10px_rgba(0,0,0,0.5)] z-20">Inspect</th>
                </tr>
              </thead>
              <tbody className="divide-y divide-zinc-900">
                {filteredResources.map((res) => {
                  const isHuman = res.actor_type === 'Human'
                  const isGitOps = res.actor_type === 'GitOps'

                  return (
                    <tr key={res.resource_id} className="hover:bg-zinc-800/40 transition group">
                      <td className="py-2 px-3 text-zinc-400">
                        <span className="px-1.5 py-0.5 rounded bg-zinc-900 border border-zinc-800 text-[11px]">
                          {res.namespace}
                        </span>
                      </td>
                      <td className="py-2 px-3 font-semibold text-zinc-200">
                        {res.kind}
                      </td>
                      <td className="py-2 px-3 font-semibold text-zinc-100 max-w-xs truncate" title={res.name}>
                        {res.name}
                      </td>
                      <td className="py-2 px-3 whitespace-nowrap">
                        {isHuman ? (
                          <span className="inline-flex items-center space-x-1 px-1.5 py-0.5 rounded text-[10px] bg-amber-950/60 text-amber-300 border border-amber-800">
                            <User className="w-2.5 h-2.5" />
                            <span>Human</span>
                          </span>
                        ) : isGitOps ? (
                          <span className="inline-flex items-center space-x-1 px-1.5 py-0.5 rounded text-[10px] bg-purple-950/60 text-purple-300 border border-purple-800">
                            <GitBranch className="w-2.5 h-2.5" />
                            <span>GitOps</span>
                          </span>
                        ) : (
                          <span className="inline-flex items-center space-x-1 px-1.5 py-0.5 rounded text-[10px] bg-zinc-900 text-zinc-400 border border-zinc-800">
                            <Cpu className="w-2.5 h-2.5" />
                            <span>Controller</span>
                          </span>
                        )}
                      </td>
                      <td className="py-2 px-3 text-zinc-400 text-[11px]">
                        {res.manager_name}
                      </td>
                      <td className="py-2 px-3 text-zinc-400 text-[11px] whitespace-nowrap">
                        {new Date(res.timestamp).toLocaleTimeString([], { hour: '2-digit', minute: '2-digit', second: '2-digit', hour12: false })}
                      </td>
                      <td className="py-2 px-3 text-right space-x-1.5 sticky right-0 bg-[#121215] group-hover:bg-[#1a1a1f] shadow-[-6px_0_10px_rgba(0,0,0,0.5)] z-10">
                        <button
                          onClick={() => onInspectSpec({
                            name: res.name,
                            kind: res.kind,
                            namespace: res.namespace,
                            timestamp: res.timestamp,
                            manager_name: res.manager_name,
                            actor_type: res.actor_type,
                            raw_json: res.raw_json,
                            yaml: res.raw_yaml,
                          })}
                          className="px-2 py-0.5 rounded bg-zinc-900 hover:bg-zinc-800 text-orange-400 border border-zinc-700 text-[11px] transition"
                          title="View spec YAML active at timestamp T"
                        >
                          YAML
                        </button>
                      </td>
                    </tr>
                  )
                })}
              </tbody>
            </table>
          </div>
        )}
      </div>
    </div>
  )
}
