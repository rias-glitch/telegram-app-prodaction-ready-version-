import { useState, useMemo } from 'react'
import { Modal } from '../ui/Overlay'
import { Button, Input } from '../ui'
import { playDice } from '../../api/games'

const BET_PRESETS = [10, 50, 100, 500]

const DICE_FACES = ['⚀', '⚁', '⚂', '⚃', '⚄', '⚅']

// Статичный кубик с точками
function DiceFace({ value, rolling }) {
  // Позиции точек для каждого значения (классическое расположение как на кубике)
  const dotPatterns = {
    1: [[50, 50]],
    2: [[25, 25], [75, 75]],
    3: [[25, 25], [50, 50], [75, 75]],
    4: [[25, 25], [75, 25], [25, 75], [75, 75]],
    5: [[25, 25], [75, 25], [50, 50], [25, 75], [75, 75]],
    6: [[25, 25], [75, 25], [25, 50], [75, 50], [25, 75], [75, 75]],
  }

  const dots = dotPatterns[value] || dotPatterns[1]

  return (
    <div className="dice-container">
      <div className={`dice-box ${rolling ? 'dice-rolling' : ''}`}>
        {dots.map((pos, i) => (
          <div
            key={i}
            className="dice-dot"
            style={{ left: `${pos[0]}%`, top: `${pos[1]}%` }}
          />
        ))}
      </div>
      <style jsx>{`
        .dice-container {
          display: flex;
          justify-content: center;
          align-items: center;
          height: 150px;
        }

        .dice-box {
          width: 120px;
          height: 120px;
          position: relative;
          background: linear-gradient(145deg, #1a1a2e, #16213e);
          border: 3px solid rgba(139, 92, 246, 0.5);
          border-radius: 16px;
          box-shadow:
            0 0 20px rgba(139, 92, 246, 0.3),
            inset 0 0 30px rgba(0, 0, 0, 0.3);
        }

        .dice-dot {
          position: absolute;
          width: 20px;
          height: 20px;
          background: linear-gradient(145deg, #8b5cf6, #a78bfa);
          border-radius: 50%;
          transform: translate(-50%, -50%);
          box-shadow: 0 0 10px rgba(139, 92, 246, 0.8);
          transition: opacity 0.1s ease;
        }

        .dice-rolling .dice-dot {
          animation: blink 0.15s ease-in-out infinite;
        }

        @keyframes blink {
          0%, 100% {
            opacity: 1;
            box-shadow: 0 0 15px rgba(139, 92, 246, 1);
          }
          50% {
            opacity: 0.3;
            box-shadow: 0 0 5px rgba(139, 92, 246, 0.3);
          }
        }
      `}</style>
    </div>
  )
}

export function DiceGame({ user, onClose, onResult }) {
  const [bet, setBet] = useState(10)
  const [target, setTarget] = useState(1)
  const [mode, setMode] = useState('exact') // 'exact', 'low', 'high'
  const [loading, setLoading] = useState(false)
  const [rolling, setRolling] = useState(false)
  const [result, setResult] = useState(null)
  const [displayValue, setDisplayValue] = useState(1)

  // Calculate win chance and multiplier based on mode
  const { winChance, multiplier } = useMemo(() => {
    if (mode === 'exact') {
      return { winChance: 16.67, multiplier: 5.5 }
    } else {
      return { winChance: 50, multiplier: 1.8 }
    }
  }, [mode])

  const handleRoll = async () => {
    if (bet <= 0 || bet > (user?.gems || 0)) return

    try {
      setLoading(true)
      setRolling(true)
      setResult(null)

      // Animate dice rolling (cycle through faces)
      const rollInterval = setInterval(() => {
        setDisplayValue(Math.floor(Math.random() * 6) + 1)
      }, 100)

      const response = await playDice(bet, target, mode)

      // Stop animation and show result
      setTimeout(() => {
        clearInterval(rollInterval)
        setDisplayValue(response.result)
        setRolling(false)
        setResult(response)
        onResult(response.gems)
      }, 1500)
    } catch (err) {
      setRolling(false)
      setResult({ error: err.message })
    } finally {
      setLoading(false)
    }
  }

  const playAgain = () => {
    setResult(null)
  }

  return (
    <Modal isOpen={true} onClose={onClose} title="Dice">
      <div className="space-y-6">
        {/* Dice display */}
        <div className="flex justify-center py-4">
          <div className={`dice-result-glow ${result ? (result.won ? 'glow-success' : 'glow-danger') : ''}`}>
            <DiceFace value={displayValue} rolling={rolling} />
          </div>
        </div>
        <style jsx>{`
          .dice-result-glow {
            border-radius: 20px;
            transition: all 0.3s ease;
          }
          .glow-success {
            filter: drop-shadow(0 0 25px rgba(34, 197, 94, 0.7));
          }
          .glow-danger {
            filter: drop-shadow(0 0 25px rgba(239, 68, 68, 0.7));
          }
        `}</style>

        {/* Result */}
        {result && !result.error && (
          <div className="text-center space-y-2">
            <div className={`text-2xl font-bold ${result.won ? 'text-success' : 'text-danger'}`}>
              {result.won ? 'ПОБЕДА!' : 'ПРОИГРЫШ'}
            </div>
            <div className="text-white/60">
              {result.won ? `+${result.win_amount}` : `-${bet}`} гемов
            </div>
            <div className="text-sm text-white/40">
              {result.mode === 'exact' && `Твой выбор: ${target} → Выпало: ${result.result}`}
              {result.mode === 'low' && `Low (1-3) → Выпало: ${result.result}`}
              {result.mode === 'high' && `High (4-6) → Выпало: ${result.result}`}
            </div>
          </div>
        )}

        {result?.error && (
          <div className="text-center text-danger">{result.error}</div>
        )}

        {/* Game controls */}
        {!result && (
          <>
            {/* Game mode selection */}
            <div className="space-y-2">
              <label className="text-sm text-white/60">Выбери режим игры</label>
              <div className="grid grid-cols-3 gap-2">
                <button
                  onClick={() => setMode('low')}
                  className={`py-3 px-2 rounded-xl font-medium transition-all ${
                    mode === 'low'
                      ? 'bg-primary text-white scale-105 shadow-lg'
                      : 'bg-white/10 text-white/60 hover:bg-white/20'
                  }`}
                >
                  <div className="text-lg font-bold">Low</div>
                  <div className="text-xs opacity-80">1-3</div>
                  <div className="text-xs text-success">{multiplier}x</div>
                </button>
                <button
                  onClick={() => setMode('high')}
                  className={`py-3 px-2 rounded-xl font-medium transition-all ${
                    mode === 'high'
                      ? 'bg-primary text-white scale-105 shadow-lg'
                      : 'bg-white/10 text-white/60 hover:bg-white/20'
                  }`}
                >
                  <div className="text-lg font-bold">High</div>
                  <div className="text-xs opacity-80">4-6</div>
                  <div className="text-xs text-success">{multiplier}x</div>
                </button>
                <button
                  onClick={() => setMode('exact')}
                  className={`py-3 px-2 rounded-xl font-medium transition-all ${
                    mode === 'exact'
                      ? 'bg-primary text-white scale-105 shadow-lg'
                      : 'bg-white/10 text-white/60 hover:bg-white/20'
                  }`}
                >
                  <div className="text-lg font-bold">Exact</div>
                  <div className="text-xs opacity-80">Выбор</div>
                  <div className="text-xs text-success">{multiplier}x</div>
                </button>
              </div>
            </div>

            {/* Pick your number (only for exact mode) */}
            {mode === 'exact' && (
              <div className="space-y-2">
                <label className="text-sm text-white/60">Выбери число (1-6)</label>
                <div className="grid grid-cols-6 gap-2">
                  {[1, 2, 3, 4, 5, 6].map((num) => (
                    <button
                      key={num}
                      onClick={() => setTarget(num)}
                      className={`aspect-square rounded-xl font-bold text-3xl transition-all ${
                        target === num
                          ? 'bg-primary text-white scale-110 shadow-lg'
                          : 'bg-white/10 text-white/60 hover:bg-white/20 hover:scale-105'
                      }`}
                    >
                      {DICE_FACES[num - 1]}
                    </button>
                  ))}
                </div>
              </div>
            )}

            {/* Range description */}
            {mode !== 'exact' && (
              <div className="bg-white/5 rounded-xl p-3 text-center">
                <div className="text-sm text-white/60">
                  {mode === 'low' ? 'Победа, если выпадет 1, 2 или 3' : 'Победа, если выпадет 4, 5 или 6'}
                </div>
              </div>
            )}

            {/* Stats */}
            <div className="grid grid-cols-2 gap-4 text-center">
              <div className="bg-white/5 rounded-xl p-3">
                <div className="text-white/60 text-sm">Шанс победы</div>
                <div className="text-xl font-bold text-success">{winChance.toFixed(2)}%</div>
              </div>
              <div className="bg-white/5 rounded-xl p-3">
                <div className="text-white/60 text-sm">Множитель</div>
                <div className="text-xl font-bold text-primary">{multiplier}x</div>
              </div>
            </div>

            {/* Bet controls */}
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

            <div className="flex justify-between text-sm text-white/60">
              <span>Баланс: {user?.gems?.toLocaleString() || 0}</span>
              <span>Возможный выигрыш: {Math.floor(bet * multiplier)}</span>
            </div>
          </>
        )}

        {/* Actions */}
        <div className="flex gap-3">
          {result ? (
            <>
              <Button variant="secondary" onClick={onClose} className="flex-1">
                Закрыть
              </Button>
              <Button onClick={playAgain} className="flex-1">
                Бросить снова
              </Button>
            </>
          ) : (
            <>
              <Button variant="secondary" onClick={onClose} className="flex-1">
                Отмена
              </Button>
              <Button
                onClick={handleRoll}
                disabled={loading || rolling || bet <= 0 || bet > (user?.gems || 0)}
                className="flex-1"
              >
                {rolling ? 'Бросаем...' : `Бросить (${bet})`}
              </Button>
            </>
          )}
        </div>
      </div>
    </Modal>
  )
}
