import React, { useState } from 'react'
import { X, Zap, AlertTriangle, Play, CheckCircle, RefreshCw } from 'lucide-react'

export default function ScenarioDrawer({ onClose, onScenarioExecuted }) {
  const [running, setRunning] = useState(null)
  const [result, setResult] = useState(null)

  const scenarios = [
    {
      id: 'oom',
      title: 'Memory Limit Squeeze -> OOMKilled',
      severity: 'critical',
      description: 'Simulates a human engineer running `kubectl edit` to slash memory limits from 512Mi to 16Mi, triggering an OOM kill (exit code 137).',
      target: 'payment-processor (Production)',
      rootCause: 'Human manual override outside GitOps pipeline',
    },
    {
      id: 'image_pull',
      title: 'Container Image Typo -> ImagePullBackOff',
      severity: 'critical',
      description: 'Simulates `kubectl set image` with an invalid tag (redis:v999.invalid-tag), triggering ImagePullBackOff / ErrImagePull.',
      target: 'order-service (Staging)',
      rootCause: 'Manual CLI container image update',
    },
    {
      id: 'crash_loop',
      title: 'Corrupted ConfigMap -> CrashLoopBackOff',
      severity: 'critical',
      description: 'Simulates modifying auth-config ConfigMap and omitting DATABASE_URL, causing pods to crash immediately upon bootstrap.',
      target: 'auth-service (Production)',
      rootCause: 'Missing required configuration key in ConfigMap',
    },
    {
      id: 'node_drain',
      title: 'Cordon Worker Node -> FailedScheduling',
      severity: 'warning',
      description: 'Simulates cordoning kind-worker (unschedulable: true), leaving no available schedulable nodes for batch workloads.',
      target: 'batch-analytics-worker (Default)',
      rootCause: 'Cluster node cordoned without workload toleration',
    },
  ]

  const runScenario = async (scenarioId) => {
    setRunning(scenarioId)
    setResult(null)
    try {
      const res = await fetch('/api/scenarios/run', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ scenario: scenarioId }),
      })
      if (res.ok) {
        const json = await res.json()
        setResult({ success: true, message: json.message || 'Scenario executed successfully!' })
        if (onScenarioExecuted) onScenarioExecuted()
      } else {
        const text = await res.text()
        setResult({ success: false, message: `Error: ${text}` })
      }
    } catch (err) {
      setResult({ success: false, message: `Failed: ${err.message}` })
    } finally {
      setRunning(null)
    }
  }

  return (
    <div className="fixed inset-0 z-50 bg-black/80 backdrop-blur-sm flex items-center justify-center p-3 sm:p-4 font-mono text-xs">
      <div className="bg-[#121215] border border-zinc-800 w-full max-w-3xl rounded flex flex-col shadow-2xl overflow-hidden max-h-[90vh]">
        {/* Header */}
        <div className="p-3 sm:p-3.5 border-b border-zinc-800 bg-[#0c0c0e] flex items-center justify-between gap-2">
          <div className="flex items-center space-x-2 min-w-0">
            <div className="p-1.5 rounded bg-purple-950/80 border border-purple-800 text-purple-400 shrink-0">
              <Zap className="w-4 h-4" />
            </div>
            <div className="min-w-0">
              <h2 className="font-bold text-zinc-100 text-xs sm:text-sm uppercase truncate">
                Chaos Simulator & Scenario Lab
              </h2>
              <p className="text-[10px] sm:text-[11px] text-zinc-500 truncate">
                Trigger real-world cluster failure modes and observe live audit recording & RCA
              </p>
            </div>
          </div>

          <button
            onClick={onClose}
            className="p-1.5 rounded bg-zinc-900 hover:bg-zinc-800 text-zinc-400 hover:text-zinc-100 border border-zinc-800 shrink-0"
          >
            <X className="w-4 h-4" />
          </button>
        </div>

        {/* Status result banner */}
        {result && (
          <div className={`p-3 border-b text-xs flex items-center space-x-2 ${
            result.success
              ? 'bg-emerald-950/20 border-emerald-900 text-emerald-300'
              : 'bg-rose-950/20 border-rose-900 text-rose-300'
          }`}>
            {result.success ? <CheckCircle className="w-4 h-4 text-emerald-400" /> : <AlertTriangle className="w-4 h-4 text-rose-400" />}
            <span>{result.message}</span>
          </div>
        )}

        {/* Scenarios List */}
        <div className="p-4 space-y-3 max-h-[70vh] overflow-y-auto">
          {scenarios.map((sc) => (
            <div
              key={sc.id}
              className="bg-[#09090b] border border-zinc-800/80 p-3.5 rounded hover:border-zinc-700 transition space-y-2"
            >
              <div className="flex flex-col sm:flex-row sm:items-start justify-between gap-3">
                <div className="space-y-1 min-w-0">
                  <div className="flex flex-wrap items-center gap-2">
                    <span
                      className={`px-1.5 py-0.2 rounded text-[10px] font-bold uppercase ${
                        sc.severity === 'critical'
                          ? 'bg-rose-950 text-rose-300 border border-rose-800'
                          : 'bg-amber-950 text-amber-300 border border-amber-800'
                      }`}
                    >
                      {sc.severity}
                    </span>
                    <span className="font-bold text-zinc-200 text-xs sm:text-sm">
                      {sc.title}
                    </span>
                  </div>
                  <p className="text-[11px] text-zinc-400 leading-relaxed">
                    {sc.description}
                  </p>
                </div>

                <button
                  onClick={() => runScenario(sc.id)}
                  disabled={running !== null}
                  className="px-3 py-1.5 rounded bg-zinc-800 hover:bg-zinc-700 disabled:opacity-50 text-purple-300 border border-purple-900/60 text-xs font-semibold flex items-center space-x-1.5 whitespace-nowrap transition shrink-0 self-start sm:self-auto"
                >
                  {running === sc.id ? (
                    <>
                      <RefreshCw className="w-3.5 h-3.5 animate-spin" />
                      <span>Injecting...</span>
                    </>
                  ) : (
                    <>
                      <Play className="w-3.5 h-3.5 text-purple-400" />
                      <span>Simulate Incident</span>
                    </>
                  )}
                </button>
              </div>

              <div className="pt-2 border-t border-zinc-900 flex flex-col sm:flex-row sm:items-center justify-between gap-1 text-[10px] text-zinc-500">
                <span>Target: <strong className="text-zinc-400">{sc.target}</strong></span>
                <span>Expected Root Cause: <strong className="text-zinc-400">{sc.rootCause}</strong></span>
              </div>
            </div>
          ))}
        </div>
      </div>
    </div>
  )
}
