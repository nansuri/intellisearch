<script setup lang="ts">
import { dadJokes } from '../config/dadJokes'
import { useRotatingLine } from '../composables/useRotatingLine'

// Friendly "researching" mascot: a smiling orb with a magnifier sweeping
// around it and a rotating dad joke so waiting feels lighter. Used for the
// initial search and for follow-up questions (compact variant).
withDefaults(
  defineProps<{
    label?: string
    note?: string
    compact?: boolean
  }>(),
  { label: 'Researching your question', note: '', compact: false }
)

const { line } = useRotatingLine(dadJokes, 8000)
</script>

<template>
  <div class="research-loading" :class="{ 'research-loading--compact': compact }" aria-busy="true">
    <div class="research-stage" aria-hidden="true">
      <span class="research-glow" />
      <div class="research-orbit">
        <span class="research-lens" />
      </div>
      <span class="research-spark research-spark--1">✦</span>
      <span class="research-spark research-spark--2">✦</span>
      <span class="research-spark research-spark--3">✦</span>
      <div class="research-orb">
        <span class="research-shine" />
        <span class="research-eye research-eye--l" />
        <span class="research-eye research-eye--r" />
        <span class="research-cheek research-cheek--l" />
        <span class="research-cheek research-cheek--r" />
        <span class="research-smile" />
      </div>
    </div>

    <div class="research-copy">
      <p class="research-label" aria-live="polite">
        {{ label }}
        <span class="research-dots" aria-hidden="true"><i /><i /><i /></span>
      </p>
      <p class="research-note">{{ note }}</p>
      <p class="research-joke" aria-hidden="true">
        <svg class="research-bulb" viewBox="0 0 24 24" width="15" height="15" aria-hidden="true">
          <path d="M9 18h6M10 21h4M12 3a6 6 0 0 0-4 10.5c.8.7 1.3 1.6 1.4 2.5h5.2c.1-.9.6-1.8 1.4-2.5A6 6 0 0 0 12 3z" fill="none" stroke="currentColor" stroke-width="1.7" stroke-linejoin="round" />
        </svg>
        <span class="research-joke-text">
          <Transition name="research-joke" mode="out-in">
            <span :key="line">{{ line }}</span>
          </Transition>
        </span>
      </p>
    </div>
  </div>
</template>

<style scoped>
.research-loading {
  position: relative;
  overflow: hidden;
  display: flex;
  align-items: center;
  gap: 22px;
  margin-top: 38px;
  padding: 22px 24px;
  border: 1px solid color-mix(in srgb, var(--color-primary) 26%, var(--color-border));
  border-radius: var(--radius-md);
  background:
    radial-gradient(circle at 12% 50%, color-mix(in srgb, var(--color-primary) 7%, transparent), transparent 55%),
    var(--color-surface);
  box-shadow: var(--shadow-search);
}
.research-loading--compact {
  margin-top: 4px;
  padding: 14px 16px;
  border-color: color-mix(in srgb, var(--color-primary) 28%, var(--color-border));
  border-radius: 16px;
  background: color-mix(in srgb, var(--color-primary) 4%, var(--color-surface));
  box-shadow: none;
}

.research-stage {
  position: relative;
  flex: 0 0 auto;
  width: 108px;
  height: 108px;
}
.research-loading--compact .research-stage {
  width: 68px;
  height: 68px;
}

.research-glow {
  position: absolute;
  inset: 12%;
  border-radius: 50%;
  background: radial-gradient(circle, color-mix(in srgb, var(--color-primary) 22%, transparent), transparent 68%);
  animation: research-breathe 3s ease-in-out infinite;
}

.research-orb {
  position: absolute;
  left: 50%;
  top: 50%;
  width: 60px;
  height: 60px;
  margin: -30px 0 0 -30px;
  border-radius: 50%;
  background: linear-gradient(160deg, var(--color-primary), color-mix(in srgb, var(--color-primary) 72%, #000));
  box-shadow: 0 10px 24px color-mix(in srgb, var(--color-primary) 34%, transparent);
  animation: research-bob 3s ease-in-out infinite;
  will-change: transform;
}
.research-loading--compact .research-orb {
  width: 38px;
  height: 38px;
  margin: -19px 0 0 -19px;
}

.research-shine {
  position: absolute;
  left: 14%;
  top: 10%;
  width: 36%;
  height: 22%;
  border-radius: 50%;
  background: rgba(255, 255, 255, .35);
  transform: rotate(-18deg);
}

.research-eye {
  position: absolute;
  top: 36%;
  width: 8px;
  height: 10px;
  border-radius: 50%;
  background: var(--color-primary-contrast);
  transform-origin: 50% 80%;
  animation: research-blink 4.2s ease-in-out infinite;
}
.research-eye--l { left: 25%; }
.research-eye--r { right: 25%; }
.research-loading--compact .research-eye {
  width: 6px;
  height: 7px;
}

.research-cheek {
  position: absolute;
  top: 54%;
  width: 9px;
  height: 5px;
  border-radius: 50%;
  background: color-mix(in srgb, var(--color-danger) 30%, transparent);
}
.research-cheek--l { left: 16%; }
.research-cheek--r { right: 16%; }
.research-loading--compact .research-cheek {
  width: 6px;
  height: 4px;
}

.research-smile {
  position: absolute;
  left: 50%;
  top: 48%;
  width: 18px;
  height: 9px;
  transform: translateX(-50%);
  border-bottom: 2.5px solid var(--color-primary-contrast);
  border-radius: 0 0 60% 60%;
  box-sizing: border-box;
}
.research-loading--compact .research-smile {
  width: 13px;
  height: 7px;
  border-bottom-width: 2px;
}

.research-orbit {
  position: absolute;
  inset: 0;
  animation: research-spin 3.4s linear infinite;
  will-change: transform;
}
.research-lens {
  --orbit-r: 40px;
  position: absolute;
  left: 50%;
  top: 50%;
  width: 21px;
  height: 21px;
  margin: -10.5px 0 0 -10.5px;
  border: 3px solid var(--color-primary);
  border-radius: 50%;
  box-sizing: border-box;
  background: color-mix(in srgb, var(--color-primary) 12%, var(--color-surface));
  transform: translateY(calc(var(--orbit-r) * -1));
}
.research-lens::after {
  content: '';
  position: absolute;
  left: 50%;
  top: -12px;
  width: 3px;
  height: 12px;
  transform: translateX(-50%);
  border-radius: 3px;
  background: var(--color-primary);
}
.research-loading--compact .research-lens {
  --orbit-r: 26px;
  width: 16px;
  height: 16px;
  margin: -8px 0 0 -8px;
  border-width: 2.5px;
}
.research-loading--compact .research-lens::after {
  top: -10px;
  width: 2.5px;
  height: 10px;
}

.research-spark {
  position: absolute;
  left: 50%;
  top: 50%;
  color: var(--color-primary);
  font-size: 12px;
  line-height: 1;
  opacity: 0;
  animation: research-spark 2.6s ease-in-out infinite;
  will-change: transform, opacity;
}
.research-spark--1 { margin: -34px 0 0 22px; animation-delay: .2s; }
.research-spark--2 { margin: -6px 0 0 -48px; animation-delay: 1.2s; }
.research-spark--3 { margin: 30px 0 0 -6px; animation-delay: 2s; }
.research-loading--compact .research-spark--1 { margin: -22px 0 0 14px; }
.research-loading--compact .research-spark--2 { margin: -4px 0 0 -32px; }
.research-loading--compact .research-spark--3 { margin: 18px 0 0 -4px; }

.research-copy {
  display: grid;
  gap: 5px;
  min-width: 0;
  flex: 1;
}
.research-loading--compact .research-copy { gap: 3px; }

.research-label {
  display: flex;
  align-items: center;
  gap: 7px;
  margin: 0;
  color: var(--color-text);
  font-size: .94rem;
  font-weight: 720;
}
.research-loading--compact .research-label {
  font-size: .82rem;
  font-weight: 680;
}
.research-note {
  margin: 0;
  color: var(--color-muted);
  font-size: .8rem;
}
.research-loading--compact .research-note {
  font-size: .74rem;
}
.research-joke {
  display: flex;
  align-items: flex-start;
  gap: 8px;
  margin: 5px 0 0;
  color: var(--color-muted);
  font-size: .74rem;
  line-height: 1.5;
}
.research-loading--compact .research-joke {
  margin-top: 3px;
  font-size: .71rem;
}
.research-bulb {
  flex: 0 0 auto;
  margin-top: 1px;
  color: var(--color-primary);
  animation: research-bulb 2.8s ease-in-out infinite;
}
.research-joke-text { min-width: 0; }

.research-dots {
  display: inline-flex;
  gap: 3px;
}
.research-dots i {
  display: block;
  width: 4px;
  height: 4px;
  border-radius: 50%;
  background: var(--color-primary);
  animation: research-dot 1.2s ease-in-out infinite;
}
.research-dots i:nth-child(2) { animation-delay: .15s; }
.research-dots i:nth-child(3) { animation-delay: .3s; }

.research-joke-enter-active,
.research-joke-leave-active {
  transition: opacity .28s ease, transform .28s ease;
}
.research-joke-enter-from { opacity: 0; transform: translateY(4px); }
.research-joke-leave-to { opacity: 0; transform: translateY(-4px); }

@keyframes research-spin {
  to { transform: rotate(360deg); }
}
@keyframes research-bob {
  0%, 100% { transform: translateY(0); }
  50% { transform: translateY(-4px); }
}
@keyframes research-breathe {
  0%, 100% { opacity: .55; transform: scale(.96); }
  50% { opacity: .9; transform: scale(1.04); }
}
@keyframes research-blink {
  0%, 90%, 100% { transform: scaleY(1); }
  94%, 96% { transform: scaleY(.12); }
}
@keyframes research-spark {
  0% { opacity: 0; transform: translateY(6px) scale(.7); }
  30% { opacity: .9; }
  70% { opacity: .5; }
  100% { opacity: 0; transform: translateY(-10px) scale(1); }
}
@keyframes research-bulb {
  0%, 100% { opacity: .6; }
  50% { opacity: 1; }
}
@keyframes research-dot {
  0%, 70%, 100% { opacity: .25; transform: translateY(0); }
  35% { opacity: 1; transform: translateY(-2px); }
}

@media (prefers-reduced-motion: reduce) {
  .research-glow,
  .research-orb,
  .research-orbit,
  .research-spark,
  .research-bulb,
  .research-dots i,
  .research-eye {
    animation: none !important;
  }
  .research-joke-enter-active,
  .research-joke-leave-active {
    transition: none !important;
  }
}

@media (max-width: 400px) {
  .research-loading {
    flex-direction: column;
    align-items: center;
    text-align: center;
    gap: 14px;
  }
  .research-joke { text-align: left; justify-content: center; }
}
</style>