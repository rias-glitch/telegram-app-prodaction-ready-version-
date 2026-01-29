export function Card({ children, className = '', onClick, variant = 'default', glow = false, ...props }) {
  const variants = {
    default: 'glass',
    light: 'glass-light',
    dark: 'glass-dark',
    gradient: 'gradient-border',
  }

  const baseClasses = `
    rounded-2xl p-4
    ${variants[variant]}
    ${onClick ? 'cursor-pointer card-hover active:scale-[0.98]' : ''}
    ${glow ? 'shadow-glow' : ''}
    transition-all duration-300
  `

  return (
    <div
      className={`${baseClasses} ${className}`}
      onClick={onClick}
      {...props}
    >
      {children}
    </div>
  )
}

export function CardHeader({ children, className = '' }) {
  return (
    <div className={`mb-3 ${className}`}>
      {children}
    </div>
  )
}

export function CardTitle({ children, className = '', gradient = false }) {
  return (
    <h3 className={`text-lg font-semibold ${gradient ? 'gradient-text' : ''} ${className}`}>
      {children}
    </h3>
  )
}

export function CardContent({ children, className = '' }) {
  return (
    <div className={className}>
      {children}
    </div>
  )
}

export function CardFooter({ children, className = '' }) {
  return (
    <div className={`mt-4 pt-4 border-t border-white/5 ${className}`}>
      {children}
    </div>
  )
}
