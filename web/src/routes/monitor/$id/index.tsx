import { useMemo } from 'react';
import { createFileRoute, useNavigate } from '@tanstack/react-router';
import { ArrowLeft } from 'lucide-react';
import { useGroupedChecks, useTimeRange, useSelectCheck } from '@/hooks';
import { ToastProvider } from '@/components/ui/toast';
import { DetailsPane } from '@/components/dashboard/DetailsPane';
import { Spinner } from '@/components/ui/spinner';
import { useHistoryModal } from '@/store';

export const Route = createFileRoute('/monitor/$id/')({
  component: MonitorViewPage,
});

function MonitorViewPage() {
  return (
    <ToastProvider>
      <MonitorView />
    </ToastProvider>
  );
}

function MonitorView() {
  const { id } = Route.useParams();
  const navigate = useNavigate();
  const timeRange = useTimeRange();
  const { data: groups, isLoading } = useGroupedChecks(timeRange);
  const historyModal = useHistoryModal();
  const selectCheck = useSelectCheck();

  const check = useMemo(() => {
    if (!groups) return null;
    for (const group of groups) {
      const found = group.checks.find((c) => c.id === Number(id));
      if (found) return found;
    }
    return null;
  }, [groups, id]);

  const handleClose = () => {
    selectCheck(null);
    navigate({ to: '/' });
  };

  const handleEdit = () => {
    navigate({ to: `/monitor/${id}/edit` });
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
        <div className="text-center">
          <div className="text-terminal-red text-lg mb-4">Monitor not found</div>
          <button
            onClick={() => navigate({ to: '/' })}
            className="text-terminal-cyan hover:text-terminal-green"
          >
            ← Back to dashboard
          </button>
        </div>
      </div>
    );
  }

  return (
    <div className="min-h-screen bg-terminal-bg text-terminal-text font-mono">
      <div className="bg-terminal-surface border-b border-terminal-border sticky top-0 z-10">
        <div className="container mx-auto px-4 py-4 flex items-center gap-4">
          <button
            onClick={handleClose}
            className="text-terminal-muted hover:text-terminal-text"
            title="Back to dashboard"
          >
            <ArrowLeft className="w-5 h-5" />
          </button>
          <h2 className="text-lg font-bold text-terminal-green">$ monitor / {check.name}</h2>
        </div>
      </div>

      <div className="container mx-auto max-w-6xl">
        <DetailsPane
          isPage={true}
          check={check}
          timeRange={timeRange}
          onClose={handleClose}
          onEditCheck={handleEdit}
          onOpenHistory={historyModal.open}
        />
      </div>
    </div>
  );
}
