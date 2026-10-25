import React, { useState, useEffect } from 'react'
import { 
  AlertTriangle, 
  ShieldAlert, 
  Clock, 
  User, 
  GitBranch, 
  Cpu, 
  CheckCircle, 
  ArrowRight, 
  Sliders, 
  Activity, 
  HelpCircle,
  ExternalLink
} from 'lucide-react'

export default function IncidentsView({ 
  incidents, 
  onViewDiff, 
  onInspectRevision,
  onTimeTravelTo 
}) {
  const [selectedIncident, setSelectedIncident] = useState(incidents[0] || null)
  const [lookbackMinutes, setLookbackMinutes] = useState(30)
  const [rcaCandidates, setRcaCandidates] = useState([])
  const [loadingRCA, setLoadingRCA] = useState(false)

  // Update selected incident if list updates
  useEffect(() => {
    if (!selectedIncident && incidents.length > 0) {
      setSelectedIncident(incidents[0])
    }
  }, [incidents])

  // Fetch RCA candidates whenever selected incident or lookback changes
  useEffect(() => {
    if (!selectedIncident) return

    const fetchRCA = async () => {
      setLoadingRCA(true)
      try {
        const res = await fetch(`/api/incidents/${selectedIncident.id}/rca?lookback=${lookbackMinutes}`)
        if (res.ok) {
          const json = await res.json()
          setRcaCandidates(json.ranked_changes || [])
        } else {
          // Fallback to pre-calculated ranked changes
          setRcaCandidates(selectedIncident.ranked_changes || [])
        }
      } catch (err) {
        console.error('Failed to fetch RCA candidates:', err)
        setRcaCandidates(selectedIncident.ranked_changes || [])
      } finally {
        setLoadingRCA(false)
      }
    }

    fetchRCA()
  }, [selectedIncident, lookbackMinutes])

  return (
    <div className="grid grid-cols-1 lg:grid-cols-12 gap-4 font-mono text-xs">
      {/* Left Column: Incidents List (4 cols) */}
      <div className="lg:col-span-4 space-y-2">
        <div className="bg-[#121215] border border-zinc-800 p-3 rounded flex items-center justify-between">
          <div className="flex items-center space-x-2">
            <AlertTriangle className="w-4 h-4 text-rose-400" />
            <span className="font-bold text-zinc-100 uppercase text-xs">
              Cluster Incidents ({incidents.length})
            </span>
          </div>
          <span className="text-[10px] text-zinc-500">Auto-detected</span>
        </div>

        <div className="space-y-2 max-h-60 sm:max-h-80 lg:max-h-[calc(100vh-220px)] overflow-y-auto pr-1">
          {incidents.length === 0 ? (
            <div className="bg-[#121215] border border-zinc-800 p-8 rounded text-center text-zinc-500">
              <CheckCircle className="w-8 h-8 text-emerald-500/50 mx-auto mb-2" />
              <p className="text-zinc-300 font-semibold">No Incidents Detected</p>
              <p className="text-[11px] text-zinc-500 mt-1">Cluster workloads are running normally.</p>
            </div>
          ) : (
            incidents.map((inc) => {
              const isSelected = selectedIncident?.id === inc.id
              const isCritical = inc.severity === 'critical'

              return (
                <div
                  key={inc.id}
                  onClick={() => setSelectedIncident(inc)}
                  className={`p-3 rounded border transition cursor-pointer text-left ${
                    isSelected
                      ? 'bg-zinc-800/80 border-rose-600/80 shadow-md'
                      : 'bg-[#121215] border-zinc-800/80 hover:border-zinc-700'
                  }`}
                >
                  <div className="flex items-start justify-between gap-2">
                    <span
                      className={`px-1.5 py-0.5 rounded text-[9px] uppercase font-bold tracking-wider ${
                        isCritical
                          ? 'bg-rose-950 text-rose-300 border border-rose-800'
                          : 'bg-amber-950 text-amber-300 border border-amber-800'
                      }`}
                    >
                      {inc.severity}
                    </span>
                    <span className="text-[10px] text-zinc-500 whitespace-nowrap">
                      {new Date(inc.detected_at).toLocaleTimeString([], { hour: '2-digit', minute: '2-digit', second: '2-digit', hour12: false })}
                    </span>
                  </div>

                  <h3 className="font-bold text-zinc-100 text-xs mt-2 leading-snug">
                    {inc.title}
                  </h3>

                  <div className="flex items-center space-x-1.5 mt-1.5 text-[11px] text-zinc-400">
                    <span className="text-zinc-300">{inc.resource_kind}</span>
                    <span>/</span>
                    <span className="truncate text-zinc-200">{inc.resource_name}</span>
                  </div>

                  <div className="text-[10px] text-zinc-500 mt-1">
                    ns: <span className="text-zinc-400">{inc.namespace || 'default'}</span>
                  </div>
                </div>
              )
            })
          )}
        </div>
      </div>

      {/* Right Column: RCA Root-Cause Investigation (8 cols) */}
      <div className="lg:col-span-8 space-y-4">
        {selectedIncident ? (
          <div className="space-y-4">
            {/* Incident Header Card */}
            <div className="bg-[#121215] border border-zinc-800 p-4 rounded space-y-3">
              <div className="flex flex-wrap items-center justify-between gap-2 border-b border-zinc-800/80 pb-3">
                <div className="space-y-1">
                  <div className="flex items-center space-x-2">
                    <span
                      className={`px-2 py-0.5 rounded text-[10px] uppercase font-bold tracking-wider ${
                        selectedIncident.severity === 'critical'
                          ? 'bg-rose-950 text-rose-300 border border-rose-800'
                          : 'bg-amber-950 text-amber-300 border border-amber-800'
                      }`}
                    >
                      {selectedIncident.severity}
                    </span>
                    <h2 className="text-sm font-bold text-zinc-100">
                      {selectedIncident.title}
                    </h2>
                  </div>
                  <div className="text-[11px] text-zinc-400">
                    Target: <span className="text-zinc-200 font-semibold">{selectedIncident.resource_kind}/{selectedIncident.resource_name}</span> in namespace <span className="text-zinc-300 font-semibold">{selectedIncident.namespace}</span>
                  </div>
                </div>

                {/* Time Travel to Incident Button */}
                <button
                  onClick={() => onTimeTravelTo(selectedIncident.detected_at, selectedIncident.namespace)}
                  className="px-2.5 py-1 rounded bg-zinc-900 hover:bg-zinc-800 text-orange-400 border border-zinc-700 text-xs flex items-center space-x-1.5 transition"
                >
                  <Clock className="w-3.5 h-3.5" />
                  <span>Time Travel to Incident</span>
                </button>
              </div>

              {/* Description message */}
              <div className="bg-[#09090b] border border-zinc-800 p-2.5 rounded text-zinc-300 text-[11px] leading-relaxed">
                <span className="text-rose-400 font-semibold mr-1.5">DIAGNOSIS:</span>
                {selectedIncident.description}
              </div>

              {/* Lookback Window Controls */}
              <div className="flex items-center justify-between pt-1">
                <div className="flex items-center space-x-2">
                  <Sliders className="w-3.5 h-3.5 text-zinc-500" />
                  <span className="text-zinc-400 text-[11px]">RCA Lookback Window:</span>
                  <div className="flex items-center space-x-1">
                    {[5, 15, 30, 60].map(mins => (
                      <button
                        key={mins}
                        onClick={() => setLookbackMinutes(mins)}
                        className={`px-2 py-0.5 rounded text-[11px] transition ${
                          lookbackMinutes === mins
                            ? 'bg-zinc-800 text-orange-400 border border-orange-500/50 font-bold'
                            : 'bg-[#09090b] text-zinc-500 hover:text-zinc-300 border border-zinc-800'
                        }`}
                      >
                        {mins}m
                      </button>
                    ))}
                  </div>
                </div>

                <div className="text-[10px] text-zinc-500">
                  {loadingRCA ? 'Ranking candidate changes...' : `${rcaCandidates.length} candidate causes scored`}
                </div>
              </div>
            </div>

            {/* Rule-Ranked Root Cause Candidates */}
            <div className="bg-[#121215] border border-zinc-800 rounded overflow-hidden">
              <div className="p-3 border-b border-zinc-800 bg-[#0c0c0e] flex items-center justify-between">
                <div className="flex items-center space-x-2">
                  <span className="text-xs font-bold text-zinc-200 uppercase">
                    RANKED ROOT CAUSE CANDIDATES
                  </span>
                  <span className="text-[10px] text-zinc-500">
                    (Scored by Actor Type, Causal Field Match, and Temporal Proximity)
                  </span>
                </div>
              </div>

              <div className="divide-y divide-zinc-900">
                {rcaCandidates.length === 0 ? (
                  <div className="p-8 text-center text-zinc-500">
                    No cluster revisions detected within the preceding {lookbackMinutes} minutes in namespace '{selectedIncident.namespace}'.
                  </div>
                ) : (
                  rcaCandidates.map((cand, idx) => {
                    const isTop = idx === 0
                    const isHuman = cand.actor_type === 'Human'
                    const isGitOps = cand.actor_type === 'GitOps'

                    return (
                      <div
                        key={cand.revision_id}
                        className={`p-4 transition ${
                          isTop
                            ? 'bg-rose-950/10 border-l-4 border-l-rose-500'
                            : 'hover:bg-zinc-800/30'
                        }`}
                      >
                        <div className="flex flex-wrap items-center justify-between gap-2">
                          <div className="flex flex-wrap items-center gap-1.5 sm:gap-2">
                            {/* Score Pill */}
                            <div
                              className={`px-2 py-0.5 rounded text-[11px] font-bold border flex items-center space-x-1 ${
                                cand.score >= 80
                                  ? 'bg-rose-950 text-rose-300 border-rose-800'
                                  : cand.score >= 50
                                  ? 'bg-amber-950 text-amber-300 border-amber-800'
                                  : 'bg-zinc-900 text-zinc-400 border-zinc-800'
                              }`}
                            >
                              <span>{cand.score}% CONFIDENCE</span>
                              {isTop && <span className="text-[9px] uppercase tracking-wider text-rose-400 ml-1">(TOP CAUSE)</span>}
                            </div>

                            {/* Actor Badge */}
                            {isHuman ? (
                              <span className="inline-flex items-center space-x-1 px-1.5 py-0.5 rounded text-[10px] bg-amber-950/80 text-amber-300 border border-amber-800">
                                <User className="w-2.5 h-2.5" />
                                <span>Human ({cand.manager_name})</span>
                              </span>
                            ) : isGitOps ? (
                              <span className="inline-flex items-center space-x-1 px-1.5 py-0.5 rounded text-[10px] bg-purple-950/80 text-purple-300 border border-purple-800">
                                <GitBranch className="w-2.5 h-2.5" />
                                <span>GitOps ({cand.manager_name})</span>
                              </span>
                            ) : (
                              <span className="inline-flex items-center space-x-1 px-1.5 py-0.5 rounded text-[10px] bg-zinc-900 text-zinc-400 border border-zinc-800">
                                <Cpu className="w-2.5 h-2.5" />
                                <span>Controller ({cand.manager_name})</span>
                              </span>
                            )}
                          </div>

                          <div className="text-[11px] text-zinc-500">
                            {new Date(cand.timestamp).toLocaleTimeString([], { hour: '2-digit', minute: '2-digit', second: '2-digit', hour12: false })}
                          </div>
                        </div>

                        {/* Resource Modified */}
                        <div className="mt-2 flex flex-wrap items-center gap-1.5 sm:gap-2 text-zinc-200">
                          <span className="font-bold text-xs">{cand.resource_kind} / {cand.resource_name}</span>
                          <span className="text-[10px] px-1 rounded bg-zinc-900 text-zinc-400 border border-zinc-800">
                            rev #{cand.revision_id}
                          </span>
                        </div>

                        {/* Diff Summary */}
                        {cand.diff_summary && (
                          <div className="mt-2 p-2 rounded bg-[#09090b] border border-zinc-800 text-zinc-300 text-[11px] font-mono">
                            <span className="text-orange-400 font-semibold mr-1.5">CHANGE:</span>
                            {cand.diff_summary}
                          </div>
                        )}

                        {/* Explanatory Reasons */}
                        {cand.reasons && cand.reasons.length > 0 && (
                          <div className="mt-2.5 space-y-1">
                            <span className="text-[10px] text-zinc-500 uppercase tracking-wider block">
                              Heuristic Causal Evidence:
                            </span>
                            <ul className="space-y-0.5">
                              {cand.reasons.map((r, i) => (
                                <li key={i} className="text-[11px] text-zinc-400 flex items-start space-x-1.5">
                                  <span className="text-rose-400">•</span>
                                  <span>{r}</span>
                                </li>
                              ))}
                            </ul>
                          </div>
                        )}

                        {/* Action buttons */}
                        <div className="mt-3 flex flex-wrap items-center gap-2 pt-2 border-t border-zinc-900">
                          <button
                            onClick={() => onViewDiff({
                              id: cand.revision_id,
                              name: cand.resource_name,
                              kind: cand.resource_kind,
                              namespace: cand.namespace,
                            })}
                            className="px-2.5 py-1 rounded bg-zinc-900 hover:bg-zinc-800 text-orange-400 border border-zinc-700 text-xs font-semibold transition"
                          >
                            Inspect YAML Diff
                          </button>
                        </div>
                      </div>
                    )
                  })
                )}
              </div>
            </div>
          </div>
        ) : (
          <div className="bg-[#121215] border border-zinc-800 p-12 rounded text-center text-zinc-500">
            Select an incident from the left to view root-cause analysis.
          </div>
        )}
      </div>
    </div>
  )
}
