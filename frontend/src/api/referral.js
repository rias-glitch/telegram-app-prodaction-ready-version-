import { api } from './client'

// Get user's referral code
export async function getReferralCode() {
  return api.get('/referral/code')
}

// Get full referral link for sharing
export async function getReferralLink() {
  return api.get('/referral/link')
}

// Get referral statistics
export async function getReferralStats() {
  return api.get('/referral/stats')
}

// Apply a referral code
export async function applyReferralCode(code) {
  return api.post('/referral/apply', { code })
}
