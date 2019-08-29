import React, { useState, useEffect } from 'react';
import { Table } from 'antd';

import { IFlameGraph } from '@/models/misc';
import { queryFlamegraph } from '@/services/misc';

interface IFlame {
  component: string;
  address: string;
  svgFullPath: string;
  svgFileName: string;
}

const tableColumns = [
  {
    title: '组件',
    dataIndex: 'component',
    key: 'component',
  },
  {
    title: '机器',
    dataIndex: 'address',
    key: 'address',
  },
  {
    title: '图片',
    dataIndex: 'url',
    key: 'url',
    render: (text: any, record: IFlame) => (
      <a target="_blank" rel="noopener noreferrer" href={record.svgFullPath}>
        {record.svgFileName}
      </a>
    ),
  },
];

function genFlames(detail: IFlameGraph): IFlame[] {
  const flames: IFlame[] = [];
  detail.items.forEach(item => {
    item.flames.forEach(flame => {
      flames.push({
        component: item.component,
        address: item.address,
        svgFullPath: flame,
        svgFileName: flame.split('/').pop() || '',
      });
    });
  });
  return flames;
}

interface FlamegraphTableProps {
  reportId: string;
}

export default function FlameGraphTable({ reportId }: FlamegraphTableProps) {
  const [loading, setLoading] = useState(false);
  const [flames, setFlames] = useState<IFlame[]>([]);

  useEffect(() => {
    async function fetchDetail() {
      if (reportId) {
        setLoading(true);
        const res: IFlameGraph | undefined = await queryFlamegraph(reportId);
        setLoading(false);
        if (res) {
          setFlames(genFlames(res));
        }
      }
    }
    fetchDetail();
  }, []);

  return <Table dataSource={flames} columns={tableColumns} pagination={false} loading={loading} />;
}
