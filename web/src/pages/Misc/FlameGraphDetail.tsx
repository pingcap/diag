import React, { useState, useEffect } from 'react';
import { Button, Modal, Table } from 'antd';
import { router } from 'umi';
import { connect } from 'dva';
import { ConnectProps, Dispatch, ConnectState } from '@/models/connect';
import { CurrentUser } from '@/models/user';
import UploadRemoteReportModal from '@/components/UploadRemoteReportModal';
import { IFlameGraph } from '@/models/misc';
import { queryFlamegraph } from '@/services/misc';

const styles = require('../style.less');

interface ReportDetailProps extends ConnectProps {
  dispatch: Dispatch;

  curUser: CurrentUser;
}

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

function FlameGraphDetail({ dispatch, match, curUser }: ReportDetailProps) {
  const reportId: string | undefined = match && match.params && (match.params as any).id;

  const [loading, setLoading] = useState(false);
  const [flames, setFlames] = useState<IFlame[]>([]);

  const [uploadRemoteModalVisible, setUploadRemoteModalVisible] = useState(false);

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

  function deleteFlamegraph() {
    Modal.confirm({
      title: '删除报告？',
      content: '你确定要删除这个报告吗？删除后不可恢复',
      okText: '删除',
      okButtonProps: { type: 'danger' },
      onOk() {
        dispatch({
          type: 'misc/deleteFlamegraph',
          payload: reportId,
        }).then((ret: boolean) => ret && router.goBack());
      },
    });
  }

  return (
    <div className={styles.container}>
      <div className={styles.list_header}>
        <h3>报告：{reportId}</h3>
        <div className={styles.space}></div>
        {curUser.role === 'admin' && (
          <React.Fragment>
            <Button
              type="primary"
              style={{ marginRight: 20 }}
              onClick={() => setUploadRemoteModalVisible(true)}
            >
              上传
            </Button>
            <Button type="primary" style={{ marginRight: 20 }}>
              <a download href={`/api/v1/perfprofiles/${reportId}.tar.gz`}>
                下载
              </a>
            </Button>
          </React.Fragment>
        )}
        <Button type="danger" onClick={deleteFlamegraph}>
          删除
        </Button>
      </div>
      <Table dataSource={flames} columns={tableColumns} pagination={false} loading={loading} />
      <UploadRemoteReportModal
        visible={uploadRemoteModalVisible}
        onClose={() => setUploadRemoteModalVisible(false)}
        uploadUrl={`/perfprofiles/${reportId}`}
      />
    </div>
  );
}

export default connect(({ user }: ConnectState) => ({ curUser: user.currentUser }))(
  FlameGraphDetail,
);
