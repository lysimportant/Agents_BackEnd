'use client';

import { useEffect, useRef } from 'react';
import { Button, Space } from 'antd';
import {
  BoldOutlined,
  ItalicOutlined,
  LinkOutlined,
  OrderedListOutlined,
  StrikethroughOutlined,
  UnorderedListOutlined,
} from '@ant-design/icons';

type RichTextEditorProps = {
  value: string;
  onChange: (value: string) => void;
  minHeight?: number;
  placeholder?: string;
};

export function RichTextEditor({ value, onChange, minHeight = 260, placeholder = '请输入内容…' }: RichTextEditorProps) {
  const editorRef = useRef<HTMLDivElement | null>(null);

  useEffect(() => {
    // 编辑器可能在异步读取文本完成后才首次挂载。不能仅用 ref 中的初始值判断，
    // 否则 value 已是新内容但 DOM 仍为空时会漏掉回填，造成“重新打开内容为空”。
    if (editorRef.current && editorRef.current.innerHTML !== value) {
      editorRef.current.innerHTML = value;
    }
  }, [value]);

  const syncContent = () => {
    onChange(editorRef.current?.innerHTML ?? '');
  };

  const runCommand = (command: string, argument?: string) => {
    editorRef.current?.focus();
    document.execCommand(command, false, argument);
    syncContent();
  };

  const createLink = () => {
    const url = window.prompt('请输入链接地址，例如 https://example.com');
    if (!url) return;
    runCommand('createLink', url);
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
        <Button size="small" icon={<LinkOutlined />} onClick={createLink}>链接</Button>
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
    </div>
  );
}
