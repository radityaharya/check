import { useEffect, useState, createContext, useContext, useCallback } from 'react';
import { cn } from '@/lib/utils';

interface Toast {
  id: string;
  message: string;
  type: 'success' | 'error' | 'info';
}

interface ToastContextValue {
  showToast: (message: string, type?: 'success' | 'error' | 'info') => void;
}

const ToastContext = createContext<ToastContextValue | null>(null);

export function useToast() {
  const context = useContext(ToastContext);
  if (!context) {
    throw new Error('useToast must be used within a ToastProvider');
  }
  return context;
}

export function ToastProvider({ children }: { children: React.ReactNode }) {
  const [toasts, setToasts] = useState<Toast[]>([]);

  const showToast = useCallback(
    (message: string, type: 'success' | 'error' | 'info' = 'info') => {
      const id = Math.random().toString(36).slice(2);
      setToasts((prev) => [...prev, { id, message, type }]);

      setTimeout(() => {
        setToasts((prev) => prev.filter((t) => t.id !== id));
      }, 2500);
    },
    []
  );

  const removeToast = useCallback((id: string) => {
    setToasts((prev) => prev.filter((t) => t.id !== id));
  }, []);

  return (
    <ToastContext.Provider value={{ showToast }}>
      {children}
      <div className="fixed top-4 right-4 z-[60] flex flex-col gap-2">
        {toasts.map((toast) => (
          <ToastItem
            key={toast.id}
            toast={toast}
            onClose={() => removeToast(toast.id)}
          />
        ))}
      </div>
    </ToastContext.Provider>
  );
}

function ToastItem({
  toast,
  onClose,
}: {
  toast: Toast;
  onClose: () => void;
}) {
  const [isVisible, setIsVisible] = useState(false);

  useEffect(() => {
    setIsVisible(true);
  }, []);

  return (
    <div
      className={cn(
        'max-w-sm bg-terminal-surface border border-terminal-border rounded-lg shadow-2xl px-4 py-3 transition-all duration-150',
        isVisible ? 'opacity-100 translate-x-0' : 'opacity-0 translate-x-4'
      )}
    >
      <div className="flex items-start gap-3">
        <div
          className={cn(
            'mt-1 w-2.5 h-2.5 rounded-full flex-shrink-0',
            toast.type === 'success' && 'bg-terminal-green',
            toast.type === 'error' && 'bg-terminal-red',
            toast.type === 'info' && 'bg-terminal-muted'
          )}
        />
        <div className="min-w-0">
          <div className="text-sm text-terminal-text">{toast.message}</div>
        </div>
        <button
          type="button"
          className="text-terminal-muted hover:text-terminal-text leading-none text-lg"
          onClick={onClose}
        >
          ×
        </button>
      </div>
    </div>
  );
}
