import { getValueFormat } from 'value-formats';

// export interface IPromConfigSection {
//   sectionKey: string,
//   title: string,

//   panels: [
//     {
//       panelKey: string,
//       title: string,
//       expand?: boolean,

//       subPanels: [
//         {
//           subPanelKey: string,
//           title: string,

//           targets: [
//             {
//               expr: string,
//               legendFormat: string,
//             }
//           ],
//           yaxis: {
//             format: string,
//             logBase: number,
//             decimals?: number,
//           }
//         }
//       ]
//     }
//   ]
// }
export interface IPromConfigSection {
  sectionKey: string;
  title: string;

  panels: IPromConfigPanel[];
}

export interface IPromConfigPanel {
  panelKey: string;
  title: string;
  expand?: boolean;

  subPanels: IPromConfigSubPanel[];
}

export interface IPromConfigSubPanel {
  subPanelKey: string;

  subPanelType?: 'line' | 'table';
  tableColumns?: [string, string];

  title: string;
  targets: IPromConfigTarget[];
  yaxis: IPromConfigYaxis;
}

export interface IPromConfigTarget {
  expr: string;
  legendFormat: string;
  desc?: string; // 'v_2_x', 'v_3_x'
}

// rename to unitFormat?
export interface IPromConfigYaxis {
  format: string;
  logBase: number;
  decimals?: number;
}

/* eslint-disable-next-line */
export const EMPHASIS_PROM_DETAIL = require('./prom-emphasis.json') as IPromConfigSection;
/* eslint-disable-next-line */
export const INSPECTION_PROM_DETAIL = require('./prom-inspection.json') as IPromConfigSection;

export function genNumberConverter(yaxis: IPromConfigYaxis) {
  const formatFunc = getValueFormat(yaxis.format);
  const valConverter = (val: number): string => {
    let { decimals } = yaxis;
    if (decimals === undefined) {
      decimals = 2;
    }
    return formatFunc(val, decimals);
  };
  return valConverter;
}
