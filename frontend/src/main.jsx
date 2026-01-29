import React from 'react'
import ReactDOM from 'react-dom/client'
import { BrowserRouter } from 'react-router-dom'
import { TonConnectUIProvider } from '@tonconnect/ui-react'
import App from './App'
import './index.css'

// Error Boundary для отладки
class ErrorBoundary extends React.Component {
  constructor(props) {
    super(props)
    this.state = { hasError: false, error: null }
  }

  static getDerivedStateFromError(error) {
    return { hasError: true, error }
  }

  render() {
    if (this.state.hasError) {
      return (
        <div style={{ padding: 20, color: 'white', background: '#1a1a2e', minHeight: '100vh' }}>
          <h1>Ошибка загрузки</h1>
          <pre style={{ whiteSpace: 'pre-wrap', color: '#ff6b6b' }}>
            {this.state.error?.toString()}
          </pre>
        </div>
      )
    }
    return this.props.children
  }
}

// Initialize Telegram WebApp
try {
  if (window.Telegram?.WebApp) {
    window.Telegram.WebApp.ready()
    window.Telegram.WebApp.expand()
  }
} catch (e) {
  console.error('Telegram WebApp init error:', e)
}

const manifestUrl = `${window.location.origin}/tonconnect-manifest.json`

ReactDOM.createRoot(document.getElementById('root')).render(
  <React.StrictMode>
    <ErrorBoundary>
      <TonConnectUIProvider manifestUrl={manifestUrl}>
        <BrowserRouter>
          <App />
        </BrowserRouter>
      </TonConnectUIProvider>
    </ErrorBoundary>
  </React.StrictMode>,
)
