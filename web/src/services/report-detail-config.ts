export interface IReportDetailConfigPanel {
  panelTitle: string;

  // data
  apiUrl: string;
  dataType: 'obj' | 'arr';

  // style
  collapse?: boolean;
  limitHeight?: boolean;
  height?: number;

  // unit
  columnUnits?: { [key: string]: string };
}

export interface IReportDetailConfigSection {
  sectionKey: string;
  sectionTitle: string;
  panels: IReportDetailConfigPanel[];
}

export type ReportDetailConfig = IReportDetailConfigSection[];

export const INSPECTION_DETAILS: ReportDetailConfig = [
  {
    sectionKey: 'global_overview',
    sectionTitle: '一、全局诊断',
    panels: [
      {
        panelTitle: 'overview',
        apiUrl: '/symptom',
        dataType: 'arr',
      },
    ],
  },
  {
    sectionKey: 'basic_info',
    sectionTitle: '二、基本信息',
    panels: [
      {
        panelTitle: '1、基本信息',
        apiUrl: '/basic',
        dataType: 'obj',
      },
      {
        panelTitle: '2、数据库基本信息',
        apiUrl: '/dbinfo',
        dataType: 'arr',
        collapse: true,
      },
      {
        panelTitle: '3、资源信息 (使用率%)',
        apiUrl: '/resource',
        dataType: 'arr',
      },
      {
        panelTitle: '4、告警信息',
        apiUrl: '/alert',
        dataType: 'arr',
      },
      {
        panelTitle: '5、慢查询信息',
        apiUrl: '/slowlog',
        dataType: 'arr',
      },
      {
        panelTitle: '6、硬件信息',
        apiUrl: '/hardware',
        dataType: 'arr',
      },
      {
        panelTitle: '7、软件信息',
        apiUrl: '/software',
        dataType: 'arr',
      },
      {
        panelTitle: '8、软件配置信息',
        apiUrl: '/config',
        dataType: 'arr',
      },
      {
        panelTitle: '9、机器 NTP 时间同步信息',
        apiUrl: '/ntp',
        dataType: 'arr',
      },
      {
        panelTitle: '10、网络质量信息',
        apiUrl: '/network',
        dataType: 'arr',
      },
      {
        panelTitle: '11、集群拓扑结构信息',
        apiUrl: '/topology',
        dataType: 'arr',
      },
      {
        panelTitle: '12、dmesg 信息',
        apiUrl: '/dmesg',
        dataType: 'arr',
        collapse: true,
      },
    ],
  },
];
