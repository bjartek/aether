"use client";

import { Connect, useFlowCurrentUser } from "@onflow/react-sdk";
import { CounterDisplay } from "../components/counter-display";
import { IncrementCounter } from "../components/increment-counter";

export default function Home() {
  const { user } = useFlowCurrentUser();

  return (
    <div className="app">
      <div className="card">
        <h1>Flow Counter App</h1>

        <div className="wallet-section">
          <Connect />
          {user?.loggedIn && (
            <div className="wallet-info">
              <p>Connected: {user.addr}</p>
            </div>
          )}
        </div>

        <div className="counter-section">
          <h2>Counter Value</h2>
          <CounterDisplay />
          <IncrementCounter />
        </div>

        <div className="info-section">
          <p>This app uses the Flow emulator running locally.</p>
          <p>Make sure the emulator is running on port 8888.</p>
        </div>
      </div>
    </div>
  );
}
