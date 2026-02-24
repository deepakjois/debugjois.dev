import { createFileRoute } from '@tanstack/react-router'
import { useQuery } from '@tanstack/react-query'
import { useAuth } from './__root'

const API_URL = import.meta.env.VITE_SITE_BACKEND_URL

export const Route = createFileRoute('/')({
  component: Index,
})

function Index() {
  const { token } = useAuth()

  const { data, isFetching, error, refetch } = useQuery({
    queryKey: ['health'],
    queryFn: async () => {
      const res = await fetch(`${API_URL}/health`, {
        headers: { Authorization: `Bearer ${token}` },
      })
      if (!res.ok) throw new Error(`HTTP ${res.status}`)
      return res.json() as Promise<{ status: string; email?: string }>
    },
    enabled: false,
  })

  return (
    <div>
      <button onClick={() => refetch()} disabled={isFetching}>
        {isFetching ? 'Checking…' : 'Check Health'}
      </button>
      {data && <p>Backend status: {data.status}{data.email && ` (user: ${data.email})`}</p>}
      {error && <p style={{ color: 'red' }}>Error: {error.message}</p>}
    </div>
  )
}
