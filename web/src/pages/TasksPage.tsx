import { useEffect, useMemo, useState } from 'react'
import { useTranslation } from 'react-i18next'
import { Button } from '@/components/ui/button'
import { Card } from '@/components/ui/card'
import { Sheet, SheetContent, SheetHeader, SheetTitle } from '@/components/ui/sheet'
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from '@/components/ui/select'
import { TableHeader, TableBody, TableHead, TableRow, TableCell } from '@/components/ui/table'
import { Badge } from '@/components/ui/badge'
import { Skeleton } from '@/components/ui/skeleton'
import { RefreshCw, Search, Play, Clock, AlertCircle, CheckCircle2, XCircle, Activity, ChevronRight, Hash, Calendar, Layers, ChevronLeft, Copy, Check } from 'lucide-react'
import { cn } from '@/lib/utils'

type Task = {
  task_id: string
  worker_name: string
  queue: string
  priority: number
  payload: any
  status: string
  last_attempt: number
  last_error?: string
  last_worker_name?: string
  trace_id?: string
  created_at: string
  updated_at: string
}

type Attempt = {
  task_id: string
  asynq_task_id?: string
  attempt: number
  status: string
  started_at: string
  finished_at?: string
  duration_ms?: number
  error?: string
  worker_name?: string
  trace_id?: string
  span_id?: string
}

const STATUS_VARIANTS: Record<string, "default" | "secondary" | "destructive" | "outline"> = {
  pending: "secondary",
  running: "default",
  success: "outline",
  fail: "destructive",
  dead: "destructive",
}

const STATUS_ICONS: Record<string, any> = {
  pending: Clock,
  running: Play,
  success: CheckCircle2,
  fail: AlertCircle,
  dead: XCircle,
}

export default function TasksPage() {
  const { t } = useTranslation()

  // Helper to format simplified relative time
  const timeAgo = (dateStr: string) => {
    const date = new Date(dateStr)
    const now = new Date()
    const diffMs = now.getTime() - date.getTime()
    const diffSec = Math.floor(diffMs / 1000)
    const diffMin = Math.floor(diffSec / 60)
    const diffHour = Math.floor(diffMin / 60)
    const diffDay = Math.floor(diffHour / 24)

    if (diffSec < 60) return t('tasks.time.justNow')
    if (diffMin < 60) return t('tasks.time.minutesAgo').replace('{0}', diffMin.toString())
    if (diffHour < 24) return t('tasks.time.hoursAgo').replace('{0}', diffHour.toString())
    return t('tasks.time.daysAgo').replace('{0}', diffDay.toString())
  }

  const [items, setItems] = useState<Task[]>([])
  const [total, setTotal] = useState(0)
  const [loading, setLoading] = useState(false)
  const [error, setError] = useState<string | null>(null)

  const [page, setPage] = useState(1)
  const [pageSize, setPageSize] = useState(20)

  const [filters, setFilters] = useState({ worker_name: '', status: '', queue: '' })
  const [workers, setWorkers] = useState<string[]>([])

  const ALL = '__all__'
  const statusOptions = useMemo(() => ['', 'pending', 'running', 'success', 'fail', 'dead'] as const, [])
  const queueOptions = useMemo(() => ['', 'critical', 'default', 'crawler'] as const, [])

  const [detailOpen, setDetailOpen] = useState(false)
  const [detailLoading, setDetailLoading] = useState(false)
  const [detailError, setDetailError] = useState<string | null>(null)
  const [selected, setSelected] = useState<{ task?: Task; attempts?: Attempt[] } | null>(null)
  const [copied, setCopied] = useState<string | null>(null)

  const queryString = useMemo(() => {
    const p = new URLSearchParams()
    if (filters.worker_name.trim()) p.set('worker_name', filters.worker_name.trim())
    if (filters.status.trim()) p.set('status', filters.status.trim())
    if (filters.queue.trim()) p.set('queue', filters.queue.trim())
    
    // Pagination params
    p.set('limit', pageSize.toString())
    p.set('offset', ((page - 1) * pageSize).toString())
    
    return p.toString()
  }, [filters, page, pageSize])

  async function refresh() {
    setLoading(true)
    setError(null)
    try {
      const res = await fetch(`/api/v1/tasks${queryString ? `?${queryString}` : ''}`)
      if (!res.ok) {
        const data = (await res.json().catch(() => null)) as { error?: string } | null
        throw new Error(data?.error || `${t('workers.requestFailed')}: ${res.status}`)
      }
      const data = (await res.json()) as { items: Task[], total: number }
      setItems(data.items ?? [])
      setTotal(data.total || 0)
    } catch (e) {
      setError(e instanceof Error ? e.message : t('workers.unknownError'))
    } finally {
      setLoading(false)
    }
  }

  async function openDetail(task: Task) {
    setSelected({ task })
    setDetailOpen(true)
    setDetailLoading(true)
    setDetailError(null)
    try {
      const res = await fetch(`/api/v1/tasks/${encodeURIComponent(task.task_id)}`)
      if (!res.ok) {
        const data = (await res.json().catch(() => null)) as { error?: string } | null
        throw new Error(data?.error || `${t('workers.requestFailed')}: ${res.status}`)
      }
      const data = (await res.json()) as { item: Task; attempts: Attempt[] }
      setSelected({ task: data.item, attempts: data.attempts })
    } catch (e) {
      setDetailError(e instanceof Error ? e.message : t('workers.unknownError'))
    } finally {
      setDetailLoading(false)
    }
  }

  async function replay(taskId: string) {
    setDetailLoading(true)
    setDetailError(null)
    try {
      const res = await fetch(`/api/v1/tasks/${encodeURIComponent(taskId)}/replay`, { method: 'POST' })
      if (!res.ok) {
        const data = (await res.json().catch(() => null)) as { error?: string } | null
        throw new Error(data?.error || `${t('workers.requestFailed')}: ${res.status}`)
      }
      await refresh()
      if (selected?.task) await openDetail(selected.task)
    } catch (e) {
      setDetailError(e instanceof Error ? e.message : t('workers.unknownError'))
    } finally {
      setDetailLoading(false)
    }
  }

  async function loadWorkers() {
    try {
      const res = await fetch('/api/v1/workers')
      if (!res.ok) return
      const data = (await res.json()) as { items: { worker_name: string }[] }
      setWorkers((data.items ?? []).map((x) => x.worker_name))
    } catch {
      // Ignore
    }
  }

  const copyToClipboard = (text: string) => {
    navigator.clipboard.writeText(text)
    setCopied(text)
    setTimeout(() => setCopied(null), 2000)
  }

  useEffect(() => {
    refresh()
    loadWorkers()
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [page, pageSize, filters])

  useEffect(() => {
     setPage(1)
  }, [filters])

  const totalPages = Math.max(1, Math.ceil(total / pageSize))

  return (
    <div className="space-y-4 h-full flex flex-col min-h-0">
      {/* Filters & Actions */}
      <div className="flex flex-col gap-4 p-1 md:flex-row md:items-center md:justify-between flex-shrink-0">
        <div className="flex flex-1 flex-wrap items-center gap-3">
           <Select
              value={filters.worker_name === '' ? ALL : filters.worker_name}
              onValueChange={(v) => setFilters((p) => ({ ...p, worker_name: v === ALL ? '' : v }))}
            >
              <SelectTrigger className="w-[180px] h-9 bg-background">
                <SelectValue placeholder={t('tasks.filters.workerName')} />
              </SelectTrigger>
              <SelectContent>
                <SelectItem value={ALL}>{t('tasks.filters.allWorkers')}</SelectItem>
                {workers.map((w) => (
                  <SelectItem key={w} value={w}>{w}</SelectItem>
                ))}
              </SelectContent>
            </Select>

            <Select
              value={filters.status === '' ? ALL : filters.status}
              onValueChange={(v) => setFilters((p) => ({ ...p, status: v === ALL ? '' : v }))}
            >
              <SelectTrigger className="w-[140px] h-9 bg-background">
                <SelectValue placeholder={t('tasks.filters.status')} />
              </SelectTrigger>
              <SelectContent>
                <SelectItem value={ALL}>{t('tasks.filters.allStatus')}</SelectItem>
                {statusOptions.filter((x) => x !== '').map((s) => (
                  <SelectItem key={s} value={s}>{s}</SelectItem>
                ))}
              </SelectContent>
            </Select>

            <Select
              value={filters.queue === '' ? ALL : filters.queue}
              onValueChange={(v) => setFilters((p) => ({ ...p, queue: v === ALL ? '' : v }))}
            >
              <SelectTrigger className="w-[140px] h-9 bg-background">
                <SelectValue placeholder={t('tasks.filters.queue')} />
              </SelectTrigger>
              <SelectContent>
                <SelectItem value={ALL}>{t('tasks.filters.allQueues')}</SelectItem>
                {queueOptions.filter((x) => x !== '').map((q) => (
                  <SelectItem key={q} value={q}>{q}</SelectItem>
                ))}
              </SelectContent>
            </Select>

            <div className="flex items-center gap-2 ml-auto md:ml-0">
               <Button 
                variant="ghost" 
                size="sm" 
                className="h-9 text-muted-foreground hover:text-foreground"
              onClick={() => {
                setFilters({ worker_name: '', status: '', queue: '' })
              }}
              >
                {t('tasks.filters.reset')}
              </Button>
            </div>
        </div>
        
        <div className="flex items-center gap-2">
           <Button variant="outline" size="sm" onClick={refresh} disabled={loading} className="h-9 bg-background">
            <RefreshCw className={cn("mr-2 h-4 w-4", loading && "animate-spin")} />
            {t('tasks.actions.refresh')}
          </Button>
        </div>
      </div>

      {error && (
        <div className="rounded-md bg-destructive/15 p-3 text-sm text-destructive flex-shrink-0">
          Error: {error}
        </div>
      )}

      {/* Data Table */}
      <Card className="shadow-sm border rounded-lg bg-card flex-1 overflow-hidden flex flex-col min-h-0">
        <div className="flex-1 overflow-auto relative">
          <table className="w-full caption-bottom text-sm">
            <TableHeader className="sticky top-0 z-20 shadow-sm bg-muted/95 backdrop-blur-sm">
              <TableRow className="hover:bg-transparent border-b border-border/60">
                <TableHead className="w-[20%] pl-6 h-10 text-xs font-semibold uppercase tracking-wider">{t('tasks.table.taskId')}</TableHead>
                <TableHead className="w-[15%] h-10 text-xs font-semibold uppercase tracking-wider">{t('tasks.table.worker')}</TableHead>
                <TableHead className="w-[12%] h-10 text-xs font-semibold uppercase tracking-wider">{t('tasks.table.queue')}</TableHead>
                <TableHead className="w-[10%] h-10 text-xs font-semibold uppercase tracking-wider">{t('tasks.table.status')}</TableHead>
                <TableHead className="w-[15%] h-10 text-xs font-semibold uppercase tracking-wider">{t('tasks.table.execution')}</TableHead>
                <TableHead className="w-[13%] h-10 text-xs font-semibold uppercase tracking-wider">{t('tasks.table.lastWorker')}</TableHead>
                <TableHead className="w-[15%] text-right pr-6 h-10 text-xs font-semibold uppercase tracking-wider">{t('tasks.table.updated')}</TableHead>
              </TableRow>
            </TableHeader>
            <TableBody>
              {loading && items.length === 0 ? (
                 Array.from({ length: 10 }).map((_, i) => (
                   <TableRow key={i} className="border-b border-border/40">
                     <TableCell className="py-3 pl-6"><Skeleton className="h-5 w-3/4" /></TableCell>
                     <TableCell><Skeleton className="h-5 w-24" /></TableCell>
                     <TableCell><Skeleton className="h-5 w-20" /></TableCell>
                     <TableCell><Skeleton className="h-5 w-20" /></TableCell>
                     <TableCell><Skeleton className="h-5 w-full" /></TableCell>
                     <TableCell><Skeleton className="h-5 w-24" /></TableCell>
                     <TableCell className="pr-6"><div className="flex justify-end"><Skeleton className="h-5 w-16" /></div></TableCell>
                   </TableRow>
                 ))
              ) : (
                items.map((t) => {
                  const StatusIcon = STATUS_ICONS[t.status] || AlertCircle
                  const isSuccess = t.status === 'success'
                  const isFail = t.status === 'fail' || t.status === 'dead'
                  
                  return (
                    <TableRow 
                      key={t.task_id} 
                      className="cursor-pointer group hover:bg-muted/40 transition-colors border-b last:border-0 border-border/40"
                      onClick={() => openDetail(t)}
                    >
                      <TableCell className="py-3 pl-6 align-top">
                        <div className="flex items-center gap-2">
                           <div className="font-mono text-xs font-medium text-foreground/90 group-hover:text-primary transition-colors truncate max-w-[200px]" title={t.task_id}>
                             {t.task_id}
                           </div>
                           <Button
                             variant="ghost" 
                             size="icon" 
                             className="h-4 w-4 text-muted-foreground opacity-0 group-hover:opacity-100 transition-opacity"
                             onClick={(e) => {
                               e.stopPropagation()
                               copyToClipboard(t.task_id)
                             }}
                           >
                             {copied === t.task_id ? <Check className="h-3 w-3 text-green-500" /> : <Copy className="h-3 w-3" />}
                           </Button>
                        </div>
                      </TableCell>

                      <TableCell className="py-3 align-top">
                        <div className="text-xs font-medium text-foreground/80 uppercase">
                          {t.worker_name}
                        </div>
                      </TableCell>

                      <TableCell className="py-3 align-top">
                        <div className="text-xs text-muted-foreground">
                          {t.queue}
                        </div>
                      </TableCell>
                      
                      <TableCell className="py-3 align-top">
                        <Badge 
                          variant={STATUS_VARIANTS[t.status] || 'outline'}
                          className={cn(
                            "capitalize gap-1.5 py-0.5 px-2 shadow-none border font-normal text-[11px]", 
                            isSuccess && "border-green-200 text-green-700 bg-green-50/50 dark:bg-green-900/10 dark:text-green-400 dark:border-green-900/30",
                            isFail && "border-red-200 text-red-700 bg-red-50/50 dark:bg-red-900/10 dark:text-red-400 dark:border-red-900/30",
                            t.status === 'pending' && "border-gray-200 bg-gray-50/50 text-gray-600 dark:bg-gray-800/50 dark:text-gray-400 dark:border-gray-700/30",
                            t.status === 'running' && "border-blue-200 bg-blue-50/50 text-blue-700 dark:bg-blue-900/10 dark:text-blue-400 dark:border-blue-900/30"
                          )}
                        >
                          <StatusIcon className="h-3 w-3" />
                          {t.status}
                        </Badge>
                      </TableCell>

                      <TableCell className="py-3 align-top">
                        <div className="flex flex-col gap-1 text-xs text-muted-foreground">
                           <div className="flex items-center gap-3">
                             <div className="flex items-center gap-1" title="Priority">
                               <Hash className="h-3 w-3 opacity-60" />
                               <span>{t.priority}</span>
                             </div>
                             <div className="flex items-center gap-1" title="Attempts">
                               <Activity className="h-3 w-3 opacity-60" />
                               <span>{t.last_attempt}</span>
                             </div>
                           </div>
                        </div>
                      </TableCell>

                      <TableCell className="py-3 align-top">
                        <div className="text-xs font-mono text-muted-foreground truncate max-w-[120px]" title={t.last_worker_name || 'N/A'}>
                          {t.last_worker_name || '-'}
                        </div>
                      </TableCell>

                      <TableCell className="py-3 pr-6 align-top text-right">
                        <div className="flex flex-col items-end gap-1">
                          <span className="text-xs font-medium text-foreground/80 whitespace-nowrap">{timeAgo(t.updated_at)}</span>
                          <span className="text-[10px] text-muted-foreground/60">
                            {new Date(t.updated_at).toLocaleTimeString([], { hour: '2-digit', minute: '2-digit' })}
                          </span>
                        </div>
                      </TableCell>
                    </TableRow>
                  )
                })
              )}
              {!loading && items.length === 0 && (
                <TableRow>
                  <TableCell colSpan={7} className="h-48 text-center">
                    <div className="flex flex-col items-center justify-center text-muted-foreground space-y-2">
                       <div className="h-12 w-12 rounded-full bg-muted/50 flex items-center justify-center">
                         <Search className="h-6 w-6 opacity-40" />
                       </div>
                       <p className="font-medium text-sm">{t('tasks.empty.title')}</p>
                       <p className="text-xs text-muted-foreground/70 max-w-xs mx-auto">{t('tasks.empty.hint')}</p>
                    </div>
                  </TableCell>
                </TableRow>
              )}
            </TableBody>
          </table>
        </div>
      </Card>
      
      {/* Pagination Controls */}
      <div className="flex items-center justify-between border-t pt-4 flex-shrink-0">
         <div className="flex items-center gap-4">
            <div className="text-xs text-muted-foreground">
              {t('tasks.pagination.showing')} <span className="font-medium text-foreground">{items.length}</span> {t('tasks.pagination.of')} <span className="font-medium text-foreground">{total}</span> {t('tasks.pagination.tasks')}
            </div>
            <div className="flex items-center gap-2">
               <span className="text-xs text-muted-foreground">{t('tasks.pagination.rowsPerPage')}</span>
               <Select value={pageSize.toString()} onValueChange={(v) => setPageSize(Number(v))}>
                 <SelectTrigger className="w-[65px] h-7 text-[11px]">
                   <SelectValue />
                 </SelectTrigger>
                 <SelectContent>
                   <SelectItem value="20">20</SelectItem>
                   <SelectItem value="50">50</SelectItem>
                   <SelectItem value="100">100</SelectItem>
                 </SelectContent>
               </Select>
            </div>
         </div>
         
         <div className="flex items-center gap-2">
            <Button
              variant="outline"
              size="sm"
              onClick={() => setPage(p => Math.max(1, p - 1))}
              disabled={page === 1 || loading}
              className="h-7 px-2 text-[11px]"
            >
              <ChevronLeft className="h-3.5 w-3.5 mr-1" />
              {t('tasks.pagination.prev')}
            </Button>
            <div className="text-[11px] font-medium mx-1 min-w-[3rem] text-center">
               {t('tasks.pagination.page').replace('{0}', page.toString()).replace('{1}', totalPages.toString())}
            </div>
            <Button
              variant="outline"
              size="sm"
              onClick={() => setPage(p => Math.min(totalPages, p + 1))}
              disabled={page === totalPages || loading}
              className="h-7 px-2 text-[11px]"
            >
              {t('tasks.pagination.next')}
              <ChevronRight className="h-3.5 w-3.5 ml-1" />
            </Button>
         </div>
      </div>

      {/* Detail Sheet */}
      <Sheet open={detailOpen} onOpenChange={setDetailOpen}>
        <SheetContent className="sm:max-w-2xl w-full overflow-y-auto pr-14">
          <SheetHeader className="pb-6 border-b">
            <SheetTitle className="flex flex-col gap-3">
              <div className="flex items-center justify-between">
                <span className="text-lg font-bold">{t('tasks.detail.title')}</span>
                <Button size="sm" variant="outline" onClick={() => replay(selected?.task?.task_id || '')} disabled={detailLoading} className="h-8 text-xs bg-primary/5 hover:bg-primary/10 border-primary/20 text-primary">
                  <RefreshCw className={cn("mr-2 h-3.5 w-3.5", detailLoading && "animate-spin")} />
                  {t('tasks.detail.replayTask')}
                </Button>
              </div>
              {selected?.task && (
                 <div className="flex items-center gap-2">
                   <Badge variant="secondary" className="font-mono font-medium text-[11px] px-2 py-0.5 bg-muted/60 text-muted-foreground">
                     {selected.task.task_id}
                   </Badge>
                   <Button variant="ghost" size="icon" className="h-6 w-6 text-muted-foreground hover:text-foreground" onClick={() => copyToClipboard(selected.task!.task_id)}>
                      {copied === selected.task.task_id ? <Check className="h-3 w-3 text-green-500" /> : <Copy className="h-3 w-3" />}
                   </Button>
                 </div>
              )}
            </SheetTitle>
          </SheetHeader>

          {detailError && <div className="mt-4 text-sm text-destructive bg-destructive/10 p-3 rounded-md border border-destructive/20">{detailError}</div>}
          
          <div className="mt-6 space-y-8">
            {selected?.task ? (
              <>
                {/* Information Grid */}
                <div className="grid grid-cols-2 md:grid-cols-3 gap-4">
                  <div className="p-3 rounded-lg bg-muted/30 border space-y-1">
                    <span className="text-[10px] text-muted-foreground uppercase tracking-widest font-bold">{t('tasks.detail.workerName')}</span>
                    <div className="font-semibold text-sm truncate">{selected.task.worker_name}</div>
                  </div>
                  <div className="p-3 rounded-lg bg-muted/30 border space-y-1">
                    <span className="text-[10px] text-muted-foreground uppercase tracking-widest font-bold">{t('tasks.detail.status')}</span>
                    <div className={cn("font-bold text-sm capitalize flex items-center gap-1.5", 
                        selected.task.status === 'success' && "text-green-600",
                        selected.task.status === 'fail' || selected.task.status === 'dead' ? "text-red-600" : "",
                        selected.task.status === 'running' && "text-blue-600",
                        selected.task.status === 'pending' && "text-gray-500"
                    )}>
                       {(() => {
                         const Icon = STATUS_ICONS[selected.task.status] || AlertCircle
                         return <Icon className="h-3.5 w-3.5" />
                       })()}
                       {selected.task.status}
                    </div>
                  </div>
                  <div className="p-3 rounded-lg bg-muted/30 border space-y-1">
                    <span className="text-[10px] text-muted-foreground uppercase tracking-widest font-bold">{t('tasks.detail.queue')}</span>
                    <div className="font-semibold text-sm truncate">{selected.task.queue}</div>
                  </div>
                  <div className="p-3 rounded-lg bg-muted/30 border space-y-1">
                    <span className="text-[10px] text-muted-foreground uppercase tracking-widest font-bold">{t('tasks.detail.attempts')}</span>
                    <div className="font-mono text-sm">{selected.task.last_attempt}</div>
                  </div>
                  <div className="p-3 rounded-lg bg-muted/30 border space-y-1">
                    <span className="text-[10px] text-muted-foreground uppercase tracking-widest font-bold">{t('tasks.detail.priority')}</span>
                    <div className="font-mono text-sm">{selected.task.priority}</div>
                  </div>
                  <div className="p-3 rounded-lg bg-muted/30 border space-y-1">
                    <span className="text-[10px] text-muted-foreground uppercase tracking-widest font-bold">{t('tasks.detail.lastWorkerInstance')}</span>
                    <div className="font-mono text-xs truncate">{selected.task.last_worker_name || '-'}</div>
                  </div>
                </div>

                {/* Error Banner */}
                {(selected.task.last_error || selected.task.status === 'fail' || selected.task.status === 'dead') && (
                  <div className="rounded-lg border border-red-200 bg-red-50/50 dark:bg-red-900/10 p-4 space-y-2">
                    <div className="flex items-center gap-2 text-red-700 dark:text-red-400 font-bold text-xs uppercase tracking-wider">
                      <AlertCircle className="h-4 w-4" />
                      {t('tasks.detail.lastError')}
                    </div>
                    <div className="text-xs font-mono text-red-600 dark:text-red-300 break-all leading-relaxed">
                      {selected.task.last_error || t('tasks.detail.noError')}
                    </div>
                  </div>
                )}

                {/* Payload JSON */}
                <div className="space-y-3">
                   <div className="flex items-center justify-between">
                     <h3 className="text-xs font-bold uppercase tracking-widest text-muted-foreground flex items-center gap-2">
                       <Layers className="h-4 w-4" />
                       {t('tasks.detail.taskPayload')}
                     </h3>
                     <Button 
                      variant="ghost" 
                      size="sm" 
                      className="h-7 text-[10px] font-bold uppercase"
                      onClick={() => copyToClipboard(JSON.stringify(selected.task!.payload, null, 2))}
                     >
                        {copied === JSON.stringify(selected.task!.payload, null, 2) ? t('tasks.detail.copied') : t('tasks.detail.copyJson')}
                     </Button>
                   </div>
                   <div className="rounded-lg border bg-muted/20 p-4 font-mono text-xs overflow-x-auto max-h-[300px]">
                     <pre className="leading-relaxed">
                       {JSON.stringify(selected.task.payload, null, 2)}
                     </pre>
                   </div>
                </div>

                {/* Execution Timeline */}
                <div className="space-y-4">
                  <div className="flex items-center justify-between border-b pb-2">
                    <h3 className="text-xs font-bold uppercase tracking-widest text-muted-foreground flex items-center gap-2">
                      <Activity className="h-4 w-4" />
                      {t('tasks.detail.executionHistory')}
                    </h3>
                  </div>

                  <div className="space-y-0 relative pl-4 border-l-2 border-muted ml-2">
                    {(selected.attempts ?? []).map((a) => (
                      <div key={`${a.task_id}-${a.attempt}`} className="mb-6 relative">
                         <div className={cn("absolute -left-[21px] top-0 h-4 w-4 rounded-full border-2 border-background shadow-sm", 
                            a.status === 'success' ? "bg-green-500" : (a.status === 'fail' || a.status === 'dead') ? "bg-red-500" : a.status === 'running' ? "bg-blue-500" : "bg-gray-400"
                         )} />
                         
                         <div className="flex flex-col gap-2.5 bg-muted/10 rounded-lg p-3 border border-transparent hover:border-border transition-colors">
                            <div className="flex items-center justify-between flex-wrap gap-2">
                               <div className="flex items-center gap-2">
                                  <span className="font-bold text-xs">{t('tasks.detail.attempt').replace('{0}', a.attempt.toString())}</span>
                                  <Badge 
                                    variant={STATUS_VARIANTS[a.status] || 'outline'}
                                    className="capitalize px-1.5 h-4.5 text-[9px] font-semibold"
                                  >
                                    {a.status}
                                  </Badge>
                               </div>
                               <div className="text-[10px] text-muted-foreground font-medium flex items-center gap-1.5">
                                 <Calendar className="h-3 w-3" />
                                 {new Date(a.started_at).toLocaleString()}
                               </div>
                            </div>
                            
                            <div className="grid grid-cols-2 gap-x-4 gap-y-1.5 text-[11px]">
                               <div className="flex items-center gap-2 text-muted-foreground">
                                 <Play className="h-3 w-3 opacity-60" />
                                 {t('tasks.detail.worker')}: <span className="text-foreground font-mono">{a.worker_name || 'N/A'}</span>
                               </div>
                               {a.duration_ms && (
                                 <div className="flex items-center gap-2 text-muted-foreground">
                                   <Clock className="h-3 w-3 opacity-60" />
                                   {t('tasks.detail.duration')}: <span className="text-foreground font-medium">{a.duration_ms}ms</span>
                                 </div>
                               )}
                               {a.trace_id && (
                                 <div className="flex items-center gap-2 text-muted-foreground col-span-2">
                                   <Search className="h-3 w-3 opacity-60" />
                                   {t('tasks.detail.traceId')}: <span className="text-foreground font-mono truncate">{a.trace_id}</span>
                                 </div>
                               )}
                            </div>

                            {a.error && (
                              <div className="rounded border bg-red-50/30 dark:bg-red-950/20 p-2 text-[10px] text-red-600 dark:text-red-400 font-mono break-all leading-normal">
                                {a.error}
                              </div>
                            )}
                         </div>
                      </div>
                    ))}
                    {(selected.attempts ?? []).length === 0 && (
                      <div className="text-xs text-muted-foreground italic py-4 bg-muted/10 rounded-lg text-center border-dashed border-2">
                        {t('tasks.detail.noHistory')}
                      </div>
                    )}
                  </div>
                </div>

                {/* Metadata */}
                <div className="pt-6 border-t grid grid-cols-2 gap-4 text-[10px] text-muted-foreground uppercase tracking-widest font-bold">
                  <div>
                    {t('tasks.detail.created')}: <span className="text-foreground ml-1">{new Date(selected.task.created_at).toLocaleString()}</span>
                  </div>
                  <div className="text-right">
                    {t('tasks.detail.updated')}: <span className="text-foreground ml-1">{new Date(selected.task.updated_at).toLocaleString()}</span>
                  </div>
                </div>
              </>
            ) : (
              <div className="space-y-6">
                <div className="grid grid-cols-2 gap-4">
                  <Skeleton className="h-16 w-full" />
                  <Skeleton className="h-16 w-full" />
                </div>
                <Skeleton className="h-48 w-full" />
                <Skeleton className="h-64 w-full" />
              </div>
            )}
          </div>
        </SheetContent>
      </Sheet>
    </div>
  )
}
