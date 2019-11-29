import React, { useEffect, useState } from 'react';

import { Tooltip, Icon } from 'antd';
import request from '@/utils/request';
import { formatDatetime } from '@/utils/datetime-util';
import { IColumnsUnit } from './report-detail-config';
import { getValueFormat } from 'value-formats';

interface IResObj {
  [key: string]: any;
}

interface IConclusion {
  status: 'error' | 'warning' | 'info';
  message: string;
}

interface IResConclusionWithData {
  conclusion: IConclusion[];
  data: any[];
}

type IResReportItem = IResObj | IResConclusionWithData | undefined;

interface IAbnormalValue {
  status: 'error' | 'warning' | 'info';
  message: string;
  value: string | number;
}

export function useReportItemQuery(
  fullApiUrl: string,
  dataType: 'obj' | 'arr',
  lessMoreColumns: string[],
  columnsUnit: IColumnsUnit,
): [IConclusion[], any[], any[], boolean] {
  const [conclusion, setConclusion] = useState<IConclusion[]>([]);
  const [tableColumns, setTableColumns] = useState<any[]>([]);
  const [dataSource, setDataSource] = useState<any[]>([]);
  const [hasAbnormal, setHasAbnormal] = useState(false);

  function toggleExpand(cellKey: string, expand: boolean) {
    const configCellEl = document.getElementById(`config_table_cell_${cellKey}`);
    if (configCellEl) {
      if (expand) {
        configCellEl.classList.add('show_detail');
      } else {
        configCellEl.classList.remove('show_detail');
      }
    }
  }

  function formatValueWithUnit(columnKey: string, value: any) {
    const unitConfig = columnsUnit[columnKey];
    if (unitConfig === undefined) {
      return value;
    }

    const formatFunc = getValueFormat(unitConfig.unitFormat);
    if (formatFunc === undefined) {
      return value;
    }

    if (unitConfig.decimals !== undefined) {
      return formatFunc(value, unitConfig.decimals, null);
    }
    return formatFunc(value, 2, null);
  }

  function parseResAsArr(res: IResConclusionWithData) {
    let containsAbnormal =
      (res.conclusion as IConclusion[]).find(item => item.status === 'error') !== undefined;
    const columns = Object.keys(res.data[0] || {}).map(key => ({
      title: key,
      dataIndex: key,
      key,
      render: (text: any, record: any) => {
        if (text === null) {
          return <span style={{ whiteSpace: 'pre-wrap' }}>none</span>;
        }
        if (text.status === 'error') {
          containsAbnormal = true;
          return (
            <div style={{ display: 'flex' }}>
              <span style={{ color: 'red', marginRight: '8px', whiteSpace: 'pre-wrap' }}>
                {formatValueWithUnit(key, (text as IAbnormalValue).value)}
              </span>
              {(text as IAbnormalValue).message && (
                <Tooltip title={(text as IAbnormalValue).message}>
                  <Icon style={{ paddingTop: '2px' }} type="question-circle" />
                </Tooltip>
              )}
            </div>
          );
        }
        if (text.status === 'warning' || text.status === 'info') {
          return (
            <span style={{ whiteSpace: 'pre-wrap' }}>{formatValueWithUnit(key, text.value)}</span>
          );
        }
        // convert the server time format to local time
        if (key === 'time' || key.endsWith('_time')) {
          text = formatDatetime(text);
        }
        if (lessMoreColumns.includes(key)) {
          return (
            <div className="config_table_cell" id={`config_table_cell_${record.key}_${key}`}>
              <div className="config_table_cell_summary">
                <a onClick={(e: any) => toggleExpand(`${record.key}_${key}`, true)}>more</a>
                <br />
                <span>{text.slice(0, 50)}...</span>
              </div>
              <div className="config_table_cell_detail">
                <a onClick={(e: any) => toggleExpand(`${record.key}_${key}`, false)}>less</a>
                <br />
                <span style={{ whiteSpace: 'pre-wrap' }}>{text}</span>
              </div>
            </div>
          );
        }
        return <span style={{ whiteSpace: 'pre-wrap' }}>{formatValueWithUnit(key, text)}</span>;
      },
    }));
    const dataArr = (res as IResConclusionWithData).data.map((item, index) => ({
      ...item,
      key: `${index}`,
    }));
    setTableColumns(columns);
    setDataSource(dataArr);
    setConclusion(res.conclusion);
    setHasAbnormal(containsAbnormal);
  }

  function parseResAsObj(res: IResObj) {
    const columns = [
      {
        title: '信息',
        dataIndex: 'field',
        key: 'field',
        render: (text: any) => <span style={{ whiteSpace: 'pre-wrap' }}>{text}</span>,
      },
      {
        title: '值',
        dataIndex: 'value',
        key: 'value',
        render: (text: any) => <span style={{ whiteSpace: 'pre-wrap' }}>{text}</span>,
      },
    ];
    const dataArr = Object.keys(res).map(key => ({
      field: key,
      // convert the server time format to local time
      value: key === 'time' || key.endsWith('_time') ? formatDatetime(res[key]) : res[key],
      key,
    }));
    setTableColumns(columns);
    setDataSource(dataArr);
  }

  useEffect(() => {
    async function fetchData() {
      const res: IResReportItem = await request(fullApiUrl);
      if (res !== undefined) {
        // exist res.data filed, and res.data is an array
        if (res.data) {
          parseResAsArr(res as IResConclusionWithData);
        } else if (dataType === 'obj') {
          // res.data doesn't exist
          // res is an object
          parseResAsObj(res as IResObj);
        }
      }
    }

    if (fullApiUrl !== '') {
      fetchData();
    }
  }, [fullApiUrl]);

  return [conclusion, tableColumns, dataSource, hasAbnormal];
}
