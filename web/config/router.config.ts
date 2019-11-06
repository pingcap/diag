export default [
  {
    path: '/user',
    component: '../layouts/UserLayout',
    routes: [
      { path: '/user', redirect: '/user/login' },
      { path: '/user/login', name: 'login', component: './User/Login' },
      {
        component: '404',
      },
    ],
  },
  {
    path: '/home',
    component: '../layouts/BlankLayout',
    routes: [
      {
        path: '/home',
        component: './Home/HomePage',
      },
    ],
  },
  {
    path: '/',
    component: '../layouts/BasicLayout',
    routes: [
      {
        path: '/',
        redirect: '/home',
      },
      {
        path: '/instances',
        name: 'instances',
        icon: 'dashboard',
        component: './Cluster/InstanceList',
        authority: ['admin'],
      },
      {
        path: '/inspection',
        name: 'inspection',
        icon: 'dashboard',
        routes: [
          {
            path: '/inspection',
            redirect: '/inspection/reports',
          },
          {
            path: '/inspection/reports',
            name: 'report_list',
            component: './Inspection/ReportList',
          },
          {
            path: '/inspection/instances/:id/reports',
            name: 'report_list',
            component: './Inspection/ReportList',
            hideInMenu: true,
          },
          {
            path: '/inspection/instances/:instanceId/reports/:id',
            name: 'report_detail',
            component: './Inspection/ReportDetail',
            hideInMenu: true,
          },
          {
            path: '/inspection/reports/:id',
            name: 'report_detail',
            component: './Inspection/ReportDetail',
            hideInMenu: true,
          },
          {
            path: '/inspection/emphasis',
            name: 'emphasis',
            component: './Emphasis/EmphasisList',
          },
          {
            path: '/inspection/instances/:id/emphasis',
            name: 'emphasis',
            component: './Emphasis/EmphasisList',
            hideInMenu: true,
          },
          {
            path: '/inspection/emphasis/:id',
            name: 'emphasis_detail',
            component: './Emphasis/EmphasisDetail',
            hideInMenu: true,
          },
          {
            path: '/inspection/instances/:instanceId/emphasis/:id',
            name: 'emphasis_detail',
            component: './Emphasis/EmphasisDetail',
            hideInMenu: true,
          },
          {
            path: '/inspection/perfprofiles',
            name: 'perf_profile',
            component: './Misc/PerfProfileList',
          },
          {
            path: '/inspection/perfprofiles/:id',
            name: 'perf_profile_detail',
            component: './Misc/PerfProfileDetail',
            hideInMenu: true,
          },
        ],
      },
      // {
      //   path: '/misc',
      //   name: 'misc',
      //   icon: 'dashboard',
      //   routes: [
      //     {
      //       path: '/misc',
      //       redirect: '/misc/perfprofiles',
      //     },
      //     {
      //       path: '/misc/perfprofiles',
      //       name: 'perf_profile',
      //       component: './Misc/PerfProfileList',
      //     },
      //     {
      //       path: '/misc/perfprofiles/:id',
      //       name: 'perf_profile_detail',
      //       component: './Misc/PerfProfileDetail',
      //       hideInMenu: true,
      //     },
      //   ],
      // },
      {
        path: '/logs',
        name: 'logs',
        icon: 'dashboard',
        component: './Log/LogList',
      },
      {
        component: './404',
      },
    ],
  },
  {
    component: './404',
  },
];
