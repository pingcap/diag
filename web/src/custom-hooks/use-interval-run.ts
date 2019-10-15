import { useEffect } from 'react';

export function useIntervalRun(run: () => void, intervalTime: number = 10 * 1000) {
  useEffect(() => {
    let timer: NodeJS.Timeout;
    function intervalRun() {
      clearTimer();
      run();
      timer = setInterval(run, intervalTime);
    }
    function clearTimer() {
      clearInterval(timer);
    }
    function pageVisibilityListener() {
      if (document.visibilityState === 'hidden') {
        clearTimer();
      } else if (document.visibilityState === 'visible') {
        intervalRun();
      }
    }
    intervalRun();
    document.addEventListener('visibilitychange', pageVisibilityListener);

    function cleanup() {
      document.removeEventListener('visibilitychange', pageVisibilityListener);
      clearTimer();
    }
    return cleanup;
  }, []);
}
