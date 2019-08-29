import React, { useState } from 'react';
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
  const [loginFailed, setLoginFailed] = useState(false);
  const { getFieldDecorator } = form;

  function handleSubmit(e: any) {
    // setLoginFailed(false);
    e.preventDefault();
    form.validateFields((err, values) => {
      if (!err) {
        dispatch({
          type: 'user/login',
          payload: values,
        }).then((res: any) => {
          setLoginFailed(res === undefined);
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
            placeholder="username"
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
            placeholder="password"
          />,
        )}
        {loginFailed && <span className={styles.error}>* 用户名或密码错误，请联系后台管理员</span>}
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
