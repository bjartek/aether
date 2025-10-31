"use client";

import { useEffect } from "react";
import { useFlowQuery } from "@onflow/react-sdk";
import GetCounterCdc from "@/cadence/scripts/GetCounter.cdc";

export function CounterDisplay() {
  const {
    data: counter,
    isLoading,
    error,
    refetch,
  } = useFlowQuery({
    cadence: GetCounterCdc,
    args: () => [],
  });

  // Auto-refresh counter every 2 seconds
  useEffect(() => {
    const interval = setInterval(() => {
      refetch();
    }, 2000);
    return () => clearInterval(interval);
  }, [refetch]);

  if (isLoading) return <div className="counter-display">Loading...</div>;
  if (error)
    return (
      <div className="counter-display error">Error: {error.message}</div>
    );

  return (
    <div className="counter-display">
      {counter !== null ? counter : "Loading..."}
    </div>
  );
}
