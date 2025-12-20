import {
  Sun,
  Moon,
  Settings as SettingsIcon,
  LogOut,
} from 'lucide-react';
import { cn } from '@/lib/utils';
import { StatusDot } from '@/components/ui/status-bar';
import type { Stats, TimeRange } from '@/types';

interface AppHeaderProps {
  darkMode: boolean;
  onToggleDarkMode: () => void;
  onOpenSettings: () => void;
  onLogout: () => void;
  sseConnected: boolean;
}

export function AppHeader({
  darkMode,
  onToggleDarkMode,
  onOpenSettings,
  onLogout,
  sseConnected,
}: AppHeaderProps) {
  return (
    <header className="bg-terminal-surface border-b border-terminal-border sticky top-0 z-40">
      <div className="container mx-auto px-6 py-4 flex justify-between items-center">
        <div className="flex items-center gap-3">
          <span className="text-terminal-muted text-xs">v1.0</span>
        </div>
        <div className="flex items-center gap-4">
          <div className="flex items-center gap-2 text-xs">
            <span className="text-terminal-muted">
              {sseConnected ? 'live' : 'offline'}
            </span>
            <StatusDot
              isUp={sseConnected}
              enabled={true}
              size="sm"
              pulse={sseConnected}
            />
          </div>
          <button
            onClick={onToggleDarkMode}
            className="text-terminal-muted hover:text-terminal-text transition p-2 rounded hover:bg-terminal-border"
            title={darkMode ? 'Switch to light mode' : 'Switch to dark mode'}
          >
            {darkMode ? (
              <Sun className="w-5 h-5" />
            ) : (
              <Moon className="w-5 h-5" />
            )}
          </button>
          <button
            onClick={onOpenSettings}
            className="text-terminal-muted hover:text-terminal-text transition p-2 rounded hover:bg-terminal-border"
            title="Settings"
          >
            <SettingsIcon className="w-5 h-5" />
          </button>
          <button
            onClick={onLogout}
            className="text-terminal-muted hover:text-terminal-text transition p-2 rounded hover:bg-terminal-border"
            title="Logout"
          >
            <LogOut className="w-5 h-5" />
          </button>
        </div>
      </div>
    </header>
  );
}

interface StatsGridProps {
  stats?: Stats;
  isLoading: boolean;
}

export function StatsGrid({ stats, isLoading }: StatsGridProps) {
  if (isLoading || !stats) {
    return (
      <div className="grid grid-cols-2 md:grid-cols-4 gap-4 mb-6">
        {[1, 2, 3, 4].map((i) => (
          <div
            key={i}
            className="bg-terminal-surface border border-terminal-border rounded-sm p-4"
          >
            <div className="h-3 w-16 rounded skeleton mb-3" />
            <div className="h-7 w-24 rounded skeleton" />
          </div>
        ))}
      </div>
    );
  }

  const uptime = stats.total_uptime || 0;

  return (
    <div className="grid grid-cols-2 md:grid-cols-4 gap-4 mb-6">
      <StatCard label="total" value={stats.total_checks || 0} />
      <StatCard label="active" value={stats.active_checks || 0} className="text-terminal-green" />
      <StatCard label="up" value={stats.up_checks || 0} className="text-terminal-green" />
      <StatCard
        label="uptime"
        value={`${uptime.toFixed(2)}%`}
        className={cn(
          uptime >= 99 && 'text-terminal-green',
          uptime >= 95 && uptime < 99 && 'text-terminal-yellow',
          uptime < 95 && 'text-terminal-red'
        )}
      />
    </div>
  );
}

interface StatCardProps {
  label: string;
  value: string | number;
  className?: string;
}

function StatCard({ label, value, className }: StatCardProps) {
  return (
    <div className="bg-terminal-surface border border-terminal-border rounded-sm p-4 hover:border-terminal-muted transition">
      <div className="text-[10px] text-terminal-muted uppercase tracking-widest mb-1">
        {label}
      </div>
      <div className={cn('text-2xl font-bold', className)}>{value}</div>
    </div>
  );
}

interface TimeRangeSelectorProps {
  value: TimeRange;
  onChange: (range: TimeRange) => void;
}

const TIME_RANGE_OPTIONS: TimeRange[] = ['15m', '30m', '60m', '1d', '30d'];

export function TimeRangeSelector({
  value,
  onChange,
}: TimeRangeSelectorProps) {
  return (
    <div className="flex flex-wrap items-center gap-2">
      <span className="text-terminal-muted uppercase tracking-widest text-xs">
        Time Range
      </span>
      {TIME_RANGE_OPTIONS.map((range) => (
        <button
          key={range}
          type="button"
          onClick={() => onChange(range)}
          className={cn(
            'px-3 py-1 rounded border text-[10px] uppercase tracking-wide',
            value === range
              ? 'border-terminal-green text-terminal-green'
              : 'border-terminal-border text-terminal-muted hover:text-terminal-text'
          )}
        >
          {range}
        </button>
      ))}
    </div>
  );
}
