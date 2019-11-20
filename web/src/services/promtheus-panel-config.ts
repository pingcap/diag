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
  title: string;
  targets: IPromConfigTarget[];
  yaxis: IPromConfigYaxis;
}

export interface IPromConfigTarget {
  expr: string;
  legendFormat: string;
}

// rename to unitFormat?
export interface IPromConfigYaxis {
  format: string;
  logBase: number;
  decimals?: number;
}

/* eslint-disable-next-line */
export const EMPHASIS_PROM_DETAIL = require('./prom-emphasis.json') as IPromConfigSection;
