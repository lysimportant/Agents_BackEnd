'use client';

import { useEffect, useState, type FormEvent } from 'react';
import { LockOutlined, MailOutlined, PhoneOutlined, SafetyCertificateOutlined, UserOutlined } from '@ant-design/icons';
import { Alert, App, Avatar, Button, Input, InputNumber, Modal, Skeleton, Tag } from 'antd';
import { API_BASE_URL } from '@/src/config/constants';
import { requestWithSession } from '@/src/services/api';
import type { AuthUser, ProfileForm, User } from '@/src/types/admin';
import styles from './ProfileDialog.module.css';

type ProfilePageProps = {
  /** authUser 表示认证用户。 */
  authUser: AuthUser;
  /** onUpdated 表示更新时间。 */
  onUpdated: (user: User) => void;
  /** onPasswordChanged 表示密码。 */
  onPasswordChanged: () => void;
};

type PasswordForm = {
  /** code 表示编码。 */
  code: string;
  /** newPassword 表示密码。 */
  newPassword: string;
  /** confirmPassword 表示密码。 */
  confirmPassword: string;
};

/** toProfileForm 实现对应业务逻辑。 */
function toProfileForm(user: AuthUser | User): ProfileForm {
  return {
    name: user.name ?? '',
    email: user.email ?? '',
    phone: user.phone ?? '',
    age: Number(user.age) || 0,
    description: user.description ?? '',
    avatarUrl: user.avatarUrl ?? '',
  };
}

/** parseProfileError 解析对应业务数据。 */
async function parseProfileError(response: Response, fallback: string) {
  try {
    /** payload 保存请求载荷。 */
    const payload = await response.json() as { error?: string };
    return payload.error || fallback;
  } catch {
    return fallback;
  }
}

/** requestProfile 实现对应业务逻辑。 */
async function requestProfile(userId: number, init: RequestInit = {}) {
  /** paths 保存路径。 */
  const paths = ['/api/profile', `/api/users/${userId}/profile`];
  for (let index = 0; index < paths.length; index += 1) {
    /** response 保存接口响应及其关联状态。 */
    const response = await requestWithSession(`${API_BASE_URL}${paths[index]}`, init);
    if (index === 0 && (response.status === 404 || response.status === 405)) continue;
    return response;
  }
  throw new Error('个人资料接口不可用');
}

/** ProfilePage 实现对应业务逻辑。 */
export function ProfilePage({ authUser, onUpdated, onPasswordChanged }: ProfilePageProps) {
  /** message 保存消息。 */
  const { message } = App.useApp();
  /** profile、setProfile 保存个人资料、个人资料。 */
  const [profile, setProfile] = useState<User | null>(null);
  /** form、setForm 负责计算或维护表单。 */
  const [form, setForm] = useState<ProfileForm>(() => toProfileForm(authUser));
  /** passwordForm、setPasswordForm 保存密码表单、密码表单。 */
  const [passwordForm, setPasswordForm] = useState<PasswordForm>({ code: '', newPassword: '', confirmPassword: '' });
  /** isLoading、setIsLoading 分别保存加载状态状态及其更新函数。 */
  const [isLoading, setIsLoading] = useState(false);
  /** isSaving、setIsSaving 分别保存保存状态状态及其更新函数。 */
  const [isSaving, setIsSaving] = useState(false);
  /** isSendingCode、setIsSendingCode 分别保存编码状态及其更新函数。 */
  const [isSendingCode, setIsSendingCode] = useState(false);
  /** isChangingPassword、setIsChangingPassword 分别保存密码状态及其更新函数。 */
  const [isChangingPassword, setIsChangingPassword] = useState(false);
  /** passwordDialogOpen、setPasswordDialogOpen 分别保存密码对话框状态及其更新函数。 */
  const [passwordDialogOpen, setPasswordDialogOpen] = useState(false);
  /** error、setError 分别保存错误状态状态及其更新函数。 */
  const [error, setError] = useState('');
  /** passwordError、setPasswordError 分别保存密码错误状态状态及其更新函数。 */
  const [passwordError, setPasswordError] = useState('');
  /** passwordMessage、setPasswordMessage 分别保存密码消息状态及其更新函数。 */
  const [passwordMessage, setPasswordMessage] = useState('');

  useEffect(() => {
    /** controller 保存请求控制器。 */
    const controller = new AbortController();
    setForm(toProfileForm(authUser));
    setProfile(null);
    setError('');
    setIsLoading(true);

    void (async () => {
      try {
        /** response 保存接口响应及其关联状态。 */
        const response = await requestProfile(authUser.id, { cache: 'no-store', signal: controller.signal });
        if (!response.ok) throw new Error(await parseProfileError(response, '加载个人资料失败'));
        /** user 保存用户。 */
        const user = await response.json() as User;
        setProfile(user);
        setForm(toProfileForm(user));
        onUpdated(user);
      /** loadError 保存错误状态。 */
      } catch (loadError) {
        if (!controller.signal.aborted) {
          setError(loadError instanceof Error ? loadError.message : '加载个人资料失败');
        }
      } finally {
        if (!controller.signal.aborted) setIsLoading(false);
      }
    })();

    return () => controller.abort();
  }, [authUser.id]);

  /** submitProfile 负责执行对应业务操作。 */
  const submitProfile = async (event: FormEvent<HTMLFormElement>) => {
    event.preventDefault();
    setError('');
    setIsSaving(true);
    try {
      /** response 保存接口响应及其关联状态。 */
      const response = await requestProfile(authUser.id, {
        method: 'PUT',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({
          ...form,
          name: form.name.trim(),
          email: form.email.trim(),
          phone: form.phone.trim(),
          description: form.description.trim(),
          avatarUrl: form.avatarUrl.trim(),
        }),
      });
      if (!response.ok) throw new Error(await parseProfileError(response, '保存个人资料失败'));
      /** user 保存用户。 */
      const user = await response.json() as User;
      setProfile(user);
      setForm(toProfileForm(user));
      onUpdated(user);
      void message.success('个人资料保存完成');
    /** saveError 保存保存状态错误状态。 */
    } catch (saveError) {
      /** errorMessage 保存错误状态消息。 */
      const errorMessage = saveError instanceof Error ? saveError.message : '保存个人资料失败';
      setError(errorMessage);
      void message.error(errorMessage);
    } finally {
      setIsSaving(false);
    }
  };

  /** sendCode 负责执行对应业务操作。 */
  const sendCode = async () => {
    setPasswordError('');
    setPasswordMessage('');
    setIsSendingCode(true);
    try {
      /** response 保存接口响应及其关联状态。 */
      const response = await requestWithSession(`${API_BASE_URL}/api/profile/password-code`, {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ email: form.email.trim() }),
      });
      if (!response.ok) throw new Error(await parseProfileError(response, '发送验证码失败'));
      setPasswordMessage('验证码已发送到当前绑定邮箱，3 分钟内有效。');
      void message.success('验证码发送完成');
    /** sendError 保存错误状态。 */
    } catch (sendError) {
      /** errorMessage 保存错误状态消息。 */
      const errorMessage = sendError instanceof Error ? sendError.message : '发送验证码失败';
      setPasswordError(errorMessage);
      void message.error(errorMessage);
    } finally {
      setIsSendingCode(false);
    }
  };

  /** changePassword 负责计算或维护密码。 */
  const changePassword = async (event: FormEvent<HTMLFormElement>) => {
    event.preventDefault();
    setPasswordError('');
    setPasswordMessage('');
    if (passwordForm.newPassword.length < 6) {
      setPasswordError('新密码至少需要 6 位。');
      return;
    }
    if (passwordForm.newPassword !== passwordForm.confirmPassword) {
      setPasswordError('两次输入的新密码不一致。');
      return;
    }
    setIsChangingPassword(true);
    try {
      /** response 保存接口响应及其关联状态。 */
      const response = await requestWithSession(`${API_BASE_URL}/api/profile/password`, {
        method: 'PUT',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ code: passwordForm.code.trim(), newPassword: passwordForm.newPassword }),
      });
      if (!response.ok) throw new Error(await parseProfileError(response, '修改密码失败'));
      setPasswordForm({ code: '', newPassword: '', confirmPassword: '' });
      setPasswordMessage('密码已修改，请使用新密码重新登录。');
      setPasswordDialogOpen(false);
      void message.success('密码修改完成');
      onPasswordChanged();
    /** changeError 保存错误状态。 */
    } catch (changeError) {
      /** errorMessage 保存错误状态消息。 */
      const errorMessage = changeError instanceof Error ? changeError.message : '修改密码失败';
      setPasswordError(errorMessage);
      void message.error(errorMessage);
    } finally {
      setIsChangingPassword(false);
    }
  };

  /** visibleUser 保存可见状态用户。 */
  const visibleUser = profile ?? authUser;
  /** avatarFallback 保存头像。 */
  const avatarFallback = Array.from(form.name.trim() || visibleUser.username || '?')[0]?.toUpperCase();

  return (
    <div className="page-stack profile-page">
      <section className="section-header-card">
        <div>
          <p className="page-kicker">账号中心</p>
          <h1>个人资料</h1>
          <span>维护个人联系方式、头像与密码安全；密码修改需要邮箱验证码确认。</span>
        </div>
      </section>

      {isLoading ? (
        <section className="panel-card">
          <div className={styles.loading} aria-label="正在加载个人资料"><Skeleton active avatar paragraph={{ rows: 5 }} /></div>
        </section>
      ) : (
        <section className="content-grid profile-layout">
          <section className="panel-card">
            <form className={styles.form} onSubmit={(event) => void submitProfile(event)}>
              <section className={styles.identity}>
                <Avatar size={88} src={form.avatarUrl || undefined}>{avatarFallback}</Avatar>
                <div>
                  <h2>{form.name || visibleUser.username}</h2>
                  <p>@{visibleUser.username}</p>
                  <div className={styles.tags}>
                    <Tag color="blue">{visibleUser.role || '未分配角色'}</Tag>
                    <Tag>{visibleUser.department || '未分配部门'}</Tag>
                    <Tag color={form.email ? 'success' : 'default'}>{form.email ? '邮箱已绑定' : '邮箱未绑定'}</Tag>
                  </div>
                </div>
              </section>

              {error && <Alert type="error" showIcon title={error} />}

              <div className={styles.fields}>
                <label>
                  显示姓名
                  <Input required maxLength={60} value={form.name} prefix={<UserOutlined />} onChange={(event) => setForm({ ...form, name: event.target.value })} placeholder="请输入姓名" />
                </label>
                <label>
                  年龄
                  <InputNumber min={0} max={150} precision={0} value={form.age || null} onChange={(age) => setForm({ ...form, age: Number(age ?? 0) })} placeholder="未填写" />
                </label>
                <label>
                  邮箱绑定
                  <Input type="email" maxLength={120} value={form.email} prefix={<MailOutlined />} onChange={(event) => setForm({ ...form, email: event.target.value })} placeholder="name@example.com" />
                </label>
                <label>
                  联系电话
                  <Input maxLength={30} value={form.phone} prefix={<PhoneOutlined />} onChange={(event) => setForm({ ...form, phone: event.target.value })} placeholder="请输入联系电话" />
                </label>
                <label className={styles.spanTwo}>
                  头像地址
                  <Input type="url" maxLength={2048} value={form.avatarUrl} onChange={(event) => setForm({ ...form, avatarUrl: event.target.value })} placeholder="https://example.com/avatar.jpg" />
                  <small>填写可公开访问的 HTTPS 图片地址，留空则使用默认头像。</small>
                </label>
                <label className={styles.spanTwo}>
                  个人描述
                  <Input.TextArea showCount maxLength={500} rows={4} value={form.description} onChange={(event) => setForm({ ...form, description: event.target.value })} placeholder="介绍你的职责、擅长方向或当前工作重点" />
                </label>
              </div>

              <div className="rich-editor-actions">
                <Button icon={<SafetyCertificateOutlined />} onClick={() => {
                  setPasswordError('');
                  setPasswordMessage('');
                  setPasswordDialogOpen(true);
                }}>修改密码</Button>
                <Button type="primary" htmlType="submit" loading={isSaving}>保存资料</Button>
              </div>
            </form>
          </section>
        </section>
      )}

      <Modal
        className={styles.passwordDialog}
        open={passwordDialogOpen}
        title="修改密码"
        footer={null}
        destroyOnHidden
        onCancel={() => {
          setPasswordDialogOpen(false);
          setPasswordForm({ code: '', newPassword: '', confirmPassword: '' });
          setPasswordError('');
          setPasswordMessage('');
        }}
      >
        <form className={styles.form} onSubmit={(event) => void changePassword(event)}>
          <p className={styles.dialogDescription}>验证码发送到当前账号绑定邮箱，3 分钟内有效。</p>
          {passwordError && <Alert type="error" showIcon title={passwordError} />}
          {passwordMessage && <Alert type="success" showIcon title={passwordMessage} />}
          <div className={styles.fields}>
            <label className={styles.spanTwo}>绑定邮箱<Input value={form.email || '当前账号未绑定邮箱'} prefix={<MailOutlined />} disabled /></label>
            <label className={styles.spanTwo}>
              邮箱验证码
              <div className="profile-code-row">
                <Input size="large" value={passwordForm.code} maxLength={6} prefix={<SafetyCertificateOutlined />} onChange={(event) => setPasswordForm({ ...passwordForm, code: event.target.value })} placeholder="请输入 6 位验证码" />
                <Button size="large" onClick={() => void sendCode()} loading={isSendingCode} disabled={!form.email.trim()}>发送验证码</Button>
              </div>
            </label>
            <label>新密码<Input.Password value={passwordForm.newPassword} prefix={<LockOutlined />} onChange={(event) => setPasswordForm({ ...passwordForm, newPassword: event.target.value })} placeholder="至少 6 位" /></label>
            <label>确认新密码<Input.Password value={passwordForm.confirmPassword} prefix={<LockOutlined />} onChange={(event) => setPasswordForm({ ...passwordForm, confirmPassword: event.target.value })} placeholder="再次输入新密码" /></label>
          </div>
          <div className="rich-editor-actions">
            <Button onClick={() => setPasswordDialogOpen(false)}>取消</Button>
            <Button type="primary" htmlType="submit" loading={isChangingPassword}>确认修改密码</Button>
          </div>
        </form>
      </Modal>
    </div>
  );
}
