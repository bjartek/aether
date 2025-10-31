"use client";

import { FlowProvider } from "@onflow/react-sdk";
import { ReactNode } from "react";
import flowJSON from "../../flow.json";

interface FlowProviderWrapperProps {
  children: ReactNode;
}

export function FlowProviderWrapper({ children }: FlowProviderWrapperProps) {
  return (
    <FlowProvider
      config={{
        // Emulator configuration
        accessNodeUrl: "http://localhost:8888",
        discoveryWallet: "http://localhost:8701/fcl/authn",
        flowNetwork: "emulator",

        // App metadata
        appDetailTitle: "Flow Counter App",
        appDetailUrl:
          typeof window !== "undefined" ? window.location.origin : "",
        appDetailIcon: "https://avatars.githubusercontent.com/u/62387156?s=200&v=4",
        appDetailDescription: "A Flow blockchain counter application",

        // Optional configuration
        computeLimit: 999,
      }}
      flowJson={flowJSON}
    >
      {children}
    </FlowProvider>
  );
}
