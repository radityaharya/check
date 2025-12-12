import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query';
import type { Probe, CreateProbeRequest, CreateProbeResponse, RegenerateTokenResponse } from '@/types';

async function fetchProbes(): Promise<Probe[]> {
  const response = await fetch('/api/probes');
  if (!response.ok) throw new Error('Failed to fetch probes');
  return response.json();
}

async function createProbe(data: CreateProbeRequest): Promise<CreateProbeResponse> {
  const response = await fetch('/api/probes', {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify(data),
  });
  if (!response.ok) {
    const error = await response.json().catch(() => ({}));
    throw new Error(error.error || 'Failed to create probe');
  }
  return response.json();
}

async function deleteProbe(id: number): Promise<void> {
  const response = await fetch(`/api/probes/${id}`, { method: 'DELETE' });
  if (!response.ok) throw new Error('Failed to delete probe');
}

async function regenerateToken(id: number): Promise<RegenerateTokenResponse> {
  const response = await fetch(`/api/probes/${id}/regenerate-token`, { method: 'POST' });
  if (!response.ok) throw new Error('Failed to regenerate token');
  return response.json();
}

export function useProbes() {
  const queryClient = useQueryClient();

  const { data: probes = [], isLoading, error } = useQuery({
    queryKey: ['probes'],
    queryFn: fetchProbes,
    refetchInterval: 30000,
  });

  const createMutation = useMutation({
    mutationFn: createProbe,
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['probes'] });
    },
  });

  const deleteMutation = useMutation({
    mutationFn: deleteProbe,
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['probes'] });
    },
  });

  const regenerateTokenMutation = useMutation({
    mutationFn: regenerateToken,
  });

  return {
    probes,
    isLoading,
    error,
    createProbe: createMutation.mutateAsync,
    deleteProbe: deleteMutation.mutateAsync,
    regenerateToken: regenerateTokenMutation.mutateAsync,
    isCreating: createMutation.isPending,
    isDeleting: deleteMutation.isPending,
    isRegenerating: regenerateTokenMutation.isPending,
  };
}

