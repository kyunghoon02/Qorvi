"use client";

import type { ReactNode } from "react";

import {
  ClerkProvider,
  SignInButton,
  SignUpButton,
  SignedIn,
  SignedOut,
  UserButton,
} from "@clerk/clerk-react";

const publishableKey =
  process.env.NEXT_PUBLIC_CLERK_PUBLISHABLE_KEY?.trim() ?? "";

export function ClerkAuthChrome({ children }: { children: ReactNode }) {
  if (!publishableKey) {
    return <>{children}</>;
  }

  return (
    <ClerkProvider publishableKey={publishableKey}>
      <div className="app-auth-shell">
        <header className="app-auth-header">
          <SignedOut>
            <SignInButton mode="modal">
              <button className="app-auth-button" type="button">
                Log in
              </button>
            </SignInButton>
            <SignUpButton mode="modal">
              <button
                className="app-auth-button app-auth-button-primary"
                type="button"
              >
                Sign up
              </button>
            </SignUpButton>
          </SignedOut>
          <SignedIn>
            <UserButton />
          </SignedIn>
        </header>
        {children}
      </div>
    </ClerkProvider>
  );
}
