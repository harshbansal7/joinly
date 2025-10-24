/**
 * Sidebar navigation component.
 */

'use client';

import Link from 'next/link';
import { usePathname } from 'next/navigation';
import {
  Bot,
  ChevronLeft,
  ChevronRight,
  LogOut,
  Video,
  Plus,
} from 'lucide-react';

import { cn } from '@/lib/utils';
import { Button } from '@/components/ui/button';
import { useUIStore } from '@/lib/store';

const navigation = [
  { name: 'Agents', href: '/agents', icon: Bot },
  { name: 'Meetings', href: '/meetings', icon: Video },
];

interface SidebarProps {
  className?: string;
}

export function Sidebar({ className }: SidebarProps) {
  const pathname = usePathname();
  const { sidebarOpen, setSidebarOpen } = useUIStore();

  return (
    <div
      className={cn(
        'flex flex-col bg-white/80 dark:bg-black backdrop-blur-lg border-r border-slate-200/50 dark:border-zinc-800 transition-all duration-300 shadow-xl',
        sidebarOpen ? 'w-64' : 'w-16',
        className
      )}
    >
      {/* Header */}
      <div className="flex items-center justify-between p-4 border-b border-slate-200/50 dark:border-zinc-800">
        <div className={cn('flex items-center', !sidebarOpen && 'justify-center')}>
          {/* <div className="p-2 bg-gradient-to-r from-blue-500 to-purple-600 rounded-lg">
            <Bot className="h-6 w-6 text-white" />
          </div> */}
          {sidebarOpen && (
            <span className="ml-3 text-lg font-bold bg-gradient-to-r from-blue-600 to-purple-600 bg-clip-text text-transparent">
              DealSense
            </span>
          )}
        </div>
        <Button
          variant="ghost"
          size="sm"
          onClick={() => setSidebarOpen(!sidebarOpen)}
          className="h-8 w-8 p-0"
        >
          {sidebarOpen ? (
            <ChevronLeft className="h-4 w-4" />
          ) : (
            <ChevronRight className="h-4 w-4" />
          )}
        </Button>
      </div>

      {/* Navigation */}
      <nav className="flex-1 px-2 py-4 space-y-1">
        {navigation.map((item) => {
          const isActive = pathname === item.href;
          return (
            <Link
              key={item.name}
              href={item.href}
              className={cn(
                'flex items-center px-3 py-2 text-sm font-medium rounded-lg transition-all duration-200',
                isActive
                  ? 'bg-blue-50 text-blue-700 border border-blue-200 dark:bg-blue-900/30 dark:text-blue-300 dark:border-blue-800'
                  : 'text-slate-700 hover:bg-slate-100 dark:text-slate-300 dark:hover:bg-slate-800'
              )}
            >
              <item.icon
                className={cn(
                  'flex-shrink-0 h-5 w-5',
                  sidebarOpen ? 'mr-3' : 'mx-auto'
                )}
              />
              {sidebarOpen && <span>{item.name}</span>}
            </Link>
          );
        })}

        {/* Create Agent Button */}
        <div className="pt-4 border-t border-slate-200/50 dark:border-zinc-800">
          <Link href="/agents/create">
            <Button
              className={cn(
                'w-full bg-gradient-to-r from-blue-600 via-purple-600 to-indigo-600 hover:from-blue-700 hover:via-purple-700 hover:to-indigo-700 text-white shadow-lg hover:shadow-xl transition-all duration-300',
                sidebarOpen ? 'justify-start gap-2 px-3' : 'justify-center p-2'
              )}
              size="sm"
            >
              <Plus className="h-4 w-4" />
              {sidebarOpen && <span className="text-sm font-semibold">Create Agent</span>}
            </Button>
          </Link>
        </div>
      </nav>

      {/* Footer */}
      <div className="p-4 border-t border-slate-200/50 dark:border-zinc-800">
        <Button
          variant="ghost"
          size="sm"
          className={cn(
            'w-full justify-start text-slate-700 hover:bg-slate-100 dark:text-slate-300 dark:hover:bg-slate-800 transition-colors duration-200',
            !sidebarOpen && 'justify-center'
          )}
        >
          <LogOut className={cn('h-4 w-4', sidebarOpen && 'mr-2')} />
          {sidebarOpen && 'Sign Out'}
        </Button>
      </div>
    </div>
  );
}
