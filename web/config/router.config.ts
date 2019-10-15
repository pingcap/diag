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
        hideChildrenInMenu: true,
        routes: [
          {
            path: '/inspection',
            redirect: '/inspection/reports',
          },
          {
            path: '/inspection/instances/:id/reports',
            name: 'report_list',
            component: './Inspection/ReportList',
          },
          {
            path: '/inspection/instances/:instanceId/reports/:id',
            name: 'report_detail',
            component: './Inspection/ReportDetail',
          },
          {
            path: '/inspection/reports/:id',
            name: 'report_detail',
            component: './Inspection/ReportDetail',
          },
          {
            path: '/inspection/reports',
            name: 'report_list',
            component: './Inspection/ReportList',
          },
        ],
      },
      {
        path: '/misc',
        name: 'misc',
        icon: 'dashboard',
        routes: [
          {
            path: '/misc',
            redirect: '/misc/perfprofiles',
          },
          {
            path: '/misc/perfprofiles',
            name: 'perf_profile',
            component: './Misc/PerfProfileList',
          },
          {
            path: '/misc/perfprofiles/:id',
            name: 'perf_profile_detail',
            component: './Misc/PerfProfileDetail',
            hideInMenu: true,
          },
        ],
      },
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
