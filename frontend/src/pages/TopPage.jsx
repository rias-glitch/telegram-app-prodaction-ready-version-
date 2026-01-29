import { useState, useEffect } from 'react'
import { Card } from '../components/ui'
import { getLeaderboard, getMyRank } from '../api/games'

const MEDALS = ['🥇', '🥈', '🥉']

export function TopPage({ user }) {
  const [leaderboard, setLeaderboard] = useState([])
  const [myRank, setMyRank] = useState(null)
  const [loading, setLoading] = useState(true)
  const [error, setError] = useState(null)

  useEffect(() => {
    loadData()
  }, [])

  const loadData = async () => {
    try {
      setLoading(true)
      const [leaderboardRes, rankRes] = await Promise.all([
        getLeaderboard(),
        user ? getMyRank() : Promise.resolve(null)
      ])
      setLeaderboard(leaderboardRes?.leaderboard || [])
      setMyRank(rankRes)
    } catch (err) {
      setError(err.message)
    } finally {
      setLoading(false)
    }
  }

  if (loading) {
    return (
      <div className="flex items-center justify-center py-12">
        <div className="text-center">
          <div className="text-4xl mb-2 animate-pulse-custom">🏆</div>
          <p className="text-white/60">Загрузка рейтинга...</p>
        </div>
      </div>
    )
  }

  if (error) {
    return (
      <div className="text-center py-12">
        <div className="text-4xl mb-2">❌</div>
        <p className="text-danger">{error}</p>
      </div>
    )
  }

  return (
    <div className="space-y-4 animate-fadeIn pb-20">
      <div className="flex items-center justify-between">
        <h1 className="text-2xl font-bold">Топ 100</h1>
        <span className="text-white/60 text-sm">За месяц</span>
      </div>

      {leaderboard.length === 0 ? (
        <Card className="text-center py-8">
          <div className="text-4xl mb-2">🏆</div>
          <p className="text-white/60">Нет игроков за этот месяц</p>
        </Card>
      ) : (
        <div className="space-y-2">
          {leaderboard.map((entry, index) => (
            <Card
              key={entry.user?.id || index}
              className={`flex items-center gap-3 ${
                index < 3 ? 'border border-primary/30' : ''
              }`}
            >
              {/* Rank */}
              <div className="w-10 text-center">
                {index < 3 ? (
                  <span className="text-2xl">{MEDALS[index]}</span>
                ) : (
                  <span className="text-white/40 font-bold">#{entry.rank || index + 1}</span>
                )}
              </div>

              {/* User info */}
              <div className="flex-1 min-w-0">
                <div className="font-semibold truncate">
                  {entry.user?.first_name || entry.user?.username || `User`}
                </div>
                {entry.user?.username && (
                  <div className="text-white/40 text-sm truncate">@{entry.user.username}</div>
                )}
              </div>

              {/* Wins count */}
              <div className="text-right">
                <div className="font-bold text-success flex items-center gap-1">
                  <span>{entry.wins_count?.toLocaleString()}</span>
                  <span className="text-sm">🏆</span>
                </div>
                <div className="text-white/40 text-xs">побед за месяц</div>
              </div>
            </Card>
          ))}
        </div>
      )}

      {/* Fixed user rank at bottom */}
      {myRank && (
        <div className="fixed bottom-20 left-0 right-0 px-4">
          <Card className="bg-gradient-to-r from-primary/20 to-secondary/20 border border-primary/30">
            <div className="flex items-center justify-between">
              <div className="flex items-center gap-3">
                <div className="w-10 h-10 rounded-full bg-primary/20 flex items-center justify-center">
                  <span className="text-lg">👤</span>
                </div>
                <div>
                  <div className="font-semibold">Вы</div>
                  <div className="text-white/60 text-sm">
                    {myRank.wins_count > 0
                      ? `${myRank.wins_count.toLocaleString()} побед`
                      : 'Пока нет побед'}
                  </div>
                </div>
              </div>
              <div className="text-right">
                <div className="text-2xl font-bold text-primary">
                  #{myRank.rank || '—'}
                </div>
                <div className="text-white/40 text-xs">ваш ранг</div>
              </div>
            </div>
          </Card>
        </div>
      )}
    </div>
  )
}
