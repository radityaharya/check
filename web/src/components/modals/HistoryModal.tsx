import { useState } from 'react';
import { Modal, ModalHeader } from '@/components/ui/modal';
import { Input } from '@/components/ui/input';
import { Spinner } from '@/components/ui/spinner';
import { StatusDot } from '@/components/ui/status-bar';
import { useCheckHistory } from '@/hooks';
import { formatDate, formatResponseTime, getStatusClass } from '@/lib/helpers';
import { cn } from '@/lib/utils';
import type { Check, CheckStatus, TimeRange } from '@/types';

interface HistoryModalProps {
  isOpen: boolean;
  onClose: () => void;
  check: Check;
  timeRange: TimeRange;
}

export function HistoryModal({
  isOpen,
  onClose,
  check,
  timeRange,
}: HistoryModalProps) {
  const {
    data: history = [],
    isLoading,
    error,
  } = useCheckHistory(check.id, timeRange);

  const [searchTerm, setSearchTerm] = useState('');
  const [filterStatus, setFilterStatus] = useState<'all' | 'success' | 'failed'>(
    'all'
  );

  // Filter history
  const filteredHistory = history.filter((log: CheckStatus) => {
    if (filterStatus === 'success' && !log.success) return false;
    if (filterStatus === 'failed' && log.success) return false;
    if (searchTerm) {
      const search = searchTerm.toLowerCase();
      return (
        log.error_message?.toLowerCase().includes(search) ||
        log.actual_value?.toLowerCase().includes(search)
      );
    }
    return true;
  });

  // Calculate stats
  const totalChecks = history.length;
  const successCount = history.filter((l: CheckStatus) => l.success).length;
  const failCount = totalChecks - successCount;
  const successRate = totalChecks > 0 ? (successCount / totalChecks) * 100 : 0;
  const avgResponseTime =
    totalChecks > 0
      ? history.reduce((sum: number, l: CheckStatus) => sum + l.response_time_ms, 0) / totalChecks
      : 0;

  return (
    <Modal isOpen={isOpen} onClose={onClose} size="lg">
      <ModalHeader onClose={onClose}>
        $ history: {check.name}
      </ModalHeader>

      {/* Stats Summary */}
      <div className="px-6 py-4 border-b border-terminal-border">
        <div className="grid grid-cols-2 md:grid-cols-4 gap-4 text-center">
          <div>
            <div className="text-[10px] uppercase text-terminal-muted tracking-widest">
              Total Checks
            </div>
            <div className="text-xl font-bold text-terminal-cyan">
              {totalChecks}
            </div>
          </div>
          <div>
            <div className="text-[10px] uppercase text-terminal-muted tracking-widest">
              Uptime
            </div>
            <div
              className={cn(
                'text-xl font-bold',
                successRate >= 99
                  ? 'text-terminal-green'
                  : successRate >= 95
                  ? 'text-terminal-yellow'
                  : 'text-terminal-red'
              )}
            >
              {successRate.toFixed(2)}%
            </div>
          </div>
          <div>
            <div className="text-[10px] uppercase text-terminal-muted tracking-widest">
              Failures
            </div>
            <div className="text-xl font-bold text-terminal-red">{failCount}</div>
          </div>
          <div>
            <div className="text-[10px] uppercase text-terminal-muted tracking-widest">
              Avg Response
            </div>
            <div className="text-xl font-bold text-terminal-text">
              {formatResponseTime(avgResponseTime)}
            </div>
          </div>
        </div>
      </div>

      {/* Filters */}
      <div className="px-6 py-3 border-b border-terminal-border flex flex-wrap gap-3 items-center">
        <Input
          type="text"
          value={searchTerm}
          onChange={(e) => setSearchTerm(e.target.value)}
          placeholder="Search logs..."
          className="flex-1 min-w-[200px] bg-terminal-surface border-terminal-border text-terminal-text text-sm"
        />
        <div className="flex gap-1">
          {(['all', 'success', 'failed'] as const).map((status) => (
            <button
              key={status}
              onClick={() => setFilterStatus(status)}
              className={cn(
                'px-3 py-1 text-[10px] uppercase tracking-widest rounded transition',
                filterStatus === status
                  ? status === 'all'
                    ? 'bg-terminal-cyan/20 text-terminal-cyan'
                    : status === 'success'
                    ? 'bg-terminal-green/20 text-terminal-green'
                    : 'bg-terminal-red/20 text-terminal-red'
                  : 'text-terminal-muted hover:text-terminal-text'
              )}
            >
              {status}
            </button>
          ))}
        </div>
      </div>

      {/* History List */}
      <div className="p-6 max-h-[50vh] overflow-y-auto">
        {isLoading ? (
          <div className="flex justify-center py-8">
            <Spinner size="lg" />
          </div>
        ) : error ? (
          <div className="text-center py-8 text-terminal-red">
            Failed to load history
          </div>
        ) : filteredHistory.length === 0 ? (
          <div className="text-center py-8 text-terminal-muted">
            {searchTerm || filterStatus !== 'all'
              ? 'No matching logs found'
              : 'No history available'}
          </div>
        ) : (
          <div className="space-y-2">
            {filteredHistory.map((log: CheckStatus) => (
              <HistoryLogItem key={log.id} log={log} />
            ))}
          </div>
        )}
      </div>

      {/* Footer */}
      <div className="px-6 py-3 border-t border-terminal-border text-[10px] text-terminal-muted">
        Showing {filteredHistory.length} of {totalChecks} logs
      </div>
    </Modal>
  );
}

interface HistoryLogItemProps {
  log: CheckStatus;
}

function HistoryLogItem({ log }: HistoryLogItemProps) {
  const [expanded, setExpanded] = useState(false);

  return (
    <div
      className={cn(
        'p-3 rounded border transition cursor-pointer',
        log.success
          ? 'bg-terminal-surface border-terminal-border hover:border-terminal-green/30'
          : 'bg-terminal-red/5 border-terminal-red/20 hover:border-terminal-red/40'
      )}
      onClick={() => setExpanded(!expanded)}
    >
      <div className="flex items-center justify-between">
        <div className="flex items-center gap-3">
          <StatusDot
            status={log.success ? 'success' : 'error'}
            pulse={false}
          />
          <div>
            <div className="text-sm font-mono">
              {formatDate(log.checked_at)}
            </div>
            {log.error_message && (
              <div
                className={cn(
                  'text-[10px] truncate max-w-[300px]',
                  log.success ? 'text-terminal-muted' : 'text-terminal-red'
                )}
              >
                {log.error_message}
              </div>
            )}
          </div>
        </div>
        <div className="flex items-center gap-4 text-xs">
          <div
            className={cn(
              'font-mono',
              getStatusClass(log.response_time_ms)
            )}
          >
            {formatResponseTime(log.response_time_ms)}
          </div>
          <div className="text-terminal-muted">
            {expanded ? '▲' : '▼'}
          </div>
        </div>
      </div>

      {expanded && (
        <div className="mt-3 pt-3 border-t border-terminal-border space-y-2">
          <DetailRow label="Status" value={log.success ? 'Success' : 'Failed'} />
          <DetailRow label="Response Time" value={`${log.response_time_ms}ms`} />
          {log.status_code && (
            <DetailRow label="HTTP Status" value={String(log.status_code)} />
          )}
          {log.error_message && (
            <DetailRow label="Error" value={log.error_message} />
          )}
          {log.actual_value && (
            <DetailRow label="Actual Value" value={log.actual_value} />
          )}
          {log.response_body && (
            <div>
              <div className="text-[10px] uppercase text-terminal-muted tracking-widest mb-1">
                Response Body
              </div>
              <pre className="text-xs text-terminal-text bg-terminal-bg p-2 rounded overflow-x-auto font-mono max-h-32 overflow-y-auto">
                {log.response_body}
              </pre>
            </div>
          )}
        </div>
      )}
    </div>
  );
}

interface DetailRowProps {
  label: string;
  value: string;
}

function DetailRow({ label, value }: DetailRowProps) {
  return (
    <div className="flex justify-between text-xs">
      <span className="text-terminal-muted">{label}</span>
      <span className="font-mono">{value}</span>
    </div>
  );
}
