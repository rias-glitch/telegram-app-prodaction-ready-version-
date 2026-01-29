import { api } from './client'

// Config
export async function getTonConfig() {
  return api.get('/ton/config')
}

// Wallet
export async function getWallet() {
  return api.get('/ton/wallet')
}

export async function connectWallet(account, proof) {
  return api.post('/ton/wallet/connect', { account, proof })
}

export async function disconnectWallet() {
  return api.delete('/ton/wallet')
}

// Deposits
export async function getDepositInfo() {
  return api.get('/ton/deposit/info')
}

export async function getDeposits() {
  return api.get('/ton/deposits')
}

// Withdrawals
export async function getWithdrawEstimate(coinsAmount) {
  return api.post('/ton/withdraw/estimate', { coins_amount: coinsAmount })
}

export async function requestWithdrawal(coinsAmount) {
  return api.post('/ton/withdraw', { coins_amount: coinsAmount })
}

export async function getWithdrawals() {
  return api.get('/ton/withdrawals')
}

export async function cancelWithdrawal(withdrawalId) {
  return api.post('/ton/withdraw/cancel', { withdrawal_id: withdrawalId })
}
