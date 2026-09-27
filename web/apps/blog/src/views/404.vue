<template>
  <section id="not-found-page" aria-labelledby="not-found-title">
    <div class="not-found-copy">
      <p class="not-found-kicker">CHANNEL 404 <span>/</span> ROUTE NOT FOUND</p>
      <h1 id="not-found-title">Signal<br /><em>lost.</em></h1>
      <p class="not-found-message">这个地址没有返回有效内容。回到首页，继续浏览已归档的信号。</p>
      <router-link class="not-found-action" to="/">
        <span>返回首页</span>
        <span aria-hidden="true">↗</span>
      </router-link>
    </div>
    <div class="not-found-console" aria-hidden="true">
      <div class="not-found-console__grid" />
      <div class="not-found-console__pulse" />
      <span class="not-found-console__code">ERR / 0x0194</span>
      <span class="not-found-console__status">NO RESPONSE</span>
    </div>
  </section>
</template>

<style lang="scss" scoped>
#not-found-page {
  display: grid;
  grid-template-columns: minmax(0, 0.9fr) minmax(280px, 1.1fr);
  gap: clamp(32px, 8vw, 128px);
  align-items: center;
  min-height: calc(100vh - var(--header-h) - 100px);
  padding: 64px 0;
}

.not-found-kicker,
.not-found-console__code,
.not-found-console__status {
  color: var(--accent);
  font-family: var(--font-mono);
  font-size: 10px;
  letter-spacing: 0.14em;
}

.not-found-kicker span {
  padding-inline: 8px;
  color: var(--text-faint);
}

.not-found-copy h1 {
  margin: 24px 0;
  color: var(--text-bright);
  font-family: var(--font-display);
  font-size: clamp(4.5rem, 11vw, 10rem);
  font-weight: 700;
  letter-spacing: -0.1em;
  line-height: 0.8;
}

.not-found-copy h1 em {
  color: var(--accent);
  font-style: normal;
}

.not-found-message {
  max-width: 34ch;
  color: var(--text-dim);
  line-height: 1.8;
}

.not-found-action {
  display: inline-flex;
  justify-content: space-between;
  gap: 32px;
  min-width: 140px;
  margin-top: 28px;
  padding: 12px 14px;
  border: 1px solid var(--accent);
  border-radius: 2px;
  color: var(--accent);
  font-family: var(--font-mono);
  font-size: 11px;
  text-decoration: none;
}

.not-found-action:hover {
  background: var(--accent);
  color: #09100a;
}

.not-found-console {
  position: relative;
  min-height: 360px;
  overflow: hidden;
  border: 1px solid var(--border-strong);
  background: var(--surface-1);
}

.not-found-console::before,
.not-found-console::after {
  position: absolute;
  width: 18px;
  height: 18px;
  content: '';
}

.not-found-console::before {
  top: -1px;
  left: -1px;
  border-top: 2px solid var(--accent);
  border-left: 2px solid var(--accent);
}

.not-found-console::after {
  right: -1px;
  bottom: -1px;
  border-right: 2px solid var(--accent-2);
  border-bottom: 2px solid var(--accent-2);
}

.not-found-console__grid {
  position: absolute;
  inset: 0;
  background-image: linear-gradient(var(--border-hairline) 1px, transparent 1px),
    linear-gradient(90deg, var(--border-hairline) 1px, transparent 1px);
  background-size: 40px 40px;
  opacity: 0.35;
}

.not-found-console__pulse {
  position: absolute;
  top: 50%;
  left: 14%;
  width: 72%;
  height: 1px;
  background: var(--accent);
  box-shadow: 0 0 22px 4px rgba(181, 255, 99, 0.36);
}

.not-found-console__pulse::after {
  position: absolute;
  top: -8px;
  right: 18%;
  width: 16px;
  height: 16px;
  border: 1px solid var(--accent);
  border-radius: 50%;
  box-shadow: 0 0 24px var(--accent);
  content: '';
}

.not-found-console__code,
.not-found-console__status {
  position: absolute;
  bottom: 20px;
}

.not-found-console__code {
  left: 20px;
}

.not-found-console__status {
  right: 20px;
  color: var(--signal);
}

@media (max-width: 767px) {
  #not-found-page {
    grid-template-columns: 1fr;
    min-height: auto;
  }

  .not-found-console {
    min-height: 240px;
  }
}
</style>
