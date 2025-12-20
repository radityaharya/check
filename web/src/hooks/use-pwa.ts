import { useEffect, useState, useCallback } from 'react';

interface BeforeInstallPromptEvent extends Event {
  prompt: () => Promise<void>;
  userChoice: Promise<{ outcome: 'accepted' | 'dismissed' }>;
}

interface UsePwaReturn {
  isUpdateAvailable: boolean;
  isOfflineReady: boolean;
  isInstalled: boolean;
  updateServiceWorker: () => Promise<void>;
  skipWaiting: () => Promise<void>;
  installApp: () => Promise<void>;
  canInstall: boolean;
}

export function usePwa(): UsePwaReturn {
  const [isUpdateAvailable, setIsUpdateAvailable] = useState(false);
  const [isOfflineReady, setIsOfflineReady] = useState(false);
  const [isInstalled, setIsInstalled] = useState(false);
  const [canInstall, setCanInstall] = useState(false);
  const [deferredPrompt, setDeferredPrompt] = useState<BeforeInstallPromptEvent | null>(null);
  const [swRegistration, setSwRegistration] = useState<ServiceWorkerRegistration | null>(null);

  useEffect(() => {
    if (typeof window === 'undefined' || !('serviceWorker' in navigator)) {
      return;
    }

    const checkInstalled = () => {
      if (window.matchMedia('(display-mode: standalone)').matches) {
        setIsInstalled(true);
      }
    };

    checkInstalled();

    const handleBeforeInstallPrompt = (e: Event) => {
      e.preventDefault();
      setDeferredPrompt(e as BeforeInstallPromptEvent);
      setCanInstall(true);
    };

    const handleAppInstalled = () => {
      setIsInstalled(true);
      setCanInstall(false);
      setDeferredPrompt(null);
    };

    window.addEventListener('beforeinstallprompt', handleBeforeInstallPrompt);
    window.addEventListener('appinstalled', handleAppInstalled);

    function checkForUpdates(reg: ServiceWorkerRegistration) {
      const checkWaiting = () => {
        if (reg.waiting && navigator.serviceWorker.controller) {
          setIsUpdateAvailable(true);
        }
      };

      checkWaiting();

      reg.addEventListener('updatefound', () => {
        const newWorker = reg.installing;
        if (!newWorker) return;

        newWorker.addEventListener('statechange', () => {
          if (newWorker.state === 'installed' && navigator.serviceWorker.controller) {
            setIsUpdateAvailable(true);
          }
        });
      });
    }

    navigator.serviceWorker
      .getRegistration()
      .then((reg) => {
        if (reg) {
          setSwRegistration(reg);
          setIsOfflineReady(true);
          checkForUpdates(reg);
        }
      })
      .catch((err) => {
        console.error('Service Worker registration check failed:', err);
      });

    navigator.serviceWorker.addEventListener('controllerchange', () => {
      window.location.reload();
    });

    return () => {
      window.removeEventListener('beforeinstallprompt', handleBeforeInstallPrompt);
      window.removeEventListener('appinstalled', handleAppInstalled);
    };
  }, []);

  const updateServiceWorker = useCallback(async () => {
    if (!swRegistration) {
      const reg = await navigator.serviceWorker.getRegistration();
      if (reg) {
        setSwRegistration(reg);
        await reg.update();
        if (reg.waiting) {
          setIsUpdateAvailable(true);
        }
      }
      return;
    }

    try {
      await swRegistration.update();
      if (swRegistration.waiting) {
        setIsUpdateAvailable(true);
      }
    } catch (err) {
      console.error('Failed to update service worker:', err);
    }
  }, [swRegistration]);

  const skipWaiting = useCallback(async () => {
    const reg = swRegistration || (await navigator.serviceWorker.getRegistration());
    if (!reg?.waiting) return;

    try {
      reg.waiting.postMessage({ type: 'SKIP_WAITING' });
      setIsUpdateAvailable(false);
    } catch (err) {
      console.error('Failed to skip waiting:', err);
    }
  }, [swRegistration]);

  const installApp = useCallback(async () => {
    if (!deferredPrompt) return;

    try {
      await deferredPrompt.prompt();
      const { outcome } = await deferredPrompt.userChoice;
      if (outcome === 'accepted') {
        setIsInstalled(true);
      }
      setDeferredPrompt(null);
      setCanInstall(false);
    } catch (err) {
      console.error('Failed to install app:', err);
    }
  }, [deferredPrompt]);

  return {
    isUpdateAvailable,
    isOfflineReady,
    isInstalled,
    updateServiceWorker,
    skipWaiting,
    installApp,
    canInstall,
  };
}

