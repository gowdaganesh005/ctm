import React, { useState, useMemo } from 'react'
import { 
  User, 
  GitBranch, 
  Cpu, 
  Plus, 
  FileEdit, 
  Trash2, 
  Search, 
  Rewind,
  Clock,
  LayoutGrid,
  Table as TableIcon
} from 'lucide-react'

export default function TimelineView({ 
  revisions, 
  namespaces, 
  onInspectRevision, 
  onViewDiff, 
  onTimeTravelTo 
}) {
  const [searchTerm, setSearchTerm] = useState('')
  const [selectedNamespace, setSelectedNamespace] = useState('ALL')
  const [selectedKind, setSelectedKind] = useState('ALL')
  const [selectedActor, setSelectedActor] = useState('ALL')
  const [viewMode, setViewMode] = useState('table') // 'table' or 'cards'

  // Extract unique kinds
  const kinds = useMemo(() => {
    const s = new Set()
    revisions.forEach(r => {
      if (r.kind) s.add(r.kind)
    })
    return Array.from(s).sort()
  }, [revisions])

  // Filter revisions
  const filteredRevisions = useMemo(() => {
    return revisions.filter(r => {
      if (selectedNamespace !== 'ALL' && r.namespace !== selectedNamespace) return false
      if (selectedKind !== 'ALL' && r.kind !== selectedKind) return false
      if (selectedActor !== 'ALL' && r.actor_type !== selectedActor) return false

      if (searchTerm) {
        const q = searchTerm.toLowerCase()
        const matchName = r.name?.toLowerCase().includes(q)
        const matchKind = r.kind?.toLowerCase().includes(q)
        const matchMgr = r.manager_name?.toLowerCase().includes(q)
        const matchNs = r.namespace?.toLowerCase().includes(q)
        if (!matchName && !matchKind && !matchMgr && !matchNs) return false
      }
      return true
    })
  }, [revisions, selectedNamespace, selectedKind, selectedActor, searchTerm])

  const actorStats = useMemo(() => {
    let human = 0, gitops = 0, controller = 0
    revisions.forEach(r => {
      if (r.actor_type === 'Human') human++
      else if (r.actor_type === 'GitOps') gitops++
      else controller++
    })
    return { human, gitops, controller, total: revisions.length }
  }, [revisions])

  const formatTime = (ts) => {
    if (!ts) return ''
    try {
      const d = new Date(ts)
      return d.toLocaleTimeString([], { hour: '2-digit', minute: '2-digit', second: '2-digit', hour12: false })
    } catch {
      return ts
    }
  }

  const formatRelative = (ts) => {
    if (!ts) return ''
    try {
      const diffMs = Date.now() - new Date(ts).getTime()
      const diffSec = Math.floor(diffMs / 1000)
      if (diffSec < 60) return `${diffSec}s ago`
      const diffMin = Math.floor(diffSec / 60)
      if (diffMin < 60) return `${diffMin}m ago`
      const diffHr = Math.floor(diffMin / 60)
      return `${diffHr}h ago`
    } catch {
      return ''
    }
  }

  return (
    <div className="space-y-4">
      {/* Metric Cards Banner - 2 cols on mobile, 4 cols on lg */}
      <div className="grid grid-cols-2 lg:grid-cols-4 gap-2.5 sm:gap-3 text-xs font-mono">
        <div className="bg-[#121215] border border-zinc-800 p-2.5 sm:p-3 rounded">
          <div className="text-zinc-500 uppercase tracking-wider text-[10px]">Total Recorded Revisions</div>
          <div className="text-lg sm:text-xl font-bold text-zinc-100 mt-1">{actorStats.total}</div>
          <div className="text-[10px] text-zinc-500 mt-0.5 truncate">Append-only audit log</div>
        </div>

        <div className="bg-[#121215] border border-amber-950/60 p-2.5 sm:p-3 rounded">
          <div className="text-amber-500/80 uppercase tracking-wider text-[10px] flex items-center justify-between">
            <span>Human Direct Edits</span>
            <User className="w-3.5 h-3.5 text-amber-400 shrink-0" />
          </div>
          <div className="text-lg sm:text-xl font-bold text-amber-300 mt-1">{actorStats.human}</div>
          <div className="text-[10px] text-amber-500/70 mt-0.5 truncate">kubectl / helm overrides</div>
        </div>

        <div className="bg-[#121215] border border-purple-950/60 p-2.5 sm:p-3 rounded">
          <div className="text-purple-400/80 uppercase tracking-wider text-[10px] flex items-center justify-between">
            <span>GitOps Deployments</span>
            <GitBranch className="w-3.5 h-3.5 text-purple-400 shrink-0" />
          </div>
          <div className="text-lg sm:text-xl font-bold text-purple-300 mt-1">{actorStats.gitops}</div>
          <div className="text-[10px] text-purple-400/70 mt-0.5 truncate">ArgoCD / Helm / Flux</div>
        </div>

        <div className="bg-[#121215] border border-zinc-800 p-2.5 sm:p-3 rounded">
          <div className="text-zinc-400 uppercase tracking-wider text-[10px] flex items-center justify-between">
            <span>Controllers & System</span>
            <Cpu className="w-3.5 h-3.5 text-zinc-500 shrink-0" />
          </div>
          <div className="text-lg sm:text-xl font-bold text-zinc-300 mt-1">{actorStats.controller}</div>
          <div className="text-[10px] text-zinc-500 mt-0.5 truncate">Kubelet / Controllers</div>
        </div>
      </div>

      {/* Filter Toolbar */}
      <div className="bg-[#121215] border border-zinc-800 p-2.5 sm:p-3 rounded flex flex-col md:flex-row md:items-center justify-between gap-2.5 font-mono text-xs">
        {/* Search Input */}
        <div className="relative w-full md:flex-1 md:min-w-[200px]">
          <Search className="w-3.5 h-3.5 text-zinc-500 absolute left-2.5 top-1/2 -translate-y-1/2" />
          <input
            type="text"
            value={searchTerm}
            onChange={(e) => setSearchTerm(e.target.value)}
            placeholder="Search resource, kind, manager..."
            className="w-full bg-[#09090b] border border-zinc-800 rounded pl-8 pr-3 py-1.5 text-zinc-200 placeholder-zinc-600 focus:outline-none focus:border-orange-500 transition text-xs"
          />
        </div>

        {/* Filter Dropdowns and View Switcher */}
        <div className="flex flex-wrap items-center justify-between md:justify-end gap-2">
          <div className="flex flex-wrap items-center gap-2">
            {/* Namespace */}
            <div className="flex items-center space-x-1">
              <span className="text-[10px] sm:text-[11px] text-zinc-500">NS:</span>
              <select
                value={selectedNamespace}
                onChange={(e) => setSelectedNamespace(e.target.value)}
                className="bg-[#09090b] border border-zinc-800 rounded px-1.5 sm:px-2 py-1 text-zinc-300 text-xs focus:outline-none focus:border-zinc-700 max-w-[120px] sm:max-w-none"
              >
                <option value="ALL">All Namespaces</option>
                {namespaces.map(ns => (
                  <option key={ns} value={ns}>{ns}</option>
                ))}
              </select>
            </div>

            {/* Kind */}
            <div className="flex items-center space-x-1">
              <span className="text-[10px] sm:text-[11px] text-zinc-500">KIND:</span>
              <select
                value={selectedKind}
                onChange={(e) => setSelectedKind(e.target.value)}
                className="bg-[#09090b] border border-zinc-800 rounded px-1.5 sm:px-2 py-1 text-zinc-300 text-xs focus:outline-none focus:border-zinc-700 max-w-[110px] sm:max-w-none"
              >
                <option value="ALL">All Kinds</option>
                {kinds.map(k => (
                  <option key={k} value={k}>{k}</option>
                ))}
              </select>
            </div>

            {/* Actor */}
            <div className="flex items-center space-x-1">
              <span className="text-[10px] sm:text-[11px] text-zinc-500">ACTOR:</span>
              <select
                value={selectedActor}
                onChange={(e) => setSelectedActor(e.target.value)}
                className="bg-[#09090b] border border-zinc-800 rounded px-1.5 sm:px-2 py-1 text-zinc-300 text-xs focus:outline-none focus:border-zinc-700 max-w-[110px] sm:max-w-none"
              >
                <option value="ALL">All</option>
                <option value="Human">Human</option>
                <option value="GitOps">GitOps</option>
                <option value="Controller">Controller</option>
              </select>
            </div>
          </div>

          {/* View Mode Toggle (Table / Responsive Cards) */}
          <div className="flex items-center bg-[#09090b] border border-zinc-800 rounded p-0.5 ml-auto md:ml-0">
            <button
              onClick={() => setViewMode('table')}
              className={`p-1 sm:px-2 sm:py-0.5 rounded text-[11px] flex items-center space-x-1 transition ${
                viewMode === 'table'
                  ? 'bg-zinc-800 text-orange-400 font-semibold shadow-none'
                  : 'text-zinc-500 hover:text-zinc-300'
              }`}
              title="Table View (with horizontal scroll)"
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
              title="Cards View (best for small screens)"
            >
              <LayoutGrid className="w-3.5 h-3.5" />
              <span className="hidden sm:inline">Cards</span>
            </button>
          </div>
        </div>
      </div>

      <div className="flex items-center justify-between text-[11px] text-zinc-500 px-1">
        <div>
          Showing <span className="text-zinc-200 font-semibold">{filteredRevisions.length}</span> of {revisions.length} revisions
        </div>
        <div className="text-[10px] text-zinc-600 hidden sm:block">
          {viewMode === 'table' ? 'Scroll horizontally if needed • Sticky actions anchored' : 'Responsive card stream'}
        </div>
      </div>

      {/* Revisions Stream Container */}
      <div className="bg-[#121215] border border-zinc-800 rounded overflow-hidden">
        {filteredRevisions.length === 0 ? (
          <div className="py-12 text-center text-zinc-500 font-mono text-xs">
            No revisions match your current filter criteria.
          </div>
        ) : viewMode === 'cards' ? (
          /* Cards View - Fully responsive, zero horizontal scroll required */
          <div className="divide-y divide-zinc-900 font-mono text-xs">
            {filteredRevisions.map((rev) => {
              const isHuman = rev.actor_type === 'Human'
              const isGitOps = rev.actor_type === 'GitOps'

              return (
                <div 
                  key={rev.id} 
                  className={`p-3 sm:p-4 hover:bg-zinc-800/40 transition flex flex-col gap-2.5 ${
                    isHuman ? 'bg-amber-950/10' : ''
                  }`}
                >
                  {/* Top line: Badges & Timestamp */}
                  <div className="flex flex-wrap items-center justify-between gap-1.5">
                    <div className="flex flex-wrap items-center gap-1.5">
                      {/* Actor Badge */}
                      {isHuman ? (
                        <span className="inline-flex items-center space-x-1 px-1.5 py-0.5 rounded text-[10px] bg-amber-950/70 text-amber-300 border border-amber-800">
                          <User className="w-2.5 h-2.5" />
                          <span>Human</span>
                        </span>
                      ) : isGitOps ? (
                        <span className="inline-flex items-center space-x-1 px-1.5 py-0.5 rounded text-[10px] bg-purple-950/70 text-purple-300 border border-purple-800">
                          <GitBranch className="w-2.5 h-2.5" />
                          <span>GitOps</span>
                        </span>
                      ) : (
                        <span className="inline-flex items-center space-x-1 px-1.5 py-0.5 rounded text-[10px] bg-zinc-900 text-zinc-400 border border-zinc-800">
                          <Cpu className="w-2.5 h-2.5" />
                          <span>Controller</span>
                        </span>
                      )}

                      {/* Action Type */}
                      {rev.change_type === 'ADDED' && (
                        <span className="inline-flex items-center space-x-1 px-1.5 py-0.5 rounded text-[10px] bg-emerald-950/60 text-emerald-300 border border-emerald-800">
                          <Plus className="w-2.5 h-2.5" />
                          <span>ADDED</span>
                        </span>
                      )}
                      {rev.change_type === 'MODIFIED' && (
                        <span className="inline-flex items-center space-x-1 px-1.5 py-0.5 rounded text-[10px] bg-sky-950/60 text-sky-300 border border-sky-800">
                          <FileEdit className="w-2.5 h-2.5" />
                          <span>MODIFIED</span>
                        </span>
                      )}
                      {rev.change_type === 'DELETED' && (
                        <span className="inline-flex items-center space-x-1 px-1.5 py-0.5 rounded text-[10px] bg-rose-950/60 text-rose-300 border border-rose-800">
                          <Trash2 className="w-2.5 h-2.5" />
                          <span>DELETED</span>
                        </span>
                      )}

                      {/* Manager */}
                      <span className="text-[10px] bg-zinc-900 px-1.5 py-0.5 rounded text-zinc-400 border border-zinc-800">
                        {rev.manager_name || 'unknown'}
                      </span>
                    </div>

                    {/* Timestamp */}
                    <div className="text-[11px] text-zinc-400 flex items-center space-x-1.5">
                      <span className="font-semibold text-zinc-300">{formatTime(rev.timestamp)}</span>
                      <span className="text-[10px] text-zinc-500">({formatRelative(rev.timestamp)})</span>
                    </div>
                  </div>

                  {/* Middle line: Kind & Resource Name + Namespace */}
                  <div className="flex flex-wrap items-center justify-between gap-2">
                    <div className="flex flex-wrap items-center gap-1.5 min-w-0">
                      <span className="font-bold text-zinc-200 text-sm">
                        {rev.kind} <span className="text-zinc-500">/</span> <span className="text-orange-300 font-mono break-all">{rev.name}</span>
                      </span>
                      <span className="text-[10px] px-1.5 py-0.5 rounded bg-zinc-900 border border-zinc-800 text-zinc-400">
                        ns: {rev.namespace || 'cluster-scoped'}
                      </span>
                    </div>

                    {/* Action buttons on card */}
                    <div className="flex items-center space-x-1.5 shrink-0 ml-auto sm:ml-0">
                      <button
                        onClick={() => onViewDiff(rev)}
                        className="px-2.5 py-1 rounded bg-zinc-900 hover:bg-zinc-800 text-orange-400 border border-orange-900/60 hover:border-orange-600 text-xs font-semibold transition"
                        title="Compare field-level diff with prior revision"
                      >
                        Diff
                      </button>

                      <button
                        onClick={() => onInspectRevision(rev)}
                        className="px-2.5 py-1 rounded bg-zinc-900 hover:bg-zinc-800 text-zinc-300 border border-zinc-800 text-xs transition"
                        title="View clean scrubbed YAML"
                      >
                        Spec
                      </button>

                      <button
                        onClick={() => onTimeTravelTo(rev.timestamp, rev.namespace)}
                        className="px-2 py-1 rounded bg-zinc-900 hover:bg-zinc-800 text-zinc-400 hover:text-zinc-100 border border-zinc-800 text-xs transition flex items-center space-x-1"
                        title="Reconstruct cluster at this timestamp"
                      >
                        <Rewind className="w-3 h-3" />
                        <span className="text-[10px]">T</span>
                      </button>
                    </div>
                  </div>
                </div>
              )
            })}
          </div>
        ) : (
          /* Table View - Wrapped in overflow-x-auto with sticky right action buttons */
          <div className="overflow-x-auto w-full">
            <table className="w-full text-left font-mono text-xs border-collapse min-w-[760px]">
              <thead>
                <tr className="border-b border-zinc-800 bg-[#0c0c0e] text-zinc-500 text-[10px] uppercase tracking-wider">
                  <th className="py-2.5 px-3 w-28">Timestamp</th>
                  <th className="py-2.5 px-3 w-24">Actor</th>
                  <th className="py-2.5 px-3 w-28">Manager</th>
                  <th className="py-2.5 px-3 w-24">Action</th>
                  <th className="py-2.5 px-3 w-32">Namespace</th>
                  <th className="py-2.5 px-3 w-28">Kind</th>
                  <th className="py-2.5 px-3 min-w-[150px]">Resource Name</th>
                  <th className="py-2.5 px-3 text-right w-36 sticky right-0 bg-[#0c0c0e] shadow-[-6px_0_10px_rgba(0,0,0,0.5)] z-20">Actions</th>
                </tr>
              </thead>
              <tbody className="divide-y divide-zinc-900">
                {filteredRevisions.map((rev) => {
                  const isHuman = rev.actor_type === 'Human'
                  const isGitOps = rev.actor_type === 'GitOps'

                  return (
                    <tr 
                      key={rev.id} 
                      className={`hover:bg-zinc-800/40 transition group ${
                        isHuman ? 'bg-amber-950/10' : ''
                      }`}
                    >
                      {/* Timestamp */}
                      <td className="py-2 px-3 whitespace-nowrap">
                        <div className="text-zinc-300 font-semibold">{formatTime(rev.timestamp)}</div>
                        <div className="text-[10px] text-zinc-500">{formatRelative(rev.timestamp)}</div>
                      </td>

                      {/* Actor Type Badge */}
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

                      {/* Manager Name */}
                      <td className="py-2 px-3 whitespace-nowrap text-zinc-300">
                        <span className="text-[11px] bg-zinc-900/80 px-1.5 py-0.5 rounded border border-zinc-800">
                          {rev.manager_name || 'unknown'}
                        </span>
                      </td>

                      {/* Change Type */}
                      <td className="py-2 px-3 whitespace-nowrap">
                        {rev.change_type === 'ADDED' && (
                          <span className="inline-flex items-center space-x-1 text-emerald-400 font-semibold text-[11px]">
                            <Plus className="w-3 h-3" />
                            <span>ADDED</span>
                          </span>
                        )}
                        {rev.change_type === 'MODIFIED' && (
                          <span className="inline-flex items-center space-x-1 text-sky-400 font-semibold text-[11px]">
                            <FileEdit className="w-3 h-3" />
                            <span>MODIFIED</span>
                          </span>
                        )}
                        {rev.change_type === 'DELETED' && (
                          <span className="inline-flex items-center space-x-1 text-rose-400 font-semibold text-[11px]">
                            <Trash2 className="w-3 h-3" />
                            <span>DELETED</span>
                          </span>
                        )}
                      </td>

                      {/* Namespace */}
                      <td className="py-2 px-3 whitespace-nowrap text-zinc-400">
                        <span className="text-[11px] px-1.5 py-0.5 rounded bg-zinc-900 border border-zinc-800">
                          {rev.namespace || 'cluster-scoped'}
                        </span>
                      </td>

                      {/* Kind */}
                      <td className="py-2 px-3 whitespace-nowrap">
                        <span className="font-semibold text-zinc-200">{rev.kind}</span>
                      </td>

                      {/* Name */}
                      <td className="py-2 px-3 font-semibold text-zinc-100 max-w-xs truncate" title={rev.name}>
                        {rev.name}
                      </td>

                      {/* Sticky Right Actions Column */}
                      <td className="py-2 px-3 text-right whitespace-nowrap space-x-1 sticky right-0 bg-[#121215] group-hover:bg-[#1a1a1f] shadow-[-6px_0_10px_rgba(0,0,0,0.5)] z-10">
                        <button
                          onClick={() => onViewDiff(rev)}
                          className="px-2 py-0.5 rounded bg-zinc-900 hover:bg-zinc-800 text-orange-400 border border-zinc-700 text-[11px] transition"
                          title="Compare field-level diff with prior revision"
                        >
                          Diff
                        </button>

                        <button
                          onClick={() => onInspectRevision(rev)}
                          className="px-2 py-0.5 rounded bg-zinc-900 hover:bg-zinc-800 text-zinc-300 border border-zinc-800 text-[11px] transition"
                          title="View clean scrubbed YAML"
                        >
                          Spec
                        </button>

                        <button
                          onClick={() => onTimeTravelTo(rev.timestamp, rev.namespace)}
                          className="px-1.5 py-0.5 rounded bg-zinc-900 hover:bg-zinc-800 text-zinc-500 hover:text-zinc-200 border border-zinc-800 text-[11px] transition"
                          title="Reconstruct cluster at this timestamp"
                        >
                          <Rewind className="w-3 h-3 inline" />
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
