import { useEffect, useRef } from 'react';
import { cn } from '@/lib/utils';

interface ModalProps {
  isOpen: boolean;
  onClose: () => void;
  children: React.ReactNode;
  className?: string;
  fullScreen?: boolean;
  size?: 'sm' | 'md' | 'lg' | 'xl';
}

const sizeClasses = {
  sm: 'max-w-md',
  md: 'max-w-lg',
  lg: 'max-w-2xl',
  xl: 'max-w-4xl',
};

export function Modal({
  isOpen,
  onClose,
  children,
  className,
  fullScreen = false,
  size = 'md',
}: ModalProps) {
  const overlayRef = useRef<HTMLDivElement>(null);

  useEffect(() => {
    const handleEscape = (e: KeyboardEvent) => {
      if (e.key === 'Escape') onClose();
    };

    if (isOpen) {
      document.addEventListener('keydown', handleEscape);
      document.body.style.overflow = 'hidden';
    }

    return () => {
      document.removeEventListener('keydown', handleEscape);
      document.body.style.overflow = '';
    };
  }, [isOpen, onClose]);

  if (!isOpen) return null;

  if (fullScreen) {
    return (
      <div className="fixed inset-0 bg-terminal-bg z-50 overflow-y-auto">
        {children}
      </div>
    );
  }

  return (
    <div
      ref={overlayRef}
      className="fixed inset-0 bg-black/80 backdrop-blur-sm flex items-center justify-center z-50 p-4"
      onClick={(e) => {
        if (e.target === overlayRef.current) onClose();
      }}
    >
      <div
        className={cn(
          'bg-terminal-surface border border-terminal-border rounded-sm shadow-2xl w-full',
          sizeClasses[size],
          className
        )}
        onClick={(e) => e.stopPropagation()}
      >
        {children}
      </div>
    </div>
  );
}

interface ModalHeaderProps {
  title?: string;
  children?: React.ReactNode;
  onClose: () => void;
  sticky?: boolean;
}

export function ModalHeader({ title, children, onClose, sticky = false }: ModalHeaderProps) {
  return (
    <div
      className={cn(
        'p-6 border-b border-terminal-border flex justify-between items-center',
        sticky && 'sticky top-0 bg-terminal-surface z-10'
      )}
    >
      <h3 className="text-lg font-bold text-terminal-green">{children || title}</h3>
      <button
        onClick={onClose}
        className="text-terminal-muted hover:text-terminal-text text-2xl leading-none"
      >
        ×
      </button>
    </div>
  );
}

interface ModalFooterProps {
  children: React.ReactNode;
  className?: string;
}

export function ModalFooter({ children, className }: ModalFooterProps) {
  return (
    <div
      className={cn(
        'p-6 pt-4 flex gap-4 border-t border-terminal-border',
        className
      )}
    >
      {children}
    </div>
  );
}
