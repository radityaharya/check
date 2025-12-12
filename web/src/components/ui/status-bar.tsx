import { cn } from '@/lib/utils';
import type { CheckStatus, Check, TimeRange } from '@/types';

interface StatusBarProps {
  history: CheckStatus[];
  check: Check;
  timeRange: TimeRange;
}

export function StatusBar({ history, check, timeRange }: StatusBarProps) {
  const statusBars = getStatusBar(history, check, timeRange);

  if (statusBars.length === 0) {
    return null;
  }

  return (
    <div className="flex gap-[2px] h-5 w-full rounded bg-terminal-border/50 overflow-visible relative">
      {statusBars.map((status, index) => (
        <div
          key={index}
          className={cn(
            'flex-1 h-full transition-all hover:opacity-70 cursor-pointer relative group/bar',
            status.success ? 'bg-terminal-green' : status.empty ? 'bg-terminal-border' : 'bg-terminal-red'
          )}
        >
          <div className="absolute bottom-full mb-1 left-1/2 -translate-x-1/2 bg-terminal-bg border border-terminal-border text-[10px] px-2 py-1 rounded opacity-0 group-hover/bar:opacity-100 pointer-events-none whitespace-nowrap z-10">
            <span>{status.empty ? 'N/A' : status.success ? 'UP' : 'DOWN'}</span>
            {status.region && (
              <>
                <span className="text-terminal-muted mx-1">|</span>
                <span className="text-terminal-cyan">{status.region}</span>
              </>
            )}
            <span className="text-terminal-muted mx-1">|</span>
            <span className="text-terminal-muted">{status.time || ''}</span>
          </div>
        </div>
      ))}
    </div>
  );
}

interface StatusDotProps {
  isUp?: boolean;
  enabled?: boolean;
  size?: 'sm' | 'md';
  glow?: boolean;
  pulse?: boolean;
  status?: 'success' | 'error' | 'warning' | 'muted';
}

export function StatusDot({
  isUp = true,
  enabled = true,
  size = 'md',
  glow = false,
  pulse = false,
  status,
}: StatusDotProps) {
  const sizeClasses = {
    sm: 'w-2 h-2',
    md: 'w-2.5 h-2.5',
  };

  // If explicit status is provided, use it
  if (status) {
    const statusColors = {
      success: 'bg-terminal-green',
      error: 'bg-terminal-red',
      warning: 'bg-terminal-yellow',
      muted: 'bg-terminal-muted',
    };
    return (
      <div
        className={cn(
          'rounded-full flex-shrink-0',
          sizeClasses[size],
          statusColors[status],
          glow && status === 'success' && 'glow-green',
          glow && status === 'error' && 'glow-red',
          pulse && 'pulse-dot'
        )}
      />
    );
  }

  return (
    <div
      className={cn(
        'rounded-full flex-shrink-0',
        sizeClasses[size],
        !enabled && 'bg-terminal-yellow',
        enabled && isUp && 'bg-terminal-green',
        enabled && !isUp && 'bg-terminal-red',
        glow && enabled && isUp && 'glow-green',
        glow && enabled && !isUp && 'glow-red',
        pulse && 'pulse-dot'
      )}
    />
  );
}

function getStatusBar(
  history: CheckStatus[] | undefined,
  check: Check,
  timeRange: TimeRange
): Array<{ success: boolean; time: string; empty: boolean; region?: string }> {
  const data = history || [];
  if (data.length === 0) {
    return [];
  }

  let limit = 40;
  const intervalSeconds = check?.interval_seconds || 60;

  if (timeRange === '15m') {
    limit = Math.min(15, Math.ceil((15 * 60) / intervalSeconds));
  } else if (timeRange === '30m') {
    limit = Math.min(30, Math.ceil((30 * 60) / intervalSeconds));
  } else if (timeRange === '60m') {
    limit = Math.min(60, Math.ceil((60 * 60) / intervalSeconds));
  } else if (timeRange === '1d') {
    limit = Math.min(96, Math.ceil((24 * 60 * 60) / intervalSeconds));
  } else if (timeRange === '30d') {
    limit = 40;
  }

  const dataToShow = data.slice(0, Math.min(limit, data.length));
  const result: Array<{ success: boolean; time: string; empty: boolean; region?: string }> = [];

  for (let i = dataToShow.length - 1; i >= 0; i--) {
    const item = dataToShow[i];
    if (!item) continue;
    result.push({
      success: item.success,
      time: new Date(item.checked_at).toLocaleString(),
      empty: false,
      region: item.region,
    });
  }

  return result;
}
