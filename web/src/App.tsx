import { useState } from 'react'
import { useTranslation } from 'react-i18next'
import WorkersPage from '@/pages/WorkersPage'
import TasksPage from '@/pages/TasksPage'
import { Sidebar } from '@/components/layout/sidebar'
import { Header } from '@/components/layout/header'

export default function App() {
  const [activeTab, setActiveTab] = useState('workers')
  const { t } = useTranslation()

  return (
    <div className="flex h-screen w-full overflow-hidden bg-background font-sans antialiased">
      {/* Sidebar for Desktop */}
      <Sidebar 
        activeTab={activeTab} 
        setActiveTab={setActiveTab} 
        className="fixed inset-y-0 left-0 z-20 hidden w-64 md:flex" 
      />

      {/* Main Content */}
      <div className="flex flex-col md:pl-64 w-full h-screen overflow-hidden">
        <Header activeTab={activeTab} setActiveTab={setActiveTab} />
        
        <main className="flex-1 overflow-hidden p-4 md:p-6 flex flex-col bg-muted/30">
          <div className="w-full h-full flex flex-col">
            {activeTab === 'workers' && <WorkersPage />}
            {activeTab === 'tasks' && <TasksPage />}
          </div>
        </main>
      </div>
    </div>
  )
}
