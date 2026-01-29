import { useState, useCallback } from 'react'
import { useNavigate } from 'react-router-dom'
import { Card, CardTitle, Button } from '../components/ui'
import { api } from '../api/client'
import { CoinFlipGame } from '../components/games/CoinFlipGame'
import { CoinFlipProGame } from '../components/games/CoinFlipProGame'
import { WheelGame } from '../components/games/WheelGame'
import { DiceGame } from '../components/games/DiceGame'
import { MinesProGame } from '../components/games/MinesProGame'

const games = [
  {
    id: 'coinflip-pro',
    icon: '🪙',
    title: 'Монетка',
    description: '10 раундов, дойди до 100x!',
    multiplier: 'x100',
    navigateTo: null,
    isHot: true,
  },
  {
    id: 'rps',
    icon: '✊',
    title: 'Камень Ножницы Бумага',
    description: 'PvE & PvP режимы',
    multiplier: 'x2',
    navigateTo: '/rps',
    hasPvP: true,
  },
  {
    id: 'mines',
    icon: '💣',
    title: 'Mines',
    description: 'PvE & PvP режимы',
    multiplier: 'x2',
    navigateTo: '/mines',
    hasPvP: true,
  },
  {
    id: 'wheel',
    icon: '🎡',
    title: 'Колесо удачи',
    description: 'Крути и получай до 10x',
    multiplier: 'x10',
    navigateTo: null,
  },
  {
    id: 'dice',
    icon: '🎲',
    title: 'Dice',
    description: 'Предсказывай исход и выигрывай!',
    multiplier: 'x100',
    navigateTo: null,
  },
  {
    id: 'mines-pro',
    icon: '💎',
    title: 'Mines v2.0',
    description: 'Многораундовые mines',
    multiplier: 'x24',
    navigateTo: null,
  },
]

export function GamesPage({ user, setUser, addGems }) {
  const navigate = useNavigate()
  const [activeGame, setActiveGame] = useState(null)
  const [claimingBonus, setClaimingBonus] = useState(false)

  const handleClaimBonus = useCallback(async () => {
    if (claimingBonus) return
    setClaimingBonus(true)
    try {
      await api.post('/profile/bonus')
      // Refresh user data
      const profile = await api.get('/profile')
      if (setUser) {
        setUser(prev => ({ ...prev, gems: profile.gems }))
      }
    } catch (err) {
      console.error('Failed to claim bonus:', err)
    } finally {
      setClaimingBonus(false)
    }
  }, [claimingBonus, setUser])

  const handleGameClick = game => {
    if (game.navigateTo) {
      navigate(game.navigateTo)
    } else {
      setActiveGame(game.id)
    }
  }

  const handleClose = () => {
    setActiveGame(null)
  }

  const handleGameResult = newGems => {
    if (setUser && newGems !== undefined) {
      setUser(prev => ({ ...prev, gems: newGems }))
    }
  }

  return (
    <div className='space-y-4 animate-fadeIn'>
      <h1 className='text-2xl font-bold'>Игры</h1>

      {user && user.gems < 100 && (
        <Card className='bg-gradient-to-r from-yellow-500/20 to-orange-500/20 border-yellow-500/30'>
          <div className='flex items-center justify-between'>
            <div>
              <p className='font-semibold text-yellow-400'>Мало средств!</p>
              <p className='text-sm text-white/60'>Получи 10,000 бесплатных гемов</p>
            </div>
            <Button
              onClick={handleClaimBonus}
              disabled={claimingBonus}
              className='bg-yellow-500 hover:bg-yellow-600 text-black font-bold'
            >
              {claimingBonus ? 'Получаем...' : 'Получить бонус'}
            </Button>
          </div>
        </Card>
      )}

      <div className='grid gap-3'>
        {games.map((game, index) => (
          <Card
            key={game.id}
            onClick={() => handleGameClick(game)}
            className={`flex items-center gap-4 group ${
              game.isHot ? 'border-orange-500/20' : ''
            }`}
            style={{ animationDelay: `${index * 50}ms` }}
          >
            {/* Icon with glow effect */}
            <div className='relative'>
              <div className='text-4xl group-hover:scale-110 transition-transform duration-300'>
                {game.icon}
              </div>
              {game.isHot && (
                <div className='absolute inset-0 bg-orange-500/20 blur-xl rounded-full' />
              )}
            </div>

            {/* Info */}
            <div className='flex-1 min-w-0'>
              <div className='flex items-center gap-2 flex-wrap'>
                <CardTitle className='group-hover:text-primary transition-colors'>
                  {game.title}
                </CardTitle>
                {game.isHot && (
                  <span className='text-[10px] bg-gradient-to-r from-orange-500 to-red-500 text-white px-2 py-0.5 rounded-full font-bold uppercase tracking-wide shadow-lg shadow-orange-500/20'>
                    Hot
                  </span>
                )}
                {game.hasPvP && (
                  <span className='text-[10px] bg-primary/20 text-primary px-2 py-0.5 rounded-full font-medium border border-primary/20'>
                    PvP
                  </span>
                )}
              </div>
              <p className='text-white/50 text-sm mt-0.5'>{game.description}</p>
            </div>

            {/* Multiplier badge */}
            <div className='flex flex-col items-end'>
              <div className='bg-gradient-to-r from-primary/20 to-secondary/20 text-primary px-3 py-1.5 rounded-xl font-bold text-sm border border-primary/10'>
                {game.multiplier}
              </div>
            </div>
          </Card>
        ))}
      </div>

      {/* PvE Games */}
      {activeGame === 'coinflip-pro' && (
        <CoinFlipProGame
          user={user}
          onClose={handleClose}
          onResult={handleGameResult}
        />
      )}

      {activeGame === 'coinflip' && (
        <CoinFlipGame
          user={user}
          onClose={handleClose}
          onResult={handleGameResult}
        />
      )}

      {activeGame === 'wheel' && (
        <WheelGame
          user={user}
          onClose={handleClose}
          onResult={handleGameResult}
        />
      )}

      {activeGame === 'dice' && (
        <DiceGame
          user={user}
          onClose={handleClose}
          onResult={handleGameResult}
        />
      )}

      {activeGame === 'mines-pro' && (
        <MinesProGame
          user={user}
          onClose={handleClose}
          onResult={handleGameResult}
        />
      )}
    </div>
  )
}
