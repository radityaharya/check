import { X } from 'lucide-react';
import { cn } from '@/lib/utils';
import { StatusDot, StatusBar } from '@/components/ui/status-bar';
import { ResponseTimeChart } from './ResponseTimeChart';
import { formatDate, getCheckTarget, getCheckTypeClass } from '@/lib/helpers';
import { useToast } from '@/components/ui/toast';
import {
  useTriggerCheck,
  useToggleCheckEnabled,
  useDeleteCheck,
  useCheckHistory,
} from '@/hooks';
import { useTimeAgoTick, formatTimeAgo } from '@/hooks/use-time-ago';
import type { Check, CheckStatus, TimeRange } from '@/types';
import { useTriggerSnapshot } from '@/hooks/use-checks';

interface DetailsPaneProps {
  check: Check | null;
  timeRange: TimeRange;
  onClose: () => void;
  onEditCheck: (check: Check) => void;
  onOpenHistory: (check: Check) => void;
}

export function DetailsPane({
  check,
  timeRange,
  onClose,
  onEditCheck,
  onOpenHistory,
}: DetailsPaneProps) {
  const { showToast } = useToast();
  const { data: history = [], isLoading: isLoadingHistory } = useCheckHistory(check?.id ?? null, timeRange);
  const triggerMutation = useTriggerCheck();
  const toggleEnabledMutation = useToggleCheckEnabled();
  const deleteMutation = useDeleteCheck();
  const snapshotMutation = useTriggerSnapshot();
  const snapshotSrc =
    check?.snapshot_url && check?.snapshot_url.length > 0
      ? `${check.snapshot_url}${check.snapshot_taken_at ? `?t=${encodeURIComponent(check.snapshot_taken_at)}` : ''}`
      : '';
  const isSnapshotCapable =
    (!!check?.url &&
      (check.url.startsWith('http://') || check.url.startsWith('https://'))) ||
    (check?.type === 'tailscale_service' &&
      !!check?.tailscale_service_host &&
      (check.tailscale_service_protocol === 'http' || check.tailscale_service_protocol === 'https'));

  // Re-render every 5 seconds to keep "time ago" displays fresh
  useTimeAgoTick(5000);

  const filteredHistory = computeStatusTransitions(history);

  const handleTrigger = async () => {
    if (!check) return;
    try {
      await triggerMutation.mutateAsync(check.id);
      showToast('Check triggered. Waiting for next result…', 'success');
    } catch {
      showToast('Failed to trigger check', 'error');
    }
  };

  const handleSnapshot = async () => {
    if (!check) return;
    try {
      await snapshotMutation.mutateAsync(check.id);
      showToast('Snapshot triggered. Refresh in a moment.', 'success');
    } catch (error) {
      const message = error instanceof Error ? error.message : 'Failed to trigger snapshot';
      showToast(message, 'error');
    }
  };

  const handleToggleEnabled = async () => {
    if (!check) return;
    try {
      await toggleEnabledMutation.mutateAsync(check);
    } catch {
      showToast('Failed to update check', 'error');
    }
  };

  const handleDelete = async () => {
    if (!check) return;
    if (!confirm('Delete this monitor?')) return;
    try {
      await deleteMutation.mutateAsync(check.id);
      showToast('Monitor deleted', 'success');
      onClose();
    } catch {
      showToast('Failed to delete check', 'error');
    }
  };

  const uptimePercent = calculateUptime(history);
  const avgLatency = calculateAvgLatency(history);
  const downCount = history.filter((h) => !h.success).length;

  return (
    <div className="bg-terminal-surface border border-terminal-border rounded-lg lg:sticky lg:top-24 max-h-[calc(100vh-8rem)] overflow-y-auto overflow-x-hidden">
      {/* <div className="p-6 border-b border-terminal-border">
        <div className="text-xs text-terminal-muted uppercase tracking-widest">
          Details
        </div>
      </div> */}

      {!check ? (
        <div className="p-6 text-terminal-muted">
          <div className="text-sm">
            Select a monitor to view graph, history, stats, and logs.
          </div>
        </div>
      ) : (
        <>
          {/* Check Header */}
          <div className="p-6 border-b border-terminal-border">
            <div className="flex items-start justify-between gap-4">
              <div className="min-w-0">
                <div className="flex items-center gap-3">
                  <StatusDot
                    isUp={check.is_up}
                    enabled={check.enabled}
                    glow={check.enabled}
                  />
                  <div className="font-bold text-terminal-text truncate">
                    {check.name}
                  </div>
                </div>
                <div className="mt-2 flex items-center gap-2 flex-wrap">
                  {!check.enabled && (
                    <span className="text-[10px] px-2 py-0.5 rounded uppercase font-bold bg-terminal-yellow/20 text-terminal-yellow">
                      paused
                    </span>
                  )}
                  <span
                    className={cn(
                      'text-[10px] px-2 py-0.5 rounded uppercase font-bold',
                      getCheckTypeClass(check.type)
                    )}
                  >
                    {check.type}
                  </span>
                  <span className="text-xs text-terminal-muted break-all">
                    {getCheckTarget(check)}
                  </span>
                </div>
                <div className="mt-2 text-xs text-terminal-muted">
                  last checked:{' '}
                  <span className="text-terminal-text" title={formatDate(check.last_checked_at)}>
                    {formatTimeAgo(check.last_checked_at)}
                  </span>
                </div>
              </div>
              <button
                type="button"
                onClick={onClose}
                className="text-terminal-muted hover:text-terminal-text text-2xl leading-none"
              >
                <X className="w-5 h-5" />
              </button>
            </div>

            <div className="mt-4 flex gap-2 flex-wrap">
              <button
                onClick={handleTrigger}
                disabled={!check.enabled}
                className="text-[10px] bg-terminal-green/20 hover:bg-terminal-green/30 text-terminal-green px-3 py-1.5 rounded uppercase tracking-wide disabled:opacity-50"
              >
                check now
              </button>
              <button
                onClick={handleToggleEnabled}
                className={cn(
                  'text-[10px] px-3 py-1.5 rounded uppercase tracking-wide',
                  check.enabled
                    ? 'bg-terminal-yellow/20 hover:bg-terminal-yellow/30 text-terminal-yellow'
                    : 'bg-terminal-green/20 hover:bg-terminal-green/30 text-terminal-green'
                )}
              >
                {check.enabled ? 'pause' : 'resume'}
              </button>
              <button
                onClick={() => onEditCheck(check)}
                className="text-[10px] bg-terminal-border hover:bg-terminal-muted text-terminal-text px-3 py-1.5 rounded uppercase tracking-wide"
              >
                edit
              </button>
              <button
                onClick={() => onOpenHistory(check)}
                className="text-[10px] bg-terminal-border hover:bg-terminal-muted text-terminal-text px-3 py-1.5 rounded uppercase tracking-wide"
              >
                changes log
              </button>
              <button
                onClick={handleSnapshot}
                disabled={snapshotMutation.isPending}
                className="text-[10px] bg-terminal-cyan/20 hover:bg-terminal-cyan/30 text-terminal-cyan px-3 py-1.5 rounded uppercase tracking-wide disabled:opacity-50"
              >
                {snapshotMutation.isPending ? 'snapshotting…' : 'snapshot now'}
              </button>
              <button
                onClick={handleDelete}
                className="text-[10px] bg-terminal-red/20 hover:bg-terminal-red/30 text-terminal-red px-3 py-1.5 rounded uppercase tracking-wide"
              >
                delete
              </button>
            </div>
          </div>

          {/* Snapshot */}
          {isSnapshotCapable && (
            <div className="p-6 border-b border-terminal-border">
              <div className="flex justify-between items-center text-xs mb-3">
                <span className="text-terminal-muted uppercase tracking-widest">Snapshot</span>
                <span className="text-terminal-muted">
                  {check.snapshot_taken_at
                    ? `captured ${formatTimeAgo(check.snapshot_taken_at)}`
                    : 'auto refresh every 6h'}
                </span>
              </div>
              {snapshotSrc ? (
                <div className="border border-terminal-border rounded overflow-hidden bg-terminal-bg">
                  <img
                    src={snapshotSrc}
                    alt="Latest monitor snapshot"
                    className="w-full aspect-video object-cover"
                  />
                </div>
              ) : (
                <div className="border border-terminal-border rounded bg-terminal-bg text-terminal-muted text-sm p-6">
                  Snapshot pending. Configure Cloudflare credentials in settings to enable previews.
                </div>
              )}
              {check.snapshot_error && (
                <div className="text-terminal-red text-xs mt-2 break-words">
                  {check.snapshot_error}
                </div>
              )}
              <div className="text-[10px] text-terminal-muted mt-2">
                Stored in the server data directory and refreshed every 6 hours.
              </div>
            </div>
          )}

          {/* Chart */}
          <div className="p-6 border-b border-terminal-border">
            <div className="flex justify-between items-center text-xs mb-3">
              <span className="text-terminal-muted">response time graph</span>
              <span className="text-terminal-muted">
                range: <span className="text-terminal-text">{timeRange}</span>
              </span>
            </div>
            {isLoadingHistory ? (
              <div className="h-40 w-full flex items-center justify-center">
                <div className="flex flex-col items-center gap-2">
                  <div className="w-5 h-5 border-2 border-terminal-green border-t-transparent rounded-full animate-spin" />
                  <span className="text-xs text-terminal-muted">Loading chart...</span>
                </div>
              </div>
            ) : (
              <ResponseTimeChart history={history} isUp={check.is_up} height="h-40" />
            )}
          </div>

          {/* Stats */}
          <div className="p-6 border-b border-terminal-border">
            <div className="text-xs text-terminal-muted uppercase tracking-widest mb-3">
              Stats
            </div>
            {isLoadingHistory ? (
              <div className="grid grid-cols-2 gap-3">
                <StatBoxSkeleton />
                <StatBoxSkeleton />
                <StatBoxSkeleton />
                <StatBoxSkeleton />
              </div>
            ) : (
              <div className="grid grid-cols-2 gap-3">
                <StatBox label="uptime" value={uptimePercent} />
                <StatBox label="avg latency" value={avgLatency} />
                <StatBox label="checks" value={String(history.length)} />
                <StatBox label="down" value={String(downCount)} />
              </div>
            )}
          </div>

          {/* Status Bar */}
          <div className="p-6 border-b border-terminal-border">
            <div className="text-xs text-terminal-muted uppercase tracking-widest mb-3">
              Status History
            </div>
            {isLoadingHistory ? (
              <div className="h-8 w-full bg-terminal-border/50 rounded animate-pulse" />
            ) : (
              <StatusBar history={history} check={check} timeRange={timeRange} />
            )}
          </div>

          {/* History (Status Changes) */}
          <div className="p-6 border-b border-terminal-border">
            <div className="flex justify-between items-center mb-3">
              <div className="text-xs text-terminal-muted uppercase tracking-widest">
                History (Status Changes)
              </div>
              <div className="text-[10px] text-terminal-muted">
                {isLoadingHistory ? '...' : `${filteredHistory.length} events`}
              </div>
            </div>
            {isLoadingHistory ? (
              <div className="space-y-2">
                {[...Array(3)].map((_, i) => (
                  <div key={i} className="h-8 bg-terminal-border/30 rounded animate-pulse" />
                ))}
              </div>
            ) : filteredHistory.length === 0 ? (
              <div className="text-terminal-muted text-sm">
                No status changes recorded yet
              </div>
            ) : (
              <div className="overflow-hidden">
                <table className="w-full text-left text-sm">
                  <thead className="bg-terminal-bg sticky top-0 text-[10px] uppercase text-terminal-muted tracking-widest">
                    <tr>
                      <th className="p-2">time</th>
                      <th className="p-2">status</th>
                      <th className="p-2">latency</th>
                    </tr>
                  </thead>
                  <tbody className="divide-y divide-terminal-border text-terminal-text">
                    {filteredHistory.slice(0, 50).map((item, idx) => (
                      <tr key={idx}>
                        <td className="p-2 text-terminal-muted">
                          {formatDate(item.checked_at)}
                        </td>
                        <td
                          className={cn(
                            'p-2',
                            item.success
                              ? 'text-terminal-green'
                              : 'text-terminal-red'
                          )}
                        >
                          {item.success ? 'UP' : 'DOWN'}
                        </td>
                        <td className="p-2 text-terminal-muted">
                          {item.response_time_ms || 0}ms
                        </td>
                      </tr>
                    ))}
                  </tbody>
                </table>
              </div>
            )}
          </div>

          {/* Logs */}
          <div className="p-6">
            <div className="flex justify-between items-center mb-3">
              <div className="text-xs text-terminal-muted uppercase tracking-widest">
                Logs
              </div>
              <div className="text-[10px] text-terminal-muted">
                {history.length ? 'latest first' : ''}
              </div>
            </div>
            {isLoadingHistory ? (
              <div className="space-y-2">
                {[...Array(3)].map((_, i) => (
                  <div key={i} className="bg-terminal-bg border border-terminal-border rounded p-3 animate-pulse">
                    <div className="flex justify-between items-center gap-3 mb-2">
                      <div className="h-3 w-24 bg-terminal-border rounded" />
                      <div className="h-4 w-10 bg-terminal-border rounded" />
                    </div>
                    <div className="h-3 w-32 bg-terminal-border rounded" />
                  </div>
                ))}
              </div>
            ) : history.length === 0 ? (
              <div className="text-terminal-muted text-sm">No logs yet</div>
            ) : (
              <div className="space-y-2">
                {history.slice(0, 50).map((entry, idx) => (
                  <div
                    key={idx}
                    className="bg-terminal-bg border border-terminal-border rounded p-3"
                  >
                    <div className="flex justify-between items-center gap-3">
                      <div className="text-xs text-terminal-muted">
                        {formatDate(entry.checked_at)}
                      </div>
                      <div
                        className={cn(
                          'text-[10px] px-2 py-0.5 rounded uppercase font-bold',
                          entry.success
                            ? 'bg-terminal-green/20 text-terminal-green'
                            : 'bg-terminal-red/20 text-terminal-red'
                        )}
                      >
                        {entry.success ? 'UP' : 'DOWN'}
                      </div>
                    </div>
                    <div className="mt-2 text-xs text-terminal-muted">
                      latency:{' '}
                      <span className="text-terminal-text">
                        {entry.response_time_ms || 0}ms
                      </span>
                      <span className="text-terminal-muted mx-2">|</span>
                      status:{' '}
                      <span className="text-terminal-text">
                        {entry.status_code || 0}
                      </span>
                    </div>
                    {(entry.error_message || entry.response_body) && (
                      <div className="mt-1 text-xs text-terminal-text whitespace-pre-wrap break-all overflow-hidden">
                        {entry.error_message || entry.response_body || '-'}
                      </div>
                    )}
                  </div>
                ))}
              </div>
            )}
          </div>
        </>
      )}
    </div>
  );
}

function StatBox({ label, value }: { label: string; value: string }) {
  return (
    <div className="bg-terminal-bg border border-terminal-border rounded p-3">
      <div className="text-[10px] text-terminal-muted uppercase tracking-widest">
        {label}
      </div>
      <div className="text-lg font-bold">{value}</div>
    </div>
  );
}

function StatBoxSkeleton() {
  return (
    <div className="bg-terminal-bg border border-terminal-border rounded p-3 animate-pulse">
      <div className="h-3 w-12 bg-terminal-border rounded mb-2" />
      <div className="h-6 w-16 bg-terminal-border rounded" />
    </div>
  );
}

function computeStatusTransitions(history: CheckStatus[]): CheckStatus[] {
  const transitions: CheckStatus[] = [];
  for (let i = 0; i < history.length; i++) {
    const current = history[i];
    const prev = history[i + 1];
    if (!prev || current.success !== prev.success) {
      transitions.push(current);
    }
  }
  return transitions;
}

function calculateUptime(history: CheckStatus[]): string {
  if (history.length === 0) return '—';
  const up = history.filter((h) => h.success).length;
  return ((up / history.length) * 100).toFixed(2) + '%';
}

function calculateAvgLatency(history: CheckStatus[]): string {
  const data = history.filter((h) => typeof h.response_time_ms === 'number');
  if (data.length === 0) return '—';
  const sum = data.reduce((acc, h) => acc + (h.response_time_ms || 0), 0);
  return Math.round(sum / data.length) + 'ms';
}
