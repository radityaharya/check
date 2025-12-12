import { createFileRoute, useNavigate } from '@tanstack/react-router';
import { useCreateCheck } from '@/hooks';
import { useToast, ToastProvider } from '@/components/ui/toast';
import { MonitorForm } from '@/components/forms/MonitorForm';
import type { CheckFormData } from '@/types';

export const Route = createFileRoute('/monitor/new')({
  component: NewMonitorPage,
});

function NewMonitorPage() {
  return (
    <ToastProvider>
      <NewMonitor />
    </ToastProvider>
  );
}

function NewMonitor() {
  const navigate = useNavigate();
  const createCheck = useCreateCheck();
  const { showToast } = useToast();

  const handleSave = async (data: CheckFormData) => {
    try {
      await createCheck.mutateAsync(data);
      showToast('Monitor created', 'success');
      navigate({ to: '/' });
    } catch (error) {
      showToast('Failed to create monitor', 'error');
      throw error;
    }
  };

  const handleCancel = () => {
    navigate({ to: '/' });
  };

  return (
    <div className="min-h-screen bg-terminal-bg text-terminal-text font-mono">
      <div className="bg-terminal-surface border-b border-terminal-border sticky top-0 z-10">
        <div className="container mx-auto px-6 py-4 flex justify-between items-center">
          <h2 className="text-lg font-bold text-terminal-green">$ new monitor</h2>
          <button
            onClick={handleCancel}
            className="text-terminal-muted hover:text-terminal-text text-2xl"
          >
            ×
          </button>
        </div>
      </div>

      <div className="container mx-auto px-6 py-8 max-w-4xl">
        <MonitorForm onSave={handleSave} onCancel={handleCancel} />
      </div>
    </div>
  );
}
