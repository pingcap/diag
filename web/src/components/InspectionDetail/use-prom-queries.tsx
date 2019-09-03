import { useEffect, useState } from 'react';
import { IPromQuery } from '@/services/prometheus-config';
import { IPromParams, promRangeQueries } from '@/services/prometheus-query';

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
      setChartLabels(labels);
      setOriChartData(data);
      setLoading(false);
    }

    query();
  }, [promQueries, promParams]);

  return [loading, chartLabels, oriChartData];
}
