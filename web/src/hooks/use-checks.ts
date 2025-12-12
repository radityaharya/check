import { useMemo } from 'react';
import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query';
import type {
  Check,
  CheckGroup,
  CheckFormData,
  CheckStatus,
  Stats,
  TimeRange,
} from '@/types';
import { useSelectedCheckId, useTimeRange } from '@/store';

// API functions
async function fetchGroupedChecks(range: TimeRange): Promise<CheckGroup[]> {
  const response = await fetch(`/api/checks/grouped?range=${encodeURIComponent(range)}`);
  if (!response.ok) throw new Error('Failed to fetch checks');
  return response.json();
}

async function fetchStats(range: TimeRange): Promise<Stats> {
  const response = await fetch(`/api/stats?range=${encodeURIComponent(range)}`);
  if (!response.ok) throw new Error('Failed to fetch stats');
  return response.json();
}

async function fetchCheckHistory(checkId: number, range: TimeRange, limit = 500): Promise<CheckStatus[]> {
  const response = await fetch(
    `/api/checks/${checkId}/history?limit=${limit}&range=${encodeURIComponent(range)}`
  );
  if (!response.ok) throw new Error('Failed to fetch history');
  return response.json();
}

async function createCheck(data: CheckFormData): Promise<Check> {
  const payload = {
    ...data,
    group_id: data.group_id ? Number(data.group_id) : null,
    tailscale_service_port: data.tailscale_service_port
      ? Number(data.tailscale_service_port)
      : undefined,
  };

  const response = await fetch('/api/checks', {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify(payload),
  });

  if (!response.ok) throw new Error('Failed to create check');
  return response.json();
}

async function updateCheck(id: number, data: CheckFormData): Promise<Check> {
  const payload = {
    ...data,
    group_id: data.group_id ? Number(data.group_id) : null,
    tailscale_service_port: data.tailscale_service_port
      ? Number(data.tailscale_service_port)
      : undefined,
  };

  const response = await fetch(`/api/checks/${id}`, {
    method: 'PUT',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify(payload),
  });

  if (!response.ok) throw new Error('Failed to update check');
  return response.json();
}

async function deleteCheck(id: number): Promise<void> {
  const response = await fetch(`/api/checks/${id}`, { method: 'DELETE' });
  if (!response.ok) throw new Error('Failed to delete check');
}

async function triggerCheck(id: number): Promise<void> {
  const response = await fetch(`/api/checks/${id}/trigger`, { method: 'POST' });
  if (!response.ok) throw new Error('Failed to trigger check');
}

async function toggleCheckEnabled(check: Check): Promise<Check> {
  const updateData = {
    ...check,
    enabled: !check.enabled,
    tag_ids: (check.tags || []).map((t) => t.id),
  };

  const response = await fetch(`/api/checks/${check.id}`, {
    method: 'PUT',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify(updateData),
  });

  if (!response.ok) throw new Error('Failed to update check');
  return response.json();
}

// React Query hooks
export function useGroupedChecks(range: TimeRange) {
  return useQuery({
    queryKey: ['checks', 'grouped', range],
    queryFn: () => fetchGroupedChecks(range),
    staleTime: 1000 * 30, // 30 seconds
  });
}

export function useStats(range: TimeRange) {
  return useQuery({
    queryKey: ['stats', range],
    queryFn: () => fetchStats(range),
    staleTime: 1000 * 30, // 30 seconds
    refetchInterval: 1000 * 60, // Refetch every minute
  });
}

export function useCheckHistory(checkId: number | null, range: TimeRange) {
  return useQuery({
    queryKey: ['checks', checkId, 'history', range],
    queryFn: () => fetchCheckHistory(checkId!, range),
    enabled: !!checkId,
    staleTime: 1000 * 30,
  });
}

export function useCreateCheck() {
  const queryClient = useQueryClient();

  return useMutation({
    mutationFn: createCheck,
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['checks'] });
      queryClient.invalidateQueries({ queryKey: ['stats'] });
    },
  });
}

export function useUpdateCheck() {
  const queryClient = useQueryClient();

  return useMutation({
    mutationFn: ({ id, data }: { id: number; data: CheckFormData }) =>
      updateCheck(id, data),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['checks'] });
      queryClient.invalidateQueries({ queryKey: ['stats'] });
    },
  });
}

export function useDeleteCheck() {
  const queryClient = useQueryClient();

  return useMutation({
    mutationFn: deleteCheck,
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['checks'] });
      queryClient.invalidateQueries({ queryKey: ['stats'] });
    },
  });
}

export function useTriggerCheck() {
  return useMutation({
    mutationFn: triggerCheck,
  });
}

export function useToggleCheckEnabled() {
  const queryClient = useQueryClient();

  return useMutation({
    mutationFn: toggleCheckEnabled,
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['checks'] });
      queryClient.invalidateQueries({ queryKey: ['stats'] });
    },
  });
}

// Helper function to update check in cache from SSE
export function updateCheckInCache(
  queryClient: ReturnType<typeof useQueryClient>,
  update: {
    check_id: number;
    is_up: boolean;
    last_checked_at: string;
    last_status: CheckStatus;
  },
  range: TimeRange
) {
  queryClient.setQueryData<CheckGroup[]>(['checks', 'grouped', range], (old) => {
    if (!old) return old;

    return old.map((group) => {
      const checkIndex = group.checks.findIndex((c) => c.id === update.check_id);
      if (checkIndex === -1) return group;

      const updatedChecks = [...group.checks];
      const check = updatedChecks[checkIndex];

      updatedChecks[checkIndex] = {
        ...check,
        is_up: update.is_up,
        last_checked_at: update.last_checked_at,
        last_status: update.last_status,
        history: [update.last_status, ...(check.history || [])].slice(0, 500),
      };

      // Recalculate group status
      let up_count = 0;
      let down_count = 0;
      let is_up = true;

      for (const c of updatedChecks) {
        if (c.enabled) {
          if (c.is_up) {
            up_count++;
          } else {
            down_count++;
            is_up = false;
          }
        }
      }

      return {
        ...group,
        checks: updatedChecks,
        is_up,
        up_count,
        down_count,
      };
    });
  });
}

// Hook to get the currently selected check from cache (reactive to SSE updates)
export function useSelectedCheck(): Check | null {
  const selectedCheckId = useSelectedCheckId();
  const timeRange = useTimeRange();
  const { data: groups } = useGroupedChecks(timeRange);

  return useMemo(() => {
    if (!selectedCheckId || !groups) return null;
    for (const group of groups) {
      const check = group.checks.find((c) => c.id === selectedCheckId);
      if (check) return check;
    }
    return null;
  }, [selectedCheckId, groups]);
}
