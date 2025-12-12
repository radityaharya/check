import { useEffect, useRef } from 'react';
import { useQueryClient } from '@tanstack/react-query';
import { useTimeRange } from '@/store';
import type { Check, CheckGroup, CheckStatus, Stats } from '@/types';

// Matches the actual SSE payload from the backend
interface SSECheckUpdate {
  check_id: number;
  check: Check;
  is_up: boolean;
  last_checked_at: string;
  last_status: CheckStatus;
}

interface UseSSEOptions {
  enabled?: boolean;
  onConnect?: () => void;
  onDisconnect?: () => void;
}

export function useSSE(options: UseSSEOptions = {}) {
  const { enabled = true, onConnect, onDisconnect } = options;

  const queryClient = useQueryClient();
  const timeRange = useTimeRange();
  const eventSourceRef = useRef<EventSource | null>(null);
  const reconnectTimeoutRef = useRef<ReturnType<typeof setTimeout> | null>(null);
  const reconnectAttempts = useRef(0);
  const isConnecting = useRef(false);

  // Store values in refs to avoid dependency issues
  const onConnectRef = useRef(onConnect);
  const onDisconnectRef = useRef(onDisconnect);
  const timeRangeRef = useRef(timeRange);

  useEffect(() => {
    onConnectRef.current = onConnect;
    onDisconnectRef.current = onDisconnect;
  }, [onConnect, onDisconnect]);

  useEffect(() => {
    timeRangeRef.current = timeRange;
  }, [timeRange]);

  useEffect(() => {
    if (!enabled) {
      // Cleanup when disabled
      if (reconnectTimeoutRef.current) {
        clearTimeout(reconnectTimeoutRef.current);
        reconnectTimeoutRef.current = null;
      }
      if (eventSourceRef.current) {
        eventSourceRef.current.close();
        eventSourceRef.current = null;
      }
      return;
    }

    const connect = () => {
      // Prevent multiple simultaneous connection attempts
      if (isConnecting.current || eventSourceRef.current?.readyState === EventSource.OPEN) {
        return;
      }

      isConnecting.current = true;

      // Close existing connection if any
      if (eventSourceRef.current) {
        eventSourceRef.current.close();
        eventSourceRef.current = null;
      }

      const eventSource = new EventSource('/api/stream/updates');
      eventSourceRef.current = eventSource;

      eventSource.onopen = () => {
        console.log('[SSE] Connected');
        isConnecting.current = false;
        reconnectAttempts.current = 0;
        onConnectRef.current?.();
      };

      // Handle named 'check_update' events from the server
      eventSource.addEventListener('check_update', (event) => {
        try {
          const update: SSECheckUpdate = JSON.parse(event.data);
          console.log('[SSE] Received check_update:', update);

          const currentRange = timeRangeRef.current;

          // Update the grouped checks cache directly (using current timeRange)
          queryClient.setQueryData<CheckGroup[]>(['checks', 'grouped', currentRange], (oldData) => {
            if (!oldData) return oldData;

            return oldData.map((group) => {
              const checkIndex = group.checks.findIndex((c) => c.id === update.check_id);
              if (checkIndex === -1) return group;

              const updatedChecks = [...group.checks];
              const existingCheck = updatedChecks[checkIndex];

              // Merge the updated check data with existing data
              updatedChecks[checkIndex] = {
                ...existingCheck,
                ...update.check,
                is_up: update.is_up,
                last_checked_at: update.last_checked_at,
                last_status: update.last_status,
                // Prepend new status to history (keep last 100)
                history: existingCheck.history
                  ? [update.last_status, ...existingCheck.history].slice(0, 100)
                  : [update.last_status],
              };

              // Recalculate group stats
              const upCount = updatedChecks.filter((c) => c.enabled && c.is_up).length;
              const downCount = updatedChecks.filter((c) => c.enabled && !c.is_up).length;

              return {
                ...group,
                checks: updatedChecks,
                is_up: downCount === 0,
                up_count: upCount,
                down_count: downCount,
              };
            });
          });

          // Update the history cache for the affected check (current timeRange)
          queryClient.setQueryData<CheckStatus[]>(
            ['checks', update.check_id, 'history', currentRange],
            (oldHistory) => {
              if (!oldHistory) return oldHistory;

              const alreadyExists = oldHistory.some((item) => {
                if (item.id && update.last_status.id) {
                  return item.id === update.last_status.id;
                }
                return item.checked_at === update.last_status.checked_at;
              });

              if (alreadyExists) return oldHistory;

              return [update.last_status, ...oldHistory].slice(0, 500);
            }
          );

          // Invalidate check stats to refresh region-specific stats
          queryClient.invalidateQueries({
            queryKey: ['check-stats', update.check_id, currentRange],
          });

          // Update stats cache directly
          queryClient.setQueryData<Stats>(['stats', currentRange], (oldStats) => {
            if (!oldStats) return oldStats;

            // We'd need to know the previous state to accurately update
            // For simplicity, just update the timestamp-related stats
            return {
              ...oldStats,
              // Stats will be recalculated on next full fetch
            };
          });
        } catch (error) {
          console.error('[SSE] Failed to parse check_update:', error);
        }
      });

      // Handle 'connected' event
      eventSource.addEventListener('connected', (event) => {
        console.log('[SSE] Server confirmed connection:', event.data);
      });

      // Handle generic messages (fallback)
      eventSource.onmessage = (event) => {
        console.log('[SSE] Generic message:', event.data);
      };

      eventSource.onerror = () => {
        console.log('[SSE] Connection error or closed');
        isConnecting.current = false;
        eventSource.close();
        eventSourceRef.current = null;
        onDisconnectRef.current?.();

        // Only reconnect if still enabled and under max attempts
        const maxAttempts = 5;
        if (reconnectAttempts.current < maxAttempts) {
          const delay = Math.min(
            1000 * Math.pow(2, reconnectAttempts.current),
            30000
          );
          console.log(`[SSE] Reconnecting in ${delay}ms (attempt ${reconnectAttempts.current + 1}/${maxAttempts})`);

          reconnectTimeoutRef.current = setTimeout(() => {
            reconnectAttempts.current++;
            connect();
          }, delay);
        } else {
          console.log('[SSE] Max reconnect attempts reached, giving up');
        }
      };
    };

    // Initial connection
    connect();

    // Cleanup on unmount or when enabled changes
    return () => {
      if (reconnectTimeoutRef.current) {
        clearTimeout(reconnectTimeoutRef.current);
        reconnectTimeoutRef.current = null;
      }
      if (eventSourceRef.current) {
        eventSourceRef.current.close();
        eventSourceRef.current = null;
      }
      isConnecting.current = false;
    };
  }, [enabled, queryClient]);

  return {
    isConnected: eventSourceRef.current?.readyState === EventSource.OPEN,
  };
}
