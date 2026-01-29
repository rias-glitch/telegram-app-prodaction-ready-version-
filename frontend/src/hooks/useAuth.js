import { useState, useEffect, useCallback } from 'react'
import { authenticate, getMe } from '../api/auth'
import { getToken } from '../api/client'

export function useAuth() {
  const [user, setUser] = useState(null)
  const [loading, setLoading] = useState(true)
  const [error, setError] = useState(null)

  const login = useCallback(async () => {
    try {
      setLoading(true)
      setError(null)

      // Check if already have token
      const existingToken = getToken()
      if (existingToken) {
        try {
          const userData = await getMe()
          setUser(userData)
          return userData
        } catch {
          // Token expired, re-authenticate
        }
      }

      // Authenticate with Telegram
      const response = await authenticate()
      setUser(response.user)
      return response.user
    } catch (err) {
      setError(err.message)
      throw err
    } finally {
      setLoading(false)
    }
  }, [])

  useEffect(() => {
    login().catch(() => {})
  }, [login])

  return { user, loading, error, login, setUser }
}
