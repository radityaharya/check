import { useMemo } from 'react';
import { createFileRoute, useNavigate } from '@tanstack/react-router';
import { useGroupedChecks, useUpdateCheck, useTimeRange } from '@/hooks';
import { useToast, ToastProvider } from '@/components/ui/toast';
import { MonitorForm } from '@/components/forms/MonitorForm';
import { Spinner } from '@/components/ui/spinner';
import type { CheckFormData } from '@/types';

export const Route = createFileRoute('/monitor/$id/edit')({
  component: EditMonitorPage,
});

function EditMonitorPage() {
  return (
    <ToastProvider>
      <EditMonitor />
    </ToastProvider>
  );
}

function EditMonitor() {
  const { id } = Route.useParams();
  const navigate = useNavigate();
  const timeRange = useTimeRange();
  const { data: groups, isLoading } = useGroupedChecks(timeRange);
  const updateCheck = useUpdateCheck();
  const { showToast } = useToast();

  const check = useMemo(() => {
    if (!groups) return null;
    for (const group of groups) {
      const found = group.checks.find((c) => c.id === Number(id));
      if (found) return found;
    }
    return null;
  }, [groups, id]);

  const handleSave = async (data: CheckFormData) => {
    try {
      await updateCheck.mutateAsync({ id: Number(id), data });
      showToast('Monitor updated', 'success');
      navigate({ to: '/' });
    } catch (error) {
      showToast('Failed to update monitor', 'error');
      throw error;
    }
  };

  const handleCancel = () => {
    navigate({ to: '/' });
  };

  if (isLoading) {
    return (
      <div className="min-h-screen bg-terminal-bg text-terminal-text font-mono flex items-center justify-center">
        <Spinner size="lg" />
      </div>
    );
  }

  if (!check) {
    return (
      <div className="min-h-screen bg-terminal-bg text-terminal-text font-mono flex items-center justify-center">
        <div className="text-terminal-red">Monitor not found</div>
      </div>
    );
  }

  return (
    <div className="min-h-screen bg-terminal-bg text-terminal-text font-mono">
      <div className="bg-terminal-surface border-b border-terminal-border sticky top-0 z-10">
        <div className="container mx-auto px-6 py-4 flex justify-between items-center">
          <h2 className="text-lg font-bold text-terminal-green">edit monitor</h2>
          <button
            onClick={handleCancel}
            className="text-terminal-muted hover:text-terminal-text text-2xl"
          >
            ×
          </button>
        </div>
      </div>

      <div className="container mx-auto px-6 py-8 max-w-4xl">
        <MonitorForm editingCheck={check} onSave={handleSave} onCancel={handleCancel} />
      </div>
    </div>
  );
}
