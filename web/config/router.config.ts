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
    path: '/',
    component: '../layouts/BasicLayout',
    routes: [
      {
        path: '/',
        redirect: '/inspection/instances',
      },
      {
        path: '/inspection',
        name: 'inspection',
        icon: 'dashboard',
        hideChildrenInMenu: true,
        routes: [
          {
            path: '/inspection',
            redirect: '/inspection/instances',
          },
          {
            path: '/inspection/instances',
            name: 'instance_list',
            component: './Inspection/InstanceList',
          },
          {
            path: '/inspection/instances/:id/reports',
            name: 'report_list',
            component: './Inspection/ReportList',
          },
          {
            path: '/inspection/reports/:id',
            name: 'report_detail',
            component: './Inspection/ReportDetail',
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
            redirect: '/misc/flamegraphs',
          },
          {
            path: '/misc/flamegraphs',
            name: 'flame_graph',
            component: './Misc/FlameGraphList',
          },
          {
            path: '/misc/flamegraphs/:id',
            name: 'flame_graph_detail',
            component: './Misc/FlameGraphDetail',
            hideInMenu: true,
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
        component: './404',
      },
    ],
  },
  {
    component: './404',
  },
];
