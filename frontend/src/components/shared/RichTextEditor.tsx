'use client';

import { useEffect, useRef, useState } from 'react';
import { Alert, Button, Input, Modal, Space } from 'antd';
import {
  BoldOutlined,
  ItalicOutlined,
  LinkOutlined,
  OrderedListOutlined,
  StrikethroughOutlined,
  UnorderedListOutlined,
} from '@ant-design/icons';

type RichTextEditorProps = {
  /** value 表示值。 */
  value: string;
  /** onChange 表示变量 onChange。 */
  onChange: (value: string) => void;
  /** minHeight 表示高度。 */
  minHeight?: number;
  /** placeholder 表示占位内容。 */
  placeholder?: string;
};

/** RichTextEditor 实现对应业务逻辑。 */
export function RichTextEditor({ value, onChange, minHeight = 260, placeholder = '请输入内容…' }: RichTextEditorProps) {
  /** editorRef 保存跨渲染周期使用的编辑器引用。 */
  const editorRef = useRef<HTMLDivElement | null>(null);
  /** selectionRef 保存跨渲染周期使用的文本选区引用。 */
  const selectionRef = useRef<Range | null>(null);
  /** linkDialogOpen、setLinkDialogOpen 分别保存对话框状态及其更新函数。 */
  const [linkDialogOpen, setLinkDialogOpen] = useState(false);
  /** linkUrl、setLinkUrl 分别保存地址状态及其更新函数。 */
  const [linkUrl, setLinkUrl] = useState('');
  /** linkError、setLinkError 分别保存错误状态状态及其更新函数。 */
  const [linkError, setLinkError] = useState('');

  useEffect(() => {
    // 编辑器可能在异步读取文本完成后才首次挂载。不能仅用 ref 中的初始值判断，
    // 否则 value 已是新内容但 DOM 仍为空时会漏掉回填，造成“重新打开内容为空”。
    if (editorRef.current && editorRef.current.innerHTML !== value) {
      editorRef.current.innerHTML = value;
    }
  }, [value]);

  /** syncContent 负责更新并保存对应业务状态。 */
  const syncContent = () => {
    onChange(editorRef.current?.innerHTML ?? '');
  };

  /** runCommand 负责计算或维护变量 runCommand。 */
  const runCommand = (command: string, argument?: string) => {
    editorRef.current?.focus();
    /** selection 保存文本选区。 */
    const selection = window.getSelection();
    if (selectionRef.current && selection) {
      selection.removeAllRanges();
      selection.addRange(selectionRef.current);
    }
    document.execCommand(command, false, argument);
    syncContent();
  };

  /** createLink 负责创建或追加对应业务记录。 */
  const createLink = () => {
    /** url 保存地址。 */
    const url = linkUrl.trim();
    if (!/^https?:\/\//i.test(url)) {
      setLinkError('请输入以 http:// 或 https:// 开头的有效地址。');
      return;
    }
    runCommand('createLink', url);
    setLinkDialogOpen(false);
    setLinkUrl('');
    setLinkError('');
  };

  /** openLinkDialog 负责计算或维护对话框。 */
  const openLinkDialog = () => {
    /** selection 保存文本选区。 */
    const selection = window.getSelection();
    if (selection?.rangeCount) selectionRef.current = selection.getRangeAt(0).cloneRange();
    setLinkUrl('');
    setLinkError('');
    setLinkDialogOpen(true);
  };

  return (
    <div className="shared-rich-editor">
      <Space wrap size={6} className="shared-rich-toolbar">
        <Button size="small" icon={<BoldOutlined />} onClick={() => runCommand('bold')}>加粗</Button>
        <Button size="small" icon={<ItalicOutlined />} onClick={() => runCommand('italic')}>斜体</Button>
        <Button size="small" icon={<StrikethroughOutlined />} onClick={() => runCommand('strikeThrough')}>删除线</Button>
        <Button size="small" onClick={() => runCommand('formatBlock', 'h2')}>H2</Button>
        <Button size="small" onClick={() => runCommand('formatBlock', 'p')}>正文</Button>
        <Button size="small" icon={<UnorderedListOutlined />} onClick={() => runCommand('insertUnorderedList')}>无序</Button>
        <Button size="small" icon={<OrderedListOutlined />} onClick={() => runCommand('insertOrderedList')}>有序</Button>
        <Button size="small" icon={<LinkOutlined />} onClick={openLinkDialog}>链接</Button>
        <Button size="small" onClick={() => runCommand('removeFormat')}>清除格式</Button>
      </Space>
      <div
        ref={editorRef}
        className="shared-rich-content"
        contentEditable
        suppressContentEditableWarning
        data-placeholder={placeholder}
        style={{ minHeight }}
        onInput={syncContent}
        onBlur={syncContent}
      />
      <Modal open={linkDialogOpen} title="插入链接" okText="插入" cancelText="取消" onOk={createLink} onCancel={() => setLinkDialogOpen(false)} destroyOnHidden>
        <Space direction="vertical" size={12} style={{ width: '100%' }}>
          {linkError && <Alert type="error" showIcon title={linkError} />}
          <Input value={linkUrl} onChange={(event) => setLinkUrl(event.target.value)} onPressEnter={createLink} placeholder="https://example.com" />
        </Space>
      </Modal>
    </div>
  );
}
