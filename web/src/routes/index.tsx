import { useEffect, useState } from 'react';
import { createFileRoute, useNavigate } from '@tanstack/react-router';
import { cn } from '@/lib/utils';
import {
  useAuthCheck,
  useLogout,
  useGroupedChecks,
  useStats,
  useDeleteCheck,
  useTriggerCheck,
  useToggleCheckEnabled,
  useCreateGroup,
  useUpdateGroup,
  useDeleteGroup,
  useCreateTag,
  useUpdateTag,
  useDeleteTag,
  useSSE,
  useSelectedCheck,
  // Store hooks
  useTimeRange,
  useSetTimeRange,
  useDarkMode,
  useToggleDarkMode,
  useSelectedCheckId,
  useSelectCheck,
  useExpandedGroups,
  useToggleGroup,
  useSSEConnected,
  useSetSSEConnected,
  useGroupModal,
  useTagModal,
  useHistoryModal,
} from '@/hooks';
import { LoadingScreen } from '@/components/ui/spinner';
import { useToast, ToastProvider } from '@/components/ui/toast';
import { AppHeader, StatsGrid, TimeRangeSelector } from '@/components/dashboard/Header';
import { MonitorsList } from '@/components/dashboard/MonitorsList';
import { DetailsPane } from '@/components/dashboard/DetailsPane';
import { GroupModal, TagModal, HistoryModal } from '@/components/modals';
import type { Check } from '@/types';

export const Route = createFileRoute('/')({
  component: DashboardPage,
});

function DashboardPage() {
  return (
    <ToastProvider>
      <Dashboard />
    </ToastProvider>
  );
}

function Dashboard() {
  const navigate = useNavigate();
  const { showToast } = useToast();

  // Auth check
  const { data: authData, isLoading: isAuthLoading, error: authError } = useAuthCheck();
  const logout = useLogout();

  // Redirect to login if not authenticated
  useEffect(() => {
    if (!isAuthLoading && (!authData?.isAuthenticated || authError)) {
      navigate({ to: '/login' });
    }
  }, [authData, authError, isAuthLoading, navigate]);

  // Global state from Zustand store
  const darkMode = useDarkMode();
  const toggleDarkMode = useToggleDarkMode();
  const timeRange = useTimeRange();
  const setTimeRange = useSetTimeRange();
  const selectedCheckId = useSelectedCheckId();
  const selectCheck = useSelectCheck();
  const selectedCheck = useSelectedCheck(); // Derived from cache - reactive to SSE!
  const expandedGroups = useExpandedGroups();
  const toggleGroup = useToggleGroup();
  const sseConnected = useSSEConnected();
  const setSSEConnected = useSetSSEConnected();

  // Modal state from store
  const groupModal = useGroupModal();
  const tagModal = useTagModal();
  const historyModal = useHistoryModal();

  // Mobile detection
  const [isMobile, setIsMobile] = useState(false);

  useEffect(() => {
    const checkMobile = () => setIsMobile(window.innerWidth < 1024);
    checkMobile();
    window.addEventListener('resize', checkMobile);
    return () => window.removeEventListener('resize', checkMobile);
  }, []);

  // Data fetching
  const { data: groupedChecks, isLoading: isLoadingChecks } = useGroupedChecks(timeRange);
  const { data: stats } = useStats(timeRange);

  // SSE for live updates
  useSSE({
    enabled: authData?.isAuthenticated === true,
    onConnect: () => setSSEConnected(true),
    onDisconnect: () => setSSEConnected(false),
  });

  // Mutations
  const deleteCheck = useDeleteCheck();
  const triggerCheck = useTriggerCheck();
  const toggleEnabled = useToggleCheckEnabled();
  const createGroup = useCreateGroup();
  const updateGroup = useUpdateGroup();
  const deleteGroup = useDeleteGroup();
  const createTag = useCreateTag();
  const updateTag = useUpdateTag();
  const deleteTag = useDeleteTag();

  // Handlers
  const handleLogout = async () => {
    try {
      await logout.mutateAsync();
      navigate({ to: '/login' });
    } catch (error) {
      showToast('Failed to logout', 'error');
    }
  };

  const handleDeleteCheck = async (check: Check) => {
    if (!confirm(`Delete monitor "${check.name}"?`)) return;
    try {
      await deleteCheck.mutateAsync(check.id);
      if (selectedCheckId === check.id) {
        selectCheck(null);
      }
      showToast('Monitor deleted', 'success');
    } catch (error) {
      showToast('Failed to delete monitor', 'error');
    }
  };

  const handleTriggerCheck = async (check: Check) => {
    try {
      await triggerCheck.mutateAsync(check.id);
      showToast(`Triggered check for ${check.name}`, 'success');
    } catch (error) {
      showToast('Failed to trigger check', 'error');
    }
  };

  const handleToggleEnabled = async (check: Check) => {
    try {
      await toggleEnabled.mutateAsync(check);
      showToast(
        check.enabled ? 'Monitor paused' : 'Monitor resumed',
        'success'
      );
    } catch (error) {
      showToast('Failed to toggle monitor', 'error');
    }
  };

  const handleSelectCheck = (check: Check) => {
    if (isMobile) {
      navigate({ to: `/monitor/${check.id}` });
    } else {
      selectCheck(check);
    }
  };

  const handleSaveGroup = async (data: { name: string; sort_order: number }) => {
    try {
      if (groupModal.editingGroup) {
        await updateGroup.mutateAsync({ id: groupModal.editingGroup.id, data });
        showToast('Group updated', 'success');
      } else {
        await createGroup.mutateAsync(data);
        showToast('Group created', 'success');
      }
    } catch (error) {
      showToast('Failed to save group', 'error');
      throw error;
    }
  };

  const handleDeleteGroup = async () => {
    if (!groupModal.editingGroup) return;
    try {
      await deleteGroup.mutateAsync(groupModal.editingGroup.id);
      showToast('Group deleted', 'success');
    } catch (error) {
      showToast('Failed to delete group', 'error');
      throw error;
    }
  };

  const handleSaveTag = async (data: { name: string; color: string }) => {
    try {
      if (tagModal.editingTag) {
        await updateTag.mutateAsync({ id: tagModal.editingTag.id, data });
        showToast('Tag updated', 'success');
      } else {
        await createTag.mutateAsync(data);
        showToast('Tag created', 'success');
      }
    } catch (error) {
      showToast('Failed to save tag', 'error');
      throw error;
    }
  };

  const handleDeleteTag = async () => {
    if (!tagModal.editingTag) return;
    try {
      await deleteTag.mutateAsync(tagModal.editingTag.id);
      showToast('Tag deleted', 'success');
    } catch (error) {
      showToast('Failed to delete tag', 'error');
      throw error;
    }
  };

  // Loading state
  if (isAuthLoading) {
    return <LoadingScreen />;
  }

  if (!authData?.isAuthenticated) {
    return <LoadingScreen />;
  }

  return (
    <div className="min-h-screen bg-terminal-bg text-terminal-text font-mono">
      {/* Header */}
      <AppHeader
        darkMode={darkMode}
        onToggleDarkMode={toggleDarkMode}
        onLogout={handleLogout}
        onOpenSettings={() => navigate({ to: '/settings' })}
        sseConnected={sseConnected}
      />

      {/* Main Content */}
      <main className="container mx-auto px-4 py-6">
        {/* Stats Grid */}
        <StatsGrid stats={stats} isLoading={isLoadingChecks} />

        {/* Time Range Selector */}
        <div className="flex flex-wrap gap-4 items-center justify-between mb-6">
          <TimeRangeSelector value={timeRange} onChange={setTimeRange} />

          {/* Action Buttons */}
          <div className="flex gap-2">
            <button
              onClick={() => navigate({ to: '/monitor/new' })}
              className="px-4 py-2 bg-terminal-green text-terminal-bg rounded font-bold text-xs uppercase tracking-wider hover:opacity-90 transition"
            >
              + Monitor
            </button>
            <button
              onClick={() => groupModal.open()}
              className="px-4 py-2 bg-terminal-surface border border-terminal-border text-terminal-text rounded text-xs uppercase tracking-wider hover:border-terminal-muted transition"
            >
              + Group
            </button>
            <button
              onClick={() => tagModal.open()}
              className="px-4 py-2 bg-terminal-surface border border-terminal-border text-terminal-text rounded text-xs uppercase tracking-wider hover:border-terminal-muted transition"
            >
              + Tag
            </button>
          </div>
        </div>

        {/* Two-Pane Layout */}
        <div className="flex gap-6">
          {/* Left: Monitors List */}
          <div
            className={cn(
              'transition-all duration-300',
              selectedCheck && !isMobile ? 'w-1/2 lg:w-2/3' : 'w-full'
            )}
          >
            <MonitorsList
              groups={groupedChecks || []}
              isLoading={isLoadingChecks}
              expandedGroups={expandedGroups}
              onToggleGroup={toggleGroup}
              selectedCheckId={selectedCheckId ?? undefined}
              onSelectCheck={handleSelectCheck}
              onEditCheck={(check) => navigate({ to: `/monitor/${check.id}/edit` })}
              onDeleteCheck={handleDeleteCheck}
              onTriggerCheck={handleTriggerCheck}
              onToggleEnabled={handleToggleEnabled}
              onShowHistory={historyModal.open}
              onEditGroup={groupModal.open}
              onEditTag={tagModal.open}
              timeRange={timeRange}
            />
          </div>

          {/* Right: Details Pane (Desktop only) */}
          {selectedCheck && !isMobile && (
            <div className="w-1/2 lg:w-1/3">
              <DetailsPane
                check={selectedCheck}
                timeRange={timeRange}
                onClose={() => selectCheck(null)}
                onEditCheck={(check) => navigate({ to: `/monitor/${check.id}/edit` })}
                onOpenHistory={historyModal.open}
              />
            </div>
          )}
        </div>
      </main>

      {/* Modals */}
      <GroupModal
        isOpen={groupModal.isOpen}
        onClose={groupModal.close}
        onSave={handleSaveGroup}
        onDelete={groupModal.editingGroup ? handleDeleteGroup : undefined}
        editingGroup={groupModal.editingGroup}
      />

      <TagModal
        isOpen={tagModal.isOpen}
        onClose={tagModal.close}
        onSave={handleSaveTag}
        onDelete={tagModal.editingTag ? handleDeleteTag : undefined}
        editingTag={tagModal.editingTag}
      />

      {historyModal.check && (
        <HistoryModal
          isOpen={historyModal.isOpen}
          onClose={historyModal.close}
          check={historyModal.check}
          timeRange={timeRange}
        />
      )}
    </div>
  );
}
