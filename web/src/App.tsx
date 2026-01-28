import { useState } from 'react'
import { useTranslation } from 'react-i18next'
import WorkersPage from '@/pages/WorkersPage'
import TasksPage from '@/pages/TasksPage'
import { Tabs, TabsList, TabsTrigger } from '@/components/ui/tabs'
import { Server, ListTodo, Activity, Languages } from 'lucide-react'
import { cn } from '@/lib/utils'
import { Button } from '@/components/ui/button'
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuItem,
  DropdownMenuTrigger,
} from '@/components/ui/dropdown-menu'

export default function App() {
  const [activeTab, setActiveTab] = useState('workers')
  const { t, i18n } = useTranslation()

  const toggleLanguage = (lang: string) => {
    i18n.changeLanguage(lang)
  }

  return (
    <div className="flex h-screen w-full overflow-hidden bg-muted/40 font-sans">
      {/* Sidebar */}
      <aside className="fixed inset-y-0 left-0 z-10 hidden w-64 flex-col border-r bg-background sm:flex">
        <div className="flex h-14 items-center border-b px-6">
          <Activity className="mr-2 h-6 w-6 text-primary" />
          <span className="font-bold text-lg tracking-tight">{t('app.title')}</span>
        </div>
        <div className="flex-1 py-6 px-4">
          <Tabs value={activeTab} onValueChange={setActiveTab} orientation="vertical" className="h-full flex-col space-y-8">
            <div className="space-y-2">
              <h2 className="mb-2 px-2 text-xs font-semibold tracking-wide text-muted-foreground uppercase">
                {t('nav.platform')}
              </h2>
              <TabsList className="flex flex-col h-auto bg-transparent p-0 space-y-1">
                <TabsTrigger 
                  value="workers" 
                  className={cn(
                    "w-full justify-start px-3 py-2 h-9 font-medium",
                    "data-[state=active]:bg-primary/10 data-[state=active]:text-primary data-[state=active]:shadow-none",
                    "hover:bg-muted text-muted-foreground hover:text-foreground transition-colors"
                  )}
                >
                  <Server className="mr-2 h-4 w-4" />
                  {t('nav.workers')}
                </TabsTrigger>
                <TabsTrigger 
                  value="tasks" 
                  className={cn(
                    "w-full justify-start px-3 py-2 h-9 font-medium",
                    "data-[state=active]:bg-primary/10 data-[state=active]:text-primary data-[state=active]:shadow-none",
                    "hover:bg-muted text-muted-foreground hover:text-foreground transition-colors"
                  )}
                >
                  <ListTodo className="mr-2 h-4 w-4" />
                  {t('nav.tasks')}
                </TabsTrigger>
              </TabsList>
            </div>
          </Tabs>
        </div>
        <div className="border-t p-4 text-xs text-muted-foreground">
          <p>{t('app.subtitle')}</p>
          <p className="mt-1">{t('app.version')}</p>
        </div>
      </aside>

      {/* Main Content */}
      <div className="flex flex-col sm:pl-64 w-full h-screen overflow-hidden">
        <header className="sticky top-0 z-10 flex h-14 items-center gap-4 border-b bg-background/95 px-6 backdrop-blur supports-[backdrop-filter]:bg-background/60 flex-none">
          <div className="flex flex-1 items-center justify-between">
            <h1 className="text-lg font-semibold">
              {activeTab === 'workers' ? t('workers.title') : t('tasks.title')}
            </h1>
            <DropdownMenu>
              <DropdownMenuTrigger asChild>
                <Button variant="outline" size="sm">
                  <Languages className="mr-2 h-4 w-4" />
                  {i18n.language === 'zh' ? '中文' : 'English'}
                </Button>
              </DropdownMenuTrigger>
              <DropdownMenuContent align="end">
                <DropdownMenuItem onClick={() => toggleLanguage('zh')}>
                  中文
                </DropdownMenuItem>
                <DropdownMenuItem onClick={() => toggleLanguage('en')}>
                  English
                </DropdownMenuItem>
              </DropdownMenuContent>
            </DropdownMenu>
          </div>
        </header>
        
        <main className="flex-1 overflow-hidden p-6 flex flex-col">
          <div className="w-full h-full flex flex-col">
            {activeTab === 'workers' && <WorkersPage />}
            {activeTab === 'tasks' && <TasksPage />}
          </div>
        </main>
      </div>
    </div>
  )
}
