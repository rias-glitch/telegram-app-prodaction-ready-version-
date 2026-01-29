import { api } from './client'

export async function playCoinFlip(bet) {
  return api.post('/game/coinflip', { bet })
}

export async function playRPS(bet, move) {
  return api.post('/game/rps', { bet, move })
}

export async function playMines(bet, pick) {
  return api.post('/game/mines', { bet, pick })
}

export async function spinCase() {
  return api.post('/game/case', {})
}

export async function getMyGames() {
  return api.get('/me/games')
}

export async function getTopUsers() {
  return api.get('/top')
}

// Wheel game
export async function playWheel(bet) {
  return api.post('/game/wheel', { bet })
}

export async function getWheelInfo() {
  return api.get('/game/wheel/info')
}

// Dice game (1-6) with modes
export async function playDice(bet, target, mode) {
  return api.post('/game/dice', { bet, target, mode })
}

export async function getDiceInfo() {
  return api.get('/game/dice/info')
}

// Mines Pro game
export async function startMinesPro(bet, minesCount) {
  return api.post('/game/mines-pro/start', { bet, mines_count: minesCount })
}

export async function revealMinesPro(cell) {
  return api.post('/game/mines-pro/reveal', { cell })
}

export async function cashoutMinesPro() {
  return api.post('/game/mines-pro/cashout', {})
}

export async function getMinesProState() {
  return api.get('/game/mines-pro/state')
}

export async function getMinesProInfo() {
  return api.get('/game/mines-pro/info')
}

// CoinFlip Pro game (multi-round)
export async function startCoinFlipPro(bet) {
  return api.post('/game/coinflip-pro/start', { bet })
}

export async function flipCoinFlipPro() {
  return api.post('/game/coinflip-pro/flip', {})
}

export async function cashoutCoinFlipPro() {
  return api.post('/game/coinflip-pro/cashout', {})
}

export async function getCoinFlipProState() {
  return api.get('/game/coinflip-pro/state')
}

export async function getCoinFlipProInfo() {
  return api.get('/game/coinflip-pro/info')
}

// Leaderboard
export async function getLeaderboard() {
  return api.get('/leaderboard')
}

export async function getMyRank() {
  return api.get('/leaderboard/rank')
}

// Upgrade system
export async function getUpgradeInfo() {
  return api.get('/upgrade/info')
}

export async function getUpgradeStatus() {
  return api.get('/upgrade/status')
}

export async function upgradeCharacter(targetLevel) {
  return api.post('/upgrade/level', { target_level: targetLevel })
}

export async function claimReferralReward(threshold) {
  return api.post('/upgrade/claim-reward', { threshold })
}
