import { useCallback, useEffect, useRef, useState } from "react";

export function useNotifications() {
  const [message, setMessage] = useState("");
  const timers = useRef<number[]>([]);

  useEffect(() => () => timers.current.forEach((timer) => window.clearTimeout(timer)), []);

  const notify = useCallback((text: string) => {
    setMessage(text);
    const timer = window.setTimeout(() => {
      setMessage("");
      timers.current = timers.current.filter((item) => item !== timer);
    }, 2600);
    timers.current.push(timer);
  }, []);

  return { message, notify, setMessage };
}
