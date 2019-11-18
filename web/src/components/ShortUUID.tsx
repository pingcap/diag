import React from 'react';
import { Tooltip } from 'antd';

export default function ShortUUID({ uuid }: { uuid: string }) {
  return (
    <Tooltip title={uuid}>
      <span style={{ cursor: 'pointer' }}>{uuid.slice(0, 6)}...</span>
    </Tooltip>
  );
}
