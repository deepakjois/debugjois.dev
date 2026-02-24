import { createContext, useContext, useState } from 'react'
import { createRootRoute, Outlet } from '@tanstack/react-router'
import { GoogleLogin, type CredentialResponse } from '@react-oauth/google'

interface AuthState {
  token: string
}

const AuthContext = createContext<AuthState | null>(null)

export function useAuth(): AuthState {
  const ctx = useContext(AuthContext)
  if (!ctx) throw new Error('useAuth must be used within an authenticated route')
  return ctx
}

function RootComponent() {
  const [token, setToken] = useState<string | null>(null)

  if (!token) {
    return (
      <div style={{ padding: '2rem' }}>
        <h1>Logger</h1>
        <p>Sign in to continue.</p>
        <GoogleLogin
          onSuccess={(res: CredentialResponse) => {
            if (res.credential) setToken(res.credential)
          }}
          onError={() => console.error('Google login failed')}
        />
      </div>
    )
  }

  return (
    <AuthContext.Provider value={{ token }}>
      <div style={{ padding: '2rem' }}>
        <header style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center', marginBottom: '1rem' }}>
          <h1 style={{ margin: 0 }}>Logger</h1>
          <button onClick={() => setToken(null)}>Sign out</button>
        </header>
        <Outlet />
      </div>
    </AuthContext.Provider>
  )
}

export const Route = createRootRoute({
  component: RootComponent,
})
