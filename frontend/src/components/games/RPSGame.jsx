import { useState } from 'react'
import { Modal } from '../ui/Overlay'
import { Button } from '../ui'
import { Input } from '../ui'
import { playRPS } from '../../api/games'

const BET_PRESETS = [10, 50, 100, 500]
const MOVES = [
  { id: 'rock', icon: '🪨', label: 'Камень' },
  { id: 'paper', icon: '📄', label: 'Бумага' },
  { id: 'scissors', icon: '✂️', label: 'Ножницы' },
]

export function RPSGame({ user, onClose, onResult, embedded = false }) {
  const [bet, setBet] = useState(100)
  const [loading, setLoading] = useState(false)
  const [result, setResult] = useState(null)
  const [selectedMove, setSelectedMove] = useState(null)

  const handlePlay = async (move) => {
    if (bet <= 0 || bet > (user?.gems || 0)) return

    try {
      setLoading(true)
      setSelectedMove(move)
      setResult(null)

      const response = await playRPS(bet, move)

      // Animation delay
      setTimeout(() => {
        setResult(response)
        onResult(response.gems)
      }, 1000)
    } catch (err) {
      setResult({ error: err.message })
    } finally {
      setLoading(false)
    }
  }

  const playAgain = () => {
    setResult(null)
    setSelectedMove(null)
  }

  const getResultText = () => {
    if (!result) return ''
    if (result.result === 1) return 'ПОБЕДА!'
    if (result.result === 0) return 'НИЧЬЯ'
    return 'ПРОИГРЫШ'
  }

  const getResultColor = () => {
    if (!result) return ''
    if (result.result === 1) return 'text-success'
    if (result.result === 0) return 'text-yellow-400'
    return 'text-danger'
  }

  const getMoveIcon = (move) => MOVES.find(m => m.id === move)?.icon || '❓'

  const gameContent = (
    <div className="space-y-6">
      {/* Battle display */}
        {result && (
          <div className="flex items-center justify-center gap-4 py-4">
            <div className="text-center">
              <div className="text-5xl mb-2">{getMoveIcon(result.move)}</div>
              <div className="text-sm text-white/60">Ты</div>
            </div>
            <div className="text-2xl text-white/40">VS</div>
            <div className="text-center">
              <div className="text-5xl mb-2">{getMoveIcon(result.bot)}</div>
              <div className="text-sm text-white/60">Бот</div>
            </div>
          </div>
        )}

        {/* Result */}
        {result && !result.error && (
          <div className="text-center space-y-2">
            <div className={`text-2xl font-bold ${getResultColor()}`}>
              {getResultText()}
            </div>
            <div className="text-white/60">
              {result.result === 1 && `+${result.awarded} гемов`}
              {result.result === -1 && `-${bet} гемов`}
              {result.result === 0 && 'Ставка возвращена'}
            </div>
          </div>
        )}

        {result?.error && (
          <div className="text-center text-danger">{result.error}</div>
        )}

        {/* Move selection */}
        {!result && (
          <>
            <div className="space-y-2">
              <label className="text-sm text-white/60">Сумма ставки</label>
              <Input
                type="number"
                value={bet}
                onChange={(e) => setBet(Math.max(1, parseInt(e.target.value) || 0))}
                min={1}
                max={user?.gems || 0}
              />
              <div className="flex gap-2">
                {BET_PRESETS.map((preset) => (
                  <button
                    key={preset}
                    onClick={() => setBet(preset)}
                    className={`flex-1 py-1 rounded-lg text-sm transition-colors ${
                      bet === preset
                        ? 'bg-primary text-white'
                        : 'bg-white/10 text-white/60 hover:bg-white/20'
                    }`}
                  >
                    {preset}
                  </button>
                ))}
              </div>
            </div>

            <div className="text-center text-white/60 text-sm">
              Баланс: {user?.gems?.toLocaleString() || 0} гемов
            </div>

            <div className="space-y-2">
              <label className="text-sm text-white/60 text-center block">Выбери свой ход</label>
              <div className="grid grid-cols-3 gap-3">
                {MOVES.map((move) => (
                  <button
                    key={move.id}
                    onClick={() => handlePlay(move.id)}
                    disabled={loading || bet <= 0 || bet > (user?.gems || 0)}
                    className="flex flex-col items-center gap-2 p-4 rounded-xl bg-white/10 hover:bg-white/20 transition-colors disabled:opacity-50"
                  >
                    <span className="text-4xl">{move.icon}</span>
                    <span className="text-sm">{move.label}</span>
                  </button>
                ))}
              </div>
            </div>
          </>
        )}

      {/* Actions */}
      {result && (
        <div className="flex gap-3">
          <Button variant="secondary" onClick={onClose} className="flex-1">
            {embedded ? 'Назад' : 'Закрыть'}
          </Button>
          <Button onClick={playAgain} className="flex-1">
            Играть снова
          </Button>
        </div>
      )}

      {!result && !embedded && (
        <Button variant="secondary" onClick={onClose} className="w-full">
          Отмена
        </Button>
      )}
    </div>
  )

  if (embedded) {
    return gameContent
  }

  return (
    <Modal isOpen={true} onClose={onClose} title="Камень Ножницы Бумага">
      {gameContent}
    </Modal>
  )
}
