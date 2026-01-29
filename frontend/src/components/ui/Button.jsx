export function Button({
  children,
  variant = 'primary',
  size = 'md',
  disabled = false,
  loading = false,
  glow = false,
  className = '',
  onClick,
  ...props
}) {
  const baseClasses = `
    font-semibold rounded-xl
    transition-all duration-200
    active:scale-95
    disabled:opacity-50 disabled:cursor-not-allowed disabled:active:scale-100
    relative overflow-hidden
  `

  const variants = {
    primary: `
      bg-gradient-to-r from-primary to-secondary text-white
      hover:shadow-glow hover:from-primary-light hover:to-secondary-light
      btn-glow
    `,
    secondary: `
      bg-white/5 text-white border border-white/10
      hover:bg-white/10 hover:border-white/20
    `,
    success: `
      bg-gradient-to-r from-success to-emerald-400 text-white
      hover:shadow-glow-success hover:from-success-light hover:to-emerald-300
    `,
    danger: `
      bg-gradient-to-r from-danger to-pink-500 text-white
      hover:shadow-glow-danger hover:from-danger-light hover:to-pink-400
    `,
    ghost: `
      bg-transparent text-white/70
      hover:bg-white/5 hover:text-white
    `,
    outline: `
      bg-transparent border border-primary/50 text-primary
      hover:bg-primary/10 hover:border-primary
    `,
  }

  const sizes = {
    xs: 'px-2.5 py-1.5 text-xs',
    sm: 'px-3 py-2 text-sm',
    md: 'px-4 py-2.5 text-base',
    lg: 'px-6 py-3 text-lg',
    xl: 'px-8 py-4 text-xl',
  }

  return (
    <button
      className={`
        ${baseClasses}
        ${variants[variant]}
        ${sizes[size]}
        ${glow ? 'shadow-glow animate-glow-pulse' : ''}
        ${className}
      `}
      disabled={disabled || loading}
      onClick={onClick}
      {...props}
    >
      {loading ? (
        <span className="flex items-center justify-center gap-2">
          <svg className="animate-spin h-4 w-4" viewBox="0 0 24 24">
            <circle
              className="opacity-25"
              cx="12"
              cy="12"
              r="10"
              stroke="currentColor"
              strokeWidth="4"
              fill="none"
            />
            <path
              className="opacity-75"
              fill="currentColor"
              d="M4 12a8 8 0 018-8V0C5.373 0 0 5.373 0 12h4zm2 5.291A7.962 7.962 0 014 12H0c0 3.042 1.135 5.824 3 7.938l3-2.647z"
            />
          </svg>
          Loading...
        </span>
      ) : (
        children
      )}
    </button>
  )
}

export function IconButton({
  children,
  variant = 'ghost',
  size = 'md',
  className = '',
  ...props
}) {
  const sizes = {
    sm: 'w-8 h-8 text-sm',
    md: 'w-10 h-10 text-base',
    lg: 'w-12 h-12 text-lg',
  }

  return (
    <Button
      variant={variant}
      className={`${sizes[size]} p-0 flex items-center justify-center ${className}`}
      {...props}
    >
      {children}
    </Button>
  )
}
