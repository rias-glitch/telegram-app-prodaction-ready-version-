import { Modal } from '../ui/Overlay'
import { Button } from '../ui'

export function GameModeSelector({ isOpen, onClose, onSelect, gameTitle }) {
  return (
    <Modal isOpen={isOpen} onClose={onClose} title={gameTitle}>
      <div className="space-y-4">
        <p className="text-center text-white/60">Choose game mode</p>

        <div className="grid gap-3">
          <button
            onClick={() => onSelect('pve')}
            className="flex items-center gap-4 p-4 rounded-xl bg-white/10 hover:bg-white/20 transition-colors text-left"
          >
            <div className="text-4xl">🤖</div>
            <div>
              <div className="font-bold">vs Bot</div>
              <div className="text-white/60 text-sm">Play against AI instantly</div>
            </div>
          </button>

          <button
            onClick={() => onSelect('pvp')}
            className="flex items-center gap-4 p-4 rounded-xl bg-gradient-to-r from-primary/20 to-secondary/20 border border-primary/30 hover:border-primary/50 transition-colors text-left"
          >
            <div className="text-4xl">⚔️</div>
            <div>
              <div className="font-bold text-primary">vs Player</div>
              <div className="text-white/60 text-sm">Find a real opponent</div>
            </div>
          </button>
        </div>

        <Button variant="ghost" onClick={onClose} className="w-full">
          Cancel
        </Button>
      </div>
    </Modal>
  )
}
