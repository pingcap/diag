export interface IUnitConfig {
  unitFormat: string;
  decimals?: number; // precision
}

export interface IColumnsUnit {
  [key: string]: IUnitConfig;
}

export interface IReportDetailConfigPanel {
  title: string;

  // data
  apiUrl: string;
  dataType: 'obj' | 'arr';

  // style
  collapse?: boolean;
  titleColor?: string; // default is 'inherit'
  limitHeight?: boolean;
  height?: number;

  // unit
  columnsUnit?: IColumnsUnit;

  // less more columns
  // if the content of one column is too much, we display only a few content at first,
  // then we can toggle to display more
  lessMoreColumns?: string[];
}

export interface IReportDetailConfigSection {
  sectionKey: string;
  title: string;
  panels: IReportDetailConfigPanel[];
}

export type ReportDetailConfig = IReportDetailConfigSection[];

export const INSPECTION_DETAILS: ReportDetailConfig = [
  {
    sectionKey: 'global_overview',
    title: '一、全局诊断',
    panels: [
      {
        title: 'overview',
        apiUrl: '/symptom',
        dataType: 'arr',
        titleColor: 'red',
      },
    ],
  },
  {
    sectionKey: 'basic_info',
    title: '二、基本信息',
    panels: [
      {
        title: '1、基本信息',
        apiUrl: '/basic',
        dataType: 'obj',
      },
      {
        title: '2、数据库基本信息',
        apiUrl: '/dbinfo',
        dataType: 'arr',
        collapse: true,
        limitHeight: true,
        height: 400,
      },
      {
        title: '3、资源信息 (使用率%)',
        apiUrl: '/resource',
        dataType: 'arr',
      },
      {
        title: '4、告警信息',
        apiUrl: '/alert',
        dataType: 'arr',
        lessMoreColumns: ['description'],
      },
      {
        title: '5、慢查询信息',
        apiUrl: '/slowlog',
        dataType: 'arr',
      },
      {
        title: '6、硬件信息',
        apiUrl: '/hardware',
        dataType: 'arr',
      },
      {
        title: '7、软件信息',
        apiUrl: '/software',
        dataType: 'arr',
      },
      {
        title: '8、软件配置信息',
        apiUrl: '/config',
        dataType: 'arr',
        lessMoreColumns: ['config'],
      },
      {
        title: '9、机器 NTP 时间同步信息',
        apiUrl: '/ntp',
        dataType: 'arr',
        columnsUnit: {
          offset: {
            unitFormat: 's',
            decimals: 3,
          },
        },
      },
      {
        title: '10、网络质量信息',
        apiUrl: '/network',
        dataType: 'arr',
        columnsUnit: {
          recv: {
            unitFormat: 'bytes',
            decimals: 2,
          },
          send: {
            unitFormat: 'bytes',
            decimals: 2,
          },
          max_duration: {
            unitFormat: 's',
            decimals: 2,
          },
          min_duration: {
            unitFormat: 's',
            decimals: 2,
          },
          avg_duration: {
            unitFormat: 's',
            decimals: 2,
          },
        },
      },
      {
        title: '11、集群拓扑结构信息',
        apiUrl: '/topology',
        dataType: 'arr',
      },
      {
        title: '12、dmesg 信息',
        apiUrl: '/dmesg',
        dataType: 'arr',
        collapse: true,
        limitHeight: true,
        height: 400,
      },
    ],
  },
];

export const EMPHASIS_DETAILS: ReportDetailConfig = [
  {
    sectionKey: 'global_overview',
    title: '一、问题定位',
    panels: [
      {
        title: 'overview',
        apiUrl: '/symptom',
        dataType: 'arr',
        titleColor: 'red',
      },
    ],
  },
];
