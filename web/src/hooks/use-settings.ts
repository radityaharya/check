import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query';
import type { Settings, TailscaleDevice, APIKey, Passkey } from '@/types';

// Settings API
async function fetchSettings(): Promise<Settings> {
  const response = await fetch('/api/settings');
  if (!response.ok) throw new Error('Failed to fetch settings');
  return response.json();
}

async function saveSettings(data: Settings): Promise<void> {
  const response = await fetch('/api/settings', {
    method: 'PUT',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify(data),
  });
  if (!response.ok) throw new Error('Failed to save settings');
}

async function testWebhook(): Promise<void> {
  const response = await fetch('/api/settings/test-webhook', { method: 'POST' });
  if (!response.ok) {
    const data = await response.json();
    throw new Error(data.error || 'Failed to test webhook');
  }
}

async function testGotify(): Promise<void> {
  const response = await fetch('/api/settings/test-gotify', { method: 'POST' });
  if (!response.ok) {
    const data = await response.json();
    throw new Error(data.error || 'Failed to test Gotify');
  }
}

async function testTailscale(): Promise<{ device_count: number }> {
  const response = await fetch('/api/settings/test-tailscale', { method: 'POST' });
  if (!response.ok) {
    const data = await response.json();
    throw new Error(data.error || 'Failed to test Tailscale');
  }
  return response.json();
}

async function testBrowserless(url: string): Promise<Blob> {
  const response = await fetch('/api/settings/test-browserless', {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify({ url }),
  });
  if (!response.ok) {
    const data = await response.json();
    throw new Error(data.error || 'Failed to test Browserless');
  }
  return response.blob();
}

// Tailscale Devices
async function fetchTailscaleDevices(): Promise<TailscaleDevice[]> {
  const response = await fetch('/api/tailscale/devices');
  if (!response.ok) return [];
  return response.json();
}

// API Keys
async function fetchAPIKeys(): Promise<APIKey[]> {
  const response = await fetch('/api/auth/apikeys');
  if (!response.ok) throw new Error('Failed to fetch API keys');
  return response.json();
}

async function createAPIKey(name: string): Promise<{ key: string }> {
  const response = await fetch('/api/auth/apikeys', {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify({ name }),
  });
  if (!response.ok) throw new Error('Failed to create API key');
  return response.json();
}

async function deleteAPIKey(id: number): Promise<void> {
  const response = await fetch('/api/auth/apikeys', {
    method: 'DELETE',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify({ id }),
  });
  if (!response.ok) throw new Error('Failed to delete API key');
}

// Passkeys
async function fetchPasskeys(): Promise<Passkey[]> {
  const response = await fetch('/api/auth/passkeys');
  if (!response.ok) throw new Error('Failed to fetch passkeys');
  return response.json();
}

async function deletePasskey(id: number): Promise<void> {
  const response = await fetch('/api/auth/passkeys', {
    method: 'DELETE',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify({ id }),
  });
  if (!response.ok) throw new Error('Failed to delete passkey');
}

// React Query hooks
export function useSettings() {
  return useQuery({
    queryKey: ['settings'],
    queryFn: fetchSettings,
    staleTime: 1000 * 60 * 5,
  });
}

export function useSaveSettings() {
  const queryClient = useQueryClient();

  return useMutation({
    mutationFn: saveSettings,
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['settings'] });
    },
  });
}

// Alias for useSaveSettings
export const useUpdateSettings = useSaveSettings;

export function useTestWebhook() {
  return useMutation({
    mutationFn: testWebhook,
  });
}

// Alias for useTestWebhook
export const useTestDiscordWebhook = useTestWebhook;

export function useTestGotify() {
  return useMutation({
    mutationFn: testGotify,
  });
}

export function useTestTailscale() {
  return useMutation({
    mutationFn: testTailscale,
  });
}

export function useTestBrowserless() {
  return useMutation({
    mutationFn: testBrowserless,
  });
}

export function useTailscaleDevices(enabled: boolean) {
  return useQuery({
    queryKey: ['tailscale', 'devices'],
    queryFn: fetchTailscaleDevices,
    enabled,
    staleTime: 1000 * 60,
  });
}

export function useAPIKeys() {
  return useQuery({
    queryKey: ['apikeys'],
    queryFn: fetchAPIKeys,
    staleTime: 1000 * 60 * 5,
  });
}

export function useCreateAPIKey() {
  const queryClient = useQueryClient();

  return useMutation({
    mutationFn: createAPIKey,
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['apikeys'] });
    },
  });
}

export function useDeleteAPIKey() {
  const queryClient = useQueryClient();

  return useMutation({
    mutationFn: deleteAPIKey,
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['apikeys'] });
    },
  });
}

export function usePasskeys() {
  return useQuery({
    queryKey: ['passkeys'],
    queryFn: fetchPasskeys,
    staleTime: 1000 * 60 * 5,
  });
}

export function useDeletePasskey() {
  const queryClient = useQueryClient();

  return useMutation({
    mutationFn: deletePasskey,
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['passkeys'] });
    },
  });
}
