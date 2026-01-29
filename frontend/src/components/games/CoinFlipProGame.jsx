import { useState, useEffect } from 'react'
import { Modal } from '../ui/Overlay'
import { Button } from '../ui'
import { Input } from '../ui'
import {
  startCoinFlipPro,
  flipCoinFlipPro,
  cashoutCoinFlipPro,
  getCoinFlipProState,
} from '../../api/games'

const BET_PRESETS = [10, 50, 100, 500]

// Multipliers for each round
const MULTIPLIERS = [
  1.0, 1.5, 2.0, 3.0, 5.0, 8.0, 12.0, 20.0, 35.0, 60.0, 100.0,
]

export function CoinFlipProGame({ user, onClose, onResult }) {
  const [bet, setBet] = useState(100)
  const [loading, setLoading] = useState(false)
  const [gameState, setGameState] = useState(null)
  const [flipping, setFlipping] = useState(false)
  const [lastFlipWin, setLastFlipWin] = useState(null)

  // Check for active game on mount
  useEffect(() => {
    checkActiveGame()
  }, [])

  const checkActiveGame = async () => {
    try {
      const state = await getCoinFlipProState()
      if (state.active) {
        setGameState(state)
      }
    } catch (err) {
      console.error('Failed to get game state:', err)
    }
  }

  const handleStart = async () => {
    if (bet <= 0 || bet > (user?.gems || 0)) return

    try {
      setLoading(true)
      const state = await startCoinFlipPro(bet)
      setGameState(state)
      setLastFlipWin(null)
    } catch (err) {
      console.error('Failed to start game:', err)
    } finally {
      setLoading(false)
    }
  }

  const handleFlip = async () => {
    if (!gameState || gameState.status !== 'active') return

    try {
      setLoading(true)
      setFlipping(true)

      const result = await flipCoinFlipPro()

      // Animation delay
      setTimeout(() => {
        setFlipping(false)
        setLastFlipWin(result.flip_win)
        setGameState(result)

        if (result.status !== 'active') {
          onResult(result.gems)
        }
      }, 1000)
    } catch (err) {
      setFlipping(false)
      console.error('Failed to flip:', err)
    } finally {
      setLoading(false)
    }
  }

  const handleCashOut = async () => {
    if (!gameState || gameState.status !== 'active') return

    try {
      setLoading(true)
      const result = await cashoutCoinFlipPro()
      setGameState(result)
      onResult(result.gems)
    } catch (err) {
      console.error('Failed to cash out:', err)
    } finally {
      setLoading(false)
    }
  }

  const handlePlayAgain = () => {
    setGameState(null)
    setLastFlipWin(null)
  }

  const isGameActive = gameState?.status === 'active'
  const isGameOver = gameState && gameState.status !== 'active'
  const currentRound = gameState?.current_round || 0
  const canCashOut = isGameActive && currentRound > 0

  return (
    <Modal isOpen={true} onClose={onClose} title='Монетка'>
      <div className='space-y-4'>
        {/* Multiplier Progress */}
        <div className='relative'>
          <div className='flex justify-between items-center mb-2'>
            <span className='text-sm text-white/60'>
              Раунды{currentRound}/10
            </span>
            <span className='text-sm font-bold text-primary'>
              x{MULTIPLIERS[currentRound]}
            </span>
          </div>

          {/* Progress bar with milestones */}
          <div className='relative h-3 bg-white/10 rounded-full overflow-hidden'>
            <div
              className='absolute h-full bg-gradient-to-r from-primary to-secondary transition-all duration-500'
              style={{ width: `${(currentRound / 10) * 100}%` }}
            />
          </div>

          {/* Multiplier labels */}
          <div className='flex justify-between mt-2 text-xs text-white/40'>
            {[1, 3, 5, 8, 10].map(r => (
              <div
                key={r}
                className={`${currentRound >= r ? 'text-primary' : ''}`}
              >
                x{MULTIPLIERS[r]}
              </div>
            ))}
          </div>
        </div>

        {/* Coin Display */}
        <div className='flex justify-center py-6'>
          <div
            className={`relative w-32 h-32 rounded-full flex items-center justify-center text-6xl
              ${flipping ? 'animate-bounce' : ''}
              ${lastFlipWin === true ? 'bg-gradient-to-br from-green-500/30 to-emerald-500/30 ring-2 ring-green-500' : ''}
              ${lastFlipWin === false ? 'bg-gradient-to-br from-red-500/30 to-pink-500/30 ring-2 ring-red-500' : ''}
              ${lastFlipWin === null ? 'bg-gradient-to-br from-yellow-500/20 to-orange-500/20' : ''}
              transition-all duration-300
            `}
          >
            {flipping ? (
              <div className='animate-spin'>🪙</div>
            ) : isGameOver ? (
              gameState.status === 'cashed_out' ? (
                '🎉'
              ) : (
                '💔'
              )
            ) : lastFlipWin === true ? (
              '✓'
            ) : lastFlipWin === false ? (
              '✗'
            ) : (
              '🪙'
            )}
          </div>
        </div>

        {/* Game Result */}
        {isGameOver && (
          <div className='text-center space-y-2 animate-fadeIn'>
            <div
              className={`text-2xl font-bold ${
                gameState.status === 'cashed_out'
                  ? 'text-success'
                  : 'text-danger'
              }`}
            >
              {gameState.status === 'cashed_out'
                ? 'ТЫ ВЫИГРАЛ!'
                : 'НЕ ПОВЕЗЛО!'}
            </div>
            {gameState.status === 'cashed_out' && (
              <div className='text-white/60'>
                Выиграно{' '}
                <span className='text-success font-bold'>
                  {gameState.win_amount}
                </span>{' '}
                гемов в x{gameState.multiplier}
              </div>
            )}
            {gameState.status === 'lost' && (
              <div className='text-white/60'>
                Проиграно{' '}
                <span className='text-danger font-bold'>{gameState.bet}</span>{' '}
                гемов после {currentRound}{' '}
                {currentRound === 1 ? 'раунда' : 'раундов'}
              </div>
            )}
          </div>
        )}

        {/* Current Win Amount */}
        {isGameActive && currentRound > 0 && (
          <div className='text-center p-3 rounded-xl bg-gradient-to-r from-green-500/20 to-emerald-500/20 border border-green-500/30'>
            <div className='text-sm text-white/60'>Текущий выигрыш</div>
            <div className='text-2xl font-bold text-success'>
              {gameState.potential_win} гемов
            </div>
            <div className='text-xs text-white/40'>
              Дальше: x{gameState.next_multiplier} (
              {Math.round(gameState.bet * gameState.next_multiplier)} гемов)
            </div>
          </div>
        )}

        {/* Bet Selection (before game) */}
        {!gameState && (
          <div className='space-y-3'>
            <label className='text-sm text-white/60'>Сумма ставки</label>
            <Input
              type='number'
              value={bet}
              onChange={e => setBet(Math.max(1, parseInt(e.target.value) || 0))}
              min={1}
              max={user?.gems || 0}
            />
            <div className='flex gap-2'>
              {BET_PRESETS.map(preset => (
                <button
                  key={preset}
                  onClick={() => setBet(preset)}
                  className={`flex-1 py-1.5 rounded-lg text-sm font-medium transition-colors ${
                    bet === preset
                      ? 'bg-primary text-white'
                      : 'bg-white/10 text-white/60 hover:bg-white/20'
                  }`}
                >
                  {preset}
                </button>
              ))}
            </div>
            <div className='text-center text-white/60 text-sm'>
              Баланс: {user?.gems?.toLocaleString() || 0} gems
            </div>
          </div>
        )}

        {/* Action Buttons */}
        <div className='flex gap-3'>
          {isGameOver ? (
            <>
              <Button variant='secondary' onClick={onClose} className='flex-1'>
                Закрыть
              </Button>
              <Button onClick={handlePlayAgain} className='flex-1'>
                Новая игра
              </Button>
            </>
          ) : isGameActive ? (
            <>
              {canCashOut && (
                <Button
                  variant='success'
                  onClick={handleCashOut}
                  disabled={loading || flipping}
                  className='flex-1'
                >
                  Забрать выигрыш ({gameState.potential_win})
                </Button>
              )}
              <Button
                onClick={handleFlip}
                disabled={loading || flipping}
                className='flex-1'
              >
                {flipping
                  ? 'Летит в воздухе...'
                  : currentRound === 0
                    ? 'Нажми,чтобы подбросить!'
                    : 'Подбросить еще раз!'}
              </Button>
            </>
          ) : (
            <>
              <Button variant='secondary' onClick={onClose} className='flex-1'>
                Закрыть
              </Button>
              <Button
                onClick={handleStart}
                disabled={loading || bet <= 0 || bet > (user?.gems || 0)}
                className='flex-1'
              >
                Начать игру ({bet})
              </Button>
            </>
          )}
        </div>

        {/* Multiplier Table */}
        {!gameState && (
          <div className='mt-4 p-3 rounded-xl bg-white/5'>
            <div className='text-sm font-medium text-white/60 mb-2'>
              Множитель
            </div>
            <div className='grid grid-cols-5 gap-1 text-center text-xs'>
              {MULTIPLIERS.slice(1).map((mult, i) => (
                <div key={i} className='p-1.5 rounded bg-white/5'>
                  <div className='text-white/40'>R{i + 1}</div>
                  <div className='font-bold text-primary'>x{mult}</div>
                </div>
              ))}
            </div>
          </div>
        )}
      </div>
    </Modal>
  )
}
