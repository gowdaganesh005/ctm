import React from 'react'
import { Clock, Rewind, UserCheck, AlertTriangle, RefreshCw, Terminal, Activity, Server } from 'lucide-react'

export default function Navbar({ 
  activeTab, 
  setActiveTab, 
  stats, 
  sseConnected, 
  incidentCount, 
  manualCount, 
  onRefresh 
}) {
  const isConnected = stats?.cluster_connected
  const inCluster = stats?.cluster_info?.in_cluster
  const totalRevs = stats?.stats?.total_revisions ?? stats?.total_revisions ?? 0

  return (
    <header className="border-b border-zinc-800 bg-[#09090b]/95 backdrop-blur sticky top-0 z-40 px-3 sm:px-6 py-2.5">
      <div className="max-w-7xl mx-auto flex flex-col md:flex-row md:items-center md:justify-between gap-3">
        
        {/* Left Side: Brand Logo and Title */}
        <div className="flex items-center justify-between">
          <div 
            className="flex items-center space-x-3 cursor-pointer group"
            onClick={() => setActiveTab('landing')}
          >
            <div className="w-8 h-8 rounded-md bg-zinc-900 border border-orange-500/40 flex items-center justify-center text-orange-400 group-hover:border-orange-400 transition-colors shrink-0">
              <Clock className="w-4 h-4 text-orange-400" />
            </div>
            <div>
              <div className="flex items-center space-x-2">
                <span className="font-mono text-sm font-bold tracking-wider text-zinc-100">
                  CLUSTER<span className="text-orange-500">TIME</span>MACHINE
                </span>
                <span className="text-[10px] font-mono px-1.5 py-0.5 rounded bg-zinc-800/80 text-orange-300 border border-zinc-700/80">
                  v1.0.0
                </span>
              </div>
              <p className="text-[10px] text-zinc-500 font-mono leading-none mt-0.5 hidden sm:block">
                In-Cluster Kubernetes Audit Recorder & RCA
              </p>
            </div>
          </div>

          {/* Mobile-only right quick actions */}
          <div className="flex md:hidden items-center space-x-2">
            <div className="flex items-center space-x-1.5 px-2 py-1 rounded bg-zinc-900 border border-zinc-800 text-[10px] font-mono">
              <span className={`w-1.5 h-1.5 rounded-full ${sseConnected ? 'bg-emerald-400 animate-pulse' : 'bg-zinc-600'}`} />
              <span className="text-zinc-400">{sseConnected ? 'LIVE' : 'POLL'}</span>
            </div>
            <button
              onClick={onRefresh}
              className="p-1.5 rounded bg-zinc-900 hover:bg-zinc-800 text-zinc-400 hover:text-zinc-200 border border-zinc-800 transition"
              title="Refresh Data"
            >
              <RefreshCw className="w-3.5 h-3.5" />
            </button>
          </div>
        </div>

        {/* Center: Navigation Tabs */}
        <nav className="flex items-center space-x-1.5 font-mono text-xs overflow-x-auto pb-1 md:pb-0 -mx-1 px-1 no-scrollbar">
          <button
            onClick={() => setActiveTab('landing')}
            className={`px-3 py-1.5 rounded-md flex items-center space-x-1.5 transition whitespace-nowrap shrink-0 text-xs ${
              activeTab === 'landing'
                ? 'bg-zinc-800 text-orange-400 border border-orange-500/40 font-medium'
                : 'text-zinc-400 hover:text-zinc-200 hover:bg-zinc-900/80 border border-transparent'
            }`}
          >
            <Terminal className="w-3.5 h-3.5 shrink-0" />
            <span>OVERVIEW</span>
          </button>

          <button
            onClick={() => setActiveTab('timeline')}
            className={`px-3 py-1.5 rounded-md flex items-center space-x-1.5 transition whitespace-nowrap shrink-0 text-xs ${
              activeTab === 'timeline'
                ? 'bg-zinc-800 text-orange-400 border border-orange-500/40 font-medium'
                : 'text-zinc-400 hover:text-zinc-200 hover:bg-zinc-900/80 border border-transparent'
            }`}
          >
            <Clock className="w-3.5 h-3.5 shrink-0" />
            <span>TIMELINE</span>
            {totalRevs > 0 && (
              <span className="text-[10px] px-1.5 py-0.2 rounded bg-zinc-900 text-orange-300 border border-zinc-700 font-semibold">
                {totalRevs}
              </span>
            )}
          </button>

          <button
            onClick={() => setActiveTab('timetravel')}
            className={`px-3 py-1.5 rounded-md flex items-center space-x-1.5 transition whitespace-nowrap shrink-0 text-xs ${
              activeTab === 'timetravel'
                ? 'bg-zinc-800 text-orange-400 border border-orange-500/40 font-medium'
                : 'text-zinc-400 hover:text-zinc-200 hover:bg-zinc-900/80 border border-transparent'
            }`}
          >
            <Rewind className="w-3.5 h-3.5 shrink-0" />
            <span>TIME TRAVEL</span>
          </button>

          <button
            onClick={() => setActiveTab('manual')}
            className={`px-3 py-1.5 rounded-md flex items-center space-x-1.5 transition whitespace-nowrap shrink-0 text-xs ${
              activeTab === 'manual'
                ? 'bg-zinc-800 text-amber-400 border border-amber-500/40 font-medium'
                : 'text-zinc-400 hover:text-zinc-200 hover:bg-zinc-900/80 border border-transparent'
            }`}
          >
            <UserCheck className="w-3.5 h-3.5 text-amber-400 shrink-0" />
            <span>MANUAL DRIFT</span>
            {manualCount > 0 && (
              <span className="text-[10px] px-1.5 py-0.2 rounded bg-amber-950/80 text-amber-300 border border-amber-800/80 font-bold">
                {manualCount}
              </span>
            )}
          </button>

          <button
            onClick={() => setActiveTab('incidents')}
            className={`px-3 py-1.5 rounded-md flex items-center space-x-1.5 transition whitespace-nowrap shrink-0 text-xs ${
              activeTab === 'incidents'
                ? 'bg-zinc-800 text-rose-400 border border-rose-500/40 font-medium'
                : 'text-zinc-400 hover:text-zinc-200 hover:bg-zinc-900/80 border border-transparent'
            }`}
          >
            <AlertTriangle className="w-3.5 h-3.5 text-rose-400 shrink-0" />
            <span>INCIDENT RCA</span>
            {incidentCount > 0 && (
              <span className="text-[10px] px-1.5 py-0.2 rounded bg-rose-950/90 text-rose-300 border border-rose-800 font-bold animate-pulse">
                {incidentCount}
              </span>
            )}
          </button>
        </nav>

        {/* Right Side: Status Badges and Refresh (Desktop) */}
        <div className="hidden md:flex items-center space-x-2 text-xs font-mono shrink-0">
          {/* Cluster Status */}
          <div className="flex items-center space-x-1.5 px-2.5 py-1 rounded-md bg-zinc-900 border border-zinc-800 text-zinc-400">
            <Server className={`w-3.5 h-3.5 ${isConnected ? 'text-emerald-400' : 'text-zinc-500'}`} />
            <span className="text-[11px] text-zinc-300 font-medium">
              {isConnected ? (inCluster ? 'CLUSTER: IN-CLUSTER' : 'CLUSTER: ONLINE') : 'STANDALONE'}
            </span>
          </div>

          {/* SSE Live Status */}
          <div className="flex items-center space-x-1.5 px-2.5 py-1 rounded-md bg-zinc-900 border border-zinc-800 text-zinc-400">
            <span className={`w-2 h-2 rounded-full ${sseConnected ? 'bg-emerald-400 animate-pulse' : 'bg-zinc-600'}`} />
            <span className="text-[11px] font-medium tracking-wide">
              {sseConnected ? 'STREAM: LIVE' : 'STREAM: POLL'}
            </span>
          </div>

          {/* Refresh Action */}
          <button
            onClick={onRefresh}
            className="p-1.5 rounded-md bg-zinc-900 hover:bg-zinc-800 text-zinc-400 hover:text-zinc-200 border border-zinc-800 transition"
            title="Refresh Data from Cluster"
          >
            <RefreshCw className="w-3.5 h-3.5" />
          </button>
        </div>

      </div>
    </header>
  )
}
