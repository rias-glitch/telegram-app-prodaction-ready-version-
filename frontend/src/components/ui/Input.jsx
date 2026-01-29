export function Input({
  type = 'text',
  value,
  onChange,
  placeholder,
  disabled = false,
  error = false,
  icon,
  className = '',
  ...props
}) {
  return (
    <div className="relative">
      {icon && (
        <div className="absolute left-4 top-1/2 -translate-y-1/2 text-white/40">
          {icon}
        </div>
      )}
      <input
        type={type}
        value={value}
        onChange={onChange}
        placeholder={placeholder}
        disabled={disabled}
        className={`
          w-full
          bg-white/5
          border ${error ? 'border-danger/50' : 'border-white/10'}
          rounded-xl
          ${icon ? 'pl-12' : 'px-4'} pr-4 py-3
          text-white placeholder-white/30
          focus:outline-none focus:border-primary/50 focus:bg-white/[0.07]
          focus:ring-2 focus:ring-primary/20
          transition-all duration-200
          disabled:opacity-50 disabled:cursor-not-allowed
          ${className}
        `}
        {...props}
      />
      {error && typeof error === 'string' && (
        <p className="mt-1.5 text-sm text-danger">{error}</p>
      )}
    </div>
  )
}

export function BetInput({ value, onChange, min = 1, max, disabled = false, balance }) {
  const presets = [10, 50, 100, 500]

  const handlePreset = (amount) => {
    if (!disabled && (!max || amount <= max)) {
      onChange(amount)
    }
  }

  const handleHalf = () => {
    if (!disabled && value > 1) {
      onChange(Math.floor(value / 2))
    }
  }

  const handleDouble = () => {
    if (!disabled) {
      const doubled = value * 2
      onChange(max ? Math.min(doubled, max) : doubled)
    }
  }

  const handleMax = () => {
    if (!disabled && max) {
      onChange(max)
    }
  }

  return (
    <div className="space-y-3">
      <div className="flex items-center gap-2">
        <Input
          type="number"
          value={value}
          onChange={(e) => onChange(Math.max(min, parseInt(e.target.value) || 0))}
          min={min}
          max={max}
          disabled={disabled}
          className="text-center text-xl font-bold"
        />
        <span className="text-2xl">💎</span>
      </div>

      {/* Preset buttons */}
      <div className="flex gap-2">
        {presets.map((amount) => (
          <button
            key={amount}
            onClick={() => handlePreset(amount)}
            disabled={disabled || (max && amount > max)}
            className={`
              flex-1 py-2 rounded-xl text-sm font-medium
              transition-all duration-200
              ${value === amount
                ? 'bg-primary/20 text-primary border border-primary/30'
                : 'bg-white/5 text-white/60 border border-white/5 hover:bg-white/10 hover:text-white'
              }
              disabled:opacity-30 disabled:cursor-not-allowed
            `}
          >
            {amount}
          </button>
        ))}
      </div>

      {/* Quick actions */}
      <div className="flex gap-2">
        <button
          onClick={handleHalf}
          disabled={disabled || value <= 1}
          className="flex-1 py-1.5 rounded-lg text-xs font-medium bg-white/5 text-white/50 hover:bg-white/10 hover:text-white transition-colors disabled:opacity-30"
        >
          ½
        </button>
        <button
          onClick={handleDouble}
          disabled={disabled || (max && value * 2 > max)}
          className="flex-1 py-1.5 rounded-lg text-xs font-medium bg-white/5 text-white/50 hover:bg-white/10 hover:text-white transition-colors disabled:opacity-30"
        >
          2×
        </button>
        {max && (
          <button
            onClick={handleMax}
            disabled={disabled || value === max}
            className="flex-1 py-1.5 rounded-lg text-xs font-medium bg-white/5 text-white/50 hover:bg-white/10 hover:text-white transition-colors disabled:opacity-30"
          >
            MAX
          </button>
        )}
      </div>

      {/* Balance display */}
      {balance !== undefined && (
        <div className="text-center text-sm text-white/40">
          Balance: <span className="text-white/60 font-medium">{balance.toLocaleString()}</span> gems
        </div>
      )}
    </div>
  )
}

export function Select({ value, onChange, options, placeholder, disabled = false, className = '' }) {
  return (
    <select
      value={value}
      onChange={onChange}
      disabled={disabled}
      className={`
        w-full
        bg-white/5
        border border-white/10
        rounded-xl
        px-4 py-3
        text-white
        focus:outline-none focus:border-primary/50
        focus:ring-2 focus:ring-primary/20
        transition-all duration-200
        disabled:opacity-50 disabled:cursor-not-allowed
        appearance-none
        cursor-pointer
        ${className}
      `}
    >
      {placeholder && (
        <option value="" disabled className="bg-dark-100">
          {placeholder}
        </option>
      )}
      {options.map((opt) => (
        <option key={opt.value} value={opt.value} className="bg-dark-100">
          {opt.label}
        </option>
      ))}
    </select>
  )
}
