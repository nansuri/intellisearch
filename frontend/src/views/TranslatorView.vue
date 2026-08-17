<script setup lang="ts">
import { computed, onMounted, onUnmounted, ref, watch } from 'vue'
import { getTranslateLanguages, translate, type TranslateLanguage } from '../services/api'
import { useToastStore } from '../stores/toast'
import PageHeader from '../components/PageHeader.vue'
import LoadingSpinner from '../components/LoadingSpinner.vue'

const toast = useToastStore()
const languages = ref<TranslateLanguage[]>([])
const languagesError = ref('')
const input = ref('')
const output = ref('')
const sourceLang = ref('auto')
const targetLang = ref('en')
const busy = ref(false)
const translateError = ref('')
const copied = ref(false)

const MAX_LENGTH = 5000
const sourceLabel = computed(() => (sourceLang.value === 'auto' ? 'Detect language' : langName(sourceLang.value)))
const targetLabel = computed(() => langName(targetLang.value))
const charCount = computed(() => `${input.value.length}/${MAX_LENGTH}`)

function langName(code: string) {
  return languages.value.find((l) => l.code === code)?.name || code
}

let timer: ReturnType<typeof setTimeout> | null = null
function scheduleTranslate() {
  if (timer) clearTimeout(timer)
  timer = setTimeout(runTranslate, 400)
}

async function runTranslate() {
  const text = input.value.trim()
  if (!text) {
    if (timer) clearTimeout(timer)
    output.value = ''
    busy.value = false
    translateError.value = ''
    return
  }
  if (text.length > MAX_LENGTH) {
    translateError.value = `Text is limited to ${MAX_LENGTH.toLocaleString()} characters.`
    return
  }
  busy.value = true
  translateError.value = ''
  try {
    const result = await translate({ q: text, source: sourceLang.value, target: targetLang.value, format: 'text' })
    output.value = result.translatedText
    copied.value = false
  } catch (e) {
    translateError.value = (e as Error).message
    output.value = ''
  } finally {
    busy.value = false
  }
}

function onInput() {
  scheduleTranslate()
}
function onClear() {
  input.value = ''
  output.value = ''
  translateError.value = ''
  if (timer) clearTimeout(timer)
}

async function swap() {
  const previousSource = sourceLang.value
  if (sourceLang.value === 'auto') {
    sourceLang.value = targetLang.value
    targetLang.value = 'en'
  } else {
    sourceLang.value = targetLang.value
    targetLang.value = previousSource
  }
  if (input.value.trim()) {
    // Swap the texts too so the translated text becomes the new input.
    const current = input.value
    input.value = output.value || ''
    output.value = current
  }
  scheduleTranslate()
}

async function copyOutput() {
  if (!output.value) return
  try {
    await navigator.clipboard.writeText(output.value)
    copied.value = true
    setTimeout(() => (copied.value = false), 1600)
  } catch {
    toast.error('Could not copy — your browser blocked clipboard access.')
  }
}

watch([sourceLang, targetLang], scheduleTranslate)
onMounted(async () => {
  try {
    languages.value = (await getTranslateLanguages()).languages
    if (!languages.value.some((l) => l.code === targetLang.value) && languages.value.length) {
      targetLang.value = languages.value[0].code
    }
  } catch (e) {
    languagesError.value = (e as Error).message
  }
})
onUnmounted(() => { if (timer) clearTimeout(timer) })
</script>

<template>
  <main class="page-shell translator-page">
    <PageHeader eyebrow="Apps" title="Translator" description="Translate text between languages — powered by LibreTranslate." />
    <p v-if="languagesError" class="translator-error" role="alert">{{ languagesError }}</p>

    <section class="translator-card">
      <div class="translator-panel translator-panel--source">
        <div class="translator-panel-head">
          <select v-model="sourceLang" class="translator-lang" aria-label="Source language">
            <option value="auto">Detect language</option>
            <option v-for="l in languages" :key="l.code" :value="l.code">{{ l.name }}</option>
          </select>
          <button type="button" class="translator-icon-btn" title="Clear" aria-label="Clear" @click="onClear">
            <svg viewBox="0 0 24 24" width="16" height="16" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round"><path d="M18 6 6 18M6 6l12 12" /></svg>
          </button>
        </div>
        <textarea
          v-model="input"
          class="translator-textarea"
          :maxlength="MAX_LENGTH"
          placeholder="Enter text"
          aria-label="Text to translate"
          @input="onInput"
        ></textarea>
        <div class="translator-panel-foot">
          <span class="translator-char-count">{{ charCount }}</span>
          <span v-if="busy" class="translator-busy"><span class="translator-busy-dot" />Translating…</span>
        </div>
      </div>

      <button type="button" class="translator-swap" title="Swap languages" aria-label="Swap languages" @click="swap">
        <svg viewBox="0 0 24 24" width="18" height="18" fill="none" stroke="currentColor" stroke-width="1.8" stroke-linecap="round" stroke-linejoin="round"><path d="M7 16V4m0 0L3 8m4-4 4 4M17 8v12m0 0 4-4m-4 4-4-4" /></svg>
      </button>

      <div class="translator-panel translator-panel--target">
        <div class="translator-panel-head">
          <select v-model="targetLang" class="translator-lang" aria-label="Target language">
            <option v-for="l in languages" :key="l.code" :value="l.code">{{ l.name }}</option>
          </select>
          <button type="button" class="translator-icon-btn" title="Copy translation" aria-label="Copy translation" :disabled="!output" @click="copyOutput">
            <svg v-if="!copied" viewBox="0 0 24 24" width="16" height="16" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><rect x="9" y="9" width="13" height="13" rx="2" /><path d="M5 15H4a2 2 0 0 1-2-2V4a2 2 0 0 1 2-2h9a2 2 0 0 1 2 2v1" /></svg>
            <svg v-else viewBox="0 0 24 24" width="16" height="16" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><path d="M20 6 9 17l-5-5" /></svg>
          </button>
        </div>
        <div class="translator-output" :aria-busy="busy">
          <LoadingSpinner v-if="busy" />
          <p v-else-if="output">{{ output }}</p>
          <p v-else-if="translateError" class="translator-output-error">{{ translateError }}</p>
          <p v-else class="translator-output-placeholder">Translation</p>
        </div>
        <div class="translator-panel-foot">
          <span v-if="copied" class="translator-copied">Copied</span>
          <span v-else class="translator-panel-tag">{{ sourceLabel }} → {{ targetLabel }}</span>
        </div>
      </div>
    </section>
  </main>
</template>

<style scoped>
.translator-page { padding-top: 8px; }
.translator-error {
  margin: 0 0 16px;
  padding: 12px 14px;
  border: 1px solid color-mix(in srgb, var(--color-danger) 35%, var(--color-border));
  border-radius: 12px;
  background: color-mix(in srgb, var(--color-danger) 8%, var(--color-surface));
  color: var(--color-danger);
  font-size: .84rem;
}
.translator-card {
  display: grid;
  grid-template-columns: 1fr auto 1fr;
  gap: 18px;
  margin-top: 24px;
  padding: 22px;
  border: 1px solid var(--color-border);
  border-radius: 20px;
  background: var(--color-surface);
  box-shadow: 0 18px 44px var(--color-shadow);
}
.translator-panel {
  display: grid;
  grid-template-rows: auto 1fr auto;
  gap: 12px;
  min-height: 260px;
  padding: 16px;
  border: 1px solid var(--color-border);
  border-radius: 14px;
  background: var(--color-bg);
}
.translator-panel-head {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 10px;
  padding-bottom: 10px;
  border-bottom: 1px solid var(--color-border);
}
.translator-lang {
  min-width: 0;
  flex: 1;
  padding: 0;
  border: 0;
  background: transparent;
  color: var(--color-text);
  font-size: .86rem;
  font-weight: 680;
  cursor: pointer;
}
.translator-icon-btn {
  display: inline-grid;
  place-items: center;
  width: 32px;
  height: 32px;
  flex: 0 0 auto;
  border: 0;
  border-radius: 8px;
  background: transparent;
  color: var(--color-muted);
  cursor: pointer;
  transition: background .14s ease, color .14s ease;
}
.translator-icon-btn:hover:not(:disabled) { background: var(--color-surface-subtle); color: var(--color-text); }
.translator-icon-btn:disabled { opacity: .4; cursor: not-allowed; }
.translator-textarea {
  width: 100%;
  min-height: 150px;
  flex: 1;
  resize: none;
  border: 0;
  outline: 0;
  background: transparent;
  color: var(--color-text);
  font-size: 1rem;
  line-height: 1.6;
}
.translator-textarea::placeholder { color: var(--color-muted); }
.translator-output {
  display: flex;
  align-items: flex-start;
  min-height: 150px;
  flex: 1;
}
.translator-output p { margin: 0; font-size: 1rem; line-height: 1.6; }
.translator-output .spinner { width: 22px; height: 22px; }
.translator-output-placeholder { color: var(--color-muted); }
.translator-output-error { color: var(--color-danger); }
.translator-panel-foot {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 10px;
  min-height: 18px;
  color: var(--color-muted);
  font-size: .74rem;
}
.translator-char-count { font-variant-numeric: tabular-nums; }
.translator-busy { display: inline-flex; align-items: center; gap: 6px; }
.translator-busy-dot {
  width: 7px;
  height: 7px;
  border-radius: 50%;
  background: var(--color-primary);
  animation: translator-pulse 1.2s ease-in-out infinite;
}
.translator-copied { color: #34d399; font-weight: 700; }
.translator-panel-tag { overflow: hidden; text-overflow: ellipsis; white-space: nowrap; }
.translator-swap {
  display: inline-grid;
  place-items: center;
  align-self: center;
  width: 42px;
  height: 42px;
  border: 1px solid var(--color-border);
  border-radius: 50%;
  background: var(--color-surface);
  color: var(--color-primary);
  cursor: pointer;
  transition: border-color .16s ease, transform .16s ease;
}
.translator-swap:hover { border-color: var(--color-primary); transform: rotate(180deg); }
@keyframes translator-pulse {
  0%, 100% { opacity: 1; }
  50% { opacity: .35; }
}
@media (max-width: 860px) {
  .translator-card { grid-template-columns: 1fr; }
  .translator-swap { justify-self: center; transform: rotate(90deg); }
  .translator-swap:hover { transform: rotate(270deg); }
}
</style>
