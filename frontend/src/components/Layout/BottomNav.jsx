import { NavLink } from 'react-router-dom'

const navItems = [
  { path: '/', icon: '🎮', label: 'Игры' },
  { path: '/top', icon: '🏆', label: 'Топ' },
  { path: '/upgrade', icon: '⚡', label: 'Прокачка' },
  { path: '/wallet', icon: '💰', label: 'Кошелёк' },
  { path: '/profile', icon: '👤', label: 'Профиль' },
]

export function BottomNav() {
  return (
    <nav className="fixed bottom-0 left-0 right-0 bg-dark-card/95 backdrop-blur-lg border-t border-white/10 pb-safe">
      <div className="flex justify-around items-center h-16 max-w-lg mx-auto">
        {navItems.map(({ path, icon, label }) => (
          <NavLink
            key={path}
            to={path}
            className={({ isActive }) =>
              `flex flex-col items-center gap-0.5 px-4 py-2 rounded-xl transition-all ${
                isActive
                  ? 'text-primary scale-105'
                  : 'text-white/60 hover:text-white/80'
              }`
            }
          >
            <span className="text-2xl">{icon}</span>
            <span className="text-xs font-medium">{label}</span>
          </NavLink>
        ))}
      </div>
    </nav>
  )
}
