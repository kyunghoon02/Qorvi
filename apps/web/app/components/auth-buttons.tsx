"use client";

import {
  SignInButton,
  SignUpButton,
  SignedIn,
  SignedOut,
  UserButton,
} from "@clerk/nextjs";

export function AuthButtons() {
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
