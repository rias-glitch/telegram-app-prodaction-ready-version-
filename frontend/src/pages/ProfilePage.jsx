import { useEffect, useState } from 'react'
import { Card, CardTitle, Button } from '../components/ui'
import { claimQuestReward } from '../api/quests'
import { getReferralLink, getReferralStats } from '../api/referral'

const GAME_ICONS = {
  coinflip: '🪙',
  rps: '✊',
  mines: '💣',
  case: '🎁',
}

const RESULT_COLORS = {
  win: 'text-success',
  lose: 'text-danger',
  draw: 'text-yellow-400',
}

export function ProfilePage({ user, games, stats, quests, fetchProfile }) {
  const [activeTab, setActiveTab] = useState('stats')
  const [claiming, setClaiming] = useState(null)
  const [referralLink, setReferralLink] = useState(null)
  const [referralStats, setReferralStats] = useState(null)
  const [loadingReferral, setLoadingReferral] = useState(false)
  const [copied, setCopied] = useState(false)
  const [dataLoaded, setDataLoaded] = useState(false)

  useEffect(() => {
    // Загружаем данные только один раз при первом рендере
    if (!dataLoaded) {
      setDataLoaded(true)
      fetchProfile()
      loadReferralData()
    }
  }, [dataLoaded, fetchProfile])

  const loadReferralData = async () => {
    // Не загружаем повторно если уже загружено
    if (referralLink !== null) return

    try {
      setLoadingReferral(true)

      // Add timeout to prevent infinite loading
      const timeout = (ms) => new Promise((_, reject) =>
        setTimeout(() => reject(new Error('Request timeout')), ms)
      )

      const [linkData, statsData] = await Promise.race([
        Promise.all([getReferralLink(), getReferralStats()]),
        timeout(10000) // 10 second timeout
      ])

      setReferralLink(linkData)
      setReferralStats(statsData?.stats || null)
    } catch (err) {
      console.error('Failed to load referral data:', err)
      // Set empty state so button shows error state
      setReferralLink({ error: err.message })
    } finally {
      setLoadingReferral(false)
    }
  }

  const handleShare = () => {
    const tg = window.Telegram?.WebApp

    // Check for errors or missing link
    if (referralLink?.error) {
      const msg = `Error: ${referralLink.error}. Try refreshing the page.`
      if (tg?.showAlert) {
        tg.showAlert(msg)
      } else {
        alert(msg)
      }
      return
    }

    if (!referralLink?.link) {
      const msg = loadingReferral
        ? 'Still loading referral link...'
        : 'Failed to load referral link. Try refreshing.'
      if (tg?.showAlert) {
        tg.showAlert(msg)
      } else {
        alert(msg)
      }
      return
    }

    const shareUrl = `https://t.me/share/url?url=${encodeURIComponent(referralLink.link)}&text=${encodeURIComponent('Join me in HardMine!')}`

    // Try openTelegramLink for t.me links
    if (tg?.openTelegramLink) {
      try {
        tg.openTelegramLink(shareUrl)
        return
      } catch (e) {
        console.error('openTelegramLink failed:', e)
      }
    }

    // Fallback to location change
    window.location.href = shareUrl
  }

  const handleCopyLink = async () => {
    if (!referralLink?.link) return

    try {
      await navigator.clipboard.writeText(referralLink.link)
      setCopied(true)
      setTimeout(() => setCopied(false), 2000)
    } catch (err) {
      console.error('Failed to copy:', err)
    }
  }

  const handleClaim = async (userQuestId) => {
    try {
      setClaiming(userQuestId)
      await claimQuestReward(userQuestId)
      fetchProfile()
    } catch (err) {
      console.error('Failed to claim reward:', err)
    } finally {
      setClaiming(null)
    }
  }

  const formatDate = (dateStr) => {
    const date = new Date(dateStr)
    return date.toLocaleDateString('ru-RU', {
      day: 'numeric',
      month: 'short',
      hour: '2-digit',
      minute: '2-digit',
    })
  }

  return (
    <div className="space-y-4 animate-fadeIn">
      {/* Profile header */}
      <Card className="text-center">
        <div className="text-5xl mb-3">👤</div>
        <h2 className="text-xl font-bold">
          {user?.first_name || user?.username || 'Player'}
        </h2>
        {user?.username && (
          <p className="text-white/60">@{user.username}</p>
        )}
        <div className="flex items-center justify-center gap-4 mt-3">
          <div className="flex items-center gap-1">
            <span className="text-xl">💎</span>
            <span className="text-xl font-bold">{user?.gems?.toLocaleString() || 0}</span>
          </div>
          <div className="flex items-center gap-1">
            <span className="text-xl">🪙</span>
            <span className="text-xl font-bold">{user?.coins?.toLocaleString() || 0}</span>
          </div>
        </div>
      </Card>

      {/* Invite Friends Section */}
      <Card className="bg-gradient-to-r from-purple-500/20 to-pink-500/20 border-purple-500/30">
        <div className="flex items-center justify-between mb-3">
          <div className="flex items-center gap-2">
            <span className="text-2xl">🎁</span>
            <div>
              <h3 className="font-bold">Пригласи друзей</h3>
              <p className="text-white/60 text-sm">Получи 500 гемов за друга!</p>
            </div>
          </div>
          {referralStats && (
            <div className="text-right">
              <div className="text-sm text-white/60">Приглашено</div>
              <div className="font-bold text-lg">{referralStats.total_referrals || 0}</div>
            </div>
          )}
        </div>

        <div className="space-y-2">
          <Button
            onClick={handleShare}
            className="w-full bg-gradient-to-r from-purple-500 to-pink-500 hover:from-purple-600 hover:to-pink-600"
          >
            {loadingReferral ? 'Загрузка...' : (referralLink?.link ? 'Поделиться ссылкой' : 'Поделиться')}
          </Button>

          {referralLink?.link && (
            <button
              onClick={handleCopyLink}
              className="w-full py-2 px-4 rounded-xl bg-white/10 hover:bg-white/20 text-sm transition-colors flex items-center justify-center gap-2"
            >
              <span className="truncate max-w-[200px]">{referralLink.code}</span>
              <span>{copied ? '✓' : '📋'}</span>
            </button>
          )}

          {referralStats?.total_earned > 0 && (
            <div className="text-center text-sm text-white/60">
              Всего заработано: <span className="text-success font-semibold">{referralStats.total_earned} 💎</span>
            </div>
          )}
        </div>
      </Card>

      {/* Tabs */}
      <div className="flex gap-2">
        {['stats', 'history', 'quests'].map((tab) => (
          <button
            key={tab}
            onClick={() => setActiveTab(tab)}
            className={`flex-1 py-2 rounded-xl font-medium transition-colors ${
              activeTab === tab
                ? 'bg-primary text-white'
                : 'bg-white/10 text-white/60 hover:bg-white/20'
            }`}
          >
            {tab === 'stats' && 'Статистика'}
            {tab === 'history' && 'История'}
            {tab === 'quests' && 'Задания'}
          </button>
        ))}
      </div>

      {/* Stats tab */}
      {activeTab === 'stats' && stats && (
        <Card>
          <CardTitle className="mb-4">Статистика за месяц</CardTitle>
          <div className="grid grid-cols-2 gap-4">
            <div className="text-center p-3 bg-white/5 rounded-xl">
              <div className="text-2xl font-bold">{stats.total_games || 0}</div>
              <div className="text-white/60 text-sm">Игр</div>
            </div>
            <div className="text-center p-3 bg-white/5 rounded-xl">
              <div className="text-2xl font-bold text-success">{stats.wins || 0}</div>
              <div className="text-white/60 text-sm">Побед</div>
            </div>
            <div className="text-center p-3 bg-white/5 rounded-xl">
              <div className="text-2xl font-bold text-danger">{stats.losses || 0}</div>
              <div className="text-white/60 text-sm">Поражений</div>
            </div>
            <div className="text-center p-3 bg-white/5 rounded-xl">
              <div className="text-2xl font-bold">
                {stats.total_games > 0
                  ? Math.round((stats.wins / stats.total_games) * 100)
                  : 0}%
              </div>
              <div className="text-white/60 text-sm">Винрейт</div>
            </div>
          </div>
          <div className="mt-4 pt-4 border-t border-white/10">
            <div className="flex justify-between">
              <span className="text-white/60">Всего выиграно</span>
              <span className="text-success font-bold">+{stats.total_won?.toLocaleString() || 0}</span>
            </div>
            <div className="flex justify-between mt-2">
              <span className="text-white/60">Всего проиграно</span>
              <span className="text-danger font-bold">-{stats.total_lost?.toLocaleString() || 0}</span>
            </div>
          </div>
        </Card>
      )}

      {/* History tab */}
      {activeTab === 'history' && (
        <div className="space-y-2">
          {(!games || games.length === 0) ? (
            <Card className="text-center py-8">
              <div className="text-4xl mb-2">🎮</div>
              <p className="text-white/60">Пока нет игр</p>
            </Card>
          ) : (
            games.slice(0, 20).map((game) => (
              <Card key={game.id} className="flex items-center gap-3">
                <div className="text-2xl">
                  {GAME_ICONS[game.game_type] || '🎮'}
                </div>
                <div className="flex-1">
                  <div className="font-medium capitalize">{game.game_type}</div>
                  <div className="text-white/40 text-xs">{formatDate(game.created_at)}</div>
                </div>
                <div className="text-right">
                  <div className={`font-bold ${RESULT_COLORS[game.result]}`}>
                    {game.result === 'win' && '+'}
                    {game.result === 'lose' && '-'}
                    {game.win_amount !== 0 ? Math.abs(game.win_amount) : game.bet_amount}
                  </div>
                  <div className="text-white/40 text-xs capitalize">{game.result}</div>
                </div>
              </Card>
            ))
          )}
        </div>
      )}

      {/* Quests tab */}
      {activeTab === 'quests' && (
        <div className="space-y-3">
          {(!quests || quests.length === 0) ? (
            <Card className="text-center py-8">
              <div className="text-4xl mb-2">📋</div>
              <p className="text-white/60">Нет активных заданий</p>
            </Card>
          ) : (
            quests.map((q) => (
              <Card key={q.quest?.id || q.user_quest_id}>
                <div className="flex items-start justify-between mb-2">
                  <div>
                    <div className="font-semibold">{q.quest?.title}</div>
                    <div className="text-white/60 text-sm">{q.quest?.description}</div>
                  </div>
                  <div className="flex flex-col gap-1 items-end">
                    {q.quest?.reward_gems > 0 && (
                      <div className="flex items-center gap-1 bg-purple-500/20 text-purple-400 px-2 py-1 rounded-lg">
                        <span>💎</span>
                        <span className="font-bold">{q.quest.reward_gems}</span>
                      </div>
                    )}
                    {q.quest?.reward_coins > 0 && (
                      <div className="flex items-center gap-1 bg-yellow-500/20 text-yellow-400 px-2 py-1 rounded-lg">
                        <span>🪙</span>
                        <span className="font-bold">{q.quest.reward_coins}</span>
                      </div>
                    )}
                    {q.quest?.reward_gk > 0 && (
                      <div className="flex items-center gap-1 bg-green-500/20 text-green-400 px-2 py-1 rounded-lg">
                        <span>⚡</span>
                        <span className="font-bold">{q.quest.reward_gk} GK</span>
                      </div>
                    )}
                  </div>
                </div>

                {/* Progress bar */}
                <div className="mt-3">
                  <div className="flex justify-between text-sm mb-1">
                    <span className="text-white/60">Прогресс</span>
                    <span>{q.current_count || 0}/{q.target_count}</span>
                  </div>
                  <div className="h-2 bg-white/10 rounded-full overflow-hidden">
                    <div
                      className="h-full bg-gradient-to-r from-primary to-secondary transition-all duration-300"
                      style={{ width: `${Math.min(q.progress || 0, 100)}%` }}
                    />
                  </div>
                </div>

                {/* Claim button */}
                {q.completed && !q.reward_claimed && q.user_quest_id && (
                  <Button
                    onClick={() => handleClaim(q.user_quest_id)}
                    disabled={claiming === q.user_quest_id}
                    size="sm"
                    className="w-full mt-3"
                  >
                    {claiming === q.user_quest_id ? 'Получаем...' : 'Получить награду'}
                  </Button>
                )}

                {q.reward_claimed && (
                  <div className="text-center text-success text-sm mt-3">
                    Награда получена!
                  </div>
                )}
              </Card>
            ))
          )}
        </div>
      )}
    </div>
  )
}
