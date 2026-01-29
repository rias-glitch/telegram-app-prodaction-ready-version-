import { useState } from 'react'
import { Card, CardTitle, Button } from '../components/ui'
import { spinCase } from '../api/games'

const CASE_COST = 100

const PRIZES = [
  { id: 1, amount: 250, color: 'from-gray-500 to-gray-600', rarity: 'Common' },
  { id: 2, amount: 500, color: 'from-green-500 to-emerald-600', rarity: 'Uncommon' },
  { id: 3, amount: 750, color: 'from-blue-500 to-indigo-600', rarity: 'Rare' },
  { id: 4, amount: 1000, color: 'from-purple-500 to-pink-600', rarity: 'Epic' },
  { id: 5, amount: 5000, color: 'from-yellow-400 to-orange-500', rarity: 'Legendary' },
]

export function CasesPage({ user, setUser }) {
  const [spinning, setSpinning] = useState(false)
  const [result, setResult] = useState(null)
  const [error, setError] = useState(null)

  const handleSpin = async () => {
    if (!user || (user.gems || 0) < CASE_COST) {
      setError('Not enough gems')
      return
    }

    try {
      setSpinning(true)
      setResult(null)
      setError(null)

      const response = await spinCase()

      // Animation delay
      setTimeout(() => {
        setSpinning(false)
        setResult(response)
        if (setUser) {
          setUser(prev => ({ ...prev, gems: response.gems }))
        }
      }, 2000)
    } catch (err) {
      setSpinning(false)
      setError(err.message)
    }
  }

  const getPrize = (id) => PRIZES.find(p => p.id === id) || PRIZES[0]

  return (
    <div className="space-y-6 animate-fadeIn">
      <h1 className="text-2xl font-bold">Cases</h1>

      {/* Case display */}
      <Card className="text-center py-8">
        <div className={`text-8xl mb-4 ${spinning ? 'animate-pulse-custom' : ''}`}>
          {result ? '💎' : '🎁'}
        </div>

        {result && (
          <div className="space-y-2 animate-slideUp">
            <div className={`text-3xl font-bold bg-gradient-to-r ${getPrize(result.case_id).color} bg-clip-text text-transparent`}>
              +{result.prize} gems!
            </div>
            <div className="text-white/60">{getPrize(result.case_id).rarity}</div>
          </div>
        )}

        {error && (
          <div className="text-danger">{error}</div>
        )}

        {!result && !error && (
          <p className="text-white/60">Open a case for {CASE_COST} gems</p>
        )}
      </Card>

      {/* Spin button */}
      <Button
        onClick={handleSpin}
        disabled={spinning || (user?.gems || 0) < CASE_COST}
        size="xl"
        className="w-full"
      >
        {spinning ? 'Opening...' : `Open Case (${CASE_COST} gems)`}
      </Button>

      {/* Balance */}
      <div className="text-center text-white/60">
        Your balance: {user?.gems?.toLocaleString() || 0} gems
      </div>

      {/* Prize table */}
      <Card>
        <CardTitle className="mb-4">Possible prizes</CardTitle>
        <div className="space-y-2">
          {PRIZES.map((prize) => (
            <div
              key={prize.id}
              className={`flex items-center justify-between p-3 rounded-xl bg-gradient-to-r ${prize.color} bg-opacity-20`}
              style={{ background: `linear-gradient(to right, rgba(0,0,0,0.3), rgba(0,0,0,0.1))` }}
            >
              <div className="flex items-center gap-3">
                <div className={`w-3 h-3 rounded-full bg-gradient-to-r ${prize.color}`} />
                <span className="font-medium">{prize.rarity}</span>
              </div>
              <span className="font-bold">{prize.amount} gems</span>
            </div>
          ))}
        </div>
      </Card>
    </div>
  )
}
