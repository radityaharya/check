import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query';
import type { Group, GroupFormData, Tag, TagFormData } from '@/types';

// Groups API
async function fetchGroups(): Promise<Group[]> {
  const response = await fetch('/api/groups');
  if (!response.ok) throw new Error('Failed to fetch groups');
  return response.json();
}

async function createGroup(data: GroupFormData): Promise<Group> {
  const response = await fetch('/api/groups', {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify(data),
  });
  if (!response.ok) throw new Error('Failed to create group');
  return response.json();
}

async function updateGroup(id: number, data: GroupFormData): Promise<Group> {
  const response = await fetch(`/api/groups/${id}`, {
    method: 'PUT',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify(data),
  });
  if (!response.ok) throw new Error('Failed to update group');
  return response.json();
}

async function deleteGroup(id: number): Promise<void> {
  const response = await fetch(`/api/groups/${id}`, { method: 'DELETE' });
  if (!response.ok) throw new Error('Failed to delete group');
}

// Tags API
async function fetchTags(): Promise<Tag[]> {
  const response = await fetch('/api/tags');
  if (!response.ok) throw new Error('Failed to fetch tags');
  return response.json();
}

async function createTag(data: TagFormData): Promise<Tag> {
  const response = await fetch('/api/tags', {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify(data),
  });
  if (!response.ok) throw new Error('Failed to create tag');
  return response.json();
}

async function updateTag(id: number, data: TagFormData): Promise<Tag> {
  const response = await fetch(`/api/tags/${id}`, {
    method: 'PUT',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify(data),
  });
  if (!response.ok) throw new Error('Failed to update tag');
  return response.json();
}

async function deleteTag(id: number): Promise<void> {
  const response = await fetch(`/api/tags/${id}`, { method: 'DELETE' });
  if (!response.ok) throw new Error('Failed to delete tag');
}

// Groups hooks
export function useGroups() {
  return useQuery({
    queryKey: ['groups'],
    queryFn: fetchGroups,
    staleTime: 1000 * 60 * 5, // 5 minutes
  });
}

export function useCreateGroup() {
  const queryClient = useQueryClient();

  return useMutation({
    mutationFn: createGroup,
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['groups'] });
      queryClient.invalidateQueries({ queryKey: ['checks'] });
    },
  });
}

export function useUpdateGroup() {
  const queryClient = useQueryClient();

  return useMutation({
    mutationFn: ({ id, data }: { id: number; data: GroupFormData }) =>
      updateGroup(id, data),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['groups'] });
      queryClient.invalidateQueries({ queryKey: ['checks'] });
    },
  });
}

export function useDeleteGroup() {
  const queryClient = useQueryClient();

  return useMutation({
    mutationFn: deleteGroup,
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['groups'] });
      queryClient.invalidateQueries({ queryKey: ['checks'] });
    },
  });
}

// Tags hooks
export function useTags() {
  return useQuery({
    queryKey: ['tags'],
    queryFn: fetchTags,
    staleTime: 1000 * 60 * 5, // 5 minutes
  });
}

export function useCreateTag() {
  const queryClient = useQueryClient();

  return useMutation({
    mutationFn: createTag,
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['tags'] });
      queryClient.invalidateQueries({ queryKey: ['checks'] });
    },
  });
}

export function useUpdateTag() {
  const queryClient = useQueryClient();

  return useMutation({
    mutationFn: ({ id, data }: { id: number; data: TagFormData }) =>
      updateTag(id, data),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['tags'] });
      queryClient.invalidateQueries({ queryKey: ['checks'] });
    },
  });
}

export function useDeleteTag() {
  const queryClient = useQueryClient();

  return useMutation({
    mutationFn: deleteTag,
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['tags'] });
      queryClient.invalidateQueries({ queryKey: ['checks'] });
    },
  });
}
