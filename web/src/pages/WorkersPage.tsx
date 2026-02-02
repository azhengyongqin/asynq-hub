import { useEffect, useState } from 'react'
import { useTranslation } from 'react-i18next'
import { Button } from '@/components/ui/button'
import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/card'
import { Badge } from '@/components/ui/badge'
import { Input } from '@/components/ui/input'
import { 
  RefreshCw, 
  Server, 
  Cpu, 
  Clock, 
  Activity, 
  Database, 
  Layers, 
  TrendingUp, 
  CheckCircle2, 
  AlertCircle, 
  Timer, 
  Gauge,
  ChevronRight,
  Zap,
  Network,
  Search,
  Settings2,
  BarChart3
} from 'lucide-react'
import { cn } from '@/lib/utils'
import {
  PieChart,
  Pie,
  Cell,
  XAxis,
  YAxis,
  CartesianGrid,
  Tooltip,
  ResponsiveContainer,
  AreaChart,
  Area
} from 'recharts'

// 队列组配置（对应后端 QueueGroupConfig）
type QueueGroupConfig = {
  name: string           // 队列组名称
  concurrency: number    // 并发数
  priorities: Record<string, number>  // 优先级权重 { critical: 50, default: 30, low: 10 }
}

type Worker = {
  worker_name: string
  base_url?: string
  redis_addr?: string
  queue_groups: QueueGroupConfig[]  // 队列组配置列表
  default_retry_count: number
  default_timeout: number
  default_delay: number
  is_enabled: boolean
  last_heartbeat_at?: string
}

type WorkerStats = {
  total_tasks: number
  pending_tasks: number
  running_tasks: number
  success_tasks: number
  failed_tasks: number
  dead_tasks: number // Note: backend doesn't seem to have dead_tasks in the struct I saw, but maybe it's in total_tasks - success - failed - pending - running?
  success_rate: number
  avg_duration_ms?: number
  queue_stats: Record<string, number>
  queue_success_stats?: Record<string, number>
  queue_failed_stats?: Record<string, number>
  queue_avg_duration?: Record<string, number>
}

// QueueStats 用于队列性能统计展示
interface QueueStats {
  queue: string
  total_tasks: number
  success_tasks: number
  failed_tasks: number
  avg_duration_ms?: number
  throughput_per_hour: number
}

type TimeSeriesData = {
  timestamp: string
  success_count: number
  fail_count: number
  avg_duration_ms?: number
}

const STATUS_COLORS: Record<string, string> = {
  success: 'hsl(var(--chart-2))',
  pending: 'hsl(var(--chart-1))',
  running: 'hsl(var(--chart-4))',
  fail: 'hsl(var(--destructive))',
  dead: 'hsl(var(--foreground))',
}

export default function WorkersPage() {
  const { t } = useTranslation()
  const [workers, setWorkers] = useState<Worker[]>([])
  const [selectedWorker, setSelectedWorker] = useState<string | null>(null)
  const [stats, setStats] = useState<WorkerStats | null>(null)
  const [timeseries, setTimeseries] = useState<TimeSeriesData[]>([])
  const [loading, setLoading] = useState(false)
  const [statsLoading, setStatsLoading] = useState(false)
  const [_error, setError] = useState<string | null>(null)

  const [searchQuery, setSearchQuery] = useState('')
  const [queueSearchQuery, setQueueSearchQuery] = useState('')

  async function fetchWorkers() {
    setLoading(true)
    setError(null)
    try {
      const res = await fetch('/api/v1/workers')
      if (!res.ok) throw new Error(`${t('workers.requestFailed')}: ${res.status}`)
      const data = (await res.json()) as { items: Worker[] }
      const items = data.items ?? []
      setWorkers(items)
      
      if (items.length > 0 && !selectedWorker) {
        setSelectedWorker(items[0].worker_name)
      }
    } catch (e) {
      setError(e instanceof Error ? e.message : t('workers.unknownError'))
    } finally {
      setLoading(false)
    }
  }

  const filteredWorkers = workers.filter(w => 
    w.worker_name.toLowerCase().includes(searchQuery.toLowerCase())
  )

  async function fetchWorkerStats(workerName: string) {
    setStatsLoading(true)
    try {
      const [statsRes, timeseriesRes] = await Promise.all([
        fetch(`/api/v1/workers/${encodeURIComponent(workerName)}/stats`),
        fetch(`/api/v1/workers/${encodeURIComponent(workerName)}/timeseries?hours=24`)
      ])

      if (statsRes.ok) {
        const data = await statsRes.json()
        setStats(data.stats)
      }

      if (timeseriesRes.ok) {
        const data = await timeseriesRes.json()
        setTimeseries(data.timeseries ?? [])
      }
    } catch (e) {
      console.error(`${t('workers.fetchStatsFailed')}:`, e)
    } finally {
      setStatsLoading(false)
    }
  }

  useEffect(() => {
    fetchWorkers()
  }, [])

  useEffect(() => {
    if (selectedWorker) {
      fetchWorkerStats(selectedWorker)
    }
  }, [selectedWorker])

  const refresh = () => {
    fetchWorkers()
    if (selectedWorker) {
      fetchWorkerStats(selectedWorker)
    }
  }

  // 准备饼图数据
  const pieData = stats
    ? [
        { name: t('workers.status.success'), value: stats.success_tasks, color: STATUS_COLORS.success },
        { name: t('workers.status.pending'), value: stats.pending_tasks, color: STATUS_COLORS.pending },
        { name: t('workers.status.running'), value: stats.running_tasks, color: STATUS_COLORS.running },
        { name: t('workers.status.fail'), value: stats.failed_tasks, color: STATUS_COLORS.fail },
        { name: t('workers.status.dead'), value: stats.dead_tasks || 0, color: STATUS_COLORS.dead },
      ].filter(d => d.value > 0)
    : []

  // 准备队列统计数据
  const processedQueueStats: QueueStats[] = stats?.queue_stats ? Object.entries(stats.queue_stats).map(([queue, total]) => ({
    queue,
    total_tasks: total,
    success_tasks: stats.queue_success_stats?.[queue] || 0,
    failed_tasks: stats.queue_failed_stats?.[queue] || 0,
    avg_duration_ms: stats.queue_avg_duration?.[queue],
    throughput_per_hour: 0 // 后端暂未直接提供，前端可根据需求计算或留空
  })) : []

  // 准备时间序列数据
  const successLabel = t('workers.stats.success')
  const failLabel = t('workers.stats.fail')
  const avgDurationLabel = t('workers.stats.avgDurationLabel')
  
  const chartData = timeseries.map((item) => ({
    time: new Date(item.timestamp).toLocaleString('zh-CN', {
      hour: '2-digit',
      minute: '2-digit',
    }),
    fullTime: new Date(item.timestamp).toLocaleString('zh-CN'),
    [successLabel]: item.success_count,
    [failLabel]: item.fail_count,
    [avgDurationLabel]: item.avg_duration_ms ? Math.round(item.avg_duration_ms) : 0,
  }))

  const selectedWorkerData = workers.find(w => w.worker_name === selectedWorker)

  return (
    <div className="flex h-full gap-6 overflow-hidden">
      {/* Sidebar: Worker List */}
      <aside className="hidden lg:flex flex-col w-72 shrink-0 gap-4 min-h-0 border-r pr-6">
        <div className="space-y-4 flex flex-col h-full">
          <div>
            <h2 className="text-xl font-bold tracking-tight mb-1">{t('workers.title')}</h2>
            <p className="text-xs text-muted-foreground">{t('workers.subtitle')}</p>
          </div>
          
          <div className="relative group">
            <Search className="absolute left-2.5 top-1/2 -translate-y-1/2 h-4 w-4 text-muted-foreground group-focus-within:text-primary transition-colors" />
            <Input
              placeholder={t('workers.searchPlaceholder', 'Search workers...')}
              className="pl-9 h-9 bg-muted/50 border-none focus-visible:ring-1"
              value={searchQuery}
              onChange={(e) => setSearchQuery(e.target.value)}
            />
          </div>

          <div className="flex-1 overflow-y-auto min-h-0 -mr-2 pr-2 space-y-1 scrollbar-thin scrollbar-thumb-muted-foreground/10 hover:scrollbar-thumb-muted-foreground/20">
            {loading && workers.length === 0 ? (
              Array.from({ length: 5 }).map((_, i) => (
                <div key={i} className="h-12 w-full animate-pulse bg-muted rounded-xl mb-2" />
              ))
            ) : (
              filteredWorkers.map((worker) => (
                <button
                  key={worker.worker_name}
                  onClick={() => setSelectedWorker(worker.worker_name)}
                  className={cn(
                    "w-full flex items-center justify-between px-3 py-3 rounded-xl transition-all duration-200 group text-left border border-transparent",
                    selectedWorker === worker.worker_name
                      ? "bg-primary text-primary-foreground shadow-lg shadow-primary/20 scale-[1.02] border-primary/10"
                      : "hover:bg-muted/80 text-foreground"
                  )}
                >
                  <div className="flex items-center gap-3 overflow-hidden">
                    <div className={cn(
                      "flex-none h-8 w-8 rounded-lg flex items-center justify-center transition-colors",
                      selectedWorker === worker.worker_name ? "bg-primary-foreground/20" : "bg-muted group-hover:bg-background"
                    )}>
                      <Server className="h-4 w-4" />
                    </div>
                    <div className="overflow-hidden">
                      <p className="text-sm font-bold truncate tracking-tight">{worker.worker_name}</p>
                      <div className="flex items-center gap-1.5 mt-0.5">
                        <div className={cn("h-1.5 w-1.5 rounded-full", worker.is_enabled ? "bg-green-500" : "bg-muted-foreground/40")} />
                        <span className={cn("text-[10px] uppercase font-black tracking-widest opacity-60", selectedWorker === worker.worker_name ? "text-primary-foreground" : "text-muted-foreground")}>
                          {worker.is_enabled ? 'Active' : 'Offline'}
                        </span>
                      </div>
                    </div>
                  </div>
                  <ChevronRight className={cn("h-4 w-4 opacity-0 -translate-x-2 transition-all", selectedWorker === worker.worker_name ? "opacity-100 translate-x-0" : "group-hover:opacity-40")} />
                </button>
              ))
            )}
            {filteredWorkers.length === 0 && !loading && (
              <div className="py-12 text-center">
                <Search className="h-8 w-8 mx-auto text-muted-foreground opacity-20 mb-3" />
                <p className="text-xs text-muted-foreground">{t('workers.noWorkerFound', 'No workers match')}</p>
              </div>
            )}
          </div>

          <div className="pt-4 mt-auto border-t">
            <Button variant="outline" size="sm" className="w-full h-9 rounded-xl font-bold uppercase tracking-widest text-[10px]" onClick={refresh} disabled={loading || statsLoading}>
              <RefreshCw className={cn("mr-2 h-3.5 w-3.5", (loading || statsLoading) && "animate-spin")} />
              {t('workers.refresh')}
            </Button>
          </div>
        </div>
      </aside>

      {/* Main Content Area */}
      <main className="flex-1 min-w-0 overflow-y-auto pr-2 pb-10 scrollbar-thin scrollbar-thumb-muted-foreground/10 hover:scrollbar-thumb-muted-foreground/20">
        <div className="lg:hidden flex gap-2 overflow-x-auto pb-4 mb-4 scrollbar-none">
          {workers.map((worker) => (
            <Button
              key={worker.worker_name}
              variant={selectedWorker === worker.worker_name ? "default" : "outline"}
              size="sm"
              className="whitespace-nowrap rounded-full px-4 h-8 shrink-0"
              onClick={() => setSelectedWorker(worker.worker_name)}
            >
              <Server className="mr-2 h-3 w-3" />
              {worker.worker_name}
            </Button>
          ))}
        </div>

        {selectedWorkerData ? (
          <div className="space-y-8 animate-in fade-in slide-in-from-bottom-2 duration-500">
            {/* 1. Metric Overview Cards */}
            <div className="grid gap-4 sm:grid-cols-2 xl:grid-cols-4">
              {[
                { 
                  label: t('workers.stats.totalTasks'), 
                  value: stats?.total_tasks || 0, 
                  icon: Zap, 
                  color: "blue",
                  extra: `${stats?.running_tasks || 0} ${t('workers.stats.running')} • ${stats?.pending_tasks || 0} ${t('workers.stats.waiting')}`
                },
                { 
                  label: t('workers.stats.successRate'), 
                  value: `${stats?.success_rate.toFixed(1) || 0}%`, 
                  icon: CheckCircle2, 
                  color: "green",
                  extra: `${stats?.success_tasks || 0} ${t('workers.stats.success')} • ${stats?.failed_tasks || 0} ${t('workers.stats.fail')}`
                },
                { 
                  label: t('workers.stats.avgDuration'), 
                  value: stats?.avg_duration_ms ? `${Math.round(stats.avg_duration_ms)}ms` : 'N/A', 
                  icon: Timer, 
                  color: "orange",
                  extra: t('workers.stats.avgDurationHint')
                },
                { 
                  label: t('workers.stats.deadTasks'), 
                  value: stats?.dead_tasks || 0, 
                  icon: AlertCircle, 
                  color: "red",
                  extra: t('workers.stats.deadTasksHint')
                }
              ].map((m, i) => (
                <Card key={i} className="border-none shadow-sm bg-muted/30 overflow-hidden relative group hover:bg-muted/50 transition-colors">
                  <div className={cn("absolute top-0 left-0 w-1 h-full", 
                    m.color === 'blue' ? "bg-blue-500" : 
                    m.color === 'green' ? "bg-green-500" : 
                    m.color === 'orange' ? "bg-orange-500" : "bg-red-500"
                  )} />
                  <CardContent className="p-5">
                    <div className="flex items-center justify-between mb-3">
                      <span className="text-[10px] font-black uppercase tracking-widest text-muted-foreground">{m.label}</span>
                      <m.icon className={cn("h-4 w-4 opacity-40 group-hover:opacity-100 transition-opacity", 
                        m.color === 'blue' ? "text-blue-500" : 
                        m.color === 'green' ? "text-green-500" : 
                        m.color === 'orange' ? "text-orange-500" : "text-red-500"
                      )} />
                    </div>
                    <div className="text-3xl font-black tracking-tight mb-1">{m.value}</div>
                    <p className="text-[10px] font-bold text-muted-foreground opacity-70 tracking-tight">{m.extra}</p>
                  </CardContent>
                </Card>
              ))}
            </div>

            {/* 2. Config & Info Section (Vertical Layout) */}
            <div className="space-y-6">
              {/* Configuration Details - Now Full Width */}
              <Card className="border-none bg-muted/20 shadow-sm overflow-hidden">
                <CardHeader className="flex flex-row items-center justify-between space-y-0 pb-6 bg-muted/10">
                  <div className="flex items-center gap-2">
                    <Settings2 className="h-4 w-4 text-primary" />
                    <CardTitle className="text-sm font-black uppercase tracking-widest text-muted-foreground">{t('workers.configTitle', 'Configuration')}</CardTitle>
                  </div>
                  <Badge variant={selectedWorkerData.is_enabled ? 'default' : 'secondary'} className="px-3 rounded-full font-black text-[10px] uppercase tracking-widest">
                    <div className={cn("mr-1.5 h-1.5 w-1.5 rounded-full animate-pulse", selectedWorkerData.is_enabled ? "bg-green-400" : "bg-muted-foreground")} />
                    {selectedWorkerData.is_enabled ? t('workers.active') : t('workers.disabled')}
                  </Badge>
                </CardHeader>
                <CardContent className="pt-6">
                  <div className="grid grid-cols-2 md:grid-cols-4 gap-8">
                    {[
                      { icon: Layers, label: t('workers.config.queueGroups'), value: selectedWorkerData.queue_groups?.length || 0 },
                      { icon: Activity, label: t('workers.config.retryCount'), value: selectedWorkerData.default_retry_count },
                      { icon: Clock, label: t('workers.config.timeout'), value: `${selectedWorkerData.default_timeout}s` },
                      { icon: Database, label: t('workers.config.redis'), value: selectedWorkerData.redis_addr?.split(':')[0] || 'Default', mono: true }
                    ].map((item, i) => (
                      <div key={i} className="space-y-2">
                        <div className="flex items-center gap-2 text-muted-foreground">
                          <item.icon className="h-3.5 w-3.5 opacity-60" />
                          <span className="text-[9px] font-black uppercase tracking-tighter">{item.label}</span>
                        </div>
                        <div className={cn("text-xl font-black tracking-tight truncate", item.mono && "font-mono text-sm")}>
                          {item.value}
                        </div>
                      </div>
                    ))}
                  </div>
                  
                  {/* Info Bar */}
                  <div className="mt-8 grid md:grid-cols-2 gap-4">
                    {selectedWorkerData.base_url && (
                      <div className="p-3 rounded-xl bg-background/50 border flex items-center gap-3 overflow-hidden group hover:border-primary/20 transition-colors">
                        <div className="h-8 w-8 rounded-lg bg-primary/10 flex items-center justify-center text-primary shrink-0">
                          <Network className="h-4 w-4" />
                        </div>
                        <div className="overflow-hidden">
                          <p className="text-[9px] font-black uppercase tracking-widest text-muted-foreground opacity-60">Base URL</p>
                          <p className="text-xs font-mono font-bold truncate">{selectedWorkerData.base_url}</p>
                        </div>
                      </div>
                    )}
                    {selectedWorkerData.last_heartbeat_at && (
                      <div className="p-3 rounded-xl bg-background/50 border flex items-center gap-3 overflow-hidden group hover:border-primary/20 transition-colors">
                        <div className="h-8 w-8 rounded-lg bg-orange-500/10 flex items-center justify-center text-orange-500 shrink-0">
                          <Clock className="h-4 w-4" />
                        </div>
                        <div className="overflow-hidden">
                          <p className="text-[9px] font-black uppercase tracking-widest text-muted-foreground opacity-60">{t('workers.config.lastHeartbeat')}</p>
                          <p className="text-xs font-bold">{new Date(selectedWorkerData.last_heartbeat_at).toLocaleString()}</p>
                        </div>
                      </div>
                    )}
                  </div>
                </CardContent>
              </Card>

              {/* Queue Groups Section - Show queue groups with priorities */}
              <Card className="border-none bg-muted/20 shadow-sm overflow-hidden flex flex-col">
                <CardHeader className="py-3 bg-muted/10 border-b flex flex-row items-center justify-between space-y-0">
                  <div className="flex items-center gap-2">
                    <Layers className="h-4 w-4 text-primary" />
                    <CardTitle className="text-sm font-black uppercase tracking-widest text-muted-foreground">{t('workers.config.queueGroups')}</CardTitle>
                  </div>
                  <div className="relative w-48 group">
                    <Search className="absolute left-2 top-1/2 -translate-y-1/2 h-3 w-3 text-muted-foreground group-focus-within:text-primary transition-colors" />
                    <Input
                      placeholder={t('workers.searchQueues', 'Search queues...')}
                      className="pl-7 h-7 text-[10px] bg-background/50 border-none focus-visible:ring-1 rounded-lg"
                      value={queueSearchQuery}
                      onChange={(e) => setQueueSearchQuery(e.target.value)}
                    />
                  </div>
                </CardHeader>
                <CardContent className="p-0">
                  <div className="max-h-[320px] overflow-y-auto scrollbar-thin scrollbar-thumb-muted-foreground/10 hover:scrollbar-thumb-muted-foreground/20">
                    <div className="divide-y divide-border/40">
                      {(selectedWorkerData.queue_groups || [])
                        .filter((qg) => qg.name.toLowerCase().includes(queueSearchQuery.toLowerCase()))
                        .map((queueGroup) => (
                        <div key={queueGroup.name} className="px-6 py-3 hover:bg-background/40 transition-colors group">
                          {/* Queue Group Header */}
                          <div className="flex items-center justify-between mb-2">
                            <div className="flex items-center gap-3 overflow-hidden">
                              <div className="h-2 w-2 rounded-full bg-primary/50 group-hover:bg-primary transition-colors shrink-0" />
                              <span className="text-[12px] font-black font-mono tracking-tight uppercase truncate">{queueGroup.name}</span>
                            </div>
                            <div className="flex items-center gap-2">
                              <Cpu className="h-3 w-3 text-muted-foreground opacity-50" />
                              <Badge variant="outline" className="h-5 text-[10px] font-black bg-background border-blue-500/20 text-blue-600 dark:text-blue-400 rounded-md px-2">
                                {t('workers.config.concurrency')}: {queueGroup.concurrency}
                              </Badge>
                            </div>
                          </div>
                          {/* Priorities */}
                          <div className="ml-5 flex flex-wrap gap-1.5">
                            {Object.entries(queueGroup.priorities || {})
                              .sort(([, a], [, b]) => b - a)
                              .map(([priority, weight]) => (
                              <Badge 
                                key={priority} 
                                variant="secondary" 
                                className={cn(
                                  "h-5 text-[9px] font-bold rounded-md px-2",
                                  priority === 'critical' && "bg-red-100 text-red-700 dark:bg-red-900/30 dark:text-red-400",
                                  priority === 'default' && "bg-slate-100 text-slate-700 dark:bg-slate-800/50 dark:text-slate-300",
                                  priority === 'low' && "bg-green-100 text-green-700 dark:bg-green-900/30 dark:text-green-400"
                                )}
                              >
                                {priority}: {weight}
                              </Badge>
                            ))}
                          </div>
                        </div>
                      ))}
                      {(selectedWorkerData.queue_groups || []).filter((qg) => qg.name.toLowerCase().includes(queueSearchQuery.toLowerCase())).length === 0 && (
                        <div className="py-8 text-center">
                          <p className="text-[10px] text-muted-foreground font-medium opacity-50 italic">{t('workers.noQueueGroups', 'No queue groups found')}</p>
                        </div>
                      )}
                    </div>
                  </div>
                </CardContent>
              </Card>
            </div>

            {/* 3. Analytics Section */}
            <div className="grid gap-6 xl:grid-cols-5">
              {/* Distribution Chart */}
              <Card className="xl:col-span-2 border-none bg-muted/30 shadow-sm relative overflow-hidden group">
                <CardHeader className="pb-2">
                  <div className="flex items-center justify-between">
                    <div className="flex items-center gap-2">
                      <BarChart3 className="h-4 w-4 text-primary" />
                      <CardTitle className="text-sm font-black uppercase tracking-widest text-muted-foreground">{t('workers.stats.taskDistribution')}</CardTitle>
                    </div>
                    <Badge variant="outline" className="text-[10px] font-black uppercase tracking-tighter border-muted-foreground/20">Overall</Badge>
                  </div>
                </CardHeader>
                <CardContent>
                  <div className="flex flex-col md:flex-row items-center gap-6 pt-4">
                    <div className="w-full max-w-[200px] aspect-square relative">
                      <ResponsiveContainer width="100%" height="100%">
                        <PieChart>
                          <Pie
                            data={pieData}
                            cx="50%"
                            cy="50%"
                            innerRadius={65}
                            outerRadius={85}
                            paddingAngle={4}
                            dataKey="value"
                            stroke="none"
                          >
                            {pieData.map((entry, index) => (
                              <Cell key={`cell-${index}`} fill={entry.color} />
                            ))}
                          </Pie>
                          <Tooltip 
                            contentStyle={{ 
                              backgroundColor: 'hsl(var(--popover))', 
                              borderColor: 'hsl(var(--border))',
                              borderRadius: '12px',
                              fontSize: '11px',
                              boxShadow: '0 8px 30px rgba(0,0,0,0.12)',
                              border: 'none',
                              padding: '8px 12px'
                            }} 
                          />
                        </PieChart>
                      </ResponsiveContainer>
                      <div className="absolute inset-0 flex flex-col items-center justify-center pointer-events-none">
                        <span className="text-3xl font-black tracking-tighter">{stats?.total_tasks || 0}</span>
                        <span className="text-[9px] uppercase font-black text-muted-foreground tracking-widest opacity-60">Tasks</span>
                      </div>
                    </div>
                    
                    <div className="flex-1 grid grid-cols-2 gap-x-6 gap-y-3 w-full">
                      {pieData.map((item, idx) => (
                        <div key={idx} className="flex flex-col gap-1">
                          <div className="flex items-center gap-2">
                            <div className="h-2 w-2 rounded-full shrink-0" style={{ backgroundColor: item.color }} />
                            <span className="text-[10px] font-black uppercase tracking-widest text-muted-foreground truncate opacity-70">{item.name}</span>
                          </div>
                          <div className="flex items-baseline gap-1 pl-4">
                            <span className="text-sm font-black">{item.value}</span>
                            <span className="text-[9px] font-bold text-muted-foreground">({((item.value / (stats?.total_tasks || 1)) * 100).toFixed(0)}%)</span>
                          </div>
                        </div>
                      ))}
                    </div>
                  </div>
                </CardContent>
              </Card>

              {/* Queue Performance List */}
              <Card className="xl:col-span-3 border-none bg-muted/30 shadow-sm flex flex-col overflow-hidden">
                <CardHeader className="pb-4">
                  <div className="flex items-center gap-2">
                    <Gauge className="h-4 w-4 text-primary" />
                    <CardTitle className="text-sm font-black uppercase tracking-widest text-muted-foreground">{t('workers.stats.queuePerformance')}</CardTitle>
                  </div>
                </CardHeader>
                <CardContent className="flex-1 min-h-0 overflow-y-auto scrollbar-thin scrollbar-thumb-muted-foreground/10 hover:scrollbar-thumb-muted-foreground/20 pr-4">
                  <div className="space-y-3">
                    {processedQueueStats.map((qs) => (
                      <div key={qs.queue} className="p-4 rounded-2xl bg-background/40 border border-transparent hover:border-primary/20 hover:bg-background/60 transition-all group">
                        <div className="flex items-center justify-between mb-3">
                          <div className="flex items-center gap-3 overflow-hidden">
                            <div className="h-8 w-8 rounded-lg bg-muted flex items-center justify-center text-muted-foreground shrink-0 group-hover:bg-primary/10 group-hover:text-primary transition-colors">
                              <Layers className="h-4 w-4" />
                            </div>
                            <span className="font-mono text-xs font-black truncate tracking-tighter uppercase">{qs.queue.split(':')[1] || qs.queue}</span>
                          </div>
                          <div className="flex items-center gap-4 text-right">
                            <div className="flex flex-col">
                              <span className="text-[9px] font-black uppercase tracking-widest text-muted-foreground opacity-60">Rate</span>
                              <span className="text-xs font-black text-green-500">{((qs.success_tasks / (qs.total_tasks || 1)) * 100).toFixed(1)}%</span>
                            </div>
                            <div className="flex flex-col">
                              <span className="text-[9px] font-black uppercase tracking-widest text-muted-foreground opacity-60">Total</span>
                              <span className="text-xs font-black">{qs.total_tasks}</span>
                            </div>
                          </div>
                        </div>
                        
                        <div className="space-y-2">
                          <div className="w-full bg-muted/50 rounded-full h-1.5 overflow-hidden">
                            <div
                              className="bg-green-500 h-full rounded-full transition-all duration-1000 shadow-[0_0_8px_rgba(34,197,94,0.4)]"
                              style={{ width: `${(qs.success_tasks / (qs.total_tasks || 1)) * 100}%` }}
                            />
                          </div>
                          <div className="flex justify-between items-center px-0.5">
                            <div className="flex items-center gap-1.5 text-[9px] font-black uppercase tracking-widest text-muted-foreground">
                              <Zap className="h-3 w-3 text-primary opacity-60" />
                              {qs.throughput_per_hour.toFixed(1)}/h
                            </div>
                            {qs.avg_duration_ms !== undefined && (
                              <div className="flex items-center gap-1.5 text-[9px] font-black uppercase tracking-widest text-orange-500">
                                <Timer className="h-3 w-3 opacity-60" />
                                {Math.round(qs.avg_duration_ms)}ms
                              </div>
                            )}
                          </div>
                        </div>
                      </div>
                    ))}
                  </div>
                </CardContent>
              </Card>
            </div>

            {/* 4. Large Trend Area Chart */}
            {chartData.length > 0 && (
              <Card className="border-none bg-muted/30 shadow-sm overflow-hidden group">
                <CardHeader className="flex flex-row items-center justify-between pb-6">
                  <div className="space-y-1">
                    <div className="flex items-center gap-2">
                      <TrendingUp className="h-4 w-4 text-primary" />
                      <CardTitle className="text-sm font-black uppercase tracking-widest text-muted-foreground">{t('workers.stats.taskTrend')}</CardTitle>
                    </div>
                    <p className="text-[10px] font-bold text-muted-foreground opacity-60 tracking-tight">{t('workers.stats.taskTrendSubtitle')}</p>
                  </div>
                  <div className="flex gap-4 p-1.5 bg-background/50 rounded-full border px-4 shadow-inner">
                    {[
                      { label: 'SUCCESS', color: "hsl(var(--chart-2))" },
                      { label: 'LATENCY', color: "hsl(var(--chart-1))" }
                    ].map(l => (
                      <div key={l.label} className="flex items-center gap-2 text-[9px] font-black tracking-widest opacity-70">
                        <div className="h-1.5 w-1.5 rounded-full" style={{ backgroundColor: l.color }} />
                        {l.label}
                      </div>
                    ))}
                  </div>
                </CardHeader>
                <CardContent>
                  <div className="h-[320px] w-full pt-4">
                    <ResponsiveContainer width="100%" height="100%">
                      <AreaChart data={chartData} margin={{ top: 10, right: 10, left: -20, bottom: 0 }}>
                        <defs>
                          <linearGradient id="colorSuccess" x1="0" y1="0" x2="0" y2="1">
                            <stop offset="5%" stopColor="hsl(var(--chart-2))" stopOpacity={0.15}/>
                            <stop offset="95%" stopColor="hsl(var(--chart-2))" stopOpacity={0}/>
                          </linearGradient>
                          <linearGradient id="colorLatency" x1="0" y1="0" x2="0" y2="1">
                            <stop offset="5%" stopColor="hsl(var(--chart-1))" stopOpacity={0.15}/>
                            <stop offset="95%" stopColor="hsl(var(--chart-1))" stopOpacity={0}/>
                          </linearGradient>
                        </defs>
                        <CartesianGrid strokeDasharray="3 3" vertical={false} stroke="hsl(var(--border))" opacity={0.4} />
                        <XAxis 
                          dataKey="time" 
                          tick={{ fontSize: 9, fontWeight: 800, fill: 'hsl(var(--muted-foreground))' }} 
                          axisLine={false}
                          tickLine={false}
                          dy={10}
                        />
                        <YAxis 
                          yAxisId="left" 
                          tick={{ fontSize: 9, fontWeight: 800, fill: 'hsl(var(--muted-foreground))' }} 
                          axisLine={false}
                          tickLine={false}
                        />
                        <YAxis 
                          yAxisId="right" 
                          orientation="right" 
                          tick={{ fontSize: 9, fontWeight: 800, fill: 'hsl(var(--muted-foreground))' }} 
                          axisLine={false}
                          tickLine={false}
                        />
                        <Tooltip 
                          cursor={{ stroke: 'hsl(var(--primary))', strokeWidth: 1, strokeDasharray: '4 4' }}
                          contentStyle={{ 
                            backgroundColor: 'hsl(var(--popover))', 
                            borderColor: 'hsl(var(--border))',
                            borderRadius: '12px',
                            fontSize: '11px',
                            boxShadow: '0 10px 40px rgba(0,0,0,0.15)',
                            border: 'none',
                            padding: '12px'
                          }}
                          labelStyle={{ fontWeight: 900, color: 'hsl(var(--foreground))', marginBottom: '8px', fontSize: '12px', letterSpacing: '0.05em' }}
                          labelFormatter={(label, payload) => payload[0]?.payload?.fullTime || label}
                        />
                        <Area 
                          yAxisId="left" 
                          type="monotone" 
                          dataKey={successLabel} 
                          stroke="hsl(var(--chart-2))" 
                          strokeWidth={4}
                          fillOpacity={1} 
                          fill="url(#colorSuccess)" 
                          activeDot={{ r: 6, strokeWidth: 0, fill: 'hsl(var(--chart-2))' }}
                        />
                        <Area 
                          yAxisId="right" 
                          type="monotone" 
                          dataKey={avgDurationLabel} 
                          stroke="hsl(var(--chart-1))" 
                          strokeWidth={4}
                          fillOpacity={1} 
                          fill="url(#colorLatency)"
                          activeDot={{ r: 6, strokeWidth: 0, fill: 'hsl(var(--chart-1))' }}
                        />
                      </AreaChart>
                    </ResponsiveContainer>
                  </div>
                </CardContent>
              </Card>
            )}
          </div>
        ) : (
          <div className="flex flex-col items-center justify-center h-full py-20 text-center space-y-6 bg-muted/10 rounded-3xl border-2 border-dashed">
            <div className="h-24 w-24 rounded-full bg-muted flex items-center justify-center relative">
              <Server className="h-10 w-10 text-muted-foreground opacity-30" />
              <div className="absolute inset-0 rounded-full border-2 border-primary/20 animate-ping" />
            </div>
            <div className="space-y-2">
              <h3 className="font-black text-xl tracking-tight">{t('workers.selectWorker')}</h3>
              <p className="text-sm text-muted-foreground max-w-sm mx-auto font-medium leading-relaxed opacity-70">{t('workers.selectWorkerHint')}</p>
            </div>
          </div>
        )}
      </main>
    </div>
  )
}
