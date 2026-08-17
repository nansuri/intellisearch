<template>
  <div class="follow-up-loading" aria-live="polite" aria-busy="true">
    <div class="follow-up-loading-glow" aria-hidden="true" />
    <div class="follow-up-loading-head">
      <span class="follow-up-loading-dot" />
      <span class="follow-up-loading-label">Researching your follow-up</span>
      <span class="follow-up-loading-dots" aria-hidden="true"><i /><i /><i /></span>
    </div>
    <div class="follow-up-loading-body">
      <div class="skeleton-line follow-up-loading-line follow-up-loading-line--1" />
      <div class="skeleton-line follow-up-loading-line follow-up-loading-line--2" />
      <div class="skeleton-line follow-up-loading-line follow-up-loading-line--3 skeleton-line--short" />
    </div>
    <p class="follow-up-loading-note">Searching the web and synthesizing a new answer…</p>
  </div>
</template>

<style scoped>
.follow-up-loading {
  position: relative;
  overflow: hidden;
  margin-top: 4px;
  padding: 18px 18px 16px;
  border: 1px solid color-mix(in srgb, var(--color-primary) 28%, var(--color-border));
  border-radius: 16px;
  background: color-mix(in srgb, var(--color-primary) 4%, var(--color-surface));
}
.follow-up-loading-glow {
  position: absolute;
  inset: -40% -20%;
  background: radial-gradient(circle at 30% 40%, color-mix(in srgb, var(--color-primary) 16%, transparent), transparent 58%);
  animation: follow-up-glow 2.4s ease-in-out infinite;
  pointer-events: none;
}
.follow-up-loading-head {
  position: relative;
  display: flex;
  align-items: center;
  gap: 8px;
  margin-bottom: 14px;
}
.follow-up-loading-dot {
  width: 8px;
  height: 8px;
  border-radius: 50%;
  background: var(--color-primary);
  animation: follow-up-pulse 1.1s ease-in-out infinite;
}
.follow-up-loading-label {
  color: var(--color-text);
  font-size: .82rem;
  font-weight: 680;
}
.follow-up-loading-dots {
  display: inline-flex;
  gap: 3px;
  margin-left: 2px;
}
.follow-up-loading-dots i {
  display: block;
  width: 4px;
  height: 4px;
  border-radius: 50%;
  background: var(--color-primary);
  animation: follow-up-dot 1.2s ease-in-out infinite;
}
.follow-up-loading-dots i:nth-child(2) { animation-delay: .15s; }
.follow-up-loading-dots i:nth-child(3) { animation-delay: .3s; }
.follow-up-loading-body { position: relative; display: grid; gap: 10px; }
.follow-up-loading-line--1 { animation-delay: 0s; }
.follow-up-loading-line--2 { animation-delay: .12s; }
.follow-up-loading-line--3 { animation-delay: .24s; }
.follow-up-loading-note {
  position: relative;
  margin: 14px 0 0;
  color: var(--color-muted);
  font-size: .74rem;
}
@keyframes follow-up-glow {
  0%, 100% { opacity: .45; transform: translateX(0); }
  50% { opacity: .85; transform: translateX(6%); }
}
@keyframes follow-up-pulse {
  0%, 100% { transform: scale(1); opacity: .65; }
  50% { transform: scale(1.25); opacity: 1; }
}
@keyframes follow-up-dot {
  0%, 70%, 100% { opacity: .25; transform: translateY(0); }
  35% { opacity: 1; transform: translateY(-2px); }
}
</style>
