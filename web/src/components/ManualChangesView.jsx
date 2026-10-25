import React from 'react'
import { UserCheck, Terminal } from 'lucide-react'

export default function ManualChangesView({ manualChanges, onViewDiff, onInspectRevision }) {
  return (
    <div className="space-y-4 font-mono text-xs">
      {/* Alert Banner */}
      <div className="bg-amber-950/20 border border-amber-900/60 p-3 sm:p-4 rounded flex flex-col sm:flex-row items-start gap-2.5 sm:gap-3">
        <div className="p-1.5 sm:p-2 rounded bg-amber-950/80 border border-amber-800 text-amber-400 mt-0.5 shrink-0">
          <Terminal className="w-4 h-4" />
        </div>
        <div className="space-y-1">
          <div className="flex flex-wrap items-center gap-2">
            <span className="font-bold text-amber-300 text-xs sm:text-sm">
              DIRECT HUMAN OVERRIDES (OUT-OF-BAND DRIFT)
            </span>
            <span className="px-2 py-0.5 rounded bg-amber-950 text-amber-200 border border-amber-800 text-[10px] font-bold">
              {manualChanges.length} DETECTED
            </span>
          </div>
          <p className="text-[10px] sm:text-[11px] text-zinc-400 leading-relaxed">
            These changes were applied directly via <span className="text-amber-300 font-semibold">kubectl</span>, CLI tools, or Kubernetes UI dashboards, bypassing GitOps pipelines (ArgoCD / Flux / Helm). Manual changes are prime candidates for operational regressions and configuration drift.
          </p>
        </div>
      </div>

      {/* Manual Changes List */}
      <div className="bg-[#121215] border border-zinc-800 rounded overflow-hidden">
        <div className="p-2.5 sm:p-3 border-b border-zinc-800 bg-[#0c0c0e] flex flex-col sm:flex-row sm:items-center justify-between gap-1">
          <span className="text-zinc-400 font-semibold uppercase text-xs">
            Recorded Human Overrides ({manualChanges.length})
          </span>
          <span className="text-[10px] sm:text-[11px] text-zinc-500">
            Ordered by newest first
          </span>
        </div>

        {manualChanges.length === 0 ? (
          <div className="py-16 text-center text-zinc-500">
            <UserCheck className="w-8 h-8 text-zinc-600 mx-auto mb-2 opacity-50" />
            <p className="text-zinc-400 font-semibold">Zero Manual Drift Detected</p>
            <p className="text-[11px] text-zinc-500 mt-0.5">
              All cluster modifications originated from certified GitOps pipelines or automated system controllers.
            </p>
          </div>
        ) : (
          <div className="divide-y divide-zinc-900">
            {manualChanges.map((ch) => {
              const dateObj = new Date(ch.timestamp)
              const timeFormatted = dateObj.toLocaleTimeString([], { hour: '2-digit', minute: '2-digit', second: '2-digit', hour12: false })
              const dateFormatted = dateObj.toISOString().split('T')[0]

              return (
                <div 
                  key={ch.id} 
                  className="p-3 sm:p-4 hover:bg-zinc-800/40 transition flex flex-col md:flex-row md:items-center justify-between gap-3 sm:gap-4"
                >
                  <div className="space-y-1.5 flex-1 min-w-0">
                    <div className="flex flex-wrap items-center gap-1.5 sm:gap-2">
                      <span className="px-1.5 py-0.5 rounded bg-amber-950/80 text-amber-300 border border-amber-800 text-[10px] font-bold flex items-center space-x-1 shrink-0">
                        <Terminal className="w-2.5 h-2.5" />
                        <span>{ch.manager_name}</span>
                      </span>

                      <span className="text-zinc-200 font-bold text-xs sm:text-sm break-all">
                        {ch.kind} / {ch.name}
                      </span>

                      <span className="px-1.5 py-0.5 rounded bg-zinc-900 text-zinc-400 border border-zinc-800 text-[10px] sm:text-[11px] shrink-0">
                        ns: {ch.namespace || 'default'}
                      </span>

                      <span className="text-[10px] sm:text-[11px] text-zinc-500 shrink-0">
                        {dateFormatted} {timeFormatted}
                      </span>
                    </div>

                    {/* Diff Summary */}
                    {ch.diff_summary ? (
                      <div className="p-2 rounded bg-[#09090b] border border-zinc-800/80 text-zinc-300 text-[11px] font-mono flex items-start sm:items-center space-x-2">
                        <span className="text-amber-400 font-semibold shrink-0">DRIFT:</span>
                        <span className="text-zinc-300 break-words">{ch.diff_summary}</span>
                      </div>
                    ) : (
                      <div className="text-[10px] sm:text-[11px] text-zinc-500 italic">
                        Resource revision recorded (metadata or spec modified)
                      </div>
                    )}
                  </div>

                  {/* Actions */}
                  <div className="flex items-center space-x-2 self-start md:self-center shrink-0">
                    <button
                      onClick={() => onViewDiff(ch)}
                      className="px-2.5 sm:px-3 py-1 rounded bg-zinc-900 hover:bg-zinc-800 text-orange-400 border border-orange-900/60 hover:border-orange-600 text-xs font-semibold transition"
                    >
                      Inspect Diff
                    </button>
                    <button
                      onClick={() => onInspectRevision(ch)}
                      className="px-2.5 py-1 rounded bg-zinc-900 hover:bg-zinc-800 text-zinc-300 border border-zinc-800 text-xs transition"
                    >
                      View Spec
                    </button>
                  </div>
                </div>
              )
            })}
          </div>
        )}
      </div>
    </div>
  )
}
