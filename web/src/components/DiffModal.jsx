import React, { useState, useEffect } from 'react'
import { X, Copy, Check, FileCode, Split, AlignJustify, User, GitBranch, Cpu } from 'lucide-react'

export default function DiffModal({ revision, onClose }) {
  const [diffData, setDiffData] = useState(null)
  const [loading, setLoading] = useState(true)
  const [viewMode, setViewMode] = useState('fields') // 'fields', 'unified', 'sideBySide'
  const [copied, setCopied] = useState(false)

  useEffect(() => {
    if (!revision?.id) return

    const fetchDiff = async () => {
      setLoading(true)
      try {
        const res = await fetch(`/api/revisions/${revision.id}/diff`)
        if (res.ok) {
          const json = await res.json()
          setDiffData(json)
        }
      } catch (err) {
        console.error('Failed to fetch diff:', err)
      } finally {
        setLoading(false)
      }
    }

    fetchDiff()
  }, [revision])

  if (!revision) return null

  const copyDiff = () => {
    if (!diffData?.unified_diff) return
    navigator.clipboard.writeText(diffData.unified_diff)
    setCopied(true)
    setTimeout(() => setCopied(false), 2000)
  }

  const renderUnifiedLines = () => {
    if (!diffData?.unified_diff) {
      return (
        <div className="p-8 text-center text-zinc-500 font-mono text-xs">
          Initial revision or no textual differences found.
        </div>
      )
    }

    const lines = diffData.unified_diff.split('\n')
    return (
      <div className="font-mono text-xs overflow-x-auto bg-[#09090b] p-3 rounded border border-zinc-800">
        {lines.map((line, idx) => {
          let lineClass = 'text-zinc-400'
          if (line.startsWith('+')) lineClass = 'text-emerald-400 bg-emerald-950/20'
          else if (line.startsWith('-')) lineClass = 'text-rose-400 bg-rose-950/20'
          else if (line.startsWith('@')) lineClass = 'text-orange-400 font-bold'

          return (
            <div key={idx} className={`px-2 py-0.5 whitespace-pre ${lineClass}`}>
              {line}
            </div>
          )
        })}
      </div>
    )
  }

  return (
    <div className="fixed inset-0 z-50 bg-black/80 backdrop-blur-sm flex items-center justify-center p-4">
      <div className="bg-[#121215] border border-zinc-800 w-full max-w-5xl max-h-[90vh] rounded flex flex-col shadow-2xl font-mono text-xs overflow-hidden">
        {/* Modal Header */}
        <div className="p-3 sm:p-3.5 border-b border-zinc-800 bg-[#0c0c0e] flex flex-col sm:flex-row sm:items-center justify-between gap-2.5">
          <div className="flex items-center space-x-2.5 sm:space-x-3">
            <div className="p-1 rounded bg-zinc-900 border border-orange-500/40 text-orange-400 shrink-0">
              <FileCode className="w-4 h-4" />
            </div>
            <div>
              <div className="flex items-center space-x-2">
                <span className="font-bold text-zinc-100 text-sm">
                  {revision.kind} / {revision.name}
                </span>
                <span className="text-[10px] px-1.5 py-0.2 rounded bg-zinc-900 text-zinc-400 border border-zinc-800">
                  rev #{revision.id}
                </span>
              </div>
              <div className="text-[10px] sm:text-[11px] text-zinc-500">
                Namespace: {revision.namespace || 'default'}
              </div>
            </div>
          </div>

          <div className="flex items-center justify-between sm:justify-end space-x-2 w-full sm:w-auto">
            {/* View Mode Buttons */}
            <div className="flex items-center bg-[#09090b] border border-zinc-800 rounded p-0.5">
              <button
                onClick={() => setViewMode('fields')}
                className={`px-2 py-1 rounded text-[11px] transition ${
                  viewMode === 'fields'
                    ? 'bg-zinc-800 text-orange-400 font-semibold'
                    : 'text-zinc-400 hover:text-zinc-200'
                }`}
              >
                Fields
              </button>
              <button
                onClick={() => setViewMode('unified')}
                className={`px-2 py-1 rounded text-[11px] transition ${
                  viewMode === 'unified'
                    ? 'bg-zinc-800 text-orange-400 font-semibold'
                    : 'text-zinc-400 hover:text-zinc-200'
                }`}
              >
                Unified
              </button>
              <button
                onClick={() => setViewMode('sideBySide')}
                className={`px-2 py-1 rounded text-[11px] transition ${
                  viewMode === 'sideBySide'
                    ? 'bg-zinc-800 text-orange-400 font-semibold'
                    : 'text-zinc-400 hover:text-zinc-200'
                }`}
              >
                Side-by-Side
              </button>
            </div>

            <div className="flex items-center space-x-1.5">
              <button
                onClick={copyDiff}
                className="p-1.5 rounded bg-zinc-900 hover:bg-zinc-800 text-zinc-300 border border-zinc-800"
                title="Copy Unified Diff"
              >
                {copied ? <Check className="w-3.5 h-3.5 text-emerald-400" /> : <Copy className="w-3.5 h-3.5" />}
              </button>

              <button
                onClick={onClose}
                className="p-1.5 rounded bg-zinc-900 hover:bg-zinc-800 text-zinc-400 hover:text-zinc-100 border border-zinc-800"
              >
                <X className="w-4 h-4" />
              </button>
            </div>
          </div>
        </div>

        {/* Modal Content */}
        <div className="p-3 sm:p-4 flex-1 overflow-y-auto space-y-3 sm:space-y-4">
          {loading ? (
            <div className="py-20 text-center text-zinc-500">
              Computing field-level diff...
            </div>
          ) : (
            <>
              {/* Revision Comparison Subheader */}
              <div className="grid grid-cols-1 sm:grid-cols-2 gap-2.5 text-[11px]">
                <div className="p-2.5 rounded bg-[#09090b] border border-zinc-800">
                  <span className="text-zinc-500 uppercase tracking-wider text-[10px] block">PREVIOUS REVISION</span>
                  {diffData?.old_revision ? (
                    <div className="mt-1 text-zinc-300 space-y-0.5">
                      <div>Rev #{diffData.old_revision.id} ({diffData.old_revision.actor_type})</div>
                      <div className="text-[10px] text-zinc-500">
                        {new Date(diffData.old_revision.timestamp).toUTCString()}
                      </div>
                    </div>
                  ) : (
                    <div className="mt-1 text-zinc-500 italic">None (Newly Created Resource)</div>
                  )}
                </div>

                <div className="p-2.5 rounded bg-[#09090b] border border-orange-950/60">
                  <span className="text-orange-400 uppercase tracking-wider text-[10px] block">CURRENT REVISION</span>
                  {diffData?.new_revision && (
                    <div className="mt-1 text-zinc-200 space-y-0.5">
                      <div>Rev #{diffData.new_revision.id} ({diffData.new_revision.actor_type} - {diffData.new_revision.manager_name})</div>
                      <div className="text-[10px] text-zinc-400">
                        {new Date(diffData.new_revision.timestamp).toUTCString()}
                      </div>
                    </div>
                  )}
                </div>
              </div>

              {/* Mode 1: Field Changes Table */}
              {viewMode === 'fields' && (
                <div className="border border-zinc-800 rounded overflow-hidden">
                  <div className="p-2.5 bg-[#0c0c0e] border-b border-zinc-800 text-zinc-400 text-xs font-semibold uppercase">
                    Field-Level Differences ({diffData?.field_changes?.length || 0})
                  </div>
                  {(!diffData?.field_changes || diffData.field_changes.length === 0) ? (
                    <div className="p-8 text-center text-zinc-500">
                      No field-level differences detected (resource creation or non-spec metadata change).
                    </div>
                  ) : (
                    <div className="overflow-x-auto w-full">
                      <table className="w-full text-left font-mono text-xs border-collapse min-w-[550px]">
                        <thead>
                          <tr className="border-b border-zinc-800 bg-[#09090b] text-zinc-500 text-[10px] uppercase">
                            <th className="py-2 px-3 w-56 sm:w-72">Path</th>
                            <th className="py-2 px-3 w-24">Type</th>
                            <th className="py-2 px-3">Old Value</th>
                            <th className="py-2 px-3">New Value</th>
                          </tr>
                        </thead>
                        <tbody className="divide-y divide-zinc-900 bg-[#09090b]">
                          {diffData.field_changes.map((fc, i) => {
                            const isAdded = fc.type === 'added'
                            const isRemoved = fc.type === 'removed'

                            return (
                              <tr key={i} className="hover:bg-zinc-800/40 transition">
                                <td className="py-2 px-3 font-semibold text-orange-300 break-all">
                                  {fc.path}
                                </td>
                                <td className="py-2 px-3 whitespace-nowrap">
                                  {isAdded ? (
                                    <span className="px-1.5 py-0.2 rounded text-[10px] bg-emerald-950 text-emerald-300 border border-emerald-800">
                                      + ADDED
                                    </span>
                                  ) : isRemoved ? (
                                    <span className="px-1.5 py-0.2 rounded text-[10px] bg-rose-950 text-rose-300 border border-rose-800">
                                      - REMOVED
                                    </span>
                                  ) : (
                                    <span className="px-1.5 py-0.2 rounded text-[10px] bg-sky-950 text-sky-300 border border-sky-800">
                                      ~ MODIFIED
                                    </span>
                                  )}
                                </td>
                                <td className="py-2 px-3 font-mono text-rose-400 bg-rose-950/10 max-w-xs truncate">
                                  {fc.old_value !== null && fc.old_value !== undefined ? JSON.stringify(fc.old_value) : '<none>'}
                                </td>
                                <td className="py-2 px-3 font-mono text-emerald-400 bg-emerald-950/10 max-w-xs truncate">
                                  {fc.new_value !== null && fc.new_value !== undefined ? JSON.stringify(fc.new_value) : '<none>'}
                                </td>
                              </tr>
                            )
                          })}
                        </tbody>
                      </table>
                    </div>
                  )}
                </div>
              )}

              {/* Mode 2: Unified Diff */}
              {viewMode === 'unified' && renderUnifiedLines()}

              {/* Mode 3: Side-by-Side YAML */}
              {viewMode === 'sideBySide' && (
                <div className="grid grid-cols-1 md:grid-cols-2 gap-3">
                  <div className="space-y-1">
                    <span className="text-[10px] text-zinc-500 uppercase tracking-wider block">PREVIOUS YAML</span>
                    <pre className="bg-[#09090b] border border-zinc-800 p-3 rounded font-mono text-xs text-zinc-400 overflow-x-auto max-h-96">
                      {diffData?.old_yaml || '# No previous revision'}
                    </pre>
                  </div>
                  <div className="space-y-1">
                    <span className="text-[10px] text-orange-400 uppercase tracking-wider block">CURRENT YAML</span>
                    <pre className="bg-[#09090b] border border-orange-950/60 p-3 rounded font-mono text-xs text-zinc-200 overflow-x-auto max-h-96">
                      {diffData?.new_yaml || '# No current revision'}
                    </pre>
                  </div>
                </div>
              )}
            </>
          )}
        </div>
      </div>
    </div>
  )
}
