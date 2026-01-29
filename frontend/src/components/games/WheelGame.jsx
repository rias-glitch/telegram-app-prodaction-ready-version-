import { useState, useEffect, useRef } from 'react'
import { Modal } from '../ui/Overlay'
import { Button, Input } from '../ui'
import { playWheel, getWheelInfo } from '../../api/games'

const BET_PRESETS = [10, 50, 100, 500]

// Default segments matching backend order
const DEFAULT_SEGMENTS = [
  { id: 1, multiplier: 0.0, color: '#4a4a4a', label: '0x' },
  { id: 2, multiplier: 0.5, color: '#e74c3c', label: '0.5x' },
  { id: 3, multiplier: 1.0, color: '#f39c12', label: '1x' },
  { id: 4, multiplier: 1.5, color: '#2ecc71', label: '1.5x' },
  { id: 5, multiplier: 2.0, color: '#3498db', label: '2x' },
  { id: 6, multiplier: 3.0, color: '#9b59b6', label: '3x' },
  { id: 7, multiplier: 5.0, color: '#e67e22', label: '5x' },
  { id: 8, multiplier: 10.0, color: '#f1c40f', label: '10x' },
]

export function WheelGame({ user, onClose, onResult }) {
  const [bet, setBet] = useState(10)
  const [loading, setLoading] = useState(false)
  const [spinning, setSpinning] = useState(false)
  const [result, setResult] = useState(null)
  const [segments, setSegments] = useState(DEFAULT_SEGMENTS)
  const [offset, setOffset] = useState(0)
  const reelRef = useRef(null)

  useEffect(() => {
    getWheelInfo().then(data => {
      if (data.segments && data.segments.length > 0) {
        setSegments(data.segments)
      }
    }).catch(() => {})
  }, [])

  const handleSpin = async () => {
    if (bet <= 0 || bet > (user?.gems || 0) || spinning) return

    try {
      setLoading(true)
      setSpinning(true)
      setResult(null)

      const response = await playWheel(bet)

      // Find winning segment index
      const winIndex = segments.findIndex(s => s.id === response.segment_id)
      const finalIndex = winIndex >= 0 ? winIndex : 0

      // Calculate animation
      const segmentWidth = 120 // Width of each segment in pixels
      const totalSpins = 3 // Full rotations

      // Calculate where the reel currently shows (normalize to 0..segments.length-1)
      const currentSegmentFloat = (offset / segmentWidth) % segments.length
      const currentSegment = Math.floor(currentSegmentFloat)

      // Calculate how far we need to travel:
      // Go through totalSpins full rotations, then land on finalIndex
      // Account for where we currently are
      let segmentsToTravel = totalSpins * segments.length + (finalIndex - currentSegment)
      if (segmentsToTravel < totalSpins * segments.length) {
        segmentsToTravel += segments.length // ensure we do at least totalSpins rotations
      }
      const totalDistance = segmentsToTravel * segmentWidth

      // Animate the reel
      const duration = 3000 // 3 seconds
      const startTime = Date.now()
      const startOffset = offset

      const animate = () => {
        const elapsed = Date.now() - startTime
        const progress = Math.min(elapsed / duration, 1)

        // Easing function (ease out cubic)
        const easeOut = 1 - Math.pow(1 - progress, 3)

        const currentOffset = startOffset + (totalDistance * easeOut)
        setOffset(currentOffset)

        if (progress < 1) {
          requestAnimationFrame(animate)
        } else {
          // Animation complete - normalize offset to prevent accumulation issues
          // Set offset to exactly where finalIndex should be (mod full cycle)
          const normalizedOffset = finalIndex * segmentWidth
          setOffset(normalizedOffset)
          setSpinning(false)
          setResult(response)
          onResult(response.gems)
        }
      }

      requestAnimationFrame(animate)

    } catch (err) {
      setSpinning(false)
      setResult({ error: err.message })
    } finally {
      setLoading(false)
    }
  }

  const playAgain = () => {
    setResult(null)
  }

  // Create extended segments array for seamless looping
  const extendedSegments = [...segments, ...segments, ...segments, ...segments, ...segments]

  return (
    <Modal isOpen={true} onClose={onClose} title="Колесо удачи">
      <div className="space-y-6">
        {/* Slot machine reel */}
        <div className="relative">
          {/* Frame */}
          <div className="bg-gradient-to-b from-yellow-600 to-yellow-800 rounded-2xl p-2 shadow-lg">
            {/* Reel container with mask */}
            <div className="relative overflow-hidden rounded-xl bg-gray-900" style={{ height: '100px' }}>
              {/* Center indicator */}
              <div className="absolute inset-y-0 left-1/2 w-1 bg-yellow-400 z-20 transform -translate-x-1/2" />
              <div className="absolute top-0 left-1/2 transform -translate-x-1/2 z-20">
                <div className="w-0 h-0 border-l-8 border-r-8 border-t-8 border-l-transparent border-r-transparent border-t-yellow-400" />
              </div>
              <div className="absolute bottom-0 left-1/2 transform -translate-x-1/2 z-20">
                <div className="w-0 h-0 border-l-8 border-r-8 border-b-8 border-l-transparent border-r-transparent border-b-yellow-400" />
              </div>

              {/* Gradient overlays for depth effect */}
              <div className="absolute inset-y-0 left-0 w-16 bg-gradient-to-r from-gray-900 to-transparent z-10 pointer-events-none" />
              <div className="absolute inset-y-0 right-0 w-16 bg-gradient-to-l from-gray-900 to-transparent z-10 pointer-events-none" />

              {/* Scrolling reel */}
              <div
                ref={reelRef}
                className="flex items-center h-full transition-none"
                style={{
                  transform: `translateX(calc(50% - 60px - ${offset % (segments.length * 120)}px))`,
                }}
              >
                {extendedSegments.map((seg, i) => (
                  <div
                    key={`${seg.id}-${i}`}
                    className="flex-shrink-0 w-[120px] h-full flex items-center justify-center text-2xl font-bold border-x border-gray-700"
                    style={{
                      backgroundColor: seg.color,
                      textShadow: '0 2px 4px rgba(0,0,0,0.5)',
                    }}
                  >
                    {seg.label}
                  </div>
                ))}
              </div>
            </div>
          </div>

          {/* Decorative lights */}
          <div className="flex justify-around mt-3">
            {[...Array(8)].map((_, i) => (
              <div
                key={i}
                className="w-3 h-3 rounded-full transition-all"
                style={{
                  backgroundColor: spinning
                    ? ['#ef4444', '#f59e0b', '#10b981', '#3b82f6'][i % 4]
                    : '#444',
                  boxShadow: spinning
                    ? `0 0 10px ${['#ef4444', '#f59e0b', '#10b981', '#3b82f6'][i % 4]}`
                    : 'none',
                  animation: spinning ? `pulse ${0.3 + i * 0.1}s infinite` : 'none',
                }}
              />
            ))}
          </div>
        </div>

        {/* Result */}
        {result && !result.error && (
          <div className="text-center space-y-2 animate-fadeIn">
            <div
              className="text-4xl font-bold"
              style={{ color: segments.find(s => s.id === result.segment_id)?.color || '#fff' }}
            >
              {result.label}
            </div>
            <div className={`text-2xl font-bold ${result.win_amount >= (result.bet || bet) ? 'text-success' : 'text-danger'}`}>
              {result.win_amount - (result.bet || bet) >= 0 ? '+' : ''}{result.win_amount - (result.bet || bet)} гемов
            </div>
            <div className="text-white/60">
              Ставка: {result.bet || bet} | Выигрыш: {result.win_amount} | Множитель: {result.multiplier}x
            </div>
          </div>
        )}

        {result?.error && (
          <div className="text-center text-danger">{result.error}</div>
        )}

        {/* Multipliers grid */}
        {!spinning && !result && (
          <div className="grid grid-cols-4 gap-2">
            {segments.map((seg) => (
              <div
                key={seg.id}
                className="text-center py-2 px-1 rounded-lg text-sm font-bold"
                style={{
                  backgroundColor: seg.color + '30',
                  color: seg.color,
                  border: `1px solid ${seg.color}50`,
                }}
              >
                {seg.label}
              </div>
            ))}
          </div>
        )}

        {/* Bet controls */}
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
                Крутить снова
              </Button>
            </>
          ) : (
            <>
              <Button variant="secondary" onClick={onClose} className="flex-1">
                Отмена
              </Button>
              <Button
                onClick={handleSpin}
                disabled={loading || spinning || bet <= 0 || bet > (user?.gems || 0)}
                className="flex-1 bg-gradient-to-r from-yellow-500 to-orange-500 hover:from-yellow-600 hover:to-orange-600"
              >
                {spinning ? '🎰 Крутим...' : `🎰 Крутить (${bet})`}
              </Button>
            </>
          )}
        </div>
      </div>
    </Modal>
  )
}
