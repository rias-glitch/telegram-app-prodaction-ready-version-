import { useState } from 'react'
import { useNavigate } from 'react-router-dom'
import { Card, CardTitle, Button } from '../components/ui'
import { MinesGame } from '../components/games/MinesGame'
import { PvPMinesGame } from '../components/games/PvPMinesGame'

export function MinesPage({ user, setUser, refreshBalance }) {
  const navigate = useNavigate()
  const [activeTab, setActiveTab] = useState('pve')
  const [gameStarted, setGameStarted] = useState(false)
  const [selectedCurrency, setSelectedCurrency] = useState('gems')
  const [selectedBet, setSelectedBet] = useState(100)

  const handleGameResult = async (newGems) => {
    if (setUser && newGems !== undefined) {
      setUser(prev => ({ ...prev, gems: newGems }))
    }
    // Обновляем баланс с сервера после игры
    if (refreshBalance) {
      await refreshBalance()
    }
  }

  const handleClose = () => {
    setGameStarted(false)
  }

  const handleStartPvP = () => {
    setGameStarted(true)
  }

  const betPresets = selectedCurrency === 'coins'
    ? [1, 5, 10, 50]
    : [100, 500, 1000, 5000]

  const getBalance = () => {
    if (selectedCurrency === 'coins') {
      return user?.coins || 0
    }
    return user?.gems || 0
  }

  return (
    <div className="space-y-4 animate-fadeIn">
      {/* Header */}
      <div className="flex items-center gap-3">
        <button
          onClick={() => navigate('/')}
          className="text-2xl hover:scale-110 transition-transform"
        >
          ←
        </button>
        <div>
          <h1 className="text-2xl font-bold flex items-center gap-2">
            <span>💣</span> Mines
          </h1>
          <p className="text-white/60 text-sm">Избегай бомб и выиграй x2</p>
        </div>
      </div>

      {/* Mode tabs */}
      <div className="flex gap-2">
        <button
          onClick={() => { setActiveTab('pve'); setGameStarted(false); }}
          className={`flex-1 py-3 rounded-xl font-medium transition-all ${
            activeTab === 'pve'
              ? 'bg-primary text-white shadow-lg shadow-primary/30'
              : 'bg-white/10 text-white/60 hover:bg-white/20'
          }`}
        >
          <div className="flex items-center justify-center gap-2">
            <span>🤖</span>
            <span>Против бота</span>
          </div>
        </button>
        <button
          onClick={() => { setActiveTab('pvp'); setGameStarted(false); }}
          className={`flex-1 py-3 rounded-xl font-medium transition-all ${
            activeTab === 'pvp'
              ? 'bg-gradient-to-r from-purple-500 to-pink-500 text-white shadow-lg shadow-purple-500/30'
              : 'bg-white/10 text-white/60 hover:bg-white/20'
          }`}
        >
          <div className="flex items-center justify-center gap-2">
            <span>⚔️</span>
            <span>Против игрока</span>
          </div>
        </button>
      </div>

      {/* PvE Mode */}
      {activeTab === 'pve' && (
        <MinesGame
          user={user}
          onClose={() => navigate('/')}
          onResult={handleGameResult}
          embedded={true}
        />
      )}

      {/* PvP Mode - Currency & Bet Selection */}
      {activeTab === 'pvp' && !gameStarted && (
        <div className="space-y-4">
          {/* Currency selection */}
          <Card>
            <CardTitle className="mb-3">Выбери валюту</CardTitle>
            <div className="grid grid-cols-2 gap-3">
              <button
                onClick={() => { setSelectedCurrency('gems'); setSelectedBet(100); }}
                className={`p-4 rounded-xl border-2 transition-all ${
                  selectedCurrency === 'gems'
                    ? 'border-cyan-400 bg-cyan-400/20'
                    : 'border-white/10 bg-white/5 hover:border-white/30'
                }`}
              >
                <div className="text-3xl mb-1">💎</div>
                <div className="font-semibold">Gems</div>
                <div className="text-white/60 text-sm">{user?.gems?.toLocaleString() || 0}</div>
              </button>
              <button
                onClick={() => { setSelectedCurrency('coins'); setSelectedBet(1); }}
                className={`p-4 rounded-xl border-2 transition-all ${
                  selectedCurrency === 'coins'
                    ? 'border-yellow-400 bg-yellow-400/20'
                    : 'border-white/10 bg-white/5 hover:border-white/30'
                }`}
              >
                <div className="text-3xl mb-1">🪙</div>
                <div className="font-semibold">Coins</div>
                <div className="text-white/60 text-sm">{user?.coins?.toLocaleString() || 0}</div>
              </button>
            </div>
          </Card>

          {/* Bet selection */}
          <Card>
            <CardTitle className="mb-3">Выбери ставку</CardTitle>
            <div className="grid grid-cols-4 gap-2 mb-4">
              {betPresets.map((bet) => (
                <button
                  key={bet}
                  onClick={() => setSelectedBet(bet)}
                  disabled={bet > getBalance()}
                  className={`py-3 rounded-xl font-medium transition-all ${
                    selectedBet === bet
                      ? selectedCurrency === 'coins'
                        ? 'bg-yellow-500 text-black'
                        : 'bg-cyan-500 text-black'
                      : 'bg-white/10 hover:bg-white/20 disabled:opacity-30 disabled:hover:bg-white/10'
                  }`}
                >
                  {bet}
                </button>
              ))}
            </div>
            <div className="flex items-center justify-between text-sm text-white/60">
              <span>Ваш баланс:</span>
              <span className="flex items-center gap-1">
                {selectedCurrency === 'coins' ? '🪙' : '💎'}
                <span className="font-semibold text-white">{getBalance().toLocaleString()}</span>
              </span>
            </div>
          </Card>

          {/* Start button */}
          <Button
            onClick={handleStartPvP}
            disabled={selectedBet > getBalance()}
            className={`w-full py-4 text-lg ${
              selectedCurrency === 'coins'
                ? 'bg-gradient-to-r from-yellow-500 to-orange-500 hover:from-yellow-600 hover:to-orange-600'
                : 'bg-gradient-to-r from-purple-500 to-pink-500 hover:from-purple-600 hover:to-pink-600'
            }`}
          >
            <span className="flex items-center justify-center gap-2">
              <span>⚔️</span>
              <span>Найти соперника</span>
              <span className="opacity-60">
                ({selectedBet} {selectedCurrency === 'coins' ? '🪙' : '💎'})
              </span>
            </span>
          </Button>

          <p className="text-center text-white/40 text-sm">
            Выиграй x2 от ставки. Ты будешь сопоставлен с игроками с такой же ставкой и валютой.
          </p>
        </div>
      )}

      {/* PvP Game Started */}
      {activeTab === 'pvp' && gameStarted && (
        <PvPMinesGame
          user={user}
          onClose={handleClose}
          onResult={handleGameResult}
          embedded={true}
          initialBet={selectedBet}
          initialCurrency={selectedCurrency}
        />
      )}
    </div>
  )
}
