import React from 'react'
import { Terminal, ArrowRight, Server, ShieldCheck, Clock, AlertTriangle, UserCheck, HardDrive } from 'lucide-react'

export default function LandingView({ stats, onNavigate }) {
  const clusterConnected = stats?.cluster_connected
  const clusterInfo = stats?.cluster_info || {}
  const statsData = stats?.stats || stats || {}

  const totalRevs = statsData.total_revisions || 0
  const humanChanges = statsData.human_changes || 0
  const totalResources = statsData.total_resources || 0
  const incidentCount = statsData.incidents_count || 0

  return (
    <div className="font-mono text-xs space-y-5 max-w-5xl mx-auto py-2">
      {/* System Status Bar */}
      <div className="flex items-center justify-between border-b border-zinc-800 pb-2 text-[11px] text-zinc-500">
        <div className="flex items-center space-x-2">
          <span className="w-2 h-2 rounded-full bg-orange-500" />
          <span className="text-zinc-300 font-semibold">cluster_runtime.md</span>
          <span className="text-zinc-600">•</span>
          <span className="text-zinc-400">mode: in-cluster audit</span>
        </div>
        <div className="flex items-center space-x-3 text-[10px]">
          <span className="px-1.5 py-0.5 rounded bg-zinc-900 border border-zinc-800 text-orange-400">
            ENGINE: ACTIVE
          </span>
          <span className="text-zinc-500">
            {clusterConnected ? 'KUBERNETES CONNECTED' : 'LOCAL CACHE'}
          </span>
        </div>
      </div>

      {/* Hero Document Block */}
      <div className="border border-zinc-800 bg-[#0c0c0e] p-5 sm:p-6 space-y-4">
        <div className="space-y-1.5 border-b border-zinc-800/80 pb-4">
          <div className="flex items-center space-x-2">
            <span className="text-orange-500 font-bold text-sm sm:text-base">#</span>
            <h1 className="text-zinc-100 font-bold text-base sm:text-lg tracking-wide">
              CLUSTER TIME MACHINE
            </h1>
            <span className="px-2 py-0.5 rounded bg-orange-950/60 border border-orange-800 text-orange-300 text-[10px] font-bold ml-2">
              IN-CLUSTER RUNTIME
            </span>
          </div>
          <p className="text-zinc-400 text-xs sm:text-[13px] leading-relaxed">
            Lightweight, append-only Kubernetes audit recorder and out-of-band drift investigator. Watches the cluster API 24/7, detects manual <span className="text-orange-400 font-semibold">kubectl</span> bypasses, and performs rule-ranked root-cause analysis on failure states.
          </p>
        </div>

        {/* Runtime Environment Matrix */}
        <div className="grid grid-cols-1 md:grid-cols-2 gap-3 pt-1">
          <div className="border border-zinc-800/90 bg-[#09090b] p-3 space-y-2">
            <div className="text-[11px] uppercase tracking-wider text-orange-400/90 font-semibold flex items-center space-x-1.5">
              <Server className="w-3.5 h-3.5 text-orange-500" />
              <span>Cluster Target Environment</span>
            </div>
            <table className="w-full text-[11px] leading-5">
              <tbody>
                <tr className="border-b border-zinc-900">
                  <td className="text-zinc-500 w-36">Host Endpoint:</td>
                  <td className="text-zinc-200 font-semibold break-all">
                    {clusterInfo.host || (clusterInfo.in_cluster ? 'kubernetes.default.svc' : 'In-Cluster Core API')}
                  </td>
                </tr>
                <tr className="border-b border-zinc-900">
                  <td className="text-zinc-500">API Server Version:</td>
                  <td className="text-zinc-300">{clusterInfo.version || 'v1.31.x (Active)'}</td>
                </tr>
                <tr className="border-b border-zinc-900">
                  <td className="text-zinc-500">Auth Method:</td>
                  <td className="text-zinc-300">
                    {clusterInfo.in_cluster ? 'Pod ServiceAccount Token' : 'client-go kubeconfig'}
                  </td>
                </tr>
                <tr>
                  <td className="text-zinc-500">Informer Watchers:</td>
                  <td className="text-emerald-400 font-semibold">12 Core & App Streams</td>
                </tr>
              </tbody>
            </table>
          </div>

          <div className="border border-zinc-800/90 bg-[#09090b] p-3 space-y-2">
            <div className="text-[11px] uppercase tracking-wider text-orange-400/90 font-semibold flex items-center space-x-1.5">
              <HardDrive className="w-3.5 h-3.5 text-orange-500" />
              <span>Persistence & Security</span>
            </div>
            <table className="w-full text-[11px] leading-5">
              <tbody>
                <tr className="border-b border-zinc-900">
                  <td className="text-zinc-500 w-36">Database:</td>
                  <td className="text-zinc-300">SQLite (Pure Go / WAL Mode)</td>
                </tr>
                <tr className="border-b border-zinc-900">
                  <td className="text-zinc-500">Storage Mount:</td>
                  <td className="text-zinc-300">/data/cluster_timemachine.db</td>
                </tr>
                <tr className="border-b border-zinc-900">
                  <td className="text-zinc-500">Secret Scrubber:</td>
                  <td className="text-orange-400 font-semibold">SHA-256 Hashed Payloads</td>
                </tr>
                <tr>
                  <td className="text-zinc-500">Actor Attribution:</td>
                  <td className="text-zinc-300">metadata.managedFields Engine</td>
                </tr>
              </tbody>
            </table>
          </div>
        </div>

        {/* Live Metrics Quad */}
        <div className="grid grid-cols-2 sm:grid-cols-4 gap-2.5 pt-2">
          <div className="p-3 bg-[#09090b] border border-zinc-800">
            <div className="text-[10px] uppercase tracking-wider text-zinc-500">Active Workloads</div>
            <div className="text-lg sm:text-xl font-bold text-zinc-100 mt-1">{totalResources}</div>
            <div className="text-[10px] text-zinc-500 mt-0.5">Discovered in cluster</div>
          </div>

          <div className="p-3 bg-[#09090b] border border-zinc-800">
            <div className="text-[10px] uppercase tracking-wider text-zinc-500">Audit Revisions</div>
            <div className="text-lg sm:text-xl font-bold text-orange-400 mt-1">{totalRevs}</div>
            <div className="text-[10px] text-zinc-500 mt-0.5">Deduplicated snapshots</div>
          </div>

          <div className="p-3 bg-[#09090b] border border-zinc-800">
            <div className="text-[10px] uppercase tracking-wider text-zinc-500">Manual Drift</div>
            <div className="text-lg sm:text-xl font-bold text-amber-400 mt-1">{humanChanges}</div>
            <div className="text-[10px] text-zinc-500 mt-0.5">Direct CLI/kubectl overrides</div>
          </div>

          <div className="p-3 bg-[#09090b] border border-zinc-800">
            <div className="text-[10px] uppercase tracking-wider text-zinc-500">Incidents</div>
            <div className={`text-lg sm:text-xl font-bold mt-1 ${incidentCount > 0 ? 'text-rose-400' : 'text-emerald-400'}`}>
              {incidentCount}
            </div>
            <div className="text-[10px] text-zinc-500 mt-0.5">Critical failure states</div>
          </div>
        </div>

        {/* Action Controls to Jump Direct to Dashboard */}
        <div className="pt-3 border-t border-zinc-800/80 flex flex-wrap items-center gap-2.5">
          <button
            onClick={() => onNavigate('timeline')}
            className="px-4 py-2 bg-orange-600 hover:bg-orange-500 text-black font-bold text-xs flex items-center space-x-1.5 transition border border-orange-500"
          >
            <span>OPEN AUDIT TIMELINE</span>
            <ArrowRight className="w-3.5 h-3.5" />
          </button>

          <button
            onClick={() => onNavigate('manual')}
            className="px-3.5 py-2 bg-zinc-900 hover:bg-zinc-800 text-amber-300 font-semibold text-xs flex items-center space-x-1.5 border border-zinc-700 transition"
          >
            <UserCheck className="w-3.5 h-3.5 text-amber-400" />
            <span>INSPECT MANUAL DRIFT ({humanChanges})</span>
          </button>

          <button
            onClick={() => onNavigate('timetravel')}
            className="px-3.5 py-2 bg-zinc-900 hover:bg-zinc-800 text-zinc-300 hover:text-zinc-100 font-semibold text-xs flex items-center space-x-1.5 border border-zinc-800 transition"
          >
            <Clock className="w-3.5 h-3.5 text-orange-400" />
            <span>TIME TRAVEL RECONSTRUCTOR</span>
          </button>

          {incidentCount > 0 && (
            <button
              onClick={() => onNavigate('incidents')}
              className="px-3.5 py-2 bg-rose-950/70 hover:bg-rose-900 text-rose-300 font-semibold text-xs flex items-center space-x-1.5 border border-rose-800 transition"
            >
              <AlertTriangle className="w-3.5 h-3.5 text-rose-400" />
              <span>INVESTIGATE INCIDENTS ({incidentCount})</span>
            </button>
          )}
        </div>
      </div>

      {/* Quick Ops Guide (Minimal Markdown block) */}
      <div className="border border-zinc-800 bg-[#09090b] p-4 space-y-2.5">
        <div className="flex items-center space-x-1.5 text-zinc-400 font-semibold text-xs">
          <Terminal className="w-3.5 h-3.5 text-orange-400" />
          <span className="uppercase tracking-wider">In-Cluster Operational Commands</span>
        </div>
        <div className="space-y-1.5 text-[11px] text-zinc-400">
          <div className="p-2 bg-[#050507] border border-zinc-800/80 rounded font-mono text-zinc-300 flex items-center justify-between">
            <span>kubectl port-forward -n ctm-system svc/ctm-service 8080:8080</span>
            <span className="text-[10px] text-zinc-500">Access Dashboard</span>
          </div>
          <div className="p-2 bg-[#050507] border border-zinc-800/80 rounded font-mono text-zinc-300 flex items-center justify-between">
            <span>kubectl edit deployment/payment-processor -n production</span>
            <span className="text-[10px] text-zinc-500">Simulate Manual Drift</span>
          </div>
        </div>
      </div>
    </div>
  )
}
