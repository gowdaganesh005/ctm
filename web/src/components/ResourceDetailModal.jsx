import React, { useState } from 'react'
import { X, Copy, Check, FileText, User, GitBranch, Cpu, Calendar } from 'lucide-react'

export default function ResourceDetailModal({ resource, onClose }) {
  const [copied, setCopied] = useState(false)

  if (!resource) return null

  const copyYAML = () => {
    if (!resource.yaml) return
    navigator.clipboard.writeText(resource.yaml)
    setCopied(true)
    setTimeout(() => setCopied(false), 2000)
  }

  const isHuman = resource.actor_type === 'Human'
  const isGitOps = resource.actor_type === 'GitOps'

  return (
    <div className="fixed inset-0 z-50 bg-black/80 backdrop-blur-sm flex items-center justify-center p-3 sm:p-4">
      <div className="bg-[#121215] border border-zinc-800 w-full max-w-4xl max-h-[85vh] rounded flex flex-col shadow-2xl font-mono text-xs overflow-hidden">
        {/* Header */}
        <div className="p-3 sm:p-3.5 border-b border-zinc-800 bg-[#0c0c0e] flex flex-col sm:flex-row sm:items-center justify-between gap-2.5">
          <div className="flex items-center space-x-2.5 sm:space-x-3">
            <div className="p-1 rounded bg-zinc-900 border border-orange-500/40 text-orange-400 shrink-0">
              <FileText className="w-4 h-4" />
            </div>
            <div className="min-w-0">
              <div className="flex flex-wrap items-center gap-1.5 sm:gap-2">
                <span className="font-bold text-zinc-100 text-xs sm:text-sm break-all">
                  {resource.kind} / {resource.name}
                </span>
                <span className="text-[10px] px-1.5 py-0.2 rounded bg-zinc-900 text-zinc-400 border border-zinc-800 shrink-0">
                  {resource.namespace || 'cluster-scoped'}
                </span>
              </div>
              <div className="text-[10px] sm:text-[11px] text-zinc-500 mt-0.5 flex flex-wrap items-center gap-1 sm:gap-2">
                <span>Managed by: {resource.manager_name || 'unknown'}</span>
                <span>•</span>
                <span>{resource.timestamp ? new Date(resource.timestamp).toUTCString() : ''}</span>
              </div>
            </div>
          </div>

          <div className="flex items-center space-x-2 self-end sm:self-auto shrink-0">
            <button
              onClick={copyYAML}
              className="p-1.5 rounded bg-zinc-900 hover:bg-zinc-800 text-zinc-300 border border-zinc-800 flex items-center space-x-1"
              title="Copy YAML"
            >
              {copied ? <Check className="w-3.5 h-3.5 text-emerald-400" /> : <Copy className="w-3.5 h-3.5" />}
              <span className="text-[11px] ml-1">{copied ? 'Copied' : 'Copy'}</span>
            </button>

            <button
              onClick={onClose}
              className="p-1.5 rounded bg-zinc-900 hover:bg-zinc-800 text-zinc-400 hover:text-zinc-100 border border-zinc-800"
            >
              <X className="w-4 h-4" />
            </button>
          </div>
        </div>

        {/* Content */}
        <div className="p-4 flex-1 overflow-y-auto">
          <pre className="bg-[#09090b] border border-zinc-800 p-4 rounded text-zinc-200 overflow-x-auto leading-relaxed font-mono text-xs">
            {resource.yaml || '# No YAML available'}
          </pre>
        </div>
      </div>
    </div>
  )
}
