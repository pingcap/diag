import { useEffect, useState } from 'react';
import { IPromQuery, IPromParams, promRangeQueries } from './prom-query';

function loopGenUniqName(
  oriName: string,
  existNames: string[],
  tryCount: number = 1,
): [string, number] {
  const newName: string = tryCount > 1 ? `${oriName}-${tryCount}` : oriName;
  if (!existNames.includes(newName)) {
    return [newName, tryCount];
  }
  return loopGenUniqName(oriName, existNames, tryCount + 1);
}

// rename the same labels
// oriLabels 中可能包含相同的元素，不能用于图表绘制，所以需要进行处理
// input:  ['timestamp', 'foo', 'bar', 'foo',   'bar',   'foo']
// oputpu: ['timestamp', 'foo', 'bar', 'foo-1', 'bar-1', 'foo-2']
export function uniqLabels(oriLabels: string[]): string[] {
  let newLabels: string[] = [];
  const duplicatedLabels: string[] = [];
  oriLabels.forEach(oriLabel => {
    const [newLabel, tryCount] = loopGenUniqName(oriLabel, newLabels);
    if (tryCount === 2) {
      duplicatedLabels.push(oriLabel);
    }
    newLabels.push(newLabel);
  });
  if (duplicatedLabels.length > 0) {
    newLabels = newLabels.map(label => (duplicatedLabels.includes(label) ? `${label}-1` : label));
  }
  return newLabels;
}

export function usePromQueries(
  promQueries: IPromQuery[],
  promParams: IPromParams,
): [boolean, string[], number[][]] {
  const [loading, setLoading] = useState(false);
  const [chartLabels, setChartLabels] = useState<string[]>([]);
  const [oriChartData, setOriChartData] = useState<number[][]>([]);

  useEffect(() => {
    async function query() {
      setLoading(true);
      const { labels, data } = await promRangeQueries(promQueries, promParams);
      setChartLabels(uniqLabels(labels));
      setOriChartData(data);
      setLoading(false);
    }

    query();
  }, [promQueries, promParams]);

  return [loading, chartLabels, oriChartData];
}
