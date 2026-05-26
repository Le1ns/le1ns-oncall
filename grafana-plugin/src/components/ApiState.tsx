import React, { useEffect, useState } from 'react';
import { Alert, LoadingPlaceholder } from '@grafana/ui';

interface Props<T> {
  load: () => Promise<T>;
  children: (data: T) => React.ReactNode;
}

export function ApiState<T>({ load, children }: Props<T>) {
  const [data, setData] = useState<T | null>(null);
  const [error, setError] = useState<Error | null>(null);

  useEffect(() => {
    let mounted = true;
    load()
      .then((value) => {
        if (mounted) {
          setData(value);
        }
      })
      .catch((err) => {
        if (mounted) {
          setError(err instanceof Error ? err : new Error(String(err)));
        }
      });

    return () => {
      mounted = false;
    };
  }, [load]);

  if (error) {
    return (
      <Alert title="OnCall API is unavailable" severity="error">
        {error.message}
      </Alert>
    );
  }

  if (!data) {
    return <LoadingPlaceholder text="Loading OnCall data" />;
  }

  return <>{children(data)}</>;
}
