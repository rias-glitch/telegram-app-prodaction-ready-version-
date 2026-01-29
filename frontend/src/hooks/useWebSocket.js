import { useState, useCallback, useRef, useEffect } from 'react'
import { getToken } from '../api/client'

export function useWebSocket(gameType = 'rps') {
  const [status, setStatus] = useState('disconnected') // disconnected, connecting, waiting, matched, playing
  const [opponent, setOpponent] = useState(null)
  const [roomId, setRoomId] = useState(null)
  const [gameState, setGameState] = useState(null)
  const [result, setResult] = useState(null)
  const [roundResult, setRoundResult] = useState(null) // Last round result for Mines
  const [moveHistory, setMoveHistory] = useState([]) // History of player's moves

  const wsRef = useRef(null)
  const handlersRef = useRef({})

  const connect = useCallback((betAmount, currency = 'gems') => {
    const token = getToken()
    if (!token) {
      console.error('No auth token')
      return
    }

    // Close existing connection if any
    if (wsRef.current) {
      wsRef.current.close()
      wsRef.current = null
    }

    let wsUrl
    const wsBase = import.meta.env.VITE_WS_URL
    if (wsBase) {
      wsUrl = `${wsBase}/ws?token=${token}&bet=${betAmount}&game=${gameType}&currency=${currency}`
    } else {
      const protocol = window.location.protocol === 'https:' ? 'wss:' : 'ws:'
      wsUrl = `${protocol}//${window.location.host}/ws?token=${token}&bet=${betAmount}&game=${gameType}&currency=${currency}`
    }

    setStatus('connecting')
    const ws = new WebSocket(wsUrl)
    wsRef.current = ws

    ws.onopen = () => {
      setStatus('waiting')
    }

    ws.onmessage = (event) => {
      try {
        const msg = JSON.parse(event.data)
        handleMessage(msg)
      } catch (err) {
        console.error('Failed to parse message:', err)
      }
    }

    ws.onclose = () => {
      setStatus('disconnected')
      setOpponent(null)
      setRoomId(null)
      wsRef.current = null
    }

    ws.onerror = (err) => {
      console.error('WebSocket error:', err)
    }
  }, [gameType])

  const handleMessage = useCallback((msg) => {
    // Backend sends data in 'payload' field
    const payload = msg.payload || {}

    switch (msg.type) {
      case 'matched':
        setStatus('matched')
        setOpponent(payload.opponent)
        setRoomId(payload.room_id)
        break

      case 'start':
        console.log('useWebSocket: received start message', payload)
        setStatus('playing')
        setGameState(payload)
        break

      case 'state':
        setGameState(payload)
        break

      case 'setup_complete':
        setGameState({ type: 'setup_complete' })
        break

      case 'round_draw':
        // Round ended in draw - trigger state update so components reset
        setGameState({ type: 'round_draw', timestamp: Date.now() })
        break

      case 'round_result':
        // Mines game - round result with move details
        console.log('useWebSocket: received round_result', payload)
        // Add unique id to ensure React detects change
        setRoundResult({ ...payload, _id: Date.now() + Math.random() })
        if (payload.history) {
          setMoveHistory(payload.history)
        }
        break

      case 'result':
        setResult(msg)
        setStatus('disconnected')
        break

      case 'opponent_left':
        setResult({ type: 'opponent_left', message: 'Opponent left the game' })
        setStatus('disconnected')
        break

      default:
        if (handlersRef.current[msg.type]) {
          handlersRef.current[msg.type](msg)
        }
    }
  }, [])

  const send = useCallback((data) => {
    console.log('useWebSocket.send: called with', JSON.stringify(data), 'wsRef.current=', !!wsRef.current, 'readyState=', wsRef.current?.readyState)
    if (wsRef.current?.readyState === WebSocket.OPEN) {
      wsRef.current.send(JSON.stringify(data))
      console.log('useWebSocket.send: message sent successfully')
    } else {
      console.error('useWebSocket.send: WebSocket not open, state=', wsRef.current?.readyState, '(0=CONNECTING, 1=OPEN, 2=CLOSING, 3=CLOSED)')
    }
  }, [])

  const disconnect = useCallback(() => {
    if (wsRef.current) {
      wsRef.current.close()
      wsRef.current = null
    }
    setStatus('disconnected')
    setOpponent(null)
    setRoomId(null)
    setGameState(null)
    setResult(null)
    setRoundResult(null)
    setMoveHistory([])
  }, [])

  const onMessage = useCallback((type, handler) => {
    handlersRef.current[type] = handler
  }, [])

  useEffect(() => {
    return () => {
      if (wsRef.current) {
        wsRef.current.close()
      }
    }
  }, [])

  return {
    status,
    opponent,
    roomId,
    gameState,
    result,
    roundResult,
    moveHistory,
    connect,
    send,
    disconnect,
    onMessage,
  }
}
