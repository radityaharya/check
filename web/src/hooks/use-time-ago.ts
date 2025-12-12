import { useState, useEffect } from 'react';

/**
 * Hook that triggers a re-render at a specified interval.
 * Useful for keeping relative time displays (like "5 seconds ago") up to date.
 */
export function useTimeAgoTick(intervalMs: number = 1000) {
  const [, setTick] = useState(0);

  useEffect(() => {
    const timer = setInterval(() => {
      setTick((t) => t + 1);
    }, intervalMs);

    return () => clearInterval(timer);
  }, [intervalMs]);
}

/**
 * Returns a formatted "time ago" string that updates live.
 * Use useTimeAgoTick() at the component level to trigger re-renders.
 */
export function formatTimeAgo(date: string | Date | null | undefined): string {
  if (!date) return 'never';
  
  const now = new Date();
  const then = new Date(date);
  const diffMs = now.getTime() - then.getTime();
  
  if (diffMs < 0) return 'just now';
  
  const seconds = Math.floor(diffMs / 1000);
  const minutes = Math.floor(seconds / 60);
  const hours = Math.floor(minutes / 60);
  const days = Math.floor(hours / 24);

  if (seconds < 60) return `${seconds}s ago`;
  if (minutes < 60) return `${minutes}m ago`;
  if (hours < 24) return `${hours}h ago`;
  return `${days}d ago`;
}
