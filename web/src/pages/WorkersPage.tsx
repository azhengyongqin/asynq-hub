import { useEffect, useState } from 'react'
import { useTranslation } from 'react-i18next'
import { Button } from '@/components/ui/button'
import { Card, CardContent, CardHeader, CardTitle, CardDescription } from '@/components/ui/card'
import { Badge } from '@/components/ui/badge'
import { Tabs, TabsContent, TabsList, TabsTrigger } from '@/components/ui/tabs'
import { RefreshCw, Server, Cpu, Clock, Activity, Database, Layers, TrendingUp, CheckCircle2, AlertCircle, Timer, Gauge } from 'lucide-react'
import { cn } from '@/lib/utils'
import {
  PieChart,
  Pie,
  Cell,
  LineChart,
  Line,
  XAxis,
  YAxis,
  CartesianGrid,
  Tooltip,
  Legend,
  ResponsiveContainer
} from 'recharts'

type Worker = {
  worker_name: string
  base_url?: string
  redis_addr?: string
  concurrency: number
  queues: Record<string, number>
  default_retry_count: number
  default_timeout: number
  default_delay: number
  is_enabled: boolean
  last_heartbeat_at?: string
}

type WorkerStats = {
  worker_name: string
  total_tasks: number
  pending_tasks: number
  running_tasks: number
  success_tasks: number
  failed_tasks: number
  dead_tasks: number
  success_rate: number
  avg_duration_ms?: number
  queue_stats: QueueStats[]
  status_distribution: Record<string, number>
}

type QueueStats = {
  queue: string
  total_tasks: number
  success_tasks: number
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
  success: '#22c55e',
  pending: '#3b82f6',
  running: '#f59e0b',
  fail: '#ef4444',
  dead: '#991b1b',
}

export default function WorkersPage() {
  const { t } = useTranslation()
  const [workers, setWorkers] = useState<Worker[]>([])
  const [selectedWorker, setSelectedWorker] = useState<string | null>(null)
  const [stats, setStats] = useState<WorkerStats | null>(null)
  const [timeseries, setTimeseries] = useState<TimeSeriesData[]>([])
  const [loading, setLoading] = useState(false)
  const [error, setError] = useState<string | null>(null)

  async function fetchWorkers() {
    setLoading(true)
    setError(null)
    try {
      const res = await fetch('/api/v1/workers')
      if (!res.ok) throw new Error(`${t('workers.requestFailed')}: ${res.status}`)
      const data = (await res.json()) as { items: Worker[] }
      setWorkers(data.items ?? [])
      
      // 默认选择第一个 worker
      if (data.items && data.items.length > 0 && !selectedWorker) {
        setSelectedWorker(data.items[0].worker_name)
      }
    } catch (e) {
      setError(e instanceof Error ? e.message : t('workers.unknownError'))
    } finally {
      setLoading(false)
    }
  }

  async function fetchWorkerStats(workerName: string) {
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
  const pieData = stats && stats.status_distribution
    ? Object.entries(stats.status_distribution).map(([status, count]) => ({
        name: t(`workers.status.${status}`) || status,
        value: count,
        color: STATUS_COLORS[status] || '#666',
      }))
    : []

  // 准备时间序列数据
  const successLabel = t('workers.stats.success')
  const failLabel = t('workers.stats.fail')
  const avgDurationLabel = t('workers.stats.avgDurationLabel')
  
  const chartData = timeseries.map((item) => ({
    time: new Date(item.timestamp).toLocaleString('zh-CN', {
      month: 'short',
      day: 'numeric',
      hour: '2-digit',
    }),
    [successLabel]: item.success_count,
    [failLabel]: item.fail_count,
    [avgDurationLabel]: item.avg_duration_ms ? Math.round(item.avg_duration_ms) : 0,
  }))

  return (
    <div className="space-y-6 h-full overflow-y-auto pr-2 pb-6">
      <div className="flex flex-col gap-4 md:flex-row md:items-center md:justify-between">
        <div>
          <h2 className="text-xl font-semibold tracking-tight">{t('workers.title')}</h2>
          <p className="text-sm text-muted-foreground">
            {t('workers.subtitle')}
          </p>
        </div>
        <Button variant="outline" size="sm" onClick={refresh} disabled={loading}>
          <RefreshCw className={cn("mr-2 h-4 w-4", loading && "animate-spin")} />
          {t('workers.refresh')}
        </Button>
      </div>

      {error && (
        <div className="rounded-md bg-destructive/15 p-3 text-sm text-destructive">
          {t('workers.error')}: {error}
        </div>
      )}

      <Tabs value={selectedWorker || undefined} onValueChange={setSelectedWorker} className="space-y-4">
        <TabsList className="inline-flex">
          {workers.map((worker) => (
            <TabsTrigger key={worker.worker_name} value={worker.worker_name} className="gap-2">
              <Server className="h-3 w-3" />
              {worker.worker_name}
            </TabsTrigger>
          ))}
        </TabsList>

        {workers.map((worker) => (
          <TabsContent key={worker.worker_name} value={worker.worker_name} className="space-y-4">
            {/* Worker 基本信息卡片 */}
            <Card>
              <CardHeader className="flex flex-row items-start justify-between space-y-0 pb-3">
                <div className="space-y-1">
                  <CardTitle className="text-base font-medium font-mono flex items-center gap-2">
                    <Server className="h-4 w-4" />
                    {worker.worker_name}
                  </CardTitle>
                  {worker.base_url && (
                    <CardDescription className="text-xs" title={worker.base_url}>
                      {worker.base_url}
                    </CardDescription>
                  )}
                </div>
                <Badge variant={worker.is_enabled ? 'default' : 'secondary'} className={cn(!worker.is_enabled && "opacity-50")}>
                  {worker.is_enabled ? t('workers.active') : t('workers.disabled')}
                </Badge>
              </CardHeader>
              <CardContent className="space-y-4">
                <div className="grid grid-cols-2 md:grid-cols-4 gap-3 text-sm">
                  <div className="flex items-center gap-2 text-muted-foreground">
                    <Cpu className="h-4 w-4" />
                    <span className="text-foreground font-medium">{worker.concurrency}</span>
                    <span className="text-xs">{t('workers.config.concurrency')}</span>
                  </div>
                  <div className="flex items-center gap-2 text-muted-foreground">
                    <Activity className="h-4 w-4" />
                    <span className="text-foreground font-medium">{worker.default_retry_count}</span>
                    <span className="text-xs">{t('workers.config.retryCount')}</span>
                  </div>
                  <div className="flex items-center gap-2 text-muted-foreground">
                    <Clock className="h-4 w-4" />
                    <span className="text-foreground font-medium">{worker.default_timeout}s</span>
                    <span className="text-xs">{t('workers.config.timeout')}</span>
                  </div>
                  {worker.redis_addr && (
                    <div className="flex items-center gap-2 text-muted-foreground col-span-2 md:col-span-1">
                      <Database className="h-3 w-3" />
                      <span className="text-xs truncate">{worker.redis_addr.split(':')[0]}</span>
                    </div>
                  )}
                </div>

                {/* 队列配置 */}
                <div className="space-y-2">
                  <div className="flex items-center gap-2 text-xs font-semibold text-muted-foreground uppercase tracking-wider">
                    <Layers className="h-3 w-3" />
                    {t('workers.config.queues')}
                  </div>
                  <div className="grid grid-cols-2 md:grid-cols-3 gap-2">
                    {Object.entries(worker.queues).map(([queueName, weight]) => (
                      <div key={queueName} className="flex items-center justify-between p-2 rounded bg-muted/30 border">
                        <span className="text-sm font-mono truncate">{queueName}</span>
                        <Badge variant="outline" className="text-xs ml-2">
                          {weight}
                        </Badge>
                      </div>
                    ))}
                  </div>
                </div>

                {worker.last_heartbeat_at && (
                  <div className="pt-2 border-t text-xs text-muted-foreground">
                    {t('workers.config.lastHeartbeat')}: {new Date(worker.last_heartbeat_at).toLocaleString('zh-CN')}
                  </div>
                )}
              </CardContent>
            </Card>

            {/* 统计概览卡片 */}
            {stats && (
              <div className="grid gap-4 md:grid-cols-2 lg:grid-cols-4">
                <Card>
                  <CardHeader className="flex flex-row items-center justify-between space-y-0 pb-2">
                    <CardTitle className="text-sm font-medium">{t('workers.stats.totalTasks')}</CardTitle>
                    <Activity className="h-4 w-4 text-muted-foreground" />
                  </CardHeader>
                  <CardContent>
                    <div className="text-2xl font-bold">{stats.total_tasks}</div>
                    <p className="text-xs text-muted-foreground mt-1">
                      {t('workers.stats.running')}: {stats.running_tasks} | {t('workers.stats.waiting')}: {stats.pending_tasks}
                    </p>
                  </CardContent>
                </Card>

                <Card>
                  <CardHeader className="flex flex-row items-center justify-between space-y-0 pb-2">
                    <CardTitle className="text-sm font-medium">{t('workers.stats.successRate')}</CardTitle>
                    <CheckCircle2 className="h-4 w-4 text-green-600" />
                  </CardHeader>
                  <CardContent>
                    <div className="text-2xl font-bold">{stats.success_rate.toFixed(1)}%</div>
                    <p className="text-xs text-muted-foreground mt-1">
                      {t('workers.stats.success')}: {stats.success_tasks} | {t('workers.stats.fail')}: {stats.failed_tasks}
                    </p>
                  </CardContent>
                </Card>

                <Card>
                  <CardHeader className="flex flex-row items-center justify-between space-y-0 pb-2">
                    <CardTitle className="text-sm font-medium">{t('workers.stats.avgDuration')}</CardTitle>
                    <Timer className="h-4 w-4 text-muted-foreground" />
                  </CardHeader>
                  <CardContent>
                    <div className="text-2xl font-bold">
                      {stats.avg_duration_ms ? `${Math.round(stats.avg_duration_ms)}ms` : 'N/A'}
                    </div>
                    <p className="text-xs text-muted-foreground mt-1">
                      {t('workers.stats.avgDurationHint')}
                    </p>
                  </CardContent>
                </Card>

                <Card>
                  <CardHeader className="flex flex-row items-center justify-between space-y-0 pb-2">
                    <CardTitle className="text-sm font-medium">{t('workers.stats.deadTasks')}</CardTitle>
                    <AlertCircle className="h-4 w-4 text-red-600" />
                  </CardHeader>
                  <CardContent>
                    <div className="text-2xl font-bold">{stats.dead_tasks}</div>
                    <p className="text-xs text-muted-foreground mt-1">
                      {t('workers.stats.deadTasksHint')}
                    </p>
                  </CardContent>
                </Card>
              </div>
            )}

            {/* 图表区域 */}
            <div className="grid gap-4 md:grid-cols-2">
              {/* 任务状态分布饼图 */}
              {stats && pieData.length > 0 && (
                <Card>
                  <CardHeader>
                    <CardTitle className="text-base flex items-center gap-2">
                      <TrendingUp className="h-4 w-4" />
                      {t('workers.stats.taskDistribution')}
                    </CardTitle>
                    <CardDescription>{t('workers.stats.taskDistributionSubtitle')}</CardDescription>
                  </CardHeader>
                  <CardContent>
                    <ResponsiveContainer width="100%" height={250}>
                      <PieChart>
                        <Pie
                          data={pieData}
                          cx="50%"
                          cy="50%"
                          labelLine={false}
                          label={({ name, percent }) => `${name} ${((percent || 0) * 100).toFixed(0)}%`}
                          outerRadius={80}
                          fill="#8884d8"
                          dataKey="value"
                        >
                          {pieData.map((entry, index) => (
                            <Cell key={`cell-${index}`} fill={entry.color} />
                          ))}
                        </Pie>
                        <Tooltip />
                      </PieChart>
                    </ResponsiveContainer>
                  </CardContent>
                </Card>
              )}

              {/* 队列性能统计 */}
              {stats && stats.queue_stats && stats.queue_stats.length > 0 && (
                <Card>
                  <CardHeader>
                    <CardTitle className="text-base flex items-center gap-2">
                      <Gauge className="h-4 w-4" />
                      {t('workers.stats.queuePerformance')}
                    </CardTitle>
                    <CardDescription>{t('workers.stats.queuePerformanceSubtitle')}</CardDescription>
                  </CardHeader>
                  <CardContent>
                    <div className="space-y-3">
                      {stats.queue_stats.map((qs) => (
                        <div key={qs.queue} className="space-y-1">
                          <div className="flex items-center justify-between text-sm">
                            <span className="font-mono font-medium">{qs.queue.split(':')[1] || qs.queue}</span>
                            <span className="text-muted-foreground">{qs.total_tasks} {t('workers.stats.tasks')}</span>
                          </div>
                          <div className="flex items-center justify-between text-xs text-muted-foreground">
                            <span>{t('workers.stats.success')}: {qs.success_tasks}</span>
                            <span>{t('workers.stats.average')}: {qs.avg_duration_ms ? `${Math.round(qs.avg_duration_ms)}ms` : 'N/A'}</span>
                            <span>{t('workers.stats.throughput')}: {qs.throughput_per_hour.toFixed(1)}/h</span>
                          </div>
                          <div className="w-full bg-muted rounded-full h-2">
                            <div
                              className="bg-green-500 h-2 rounded-full transition-all"
                              style={{ width: `${(qs.success_tasks / qs.total_tasks) * 100}%` }}
                            />
                          </div>
                        </div>
                      ))}
                    </div>
                  </CardContent>
                </Card>
              )}
            </div>

            {/* 时间序列趋势图 */}
            {chartData.length > 0 && (
              <div className="grid gap-4 md:grid-cols-2">
                <Card className="md:col-span-2">
                  <CardHeader>
                    <CardTitle className="text-base flex items-center gap-2">
                      <TrendingUp className="h-4 w-4" />
                      {t('workers.stats.taskTrend')}
                    </CardTitle>
                    <CardDescription>{t('workers.stats.taskTrendSubtitle')}</CardDescription>
                  </CardHeader>
                  <CardContent>
                    <ResponsiveContainer width="100%" height={300}>
                      <LineChart data={chartData}>
                        <CartesianGrid strokeDasharray="3 3" />
                        <XAxis dataKey="time" tick={{ fontSize: 12 }} />
                        <YAxis yAxisId="left" tick={{ fontSize: 12 }} />
                        <YAxis yAxisId="right" orientation="right" tick={{ fontSize: 12 }} />
                        <Tooltip />
                        <Legend />
                        <Line yAxisId="left" type="monotone" dataKey={successLabel} stroke="#22c55e" strokeWidth={2} />
                        <Line yAxisId="left" type="monotone" dataKey={failLabel} stroke="#ef4444" strokeWidth={2} />
                        <Line yAxisId="right" type="monotone" dataKey={avgDurationLabel} stroke="#3b82f6" strokeWidth={2} />
                      </LineChart>
                    </ResponsiveContainer>
                  </CardContent>
                </Card>
              </div>
            )}
          </TabsContent>
        ))}
      </Tabs>

      {workers.length === 0 && !loading && (
        <div className="col-span-full py-12 text-center text-muted-foreground border rounded-lg border-dashed">
          <Server className="mx-auto h-10 w-10 opacity-20 mb-2" />
          <p>{t('workers.noWorkers')}</p>
          <p className="text-xs">{t('workers.noWorkersHint')}</p>
        </div>
      )}
    </div>
  )
}
