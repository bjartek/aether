import type { Metadata } from "next";
import { FlowProviderWrapper } from "../components/flow-provider-wrapper";
import "./globals.css";

export const metadata: Metadata = {
  title: "Flow Counter App",
  description: "A Flow blockchain counter application using @onflow/react-sdk",
};

export default function RootLayout({
  children,
}: Readonly<{
  children: React.ReactNode;
}>) {
  return (
    <html lang="en">
      <body>
        <FlowProviderWrapper>{children}</FlowProviderWrapper>
      </body>
    </html>
  );
}
