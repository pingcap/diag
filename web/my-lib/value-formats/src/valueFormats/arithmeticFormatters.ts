import { toFixed } from './valueFormats';
import { DecimalCount } from '../types/displayValue';

// 百分比，进制转换

// 原始数值就表示百分比
// 单纯地加上 % 号
// 0.7 => 0.7%
export function toPercent(size: number, decimals: DecimalCount) {
  if (size === null) {
    return '';
  }
  return toFixed(size, decimals) + '%';
}

// 乘以 100 后加上 % 号
// 0.7 => 70%
export function toPercentUnit(size: number, decimals: DecimalCount) {
  if (size === null) {
    return '';
  }
  return toFixed(100 * size, decimals) + '%';
}

export function toHex0x(value: number, decimals: DecimalCount) {
  if (value == null) {
    return '';
  }
  const hexString = toHex(value, decimals);
  if (hexString.substring(0, 1) === '-') {
    return '-0x' + hexString.substring(1);
  }
  return '0x' + hexString;
}

export function toHex(value: number, decimals: DecimalCount) {
  if (value == null) {
    return '';
  }
  return parseFloat(toFixed(value, decimals))
    .toString(16)
    .toUpperCase();
}

export function sci(value: number, decimals: DecimalCount) {
  if (value == null) {
    return '';
  }
  return value.toExponential(decimals as number);
}
