"use client";

import type { ComponentType, ReactNode } from "react";

import * as ClerkReact from "@clerk/clerk-react";
import * as ClerkNext from "@clerk/nextjs";

type ButtonWrapperProps = {
  children?: ReactNode;
  mode?: "modal" | "redirect";
};

type VisibilityWrapperProps = {
  children?: ReactNode;
};

function renderChildren({ children }: VisibilityWrapperProps) {
  return <>{children ?? null}</>;
}

function renderNothing() {
  return null;
}

const SignInButton =
  ((ClerkNext as Record<string, unknown>).SignInButton ??
    (ClerkReact as Record<string, unknown>).SignInButton ??
    renderChildren) as ComponentType<ButtonWrapperProps>;

const SignUpButton =
  ((ClerkNext as Record<string, unknown>).SignUpButton ??
    (ClerkReact as Record<string, unknown>).SignUpButton ??
    renderChildren) as ComponentType<ButtonWrapperProps>;

const SignedIn =
  ((ClerkNext as Record<string, unknown>).SignedIn ??
    (ClerkReact as Record<string, unknown>).SignedIn ??
    renderNothing) as ComponentType<VisibilityWrapperProps>;

const SignedOut =
  ((ClerkNext as Record<string, unknown>).SignedOut ??
    (ClerkReact as Record<string, unknown>).SignedOut ??
    renderChildren) as ComponentType<VisibilityWrapperProps>;

const UserButton =
  ((ClerkNext as Record<string, unknown>).UserButton ??
    (ClerkReact as Record<string, unknown>).UserButton ??
    renderNothing) as ComponentType;

const clerkPublishableKey =
  process.env.NEXT_PUBLIC_CLERK_PUBLISHABLE_KEY?.trim() ?? "";

export function AuthButtons() {
  if (!clerkPublishableKey) {
    return null;
  }

  return (
    <div className="app-auth-container">
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
    </div>
  );
}
