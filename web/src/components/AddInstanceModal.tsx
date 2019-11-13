import React, { useEffect, useState } from 'react';
import { Modal, Icon, Upload, message, Tabs, Input } from 'antd';
import { UploadProps, UploadChangeParam } from 'antd/lib/upload';
import { IInstance } from '@/models/inspection';
import { oriRequest } from '@/utils/request';
import { createInstance } from '@/services/inspection';

const { Dragger } = Upload;
const { TabPane } = Tabs;
const { TextArea } = Input;

const styles = require('./AddInstanceModal.less');

interface Props {
  visible: boolean;
  onClose: () => void;
  onData: (instance: IInstance) => void;
}

function AddInstanceModal({ visible, onClose, onData }: Props) {
  const [instanceConfig, setInstanceConfig] = useState('');
  const [configTemplate, setConfigTemplate] = useState('');
  const [configTemplateVisible, setConfigTemplateVisible] = useState(false);
  const [activeTab, setActiveTab] = useState('1');
  const [fileList, setFileList] = useState<any[]>([]);

  const uploadProps: UploadProps = {
    name: 'file',
    accept: '.ini',
    action: '/api/v1/instances/file',
    showUploadList: true,
    onChange(info: UploadChangeParam) {
      setFileList(info.fileList.slice(-1));
      const { status } = info.file;
      if (status === 'done') {
        message.success(`${info.file.name} 上传成功！`);
        onData(info.file.response as IInstance);
        onClose();
        setFileList([]);
      } else if (status === 'error') {
        message.error(`${info.file.name} 上传失败，错误：${info.file.response.message}`);
      }
    },
  };

  useEffect(() => {
    oriRequest('/add-instance-config-template.json').then(data => {
      setConfigTemplate(JSON.stringify(data, null, 2));
    });
  }, []);

  function handleOk() {
    let configObj;
    try {
      configObj = JSON.parse(instanceConfig);
    } catch (error) {
      message.error('配置内容不是合法的 json 格式！');
      return;
    }

    createInstance(configObj).then(res => {
      if (res !== undefined) {
        message.success('创建实例成功！');
        onData(res as IInstance);
        onClose();
        setInstanceConfig('');
      } else {
        message.error('创建实例失败！');
      }
    });
  }

  return (
    <Modal
      title="添加实例"
      visible={visible}
      onCancel={onClose}
      onOk={handleOk}
      okButtonProps={{ hidden: activeTab === '1', disabled: instanceConfig.length === 0 }}
    >
      <Tabs type="card" activeKey={activeTab} onChange={setActiveTab}>
        <TabPane tab="通过上传配置文件" key="1">
          <Dragger {...uploadProps} fileList={fileList}>
            <p className="ant-upload-drag-icon">
              <Icon type="inbox" />
            </p>
            <p className="ant-upload-text">点击或将文件拖拽到这里上传</p>
            <p className="ant-upload-hint">仅支持目标集群创建时的 tidb-ansible/inventory.ini</p>
          </Dragger>
        </TabPane>

        <TabPane tab="通过输入配置内容" key="2">
          <TextArea
            autosize={{ minRows: 6, maxRows: 20 }}
            value={instanceConfig}
            onChange={e => setInstanceConfig(e.target.value)}
          />
          <div className={styles.config_template_trigger}>
            配置内容模板示例
            <Icon
              type={configTemplateVisible ? 'caret-up' : 'caret-down'}
              onClick={() => setConfigTemplateVisible(prevVisible => !prevVisible)}
            />
          </div>
          {configTemplateVisible && (
            <TextArea autosize={{ minRows: 6, maxRows: 8 }} value={configTemplate} />
          )}
        </TabPane>
      </Tabs>
    </Modal>
  );
}

export default AddInstanceModal;
