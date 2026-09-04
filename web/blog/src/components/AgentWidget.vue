<template>
  <section :class="['agent-widget', { 'agent-widget--embedded': embedded }]" aria-label="Benetnasch 公开对话">
    <button
      v-if="!embedded && !opened"
      class="agent-widget__entry"
      type="button"
      aria-haspopup="dialog"
      aria-expanded="false"
      @click="opened = true">
      <span class="agent-widget__orb" aria-hidden="true">✦</span>
      <span>和 Benetnasch 聊聊</span>
    </button>

    <div v-if="embedded || opened" class="agent-widget__panel" role="dialog" aria-modal="false" aria-label="Benetnasch 对话">
      <header class="agent-widget__header">
        <div>
          <strong>Benetnasch</strong>
          <small>只查阅公开文章</small>
        </div>
        <button v-if="!embedded" class="agent-widget__close" type="button" aria-label="关闭对话" @click="closePanel">×</button>
      </header>

      <div ref="messageList" class="agent-widget__messages" aria-live="polite">
        <article v-if="messages.length === 0" class="agent-widget__welcome">
          <p>可以问我这里的文章、分类或标签。</p>
          <small>语音播报未启用；你可以随时清除这段短期会话。</small>
        </article>
        <article v-for="(message, index) in messages" :key="index" :class="['agent-widget__message', message.role]">
          <p>{{ message.content || (message.role === 'assistant' && streaming ? '正在查阅……' : '') }}</p>
          <ul v-if="message.citations.length > 0" class="agent-widget__citations">
            <li v-for="citation in message.citations" :key="citation.articleId + ':' + citation.url">
              <a :href="citation.url || '/articles/' + citation.articleId" target="_blank" rel="noopener noreferrer">
                {{ citation.title || `文章 #${citation.articleId}` }}
              </a>
            </li>
          </ul>
        </article>
        <p v-if="errorMessage" class="agent-widget__error" role="alert">{{ errorMessage }}</p>
      </div>

      <form class="agent-widget__composer" @submit.prevent="sendMessage">
        <textarea
          v-model="input"
          rows="2"
          maxlength="4000"
          :disabled="streaming"
          placeholder="输入你想了解的内容"
          aria-label="对话内容"
          @keydown.enter.exact.prevent="sendMessage" />
        <div class="agent-widget__actions">
          <button type="button" class="agent-widget__clear" :disabled="streaming" @click="clearConversation">清除会话</button>
          <button type="submit" class="agent-widget__send" :disabled="streaming || input.trim() === ''">发送</button>
        </div>
      </form>
    </div>
  </section>
</template>

<script lang="ts">
import { computed, defineComponent, nextTick, onBeforeUnmount, ref } from 'vue'

interface AgentCitation {
  articleId: number
  title: string
  url?: string
}

interface AgentEventPayload {
  sessionId?: string
  turnId?: string
  seq?: number
  replay?: boolean
  text?: string
  opening?: string
  citation?: AgentCitation
  message?: string
}

interface AgentMessage {
  role: 'user' | 'assistant'
  content: string
  citations: AgentCitation[]
}

const SESSION_STORAGE_KEY = 'benetnasch:agent-session:v1'

export default defineComponent({
  name: 'AgentWidget',
  props: {
    embedded: {
      type: Boolean,
      default: false
    }
  },
  setup(props) {
    const opened = ref(props.embedded)
    const input = ref('')
    const messages = ref<AgentMessage[]>([])
    const errorMessage = ref('')
    const streaming = ref(false)
    const messageList = ref<HTMLElement | null>(null)
    const sessionId = ref(readSessionID())
    const activeTurnID = ref('')
    const lastSeq = ref(0)
    const doneReceived = ref(false)
    let controller: AbortController | null = null

    const scrollToLatest = () => {
      void nextTick(() => {
        if (messageList.value) messageList.value.scrollTop = messageList.value.scrollHeight
      })
    }

    const sendMessage = async () => {
      const message = input.value.trim()
      if (!message || streaming.value) return
      errorMessage.value = ''
      messages.value.push({ role: 'user', content: message, citations: [] })
      const assistant: AgentMessage = { role: 'assistant', content: '', citations: [] }
      messages.value.push(assistant)
      input.value = ''
      streaming.value = true
      activeTurnID.value = ''
      lastSeq.value = 0
      doneReceived.value = false
      controller = new AbortController()
      scrollToLatest()
      try {
        const response = await fetch('/api/agent/chat', {
          method: 'POST',
          headers: {
            Accept: 'text/event-stream',
            'Content-Type': 'application/json',
            'X-Request-ID': requestID()
          },
          body: JSON.stringify({ sessionId: sessionId.value, message }),
          signal: controller.signal
        })
        if (!response.ok || !response.body) throw new Error('agent request failed')
        await consumeSSE(response, (eventName, payload) => applyEvent(eventName, payload, assistant))
      } catch (error) {
        if (!(error instanceof DOMException && error.name === 'AbortError')) {
          let resumed = false
          if (sessionId.value && activeTurnID.value) {
            try {
              const replayResponse = await fetch(
                '/api/agent/sessions/' +
                  encodeURIComponent(sessionId.value) +
                  '/events?afterSeq=' +
                  encodeURIComponent(String(lastSeq.value)) +
                  '&turnId=' +
                  encodeURIComponent(activeTurnID.value),
                { headers: { Accept: 'text/event-stream' } }
              )
              if (replayResponse.ok && replayResponse.body) {
                const replayed = await consumeSSE(replayResponse, (eventName, payload) => applyEvent(eventName, payload, assistant))
                resumed = replayed > 0 || doneReceived.value
              }
            } catch (_resumeError) {
              resumed = false
            }
          }
          if (!resumed) errorMessage.value = '我现在暂时离线，请稍后再来。'
        }
      } finally {
        streaming.value = false
        controller = null
        scrollToLatest()
      }
    }

    const closePanel = () => {
      controller?.abort()
      opened.value = false
    }

    const clearConversation = async () => {
      controller?.abort()
      if (sessionId.value) {
        try {
          await fetch('/api/agent/sessions/' + encodeURIComponent(sessionId.value), { method: 'DELETE' })
        } catch (_error) {
          // Local state is still cleared; the Redis session will expire on its own.
        }
      }
      sessionId.value = ''
      activeTurnID.value = ''
      lastSeq.value = 0
      doneReceived.value = false
      sessionStorage.removeItem(SESSION_STORAGE_KEY)
      messages.value = []
      errorMessage.value = ''
      streaming.value = false
    }

    const rememberSession = (value: string) => {
      if (!value || value === sessionId.value) return
      sessionId.value = value
      sessionStorage.setItem(SESSION_STORAGE_KEY, value)
    }

    const applyEvent = (eventName: string, payload: AgentEventPayload, assistant: AgentMessage) => {
      if (payload.sessionId) rememberSession(payload.sessionId)
      if (payload.turnId) activeTurnID.value = payload.turnId
      if (typeof payload.seq === 'number' && payload.seq > lastSeq.value) lastSeq.value = payload.seq
      if (eventName === 'meta' && payload.opening && assistant.content === '') {
        assistant.content = payload.opening
      } else if (eventName === 'delta' && payload.text) {
        assistant.content += payload.text
      } else if (eventName === 'citation' && payload.citation) {
        const exists = assistant.citations.some((item) => item.articleId === payload.citation?.articleId)
        if (!exists) assistant.citations.push(payload.citation)
      } else if (eventName === 'error') {
        errorMessage.value = payload.message || '对话暂时不可用'
      } else if (eventName === 'done') {
        doneReceived.value = true
      }
      scrollToLatest()
    }

    onBeforeUnmount(() => controller?.abort())

    return {
      opened,
      embedded: computed(() => props.embedded),
      input,
      messages,
      errorMessage,
      streaming,
      messageList,
      sendMessage,
      closePanel,
      clearConversation
    }
  }
})

function readSessionID(): string {
  try {
    return sessionStorage.getItem(SESSION_STORAGE_KEY) || ''
  } catch (_error) {
    return ''
  }
}

function requestID(): string {
  if (typeof crypto !== 'undefined' && typeof crypto.randomUUID === 'function') return crypto.randomUUID()
  return 'agent-' + Date.now().toString(36) + '-' + Math.random().toString(36).slice(2)
}

async function consumeSSE(response: Response, onEvent: (name: string, payload: AgentEventPayload) => void): Promise<number> {
  const reader = response.body?.getReader()
  if (!reader) throw new Error('agent stream is unavailable')
  const decoder = new TextDecoder()
  let buffer = ''
  let eventCount = 0
  const dispatch = (raw: string) => {
    let name = 'message'
    const data: string[] = []
    for (const line of raw.split(/\r?\n/)) {
      if (line.startsWith('event:')) name = line.slice(6).trim()
      if (line.startsWith('data:')) data.push(line.slice(5).trimStart())
    }
    if (data.length === 0) return
    try {
      onEvent(name, JSON.parse(data.join('\n')) as AgentEventPayload)
      eventCount++
    } catch (_error) {
      // Ignore malformed optional event frames; the backend validates its own payload.
    }
  }
  while (true) {
    const { done, value } = await reader.read()
    buffer += decoder.decode(value || new Uint8Array(), { stream: !done })
    const frames = buffer.split(/\r?\n\r?\n/)
    buffer = frames.pop() || ''
    frames.forEach(dispatch)
    if (done) {
      if (buffer.trim()) dispatch(buffer)
      return eventCount
    }
  }
}
</script>

<style lang="scss" scoped>
.agent-widget {
  position: fixed;
  right: 1.25rem;
  bottom: 1.25rem;
  z-index: 1100;
  color: var(--text-normal);
}

.agent-widget--embedded {
  position: relative;
  right: auto;
  bottom: auto;
  z-index: auto;
}

.agent-widget--embedded .agent-widget__panel {
  width: 100%;
  max-height: none;
}

.agent-widget__entry,
.agent-widget__panel {
  background: var(--background-secondary);
  border: 1px solid var(--bg-accent-55);
  box-shadow: var(--accent-shadow);
}

.agent-widget__entry {
  display: flex;
  align-items: center;
  gap: 0.5rem;
  border-radius: 999px;
  padding: 0.65rem 1rem;
  color: var(--text-bright);
  cursor: pointer;
}

.agent-widget__orb {
  color: var(--text-accent);
  font-size: 1.1rem;
}

.agent-widget__panel {
  display: flex;
  flex-direction: column;
  width: min(24rem, calc(100vw - 2rem));
  max-height: min(38rem, calc(100vh - 2rem));
  border-radius: 1rem;
  overflow: hidden;
}

.agent-widget__header,
.agent-widget__actions {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 0.75rem;
}

.agent-widget__header {
  padding: 0.85rem 1rem;
  border-bottom: 1px solid var(--bg-accent-05);
}

.agent-widget__header strong,
.agent-widget__header small {
  display: block;
}

.agent-widget__header small {
  margin-top: 0.15rem;
  color: var(--text-dim);
  font-size: 0.72rem;
}

.agent-widget__close {
  border: 0;
  background: transparent;
  color: var(--text-dim);
  font-size: 1.5rem;
  cursor: pointer;
}

.agent-widget__messages {
  flex: 1;
  min-height: 12rem;
  padding: 1rem;
  overflow-y: auto;
}

.agent-widget__welcome,
.agent-widget__message {
  margin: 0 0 0.75rem;
  padding: 0.7rem 0.8rem;
  border-radius: 0.75rem;
  line-height: 1.6;
  white-space: pre-wrap;
  overflow-wrap: anywhere;
}

.agent-widget__welcome {
  background: var(--bg-accent-05);
}

.agent-widget__welcome p,
.agent-widget__welcome small,
.agent-widget__message p {
  margin: 0;
}

.agent-widget__welcome small {
  color: var(--text-dim);
  font-size: 0.75rem;
}

.agent-widget__message.user {
  margin-left: 1.5rem;
  background: var(--bg-sub-accent-55);
  color: var(--text-bright);
}

.agent-widget__message.assistant {
  margin-right: 1.5rem;
  background: var(--bg-accent-05);
}

.agent-widget__citations {
  margin: 0.5rem 0 0;
  padding-left: 1.1rem;
  font-size: 0.78rem;
}

.agent-widget__citations a {
  color: var(--text-a);
}

.agent-widget__error {
  margin: 0.5rem 0 0;
  color: var(--custom-quote-danger);
  font-size: 0.82rem;
}

.agent-widget__composer {
  padding: 0.75rem;
  border-top: 1px solid var(--bg-accent-05);
}

.agent-widget__composer textarea {
  display: block;
  width: 100%;
  box-sizing: border-box;
  resize: vertical;
  border: 1px solid var(--bg-accent-55);
  border-radius: 0.65rem;
  padding: 0.6rem;
  background: var(--background-primary);
  color: var(--text-normal);
}

.agent-widget__actions {
  margin-top: 0.55rem;
}

.agent-widget__clear,
.agent-widget__send {
  border: 0;
  border-radius: 0.55rem;
  padding: 0.45rem 0.75rem;
  cursor: pointer;
}

.agent-widget__clear {
  background: transparent;
  color: var(--text-dim);
}

.agent-widget__send {
  background: var(--text-accent);
  color: var(--background-primary);
}

.agent-widget__clear:disabled,
.agent-widget__send:disabled,
.agent-widget__composer textarea:disabled {
  cursor: not-allowed;
  opacity: 0.55;
}

@media (max-width: 640px) {
  .agent-widget {
    right: 0.75rem;
    bottom: 0.75rem;
  }
}
</style>
