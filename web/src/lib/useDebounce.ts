import { useEffect, useState } from 'react'

// useDebouncedValue returns `value` after it has been stable for `delay` ms,
// so typing in a search box doesn't fire a request per keystroke.
export function useDebouncedValue<T>(value: T, delay = 300): T {
  const [debounced, setDebounced] = useState(value)

  useEffect(() => {
    const t = setTimeout(() => setDebounced(value), delay)
    return () => clearTimeout(t)
  }, [value, delay])

  return debounced
}
