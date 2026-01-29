import { api, setToken } from './client'

export async function authenticate() {
  const tg = window.Telegram?.WebApp

  if (!tg?.initData) {
    throw new Error('Telegram WebApp not available')
  }

  const response = await api.post('/auth', {
    init_data: tg.initData,
  })

  if (response.token) {
    setToken(response.token)
  }

  return response
}

export async function getMe() {
  return api.get('/me')
}

export async function getMyProfile() {
  return api.get('/profile')
}

export async function updateBalance(amount, reason) {
  return api.post('/profile/balance', { amount, reason })
}
