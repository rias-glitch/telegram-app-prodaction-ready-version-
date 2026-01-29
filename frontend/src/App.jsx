import { Routes, Route } from 'react-router-dom'
import { Layout } from './components/Layout'
import { useAuth } from './hooks/useAuth'
import { useProfile } from './hooks/useProfile'
import { GamesPage } from './pages/GamesPage'
import { CasesPage } from './pages/CasesPage'
import { TopPage } from './pages/TopPage'
import { ProfilePage } from './pages/ProfilePage'
import { WalletPage } from './pages/WalletPage'
import { UpgradePage } from './pages/UpgradePage'
import { RPSPage } from './pages/RPSPage'
import { MinesPage } from './pages/MinesPage'

function App() {
  const { user, loading, error, setUser } = useAuth()
  const { games, stats, quests, fetchProfile, addGems, refreshBalance } = useProfile(
    user,
    setUser
  )

  if (loading) {
    return (
      <div className='min-h-screen flex items-center justify-center'>
        <div className='text-center'>
          <div className='text-4xl mb-4 animate-pulse-custom'>🍆🍆🍆</div>
          <p className='text-white/60'>Загрузка...</p>
        </div>
      </div>
    )
  }

  if (error) {
    return (
      <div className='min-h-screen flex items-center justify-center p-4'>
        <div className='text-center'>
          <div className='text-4xl mb-4'>❌</div>
          <p className='text-danger mb-2'>Failed to connect</p>
          <p className='text-white/60 text-sm'>{error}</p>
        </div>
      </div>
    )
  }

  return (
    <Layout user={user}>
      <Routes>
        <Route
          path='/'
          element={
            <GamesPage user={user} setUser={setUser} addGems={addGems} refreshBalance={refreshBalance} />
          }
        />
        <Route
          path='/cases'
          element={
            <CasesPage user={user} setUser={setUser} addGems={addGems} />
          }
        />
        <Route path='/top' element={<TopPage user={user} />} />
        <Route
          path='/upgrade'
          element={<UpgradePage user={user} setUser={setUser} />}
        />
        <Route
          path='/profile'
          element={
            <ProfilePage
              user={user}
              games={games}
              stats={stats}
              quests={quests}
              fetchProfile={fetchProfile}
            />
          }
        />
        <Route path='/wallet' element={<WalletPage user={user} refreshBalance={refreshBalance} />} />
        <Route
          path='/rps'
          element={<RPSPage user={user} setUser={setUser} refreshBalance={refreshBalance} />}
        />
        <Route
          path='/mines'
          element={<MinesPage user={user} setUser={setUser} refreshBalance={refreshBalance} />}
        />
      </Routes>
    </Layout>
  )
}

export default App
