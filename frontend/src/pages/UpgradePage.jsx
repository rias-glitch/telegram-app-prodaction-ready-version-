import { useState, useEffect, useCallback } from 'react'
import { Card, Button } from '../components/ui'
import { getUpgradeStatus, upgradeCharacter, claimReferralReward } from '../api/games'
import { getReferralStats } from '../api/referral'

const LEVEL_NAMES = [
  'Beginner',     // 1
  'Rookie',       // 2
  'Amateur',      // 3
  'Skilled',      // 4
  'Expert',       // 5
  'Master',       // 6
  'Champion',     // 7
  'Legend',       // 8
  'Mythic',       // 9
  'Divine',       // 10
]

const LEVEL_COLORS = [
  'from-gray-500 to-gray-600',      // 1
  'from-green-500 to-green-600',    // 2
  'from-blue-500 to-blue-600',      // 3
  'from-purple-500 to-purple-600',  // 4
  'from-pink-500 to-pink-600',      // 5
  'from-orange-500 to-orange-600',  // 6
  'from-red-500 to-red-600',        // 7
  'from-yellow-500 to-amber-500',   // 8
  'from-cyan-400 to-blue-500',      // 9
  'from-violet-500 to-purple-600',  // 10
]

export function UpgradePage({ user, setUser }) {
  const [status, setStatus] = useState(null)
  const [loading, setLoading] = useState(true)
  const [upgrading, setUpgrading] = useState(false)
  const [claiming, setClaiming] = useState(null)
  const [error, setError] = useState(null)

  const loadStatus = useCallback(async () => {
    try {
      setLoading(true)
      const data = await getUpgradeStatus()
      setStatus(data)
    } catch (err) {
      setError(err.message)
    } finally {
      setLoading(false)
    }
  }, [])

  useEffect(() => {
    loadStatus()
  }, [loadStatus])

  const handleUpgrade = async () => {
    if (!status || upgrading) return
    const nextLevel = status.character_level + 1
    if (nextLevel > 10) return

    try {
      setUpgrading(true)
      const result = await upgradeCharacter(nextLevel)
      if (result.success) {
        setStatus(prev => ({
          ...prev,
          character_level: result.new_level,
          gk: result.gk,
          next_level_cost: result.next_level_cost
        }))
        if (setUser) {
          setUser(prev => ({ ...prev, character_level: result.new_level, gk: result.gk }))
        }
      }
    } catch (err) {
      setError(err.message)
    } finally {
      setUpgrading(false)
    }
  }

  const handleClaimReward = async (threshold) => {
    if (claiming) return

    try {
      setClaiming(threshold)
      const result = await claimReferralReward(threshold)
      if (result.success) {
        setStatus(prev => ({
          ...prev,
          gk: result.gk,
          available_rewards: prev.available_rewards.filter(r => r.threshold !== threshold)
        }))
        if (setUser) {
          setUser(prev => ({ ...prev, gk: result.gk }))
        }
      }
    } catch (err) {
      setError(err.message)
    } finally {
      setClaiming(null)
    }
  }

  if (loading) {
    return (
      <div className="flex items-center justify-center py-12">
        <div className="text-center">
          <div className="text-4xl mb-2 animate-pulse-custom">⚡</div>
          <p className="text-white/60">Загрузка...</p>
        </div>
      </div>
    )
  }

  if (error && !status) {
    return (
      <div className="text-center py-12">
        <div className="text-4xl mb-2">❌</div>
        <p className="text-danger">{error}</p>
        <Button onClick={loadStatus} className="mt-4">Повторить</Button>
      </div>
    )
  }

  const level = status?.character_level || 1
  const gk = status?.gk || 0
  const nextCost = status?.next_level_cost || 0
  const canUpgrade = level < 10 && gk >= nextCost

  return (
    <div className="space-y-4 animate-fadeIn">
      <h1 className="text-2xl font-bold">Прокачка</h1>

      {/* GK Balance */}
      <Card className="bg-gradient-to-r from-yellow-500/20 to-orange-500/20 border-yellow-500/30">
        <div className="flex items-center justify-between">
          <div>
            <p className="text-white/60 text-sm">Баланс GK</p>
            <p className="text-2xl font-bold text-yellow-400">{gk.toLocaleString()}</p>
          </div>
          <div className="text-4xl">🔑</div>
        </div>
      </Card>

      {/* Character Level */}
      <Card>
        <div className="text-center">
          <div className={`w-24 h-24 mx-auto rounded-full bg-gradient-to-br ${LEVEL_COLORS[level - 1]} flex items-center justify-center mb-4 shadow-lg`}>
            <span className="text-4xl font-bold text-white">{level}</span>
          </div>
          <h2 className="text-xl font-bold mb-1">{LEVEL_NAMES[level - 1]}</h2>
          <p className="text-white/60">Уровень {level}/10</p>

          {/* Progress bar */}
          <div className="mt-4 h-2 bg-white/10 rounded-full overflow-hidden">
            <div
              className={`h-full bg-gradient-to-r ${LEVEL_COLORS[level - 1]} transition-all duration-500`}
              style={{ width: `${(level / 10) * 100}%` }}
            />
          </div>
        </div>

        {/* Upgrade Button */}
        {level < 10 ? (
          <div className="mt-6">
            <Button
              onClick={handleUpgrade}
              disabled={!canUpgrade || upgrading}
              loading={upgrading}
              className="w-full"
              variant={canUpgrade ? 'primary' : 'secondary'}
            >
              {upgrading ? 'Прокачиваем...' : (
                <>
                  Прокачать до уровня {level + 1}
                  <span className="ml-2 text-yellow-400">{nextCost.toLocaleString()} GK</span>
                </>
              )}
            </Button>
            {!canUpgrade && gk < nextCost && (
              <p className="text-center text-white/40 text-sm mt-2">
                Нужно ещё {(nextCost - gk).toLocaleString()} GK
              </p>
            )}
          </div>
        ) : (
          <div className="mt-6 text-center">
            <span className="text-2xl">🏆</span>
            <p className="text-success font-semibold mt-2">Максимальный уровень достигнут!</p>
          </div>
        )}
      </Card>

      {/* Referral Rewards */}
      <Card>
        <h3 className="font-semibold mb-4 flex items-center gap-2">
          <span>👥</span>
          Награды за рефералов
        </h3>

        <div className="flex items-center justify-between mb-4 p-3 bg-white/5 rounded-xl">
          <div>
            <p className="text-white/60 text-sm">Приглашено</p>
            <p className="text-xl font-bold">{status?.total_referrals || 0}</p>
          </div>
          <div className="text-right">
            <p className="text-white/60 text-sm">Всего заработано</p>
            <p className="text-xl font-bold text-yellow-400">{status?.referral_earnings?.toLocaleString() || 0} coins</p>
          </div>
        </div>

        {/* Reward milestones */}
        <div className="space-y-2">
          {status?.referral_rewards && Object.entries(status.referral_rewards)
            .sort((a, b) => Number(a[0]) - Number(b[0]))
            .map(([threshold, reward]) => {
              const t = Number(threshold)
              const reached = (status?.total_referrals || 0) >= t
              const available = status?.available_rewards?.find(r => r.threshold === t)
              const isClaiming = claiming === t

              return (
                <div
                  key={threshold}
                  className={`flex items-center justify-between p-3 rounded-xl transition-colors ${
                    reached ? 'bg-success/10 border border-success/30' : 'bg-white/5'
                  }`}
                >
                  <div className="flex items-center gap-3">
                    <div className={`w-8 h-8 rounded-full flex items-center justify-center ${
                      reached ? 'bg-success/20 text-success' : 'bg-white/10 text-white/40'
                    }`}>
                      {reached ? '✓' : t}
                    </div>
                    <div>
                      <p className="font-medium">{t} реферал{t > 1 ? 'ов' : ''}</p>
                      <p className="text-sm text-yellow-400">+{reward.toLocaleString()} GK</p>
                    </div>
                  </div>
                  {available ? (
                    <Button
                      size="sm"
                      onClick={() => handleClaimReward(t)}
                      disabled={isClaiming}
                      loading={isClaiming}
                    >
                      Забрать
                    </Button>
                  ) : reached ? (
                    <span className="text-success text-sm">Получено</span>
                  ) : (
                    <span className="text-white/40 text-sm">
                      ещё {t - (status?.total_referrals || 0)}
                    </span>
                  )}
                </div>
              )
            })}
        </div>
      </Card>

      {/* Upgrade costs info */}
      <Card>
        <h3 className="font-semibold mb-4 flex items-center gap-2">
          <span>📊</span>
          Стоимость прокачки
        </h3>
        <div className="grid grid-cols-2 gap-2">
          {status?.costs && Object.entries(status.costs)
            .sort((a, b) => Number(a[0]) - Number(b[0]))
            .map(([lvl, cost]) => {
              const l = Number(lvl)
              const isCompleted = level >= l
              const isCurrent = level === l - 1

              return (
                <div
                  key={lvl}
                  className={`flex items-center justify-between p-2 rounded-lg ${
                    isCompleted ? 'bg-success/10' : isCurrent ? 'bg-primary/10 border border-primary/30' : 'bg-white/5'
                  }`}
                >
                  <span className="text-sm">Lv.{l}</span>
                  <span className={`text-sm font-medium ${isCompleted ? 'text-success line-through' : 'text-yellow-400'}`}>
                    {cost.toLocaleString()} GK
                  </span>
                </div>
              )
            })}
        </div>
      </Card>

      {/* How to earn GK */}
      <Card className="bg-white/5">
        <h3 className="font-semibold mb-3 flex items-center gap-2">
          <span>💡</span>
          Как заработать GK
        </h3>
        <ul className="text-white/60 text-sm space-y-2">
          <li className="flex items-start gap-2">
            <span>👥</span>
            <span>Приглашай друзей и получай GK когда они присоединяются</span>
          </li>
          <li className="flex items-start gap-2">
            <span>💰</span>
            <span>Получай 50% от комиссии за вывод твоих рефералов</span>
          </li>
        </ul>
      </Card>
    </div>
  )
}
