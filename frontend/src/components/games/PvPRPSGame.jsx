import { useState, useEffect, useRef } from 'react'
import { Modal } from '../ui/Overlay'
import { Button } from '../ui'
import { useWebSocket } from '../../hooks/useWebSocket'

const MOVES = [
  { id: 'rock', icon: '🪨', label: 'Rock' },
  { id: 'paper', icon: '📄', label: 'Paper' },
  { id: 'scissors', icon: '✂️', label: 'Scissors' },
]

const TURN_TIMEOUT = 20 // seconds

export function PvPRPSGame({
  user,
  onClose,
  onResult,
  embedded = false,
  initialBet = 0,
  initialCurrency = 'gems',
}) {
  const {
    status,
    opponent,
    roomId,
    gameState,
    result,
    connect,
    send,
    disconnect,
  } = useWebSocket('rps')

  const [selectedMove, setSelectedMove] = useState(null)
  const [waiting, setWaiting] = useState(false)
  const [round, setRound] = useState(1)
  const [timer, setTimer] = useState(TURN_TIMEOUT)
  const [bet, setBet] = useState(initialBet)
  const [currency, setCurrency] = useState(initialCurrency)
  const [showingDraw, setShowingDraw] = useState(null) // { yourMove, opponentMove }
  const timerRef = useRef(null)
  const lastStartTimestamp = useRef(null)

  useEffect(() => {
    // Auto connect when component mounts
    connect(initialBet, initialCurrency)

    return () => {
      disconnect()
      stopTimer()
    }
  }, [initialBet, initialCurrency])

  useEffect(() => {
    if (result) {
      // Game finished
      setWaiting(false)
      stopTimer()
      // Notify parent to refresh balance
      if (onResult) {
        onResult()
      }
    }
  }, [result, onResult])

  // Handle round_draw - show draw visualization then update round
  useEffect(() => {
    if (status === 'playing' && gameState?.type === 'round_draw') {
      console.log('PvPRPSGame: round_draw received', gameState)

      // Show draw visualization
      setShowingDraw({
        yourMove: gameState.your_move,
        opponentMove: gameState.opponent_move,
        round: gameState.round || round,
      })

      // Hide after 2.5 seconds and increment round
      setTimeout(() => {
        setShowingDraw(null)
        setRound(prev => prev + 1)
      }, 2500)
    }
  }, [gameState, status])

  // Handle start signal - reset state for new round
  useEffect(() => {
    if (gameState?.type === 'start' && gameState?.timestamp) {
      // Only reset if this is a NEW start (not the same one)
      if (lastStartTimestamp.current !== gameState.timestamp) {
        console.log('PvPRPSGame: new round started, timestamp=', gameState.timestamp)
        lastStartTimestamp.current = gameState.timestamp
        setSelectedMove(null)
        setWaiting(false)
        startTimer()
      }
    }
  }, [gameState])

  // Start timer when matched
  useEffect(() => {
    if (status === 'matched') {
      startTimer()
    }
  }, [status])

  // Timer logic
  const startTimer = () => {
    stopTimer()
    setTimer(TURN_TIMEOUT)
    timerRef.current = setInterval(() => {
      setTimer(prev => {
        if (prev <= 1) {
          stopTimer()
          return 0
        }
        return prev - 1
      })
    }, 1000)
  }

  const stopTimer = () => {
    if (timerRef.current) {
      clearInterval(timerRef.current)
      timerRef.current = null
    }
  }

  const handleMove = move => {
    // Prevent double moves
    if (selectedMove !== null || waiting) {
      console.log('PvPRPSGame: handleMove blocked - already moved or waiting')
      return
    }
    setSelectedMove(move)
    setWaiting(true)
    stopTimer()
    send({ type: 'move', value: move })
  }

  const handlePlayAgain = () => {
    setSelectedMove(null)
    setWaiting(false)
    setRound(1)
    setTimer(TURN_TIMEOUT)
    setShowingDraw(null)
    stopTimer()
    disconnect()
    // Increased timeout to ensure WebSocket is fully closed before reconnecting
    setTimeout(() => connect(initialBet, initialCurrency), 500)
  }

  const getMoveIcon = move => MOVES.find(m => m.id === move)?.icon || '❓'

  const getResultText = () => {
    if (!result?.payload) return ''
    const you = result.payload.you
    if (you === 'cancelled') return 'ИГРА ОТМЕНЕНА'
    if (you === 'win') return 'ТЫ ВЫИГРАЛ!'
    if (you === 'lose') return 'ТЫ ПРОИГРАЛ!'
    return 'НИЧЬЯ'
  }

  const getResultColor = () => {
    if (!result?.payload) return ''
    const you = result.payload.you
    if (you === 'cancelled') return 'text-yellow-400'
    if (you === 'win') return 'text-green-400'
    if (you === 'lose') return 'text-red-400'
    return 'text-yellow-400'
  }

  const getOpponentName = () => {
    if (opponent?.first_name) return opponent.first_name
    if (opponent?.username) return `@${opponent.username}`
    if (opponent?.id) return `Игрок#${opponent.id}`
    return 'Противник'
  }

  // Timer progress percentage
  const timerProgress = (timer / TURN_TIMEOUT) * 100
  const timerColor =
    timer > 10 ? 'bg-green-500' : timer > 5 ? 'bg-yellow-500' : 'bg-red-500'

  const currencyIcon = currency === 'coins' ? '🪙' : '💎'

  const gameContent = (
    <div className='space-y-4'>
      {/* Opponent info */}
      {opponent && status !== 'connecting' && status !== 'waiting' && (
        <div className='flex items-center justify-center gap-3 p-3 bg-white/5 rounded-xl'>
          <div className='w-10 h-10 rounded-full bg-gradient-to-br from-blue-500 to-purple-500 flex items-center justify-center text-lg font-bold'>
            {getOpponentName().charAt(0).toUpperCase()}
          </div>
          <div>
            <div className='text-sm text-white/60'>Игра против</div>
            <div className='font-semibold'>{getOpponentName()}</div>
          </div>
        </div>
      )}

      {/* Timer bar */}
      {(status === 'playing' || status === 'matched') &&
        !result &&
        !waiting &&
        !showingDraw && (
          <div className='space-y-1'>
            <div className='flex justify-between text-sm'>
              <span className='text-white/60'>Оставшееся время</span>
              <span
                className={
                  timer <= 5
                    ? 'text-red-400 font-bold animate-pulse'
                    : 'text-white/80'
                }
              >
                {timer}s
              </span>
            </div>
            <div className='h-2 bg-white/10 rounded-full overflow-hidden'>
              <div
                className={`h-full ${timerColor} transition-all duration-1000 ease-linear`}
                style={{ width: `${timerProgress}%` }}
              />
            </div>
          </div>
        )}

      {/* Status indicator */}
      <div className='text-center'>
        {status === 'connecting' && (
          <div className='flex items-center justify-center gap-2 text-white/60'>
            <div className='w-2 h-2 bg-yellow-400 rounded-full animate-pulse' />
            Соединение...
          </div>
        )}
        {status === 'waiting' && (
          <div className='flex items-center justify-center gap-2 text-white/60'>
            <div className='w-2 h-2 bg-primary rounded-full animate-pulse' />
            Поиск противника...
          </div>
        )}
        {status === 'matched' && !result && (
          <div className='flex items-center justify-center gap-2 text-green-400'>
            <div className='w-2 h-2 bg-green-400 rounded-full' />
            Противник найден!
          </div>
        )}
        {status === 'playing' && !result && !showingDraw && (
          <div className='flex items-center justify-center gap-2 text-primary font-medium'>
            <div className='w-2 h-2 bg-primary rounded-full animate-pulse' />
            {waiting
              ? `Ждем действие ${getOpponentName()}...`
              : `Раунд ${round} - Выбирай!`}
          </div>
        )}
      </div>

      {/* Round Draw display */}
      {showingDraw && !result && (
        <div className='text-center space-y-4 animate-fadeIn'>
          <div className='text-lg font-bold text-yellow-400 mb-2'>
            Раунд {showingDraw.round} - Ничья!
          </div>
          <div className='flex items-center justify-center gap-6 py-4'>
            <div className='text-center'>
              <div className='text-6xl mb-2 animate-bounce'>
                {getMoveIcon(showingDraw.yourMove)}
              </div>
              <div className='text-sm text-white/60'>Ты</div>
            </div>
            <div className='text-3xl text-yellow-400 font-bold'>=</div>
            <div className='text-center'>
              <div className='text-6xl mb-2 animate-bounce'>
                {getMoveIcon(showingDraw.opponentMove)}
              </div>
              <div className='text-sm text-white/60'>{getOpponentName()}</div>
            </div>
          </div>
          <div className='text-white/60 text-sm'>
            Одинаковый выбор! Следующий раунд...
          </div>
        </div>
      )}

      {/* Result display */}
      {result && (
        <div className='text-center space-y-4 animate-slideUp'>
          {/* Cancelled - no opponent found */}
          {result.payload?.you === 'cancelled' && (
            <div className='text-7xl mb-4'>🔍</div>
          )}

          {/* Battle display - only show if game was played */}
          {result.payload?.details && result.payload?.you !== 'cancelled' && (
            <div className='flex items-center justify-center gap-6 py-4'>
              <div className='text-center'>
                <div className='text-6xl mb-2 transform hover:scale-110 transition-transform'>
                  {getMoveIcon(result.payload.details.yourMove || selectedMove)}
                </div>
                <div className='text-sm text-white/60'>Ты</div>
              </div>
              <div className='text-3xl text-white/40 font-bold'>VS</div>
              <div className='text-center'>
                <div className='text-6xl mb-2 transform hover:scale-110 transition-transform'>
                  {getMoveIcon(result.payload.details.opponentMove)}
                </div>
                <div className='text-sm text-white/60'>{getOpponentName()}</div>
              </div>
            </div>
          )}

          <div className={`text-3xl font-bold ${getResultColor()}`}>
            {getResultText()}
          </div>

          {result.payload?.reason === 'no_opponent' && (
            <div className='text-white/60'>
              Противник не найден. Ставка возвращена.
            </div>
          )}

          {result.payload?.reason === 'opponent_left' && (
            <div className='text-white/60'>
              {getOpponentName()} вышел из игры
            </div>
          )}
        </div>
      )}

      {/* Move selection */}
      {(status === 'playing' || status === 'matched') &&
        !result &&
        !selectedMove &&
        !showingDraw && (
          <div className='space-y-3'>
            <label className='text-sm text-white/60 text-center block'>
              Сделай свой ход
            </label>
            <div className='grid grid-cols-3 gap-3'>
              {MOVES.map(move => (
                <button
                  key={move.id}
                  onClick={() => handleMove(move.id)}
                  disabled={waiting}
                  className='flex flex-col items-center gap-2 p-4 rounded-xl bg-white/10 hover:bg-white/20 border-2 border-transparent hover:border-primary/50 transition-all transform active:scale-95 disabled:opacity-50'
                >
                  <span className='text-5xl'>{move.icon}</span>
                  <span className='text-sm font-medium'>{move.label}</span>
                </button>
              ))}
            </div>
          </div>
        )}

      {/* Selected move display while waiting */}
      {waiting && !result && !showingDraw && (
        <div className='text-center py-8'>
          <div className='text-7xl mb-4 animate-pulse-custom'>
            {getMoveIcon(selectedMove)}
          </div>
          <p className='text-white/60'>Ждем ход от {getOpponentName()}...</p>
        </div>
      )}

      {/* Searching animation */}
      {(status === 'waiting' || status === 'connecting') && (
        <div className='flex justify-center py-8'>
          <div className='text-6xl animate-pulse-custom'>
            <span className='inline-block animate-bounce'>⚔️</span>
          </div>
        </div>
      )}

      {/* Bet info */}
      {initialBet > 0 && (
        <div className='text-center text-white/60 text-sm'>
          Ставка: {initialBet} {currencyIcon}
        </div>
      )}

      {/* Actions */}
      <div className='flex gap-3 pt-2'>
        {result ? (
          <>
            <Button variant='secondary' onClick={onClose} className='flex-1'>
              {embedded ? 'Назад' : 'Закрыть'}
            </Button>
            <Button onClick={handlePlayAgain} className='flex-1'>
              Играть снова
            </Button>
          </>
        ) : (
          <Button variant='secondary' onClick={onClose} className='w-full'>
            Назад
          </Button>
        )}
      </div>
    </div>
  )

  if (embedded) {
    return gameContent
  }

  return (
    <Modal isOpen={true} onClose={onClose} title='PvP Камень Ножницы Бумага'>
      {gameContent}
    </Modal>
  )
}
