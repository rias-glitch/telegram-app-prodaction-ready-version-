import { useState } from 'react'
import { Modal } from '../ui/Overlay'
import { Button } from '../ui'
import { Input } from '../ui'
import { playCoinFlip } from '../../api/games'

const BET_PRESETS = [10, 50, 100, 500]

export function CoinFlipGame({ user, onClose, onResult }) {
  const [bet, setBet] = useState(10)
  const [loading, setLoading] = useState(false)
  const [result, setResult] = useState(null)
  const [flipping, setFlipping] = useState(false)

  const handlePlay = async () => {
    if (bet <= 0 || bet > (user?.gems || 0)) return

    try {
      setLoading(true)
      setFlipping(true)
      setResult(null)

      const response = await playCoinFlip(bet)

      // Animation delay
      setTimeout(() => {
        setFlipping(false)
        setResult(response)
        onResult(response.gems)
      }, 1500)
    } catch (err) {
      setFlipping(false)
      setResult({ error: err.message })
    } finally {
      setLoading(false)
    }
  }

  const playAgain = () => {
    setResult(null)
  }

  return (
    <Modal isOpen={true} onClose={onClose} title="Coin Flip">
      <div className="space-y-6">
        {/* Coin animation */}
        <div className="flex justify-center py-8">
          <div
            className={`text-8xl transition-transform duration-500 ${
              flipping ? 'animate-spin-slow' : ''
            }`}
          >
            {result ? (result.win ? '🌟' : '💔') : '🪙'}
          </div>
        </div>

        {/* Result */}
        {result && !result.error && (
          <div className="text-center space-y-2">
            <div className={`text-2xl font-bold ${result.win ? 'text-success' : 'text-danger'}`}>
              {result.win ? 'YOU WON!' : 'YOU LOST'}
            </div>
            <div className="text-white/60">
              {result.win ? `+${result.awarded} gems` : `-${bet} gems`}
            </div>
          </div>
        )}

        {result?.error && (
          <div className="text-center text-danger">{result.error}</div>
        )}

        {/* Bet controls */}
        {!result && (
          <>
            <div className="space-y-2">
              <label className="text-sm text-white/60">Bet amount</label>
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
              Balance: {user?.gems?.toLocaleString() || 0} gems
            </div>
          </>
        )}

        {/* Actions */}
        <div className="flex gap-3">
          {result ? (
            <>
              <Button variant="secondary" onClick={onClose} className="flex-1">
                Close
              </Button>
              <Button onClick={playAgain} className="flex-1">
                Play Again
              </Button>
            </>
          ) : (
            <>
              <Button variant="secondary" onClick={onClose} className="flex-1">
                Cancel
              </Button>
              <Button
                onClick={handlePlay}
                disabled={loading || flipping || bet <= 0 || bet > (user?.gems || 0)}
                className="flex-1"
              >
                {loading ? 'Flipping...' : `Flip (${bet})`}
              </Button>
            </>
          )}
        </div>
      </div>
    </Modal>
  )
}
