(function () {
  'use strict';

  function mount(options) {
    options = options || {};
    var script = options.script || document.currentScript;
    var globalConfig = window.SOCKET_CUSTOMER_CONFIG || {};
    var apiBase = String(options.apiBase || (script && script.dataset.apiBase) || globalConfig.apiBase || window.location.origin).replace(/\/$/, '');
    var title = String(options.title || (script && script.dataset.title) || globalConfig.title || '在线客服');
    var color = String(options.color || (script && script.dataset.color) || globalConfig.color || '#1677ff');
    var position = String(options.position || (script && script.dataset.position) || globalConfig.position || 'right') === 'left' ? 'left' : 'right';
    var sessionKey = String(options.sessionKey || (script && script.dataset.sessionKey) || globalConfig.sessionKey || 'default');
    var storageKey = 'socket-customer:' + apiBase + ':' + sessionKey;
    var session = readSession(storageKey);
    var socket = null;
    var opened = false;
    var started = false;
    var messages = [];
    var messageIds = {};
    var reconnectTimer = 0;

    var host = document.createElement('div');
    host.setAttribute('data-socket-customer-widget', '');
    document.body.appendChild(host);
    var root = host.attachShadow({ mode: 'open' });
    root.innerHTML = '<style>' + styles(position) + '</style>' +
      '<button class="launcher" type="button" aria-label="打开在线客服"><span class="pulse"></span><span class="chat-icon">◌</span></button>' +
      '<section class="panel" aria-label="' + escapeAttribute(title) + '" hidden>' +
        '<header><div><strong>' + escapeText(title) + '</strong><small class="status">点击连接客服</small></div><button class="close" type="button" aria-label="关闭">×</button></header>' +
        '<main class="messages"><div class="welcome">您好，有什么可以帮您？</div></main>' +
        '<div class="tools"><button class="emoji" type="button" aria-label="表情">😊</button><button class="file" type="button" aria-label="发送图片或文件">📎</button><input class="file-input" type="file" hidden><span class="session-id"></span></div>' +
        '<div class="emoji-panel" hidden></div>' +
        '<form><textarea rows="2" maxlength="4000" placeholder="输入消息…"></textarea><button class="send" type="submit">发送</button></form>' +
      '</section>';
    root.host.style.setProperty('--socket-color', color);

    var launcher = root.querySelector('.launcher');
    var panel = root.querySelector('.panel');
    var closeButton = root.querySelector('.close');
    var status = root.querySelector('.status');
    var messageList = root.querySelector('.messages');
    var form = root.querySelector('form');
    var textarea = root.querySelector('textarea');
    var fileButton = root.querySelector('.file');
    var fileInput = root.querySelector('.file-input');
    var emojiButton = root.querySelector('.emoji');
    var emojiPanel = root.querySelector('.emoji-panel');
    var sessionLabel = root.querySelector('.session-id');
    var emojiList = ['😀', '😁', '😂', '😊', '😍', '🤝', '👍', '🎉', '❤️', '🙏', '📦', '✅'];
    emojiList.forEach(function (emoji) {
      var button = document.createElement('button');
      button.type = 'button';
      button.textContent = emoji;
      button.addEventListener('click', function () {
        textarea.value += emoji;
        emojiPanel.hidden = true;
        textarea.focus();
      });
      emojiPanel.appendChild(button);
    });

    launcher.addEventListener('click', function () {
      opened = true;
      panel.hidden = false;
      launcher.hidden = true;
      if (!started) {
        started = true;
        connect();
      }
      textarea.focus();
    });
    closeButton.addEventListener('click', function () {
      opened = false;
      panel.hidden = true;
      launcher.hidden = false;
    });
    emojiButton.addEventListener('click', function () {
      emojiPanel.hidden = !emojiPanel.hidden;
    });
    fileButton.addEventListener('click', function () { fileInput.click(); });
    fileInput.addEventListener('change', function () {
      var file = fileInput.files && fileInput.files[0];
      if (file) uploadFile(file);
      fileInput.value = '';
    });
    form.addEventListener('submit', function (event) {
      event.preventDefault();
      var content = textarea.value.trim();
      if (!content) return;
      if (!socket || socket.readyState !== WebSocket.OPEN) {
        status.textContent = '正在重连，请稍后发送';
        return;
      }
      socket.send(JSON.stringify({ type: 'message', messageType: 'text', content: content }));
      textarea.value = '';
    });
    textarea.addEventListener('keydown', function (event) {
      if (event.ctrlKey && event.key === 'Enter') form.requestSubmit();
    });

    function connect() {
      window.clearTimeout(reconnectTimer);
      status.textContent = '正在连接…';
      var url = new URL('/api/socket/customer', apiBase);
      url.protocol = url.protocol === 'https:' ? 'wss:' : 'ws:';
      if (session.id && session.token) {
        url.searchParams.set('conversationId', session.id);
        url.searchParams.set('visitorToken', session.token);
      }
      url.searchParams.set('visitorName', options.visitorName || '网站访客');
      socket = new WebSocket(url.toString());
      socket.onopen = function () { status.textContent = '客服已连接'; };
      socket.onmessage = function (event) {
        var envelope;
        try { envelope = JSON.parse(String(event.data)); } catch (_) { return; }
        if (envelope.type === 'session' && envelope.conversation) {
          session = { id: envelope.conversation.id, token: envelope.visitorToken || session.token };
          writeSession(storageKey, session);
          sessionLabel.textContent = '会话 ' + session.id.slice(-8);
          status.textContent = '客服已连接';
        } else if (envelope.type === 'history' && Array.isArray(envelope.messages)) {
          messages = [];
          messageIds = {};
          envelope.messages.forEach(addMessage);
        } else if (envelope.type === 'message' && envelope.message) {
          addMessage(envelope.message);
        } else if (envelope.type === 'error') {
          status.textContent = envelope.error || '连接异常';
          if (String(envelope.error || '').indexOf('凭证') >= 0) {
            session = {};
            writeSession(storageKey, session);
          }
        }
      };
      socket.onclose = function () {
        status.textContent = '连接中断，正在重连…';
        reconnectTimer = window.setTimeout(connect, 1800);
      };
      socket.onerror = function () { socket.close(); };
    }

    function addMessage(message) {
      if (!message || messageIds[message.id]) return;
      messageIds[message.id] = true;
      messages.push(message);
      var row = document.createElement('div');
      row.className = 'message-row ' + (message.senderType === 'visitor' ? 'visitor' : 'agent');
      var bubble = document.createElement('div');
      bubble.className = 'bubble';
      var meta = document.createElement('small');
      meta.textContent = (message.senderName || (message.senderType === 'visitor' ? '我' : '客服')) + ' · ' + formatTime(message.createdAt);
      bubble.appendChild(meta);
      if (message.messageType === 'image' || message.messageType === 'file') {
        renderAttachment(bubble, message);
      } else {
        var copy = document.createElement('p');
        copy.textContent = message.content || '';
        bubble.appendChild(copy);
      }
      row.appendChild(bubble);
      messageList.appendChild(row);
      messageList.scrollTop = messageList.scrollHeight;
    }

    function renderAttachment(container, message) {
      var button = document.createElement('button');
      button.type = 'button';
      button.className = 'attachment';
      button.textContent = (message.messageType === 'image' ? '🖼 ' : '📄 ') + (message.attachmentName || '聊天文件');
      container.appendChild(button);
      fetchAttachment(message).then(function (blob) {
        var objectUrl = URL.createObjectURL(blob);
        if (message.messageType === 'image') {
          var image = document.createElement('img');
          image.alt = message.attachmentName || '聊天图片';
          image.src = objectUrl;
          image.addEventListener('click', function () { window.open(objectUrl, '_blank', 'noopener'); });
          container.replaceChild(image, button);
        } else {
          button.addEventListener('click', function () {
            var link = document.createElement('a');
            link.href = objectUrl;
            link.download = message.attachmentName || 'socket-file';
            link.click();
          });
        }
      }).catch(function () { button.textContent += '（加载失败）'; });
    }

    function fetchAttachment(message) {
      return fetch(apiBase + '/api/socket/customer/' + encodeURIComponent(session.id) + '/files/' + message.id, {
        headers: { 'X-Socket-Visitor-Token': session.token }
      }).then(function (response) {
        if (!response.ok) throw new Error('attachment');
        return response.blob();
      });
    }

    function uploadFile(file) {
      if (!session.id || !session.token) return;
      if (file.size > 32 * 1024 * 1024) {
        status.textContent = '文件不能超过 32 MiB';
        return;
      }
      status.textContent = '正在发送文件…';
      var formData = new FormData();
      formData.append('file', file);
      fetch(apiBase + '/api/socket/customer/' + encodeURIComponent(session.id) + '/files', {
        method: 'POST',
        headers: { 'X-Socket-Visitor-Token': session.token },
        body: formData
      }).then(function (response) {
        if (!response.ok) throw new Error('upload');
        return response.json();
      }).then(function (message) {
        addMessage(message);
        status.textContent = '客服已连接';
      }).catch(function () { status.textContent = '文件发送失败'; });
    }

    return { open: function () { launcher.click(); }, close: function () { closeButton.click(); }, destroy: function () { window.clearTimeout(reconnectTimer); if (socket) socket.close(); host.remove(); } };
  }

  function readSession(key) {
    try { return JSON.parse(localStorage.getItem(key) || '{}') || {}; } catch (_) { return {}; }
  }
  function writeSession(key, value) {
    try { localStorage.setItem(key, JSON.stringify(value)); } catch (_) {}
  }
  function formatTime(value) {
    var date = new Date(value);
    return Number.isNaN(date.getTime()) ? '' : date.toLocaleTimeString('zh-CN', { hour: '2-digit', minute: '2-digit' });
  }
  function escapeText(value) { return String(value).replace(/[&<>]/g, function (char) { return ({ '&': '&amp;', '<': '&lt;', '>': '&gt;' })[char]; }); }
  function escapeAttribute(value) { return escapeText(value).replace(/"/g, '&quot;'); }
  function styles(position) {
    var side = position === 'left' ? 'left:24px' : 'right:24px';
    return ':host{--socket-color:#1677ff;position:fixed;z-index:2147483000;bottom:24px;' + side + ';font-family:Arial,"Microsoft YaHei",sans-serif;color:#172033}' +
      '*{box-sizing:border-box}.launcher{width:58px;height:58px;border:0;border-radius:50%;color:white;background:linear-gradient(135deg,var(--socket-color),color-mix(in srgb,var(--socket-color) 60%,#8b5cf6));box-shadow:0 12px 32px color-mix(in srgb,var(--socket-color) 42%,transparent);cursor:pointer;font-size:30px;position:relative}.pulse{position:absolute;inset:-6px;border:2px solid color-mix(in srgb,var(--socket-color) 48%,transparent);border-radius:50%;animation:pulse 2s infinite}.chat-icon:before{content:"💬";font-size:27px}' +
      '.panel{width:min(370px,calc(100vw - 32px));height:min(590px,calc(100vh - 48px));border:1px solid rgba(148,163,184,.34);border-radius:18px;background:#fff;box-shadow:0 24px 70px rgba(15,23,42,.22);overflow:hidden}.panel[hidden],.launcher[hidden]{display:none}.panel header{height:72px;padding:14px 16px;color:#fff;background:linear-gradient(135deg,var(--socket-color),color-mix(in srgb,var(--socket-color) 58%,#8b5cf6));display:flex;align-items:center;justify-content:space-between}.panel header div{display:grid;gap:4px}.panel header strong{font-size:17px}.panel header small{opacity:.88}.close{border:0;background:rgba(255,255,255,.14);color:#fff;border-radius:9px;width:34px;height:34px;font-size:24px;cursor:pointer}' +
      '.messages{height:390px;overflow:auto;padding:15px;background:#f6f8fc}.welcome{margin:4px auto 16px;padding:8px 12px;width:max-content;max-width:92%;border-radius:999px;background:#e9eef7;color:#64748b;font-size:12px}.message-row{display:flex;margin:10px 0}.message-row.visitor{justify-content:flex-end}.bubble{max-width:80%;padding:9px 11px;border:1px solid #dce3ee;border-radius:13px;background:#fff;box-shadow:0 4px 12px rgba(15,23,42,.05)}.visitor .bubble{border-color:color-mix(in srgb,var(--socket-color) 32%,#dce3ee);background:color-mix(in srgb,var(--socket-color) 10%,#fff)}.bubble small{display:block;margin-bottom:5px;color:#8a96a8;font-size:10px}.bubble p{margin:0;white-space:pre-wrap;word-break:break-word}.bubble img{display:block;max-width:100%;max-height:220px;border-radius:8px;cursor:pointer}.attachment{border:0;background:transparent;color:var(--socket-color);cursor:pointer;text-align:left}' +
      '.tools{height:38px;padding:4px 12px;border-top:1px solid #e2e8f0;display:flex;align-items:center;gap:5px}.tools button{border:0;background:transparent;border-radius:7px;padding:5px;cursor:pointer}.tools button:hover{background:#eef2f8}.session-id{margin-left:auto;color:#94a3b8;font-size:10px}.emoji-panel{position:absolute;bottom:118px;width:230px;padding:8px;border:1px solid #dce3ee;border-radius:12px;background:#fff;box-shadow:0 12px 30px rgba(15,23,42,.18);display:grid;grid-template-columns:repeat(6,1fr);gap:3px}.emoji-panel[hidden]{display:none}.emoji-panel button{border:0;border-radius:7px;background:transparent;padding:5px;font-size:19px;cursor:pointer}.emoji-panel button:hover{background:#eef2f8}' +
      'form{height:90px;padding:8px 10px 10px;display:grid;grid-template-columns:1fr auto;gap:8px;border-top:1px solid #e2e8f0}textarea{width:100%;resize:none;border:1px solid #d5ddea;border-radius:10px;padding:9px;outline:none;font:inherit}textarea:focus{border-color:var(--socket-color);box-shadow:0 0 0 3px color-mix(in srgb,var(--socket-color) 14%,transparent)}.send{align-self:end;border:0;border-radius:9px;padding:9px 13px;color:#fff;background:var(--socket-color);cursor:pointer}' +
      '@keyframes pulse{0%,100%{transform:scale(.92);opacity:.4}50%{transform:scale(1.12);opacity:.05}}@media(prefers-reduced-motion:reduce){.pulse{animation:none}}@media(max-width:480px){:host{bottom:12px;' + (position === 'left' ? 'left:12px' : 'right:12px') + '}.panel{height:calc(100vh - 24px)}.messages{height:calc(100vh - 224px)}}';
  }

  window.SocketCustomerWidget = { mount: mount };
  var autoScript = document.currentScript;
  if (document.readyState === 'loading') {
    document.addEventListener('DOMContentLoaded', function () { mount({ script: autoScript }); }, { once: true });
  } else {
    mount({ script: autoScript });
  }
})();
