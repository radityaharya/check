import { useEffect, useState } from 'react';
import { Button } from '@/components/ui/button';
import { Modal, ModalHeader, ModalFooter } from '@/components/ui/modal';
import { usePwa } from '@/hooks/use-pwa';
import { RefreshCw, Download, X } from 'lucide-react';

export function PwaUpdateNotification() {
  const { isUpdateAvailable, skipWaiting, canInstall, installApp } = usePwa();
  const [showUpdateModal, setShowUpdateModal] = useState(false);
  const [showInstallPrompt, setShowInstallPrompt] = useState(false);

  useEffect(() => {
    if (isUpdateAvailable) {
      setShowUpdateModal(true);
    }
  }, [isUpdateAvailable]);

  useEffect(() => {
    if (canInstall) {
      const timer = setTimeout(() => {
        setShowInstallPrompt(true);
      }, 3000);
      return () => clearTimeout(timer);
    }
  }, [canInstall]);

  const handleUpdate = async () => {
    await skipWaiting();
    setShowUpdateModal(false);
  };

  const handleInstall = async () => {
    await installApp();
    setShowInstallPrompt(false);
  };

  return (
    <>
      <Modal isOpen={showUpdateModal} onClose={() => setShowUpdateModal(false)} size="sm">
        <ModalHeader title="Update Available" onClose={() => setShowUpdateModal(false)}>
          <div className="flex items-center gap-2">
            <RefreshCw className="size-5" />
            <span>Update Available</span>
          </div>
        </ModalHeader>
        <div className="p-6">
          <p className="text-terminal-text mb-4">
            A new version of the app is available. Would you like to update now?
          </p>
          <p className="text-sm text-terminal-muted">
            The app will reload after updating.
          </p>
        </div>
        <ModalFooter>
          <Button variant="outline" onClick={() => setShowUpdateModal(false)}>
            Later
          </Button>
          <Button onClick={handleUpdate} className="gap-2">
            <RefreshCw className="size-4" />
            Update Now
          </Button>
        </ModalFooter>
      </Modal>

      {canInstall && (
        <div
          className={`fixed bottom-4 right-4 bg-terminal-surface border border-terminal-border rounded-sm shadow-2xl p-4 max-w-sm z-50 transition-all duration-300 ${
            showInstallPrompt
              ? 'opacity-100 translate-y-0'
              : 'opacity-0 translate-y-4 pointer-events-none'
          }`}
        >
          <div className="flex items-start gap-3">
            <Download className="size-5 text-terminal-green flex-shrink-0 mt-0.5" />
            <div className="flex-1 min-w-0">
              <h3 className="text-sm font-semibold text-terminal-text mb-1">
                Install App
              </h3>
              <p className="text-xs text-terminal-muted mb-3">
                Install this app on your device for a better experience.
              </p>
              <div className="flex gap-2">
                <Button
                  size="sm"
                  variant="outline"
                  onClick={() => setShowInstallPrompt(false)}
                  className="flex-1"
                >
                  <X className="size-3" />
                </Button>
                <Button size="sm" onClick={handleInstall} className="flex-1 gap-1">
                  <Download className="size-3" />
                  Install
                </Button>
              </div>
            </div>
          </div>
        </div>
      )}
    </>
  );
}

