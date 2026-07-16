'use client';

import { useEffect, useRef, useState, type FormEvent } from 'react';
import { MailOutlined, PhoneOutlined, UserOutlined } from '@ant-design/icons';
import { Alert, Avatar, Button, Input, InputNumber, Modal, Skeleton, Tag } from 'antd';
import { API_BASE_URL } from '../lib/constants';
import { requestWithSession } from '../lib/api';
import type { AuthUser, ProfileForm, User } from '../types/admin';
import styles from './ProfileDialog.module.css';

type ProfileDialogProps = {
  authUser: AuthUser;
  open: boolean;
  onClose: () => void;
  onUpdated: (user: User) => void;
};

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

async function parseProfileError(response: Response, fallback: string) {
  try {
    const payload = await response.json() as { error?: string };
    return payload.error || fallback;
  } catch {
    return fallback;
  }
}

async function requestProfile(userId: number, init: RequestInit = {}) {
  const paths = ['/api/profile', `/api/users/${userId}/profile`];
  for (let index = 0; index < paths.length; index += 1) {
    const response = await requestWithSession(`${API_BASE_URL}${paths[index]}`, init);
    if (index === 0 && (response.status === 404 || response.status === 405)) continue;
    return response;
  }
  throw new Error('个人资料接口不可用');
}

export function ProfileDialog({ authUser, open, onClose, onUpdated }: ProfileDialogProps) {
  const formRef = useRef<HTMLFormElement>(null);
  const [profile, setProfile] = useState<User | null>(null);
  const [form, setForm] = useState<ProfileForm>(() => toProfileForm(authUser));
  const [isLoading, setIsLoading] = useState(false);
  const [isSaving, setIsSaving] = useState(false);
  const [error, setError] = useState('');

  useEffect(() => {
    if (!open) return;
    const controller = new AbortController();
    setForm(toProfileForm(authUser));
    setProfile(null);
    setError('');
    setIsLoading(true);

    void (async () => {
      try {
        const response = await requestProfile(authUser.id, { cache: 'no-store', signal: controller.signal });
        if (!response.ok) throw new Error(await parseProfileError(response, '加载个人资料失败'));
        const user = await response.json() as User;
        setProfile(user);
        setForm(toProfileForm(user));
        onUpdated(user);
      } catch (loadError) {
        if (!controller.signal.aborted) {
          setError(loadError instanceof Error ? loadError.message : '加载个人资料失败');
        }
      } finally {
        if (!controller.signal.aborted) setIsLoading(false);
      }
    })();

    return () => controller.abort();
  }, [authUser.id, open]);

  const submit = async (event: FormEvent<HTMLFormElement>) => {
    event.preventDefault();
    setError('');
    setIsSaving(true);
    try {
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
      const user = await response.json() as User;
      setProfile(user);
      setForm(toProfileForm(user));
      onUpdated(user);
      onClose();
    } catch (saveError) {
      setError(saveError instanceof Error ? saveError.message : '保存个人资料失败');
    } finally {
      setIsSaving(false);
    }
  };

  const visibleUser = profile ?? authUser;
  const avatarFallback = Array.from(form.name.trim() || visibleUser.username || '?')[0]?.toUpperCase();

  return (
    <Modal
      open={open}
      title="个人资料"
      width={720}
      destroyOnHidden
      mask={{ closable: !isSaving }}
      onCancel={isSaving ? undefined : onClose}
      footer={[
        <Button key="cancel" disabled={isSaving} onClick={onClose}>取消</Button>,
        <Button key="save" type="primary" loading={isSaving} disabled={isLoading} onClick={() => formRef.current?.requestSubmit()}>保存资料</Button>,
      ]}
    >
      {isLoading ? (
        <div className={styles.loading} aria-label="正在加载个人资料"><Skeleton active avatar paragraph={{ rows: 5 }} /></div>
      ) : (
        <form ref={formRef} className={styles.form} onSubmit={(event) => void submit(event)}>
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
        </form>
      )}
    </Modal>
  );
}
