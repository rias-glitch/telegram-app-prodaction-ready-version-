import { BottomNav } from './BottomNav'

export function Layout({ children, user }) {
  return (
    <div className="min-h-screen flex flex-col">
      {/* Header with balance */}
      <header className="sticky top-0 z-40 bg-dark/95 backdrop-blur-lg border-b border-white/10">
        <div className="flex items-center justify-between px-4 py-3 max-w-lg mx-auto">
          <div className="flex items-center gap-2">
            <span className="text-xl">🎮</span>
            <span className="font-bold gradient-text">Games</span>
          </div>
          {user && (
            <div className="flex items-center gap-2">
              <div className="flex items-center gap-1.5 bg-white/10 px-2.5 py-1.5 rounded-full">
                <span className="text-base">🪙</span>
                <span className="font-bold text-sm">{user.coins?.toLocaleString() || 0}</span>
              </div>
              <div className="flex items-center gap-1.5 bg-white/10 px-2.5 py-1.5 rounded-full">
                <span className="text-base">💎</span>
                <span className="font-bold text-sm">{user.gems?.toLocaleString() || 0}</span>
              </div>
            </div>
          )}
        </div>
      </header>

      {/* Main content */}
      <main className="flex-1 overflow-y-auto pb-20">
        <div className="max-w-lg mx-auto px-4 py-4">
          {children}
        </div>
      </main>

      {/* Bottom navigation */}
      <BottomNav />
    </div>
  )
}
