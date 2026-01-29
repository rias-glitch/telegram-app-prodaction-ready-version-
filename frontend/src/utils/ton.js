import { Address } from '@ton/core'

/**
 * Конвертирует адрес TON в user-friendly формат
 * @param {string} address - адрес в любом формате (raw или user-friendly)
 * @param {boolean} bounceable - true для EQ..., false для UQ...
 * @returns {string} адрес в user-friendly формате
 */
export function toUserFriendlyAddress(address, bounceable = false) {
  if (!address) return ''

  try {
    const parsed = Address.parse(address)
    return parsed.toString({
      bounceable,
      testOnly: false,
    })
  } catch (err) {
    console.error('Failed to parse address:', err)
    // Если не удалось распарсить, возвращаем как есть
    return address
  }
}

/**
 * Сокращает адрес для отображения
 * @param {string} address - полный адрес
 * @param {number} startChars - количество символов в начале
 * @param {number} endChars - количество символов в конце
 * @returns {string} сокращённый адрес
 */
export function shortenAddress(address, startChars = 6, endChars = 4) {
  if (!address) return ''
  if (address.length <= startChars + endChars + 3) return address
  return `${address.slice(0, startChars)}...${address.slice(-endChars)}`
}
