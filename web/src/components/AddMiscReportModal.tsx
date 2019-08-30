import React, { useState } from 'react';
import { Modal, message, Select } from 'antd';
import { IFormatInstance } from '@/models/inspection';
import { queryInstanceComponents } from '@/services/misc';

const { Option } = Select;
const styles = require('./AddMiscReportModal.less');

interface IComponent {
  name: string;
  ip: string;
  port: string;
}

interface Props {
  visible: boolean;
  onClose: () => void;

  instances: IFormatInstance[];

  onData: (instanceId: string, node: string) => Promise<any>;
}

function AddMiscReportModal({ visible, onClose, instances, onData }: Props) {
  const [submitting, setSubmitting] = useState(false);
  const [instanceId, setInstanceId] = useState('');

  const [nodeList, setNodeList] = useState<IComponent[]>([]);
  const [node, setNode] = useState('');

  async function submit() {
    setSubmitting(true);
    try {
      await onData(instanceId, node);
      onClose();
      message.success(`正在收集 ${instanceId} 的新报告！`);
    } catch (err) {
      // TODO
    }
    setSubmitting(false);
  }

  async function fetchNodeList(instId: string) {
    const res: IComponent[] | undefined = await queryInstanceComponents(instId);
    setNodeList(res || []);
  }

  function handleInstanceChange(value: string | undefined) {
    // 如果用户对 select 进行了 clear，value 为 undefined
    setInstanceId(value || '');

    // reset
    setNodeList([]);
    if (node !== 'all') {
      setNode('');
    }
    if (value) {
      fetchNodeList(value);
    }
  }

  function handleNodeChange(value: string | undefined) {
    setNode(value || '');
  }

  return (
    <Modal
      visible={visible}
      onCancel={onClose}
      onOk={submit}
      title="获取报告"
      confirmLoading={submitting}
      okButtonProps={{ disabled: instanceId === '' || node === '' }}
    >
      <div className={styles.modal_item}>
        <span>选择集群实例</span>
        <Select style={{ width: 200 }} onChange={handleInstanceChange}>
          {instances.map(item => (
            <Option value={item.uuid} key={item.uuid}>
              {item.name}
            </Option>
          ))}
        </Select>
      </div>
      <div className={styles.modal_item}>
        <span>选择节点</span>
        <Select style={{ width: 200 }} onChange={handleNodeChange}>
          {nodeList.map(item => (
            <Option value={`${item.name}:${item.ip}:${item.port}`} key={JSON.stringify(item)}>
              {item.name}-{item.ip}:{item.port}
            </Option>
          ))}
          <Option value="all">ALL</Option>
        </Select>
      </div>
    </Modal>
  );
}

export default AddMiscReportModal;
