import React from 'react';
import { Form, Input, Icon, Button } from 'antd';
import { WrappedFormUtils } from 'antd/lib/form/Form';
import { connect } from 'dva';
import { ConnectProps, ConnectState, Dispatch } from '@/models/connect';

const styles = require('./Login.less');

interface LoginProps extends ConnectProps {
  form: WrappedFormUtils;
  dispatch: Dispatch;
  logging: boolean;
}

function Login({ form, dispatch, logging }: LoginProps) {
  const { getFieldDecorator } = form;

  function handleSubmit(e: any) {
    e.preventDefault();
    form.validateFields((err, values) => {
      console.log(values);
      if (!err) {
        dispatch({
          type: 'user/login',
          payload: values,
        });
      }
    });
  }

  return (
    <Form className={styles.container} onSubmit={handleSubmit}>
      <Form.Item>
        {getFieldDecorator('username', {
          rules: [{ required: true, message: '请输入用户名' }],
        })(
          <Input
            prefix={<Icon type="user" style={{ color: 'rgba(0,0,0,.25)' }} />}
            placeholder="username (admin or dba)"
          />,
        )}
      </Form.Item>
      <Form.Item>
        {getFieldDecorator('password', {
          rules: [{ required: true, message: '请输入密码' }],
        })(
          <Input
            prefix={<Icon type="lock" style={{ color: 'rgba(0,0,0,.25)' }} />}
            type="password"
            placeholder="password (tidb)"
          />,
        )}
      </Form.Item>
      <Form.Item>
        <Button type="primary" htmlType="submit" className={styles.action_btn} loading={logging}>
          登录
        </Button>
      </Form.Item>
    </Form>
  );
}

const WrappedLogin = Form.create({ name: 'login_form' })(Login);

export default connect(({ loading }: ConnectState) => ({
  logging: loading.effects['user/login'],
}))(WrappedLogin);
