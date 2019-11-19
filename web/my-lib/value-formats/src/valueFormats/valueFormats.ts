import { getCategories } from './categories';
import { DecimalCount } from '../types/displayValue';

export type ValueFormatter = (
  value: number,
  decimals?: DecimalCount,
  scaledDecimals?: DecimalCount,
  isUtc?: boolean,
) => string;

export interface ValueFormat {
  name: string;
  id: string;
  fn: ValueFormatter;
}

export interface ValueFormatCategory {
  name: string;
  formats: ValueFormat[];
}

interface ValueFormatterIndex {
  [id: string]: ValueFormatter;
}

// Globals & formats cache
let categories: ValueFormatCategory[] = [];
const index: ValueFormatterIndex = {};
let hasBuiltIndex = false;

/////////////////////////////////////////////////////////////////

// 固定小数位数
export function toFixed(value: number, decimals?: DecimalCount): string {
  if (value === null) {
    return '';
  }
  if (value === Number.NEGATIVE_INFINITY || value === Number.POSITIVE_INFINITY) {
    return value.toLocaleString();
  }

  // 如果 decimals 为空，则 factor (系数因子?) 为 1 (即 10^0 = 1)
  const factor = decimals ? Math.pow(10, Math.max(0, decimals)) : 1;
  // Math.round() 进行四舍五入
  // 哦，这句话的作用就是用来抛弃掉多余的小数
  // 比如 12.346，约定精确度为保留小数 2 位，目标值是 12.35
  // 则先乘以 10^2 = 100，得到 1234.6，对它进行四舍五入，得到 1235，再重新除以 100，得到目标值 12.35
  // 那就这已经得到目标值了啊，为什么不是直接返回呢？
  // 考虑特殊情况 12.3，约定精确度为 2，预期值是 12.30，但到这一步只能得到 12.3
  const formatted = String(Math.round(value * factor) / factor);

  // if exponent return directly
  // 什么时候会有指数出现? 估计是前面 String 的作用
  if (formatted.indexOf('e') !== -1 || value === 0) {
    return formatted;
  }

  // If tickDecimals was specified, ensure that we have exactly that
  // much precision; otherwise default to the value's own precision.
  if (decimals != null) {
    // 补末尾的 0
    // 12.3，保留 2 位小数，则得到 12.30，末尾要补一个 0
    const decimalPos = formatted.indexOf('.');
    const precision = decimalPos === -1 ? 0 : formatted.length - decimalPos - 1;
    if (precision < decimals) {
      return (
        (precision ? formatted : formatted + '.') + String(factor).substr(1, decimals - precision)
      );
    }
  }

  return formatted;
}

export function toFixedScaled(
  value: number,
  decimals: DecimalCount,
  scaledDecimals: DecimalCount,
  additionalDecimals: number,
  ext?: string,
) {
  if (scaledDecimals === null || scaledDecimals === undefined) {
    return toFixed(value, decimals) + ext;
  } else {
    return toFixed(value, scaledDecimals + additionalDecimals) + ext;
  }

  return toFixed(value, decimals) + ext;
}

export function toFixedUnit(unit: string): ValueFormatter {
  return (size: number, decimals?: DecimalCount) => {
    if (size === null) {
      return '';
    }
    return toFixed(size, decimals) + ' ' + unit;
  };
}

// Formatter which scales the unit string geometrically according to the given
// numeric factor. Repeatedly scales the value down by the factor until it is
// less than the factor in magnitude, or the end of the array is reached.
// factor 是比例系数，比如计算磁盘大小的 1024，计算时间的 60 (可是时间有 60 和 1000 两个 factor 耶...)
export function scaledUnits(factor: number, extArray: string[]) {
  return (size: number, decimals?: DecimalCount, scaledDecimals?: DecimalCount) => {
    if (size === null) {
      return '';
    }
    if (size === Number.NEGATIVE_INFINITY || size === Number.POSITIVE_INFINITY || isNaN(size)) {
      return size.toLocaleString();
    }

    let steps = 0;
    const limit = extArray.length;

    // 循环计算直到 size 小于 factor，得到 steps，通过 steps 得到单位
    while (Math.abs(size) >= factor) {
      steps++;
      size /= factor;

      if (steps >= limit) {
        return 'NA';
      }
    }

    // 如果 scaledDecimals 不为空，则以 scaledDecimals 为准
    // 否则以 decimals 为准
    if (steps > 0 && scaledDecimals !== null && scaledDecimals !== undefined) {
      decimals = scaledDecimals + 3 * steps; // 为什么是乘以 3，大概是以 factor = 1000 为准，1000 表示 3 个小数点
    }

    return toFixed(size, decimals) + extArray[steps];
  };
}

export function locale(value: number, decimals: DecimalCount) {
  if (value == null) {
    return '';
  }
  return value.toLocaleString(undefined, { maximumFractionDigits: decimals as number });
}

export function simpleCountUnit(symbol: string) {
  const units = ['', 'K', 'M', 'B', 'T'];
  const scaler = scaledUnits(1000, units);
  return (size: number, decimals?: DecimalCount, scaledDecimals?: DecimalCount) => {
    if (size === null) {
      return '';
    }
    const scaled = scaler(size, decimals, scaledDecimals);
    return scaled + ' ' + symbol;
  };
}

/////////////////////////////////////////////////////////////////

function buildFormats() {
  categories = getCategories();

  for (const cat of categories) {
    for (const format of cat.formats) {
      index[format.id] = format.fn;
    }
  }

  hasBuiltIndex = true;
}

export function getValueFormat(id: string): ValueFormatter {
  if (!hasBuiltIndex) {
    buildFormats();
  }

  return index[id];
}

export function getValueFormatterIndex(): ValueFormatterIndex {
  if (!hasBuiltIndex) {
    buildFormats();
  }

  return index;
}

export function getValueFormats() {
  if (!hasBuiltIndex) {
    buildFormats();
  }

  return categories.map(cat => {
    return {
      text: cat.name,
      submenu: cat.formats.map(format => {
        return {
          text: format.name,
          value: format.id,
        };
      }),
    };
  });
}
