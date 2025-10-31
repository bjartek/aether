"use client";

import { useFlowMutate } from "@onflow/react-sdk";
import IncrementCounterCdc from "@/cadence/transactions/IncrementCounter.cdc";

export function IncrementCounter() {
  const { mutate, isPending, isSuccess, error } = useFlowMutate();

  const handleIncrement = () => {
    mutate({
      cadence: IncrementCounterCdc,
      args: () => [],
    });
  };

  return (
    <div>
      <button
        onClick={handleIncrement}
        disabled={isPending}
        className="btn btn-primary btn-large"
      >
        {isPending ? "Processing..." : "Increment Counter"}
      </button>

      {isSuccess && (
        <div className="status-message">Transaction successful!</div>
      )}

      {error && (
        <div className="status-message error">Error: {error.message}</div>
      )}
    </div>
  );
}
