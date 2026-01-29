import { api } from './client'

export async function getQuests() {
  return api.get('/quests')
}

export async function getMyQuests() {
  return api.get('/me/quests')
}

export async function claimQuestReward(questId) {
  return api.post(`/quests/${questId}/claim`)
}
