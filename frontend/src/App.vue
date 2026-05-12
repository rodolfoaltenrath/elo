<script setup>
import { computed, onMounted, ref } from 'vue'
import {
  AutoFix,
  CheckStatus,
  InstallDriverFile,
  LaunchProcessFile,
  SelectDriverFile,
  SelectProcessFile
} from '../wailsjs/go/main/App'

const status = ref(null)
const selectedFile = ref('')
const busy = ref('')
const notice = ref({ type: '', text: '' })
const isDragging = ref(false)

const statusCards = computed(() => {
  if (!status.value) return []

  return [
    status.value.java8,
    status.value.tokenService,
    status.value.tokenReader,
    status.value.certificate,
    status.value.browserPkcs11
  ]
})

const technicalCards = computed(() => {
  if (!status.value) return []
  return [
    ...status.value.packages
  ]
})

const readyLabel = computed(() => {
  if (!status.value) return 'Verificando ambiente'
  return status.value.allOk ? 'Ambiente pronto' : 'Ajustes necessários'
})

const healthSummary = computed(() => {
  const cards = [...statusCards.value, ...technicalCards.value]
  if (!cards.length) return { ok: 0, total: 0 }
  return {
    ok: cards.filter((item) => item.ok).length,
    total: cards.length
  }
})

const nextAction = computed(() => {
  if (!status.value) {
    return {
      title: 'Verificando o computador',
      text: 'Aguarde alguns segundos enquanto o ambiente é analisado.',
      tone: 'wait'
    }
  }

  const failing = [...statusCards.value, ...technicalCards.value].find((item) => !item.ok)
  if (failing) {
    return {
      title: failing.name,
      text: failing.remediation || failing.detail,
      tone: 'warn'
    }
  }

  if (!selectedFile.value) {
    return {
      title: 'Tudo pronto',
      text: 'Selecione o arquivo do tribunal para abrir o assinador.',
      tone: 'ok'
    }
  }

  return {
    title: 'Arquivo selecionado',
    text: 'Clique em Abrir com Java 8 para iniciar o assinador.',
    tone: 'ok'
  }
})

const workflowSteps = computed(() => [
  {
    label: 'Preparar',
    ok: Boolean(status.value?.allOk),
    active: !status.value?.allOk
  },
  {
    label: 'Selecionar',
    ok: Boolean(selectedFile.value),
    active: Boolean(status.value?.allOk && !selectedFile.value)
  },
  {
    label: 'Assinar',
    ok: false,
    active: Boolean(status.value?.allOk && selectedFile.value)
  }
])

const fileName = computed(() => {
  if (!selectedFile.value) return ''
  return selectedFile.value.split(/[\\/]/).pop()
})

const driverHelp = computed(() => {
  if (!status.value || status.value.tokenReader.ok) return null

  const detail = `${status.value.tokenReader.detail} ${status.value.tokenReader.remediation}`.toLowerCase()
  if (!detail.includes('driver') && !detail.includes('fabricante') && !detail.includes('opensc')) {
    return null
  }

  return {
    token: 'Giesecke & Devrient StarSign CUT S',
    driver: 'SafeSign Identity Client para Linux',
    search: 'SafeSign Identity Client Linux StarSign CUT S download',
    library: 'libaetpkss.so',
    formats: '.deb, .rpm, .tar.gz ou .so'
  }
})

onMounted(refreshStatus)

async function refreshStatus() {
  await runBusy('status', async () => {
    status.value = await CheckStatus()
  })
}

async function fixProblems() {
  await runBusy('fix', async () => {
    const result = await AutoFix()
    setNotice(result.ok ? 'success' : 'error', result.message)
    status.value = await CheckStatus()
  })
}

async function chooseFile() {
  await runBusy('file', async () => {
    const path = await SelectProcessFile()
    if (path) {
      selectedFile.value = path
      setNotice('success', 'Arquivo selecionado.')
    }
  })
}

async function installDriver() {
  await runBusy('driver', async () => {
    const path = await SelectDriverFile()
    if (!path) return

    const result = await InstallDriverFile(path)
    setNotice(result.ok ? 'success' : 'error', result.message)
    status.value = await CheckStatus()
  })
}

async function launchFile() {
  if (!selectedFile.value) {
    setNotice('error', 'Selecione um arquivo .jnlp ou .jar.')
    return
  }

  await runBusy('launch', async () => {
    const result = await LaunchProcessFile(selectedFile.value)
    setNotice(result.ok ? 'success' : 'error', result.message)
    status.value = await CheckStatus()
  })
}

async function copyDiagnostics() {
  if (!status.value) return

  const lines = [
    'Elo - Relatório do ambiente',
    `Verificado em: ${status.value.checkedAt}`,
    `Arquivo selecionado: ${selectedFile.value || 'nenhum'}`,
    '',
    ...[...statusCards.value, ...technicalCards.value].map((item) => {
      const state = item.ok ? 'OK' : 'PENDENTE'
      return `${state} - ${item.name}: ${item.detail}${item.ok ? '' : ` | ${item.remediation}`}`
    })
  ]

  await navigator.clipboard.writeText(lines.join('\n'))
  setNotice('success', 'Relatório copiado.')
}

function handleDrop(event) {
  isDragging.value = false
  const file = event.dataTransfer?.files?.[0]

  if (!file) return

  const path = file.path || file.webkitRelativePath
  if (!path) {
    setNotice('error', 'Use o botão Selecionar arquivo neste ambiente.')
    return
  }

  if (!/\.(jnlp|jar)$/i.test(path)) {
    setNotice('error', 'Use apenas arquivos .jnlp ou .jar.')
    return
  }

  selectedFile.value = path
  setNotice('success', 'Arquivo pronto para execução.')
}

async function runBusy(name, task) {
  if (busy.value) return

  busy.value = name
  notice.value = { type: '', text: '' }

  try {
    await task()
  } catch (error) {
    setNotice('error', error?.message || String(error))
  } finally {
    busy.value = ''
  }
}

function setNotice(type, text) {
  notice.value = { type, text }
}
</script>

<template>
  <main class="shell">
    <section class="topbar">
      <div>
        <p class="eyebrow">Linux + Certificado Digital</p>
        <h1>Elo</h1>
      </div>

      <button class="ghost-button" :disabled="Boolean(busy)" @click="refreshStatus">
        <span class="button-icon">↻</span>
        Atualizar
      </button>
    </section>

    <section class="hero">
      <div class="hero-copy">
        <p class="state-pill" :class="{ ok: status?.allOk }">
          <span></span>
          {{ readyLabel }}
        </p>
        <h2>{{ nextAction.title }}</h2>
        <p>{{ nextAction.text }}</p>

        <div class="workflow">
          <div
            v-for="step in workflowSteps"
            :key="step.label"
            class="workflow-step"
            :class="{ ok: step.ok, active: step.active }"
          >
            <span></span>
            {{ step.label }}
          </div>
        </div>
      </div>

      <div class="action-panel">
        <div class="health-meter">
          <strong>{{ healthSummary.ok }}/{{ healthSummary.total || '...' }}</strong>
          <span>itens prontos</span>
        </div>

        <button class="primary-button" :disabled="Boolean(busy)" @click="fixProblems">
          <span v-if="busy === 'fix'" class="spinner"></span>
          <span v-else class="button-icon">✓</span>
          {{ busy === 'fix' ? 'Corrigindo...' : 'Corrigir tudo automaticamente' }}
        </button>

        <button class="secondary-button" :disabled="Boolean(busy)" @click="installDriver">
          <span v-if="busy === 'driver'" class="spinner"></span>
          <span v-else class="button-icon">＋</span>
          {{ busy === 'driver' ? 'Instalando...' : 'Instalar driver baixado' }}
        </button>

        <button class="tertiary-button" :disabled="Boolean(busy) || !status" @click="copyDiagnostics">
          <span class="button-icon">⧉</span>
          Copiar relatório
        </button>

        <p class="panel-note">
          Quando tudo estiver verde, o computador está pronto. Se o tribunal ainda falhar,
          o próximo suspeito é sessão expirada ou instabilidade do próprio sistema.
        </p>
      </div>
    </section>

    <section v-if="notice.text" class="notice" :class="notice.type">
      {{ notice.text }}
    </section>

    <section v-if="driverHelp" class="driver-guide">
      <div class="section-title">
        <p>Driver Recomendado</p>
        <span>{{ driverHelp.token }}</span>
      </div>

      <div class="driver-guide-grid">
        <div class="driver-summary">
          <small>Procure por</small>
          <strong>{{ driverHelp.driver }}</strong>
          <code>{{ driverHelp.search }}</code>
        </div>

        <ol class="driver-steps">
          <li>Busque esse nome no site da sua certificadora ou do fabricante do token.</li>
          <li>Baixe a versão Linux em {{ driverHelp.formats }}.</li>
          <li>Clique em Instalar driver baixado e selecione o arquivo.</li>
          <li>Reconecte o token e clique em Atualizar.</li>
        </ol>
      </div>
    </section>

    <section class="content-grid">
      <div class="status-board">
        <div class="section-title">
          <p>Diagnóstico</p>
          <span v-if="status">Última verificação: {{ status.checkedAt }}</span>
        </div>

        <div v-if="!status" class="loading-block">
          <span class="spinner"></span>
          Verificando dependências
        </div>

        <div v-else class="cards">
          <article v-for="item in statusCards" :key="item.name" class="status-card">
            <div class="status-head">
              <span class="status-dot" :class="{ ok: item.ok }"></span>
              <strong>{{ item.name }}</strong>
              <em :class="{ ok: item.ok }">{{ item.ok ? 'Pronto' : 'Ação' }}</em>
            </div>
            <p>{{ item.detail }}</p>
            <small v-if="!item.ok">{{ item.remediation }}</small>
          </article>
        </div>

        <details v-if="technicalCards.length" class="technical-details">
          <summary>Detalhes técnicos</summary>
          <div class="mini-cards">
            <article v-for="item in technicalCards" :key="item.name" class="mini-card">
              <span class="status-dot" :class="{ ok: item.ok }"></span>
              <strong>{{ item.name }}</strong>
              <small>{{ item.detail }}</small>
            </article>
          </div>
        </details>
      </div>

      <div class="launcher">
        <div class="section-title">
          <p>Executar Arquivo</p>
          <span>.jnlp ou .jar</span>
        </div>

        <button
          class="dropzone"
          :class="{ dragging: isDragging, loaded: selectedFile }"
          @click="chooseFile"
          @dragover.prevent="isDragging = true"
          @dragleave.prevent="isDragging = false"
          @drop.prevent="handleDrop"
        >
          <span class="drop-icon">⇣</span>
          <strong>{{ fileName || 'Arraste o arquivo do tribunal aqui' }}</strong>
          <small>{{ selectedFile || 'ou clique para selecionar no computador' }}</small>
        </button>

        <button class="launch-button" :disabled="Boolean(busy) || !selectedFile" @click="launchFile">
          <span v-if="busy === 'launch'" class="spinner"></span>
          <span v-else class="button-icon">▶</span>
          {{ busy === 'launch' ? 'Iniciando...' : 'Abrir com Java 8' }}
        </button>
      </div>
    </section>
  </main>
</template>

<style>
:root {
  color: #edf2f4;
  background: #090d12;
  font-family:
    Inter, ui-sans-serif, system-ui, -apple-system, BlinkMacSystemFont, "Segoe UI",
    sans-serif;
  font-synthesis: none;
  text-rendering: geometricPrecision;
}

* {
  box-sizing: border-box;
}

body {
  min-width: 360px;
  min-height: 100vh;
  margin: 0;
  background:
    radial-gradient(circle at 20% 10%, rgba(50, 121, 168, 0.24), transparent 30%),
    linear-gradient(135deg, #090d12 0%, #111821 54%, #17130f 100%);
}

button {
  font: inherit;
}

.shell {
  width: min(1180px, calc(100vw - 40px));
  min-height: 100vh;
  margin: 0 auto;
  padding: 28px 0 36px;
}

.topbar,
.hero,
.content-grid,
.section-title,
.status-head {
  display: flex;
  align-items: center;
}

.topbar {
  justify-content: space-between;
  gap: 18px;
  margin-bottom: 24px;
}

.eyebrow,
.section-title p {
  margin: 0;
  color: #59c3c3;
  font-size: 0.78rem;
  font-weight: 800;
  letter-spacing: 0;
  text-transform: uppercase;
}

h1,
h2,
p {
  margin-top: 0;
}

h1 {
  margin-bottom: 0;
  font-size: clamp(1.6rem, 4vw, 2.4rem);
}

h2 {
  max-width: 720px;
  margin-bottom: 14px;
  font-size: clamp(2rem, 5vw, 4.4rem);
  line-height: 0.98;
}

.hero {
  justify-content: space-between;
  gap: 28px;
  min-height: 286px;
  padding: 34px 0;
  border-top: 1px solid rgba(255, 255, 255, 0.1);
  border-bottom: 1px solid rgba(255, 255, 255, 0.1);
}

.hero-copy > p:last-child {
  max-width: 640px;
  margin-bottom: 0;
  color: #adbac7;
  font-size: 1.04rem;
  line-height: 1.7;
}

.workflow {
  display: flex;
  flex-wrap: wrap;
  gap: 10px;
  margin-top: 22px;
}

.workflow-step {
  display: inline-flex;
  align-items: center;
  gap: 9px;
  min-height: 36px;
  padding: 0 12px;
  border: 1px solid rgba(255, 255, 255, 0.1);
  border-radius: 999px;
  color: #8f9dad;
  background: rgba(255, 255, 255, 0.04);
  font-size: 0.9rem;
  font-weight: 850;
}

.workflow-step span {
  width: 9px;
  height: 9px;
  border-radius: 50%;
  background: #647386;
}

.workflow-step.active {
  border-color: rgba(255, 181, 71, 0.36);
  color: #ffd28a;
  background: rgba(255, 181, 71, 0.08);
}

.workflow-step.active span {
  background: #ffb547;
  box-shadow: 0 0 14px rgba(255, 181, 71, 0.64);
}

.workflow-step.ok {
  border-color: rgba(58, 214, 159, 0.3);
  color: #a7f2d4;
  background: rgba(58, 214, 159, 0.08);
}

.workflow-step.ok span {
  background: #3ad69f;
  box-shadow: 0 0 14px rgba(58, 214, 159, 0.64);
}

.state-pill {
  display: inline-flex;
  align-items: center;
  gap: 10px;
  margin-bottom: 18px;
  padding: 8px 12px;
  border: 1px solid rgba(255, 181, 71, 0.32);
  border-radius: 999px;
  color: #ffd28a;
  background: rgba(255, 181, 71, 0.08);
  font-size: 0.9rem;
  font-weight: 800;
}

.state-pill span,
.status-dot {
  width: 10px;
  height: 10px;
  border-radius: 50%;
  background: #ffb547;
  box-shadow: 0 0 18px rgba(255, 181, 71, 0.72);
}

.state-pill.ok {
  border-color: rgba(58, 214, 159, 0.35);
  color: #70e6bb;
  background: rgba(58, 214, 159, 0.08);
}

.state-pill.ok span,
.status-dot.ok {
  background: #3ad69f;
  box-shadow: 0 0 18px rgba(58, 214, 159, 0.72);
}

.action-panel,
.status-board,
.launcher,
.driver-guide,
.notice {
  border: 1px solid rgba(255, 255, 255, 0.1);
  border-radius: 8px;
  background: rgba(13, 19, 27, 0.78);
  box-shadow: 0 24px 80px rgba(0, 0, 0, 0.32);
}

.action-panel {
  width: min(330px, 100%);
  flex: 0 0 auto;
  padding: 18px;
}

.health-meter {
  display: flex;
  min-height: 58px;
  align-items: baseline;
  justify-content: center;
  gap: 8px;
  margin-bottom: 14px;
  border: 1px solid rgba(58, 214, 159, 0.18);
  border-radius: 8px;
  background: rgba(58, 214, 159, 0.06);
}

.health-meter strong {
  color: #70e6bb;
  font-size: 1.55rem;
}

.health-meter span {
  color: #aeb9c5;
  font-size: 0.9rem;
}

.panel-note {
  margin: 12px 0 0;
  color: #9aa8b6;
  font-size: 0.88rem;
  line-height: 1.55;
}

.ghost-button,
.primary-button,
.launch-button {
  display: inline-flex;
  align-items: center;
  justify-content: center;
  gap: 10px;
  min-height: 46px;
  border: 0;
  border-radius: 8px;
  color: #f8fbff;
  cursor: pointer;
  transition:
    transform 150ms ease,
    opacity 150ms ease,
    background 150ms ease;
}

.ghost-button {
  padding: 0 16px;
  border: 1px solid rgba(255, 255, 255, 0.12);
  background: rgba(255, 255, 255, 0.06);
}

.primary-button,
.launch-button {
  width: 100%;
  min-height: 58px;
  background: linear-gradient(135deg, #1f9d8a, #3279a8);
  font-size: 1rem;
  font-weight: 900;
}

.secondary-button,
.tertiary-button,
.launch-button {
  margin-top: 16px;
}

.secondary-button,
.tertiary-button {
  display: inline-flex;
  width: 100%;
  min-height: 52px;
  align-items: center;
  justify-content: center;
  gap: 10px;
  border: 1px solid rgba(89, 195, 195, 0.32);
  border-radius: 8px;
  color: #f8fbff;
  background: rgba(89, 195, 195, 0.1);
  cursor: pointer;
  font-weight: 850;
  transition:
    transform 150ms ease,
    opacity 150ms ease,
    background 150ms ease;
}

.tertiary-button {
  min-height: 46px;
  border-color: rgba(255, 255, 255, 0.12);
  background: rgba(255, 255, 255, 0.055);
  font-size: 0.92rem;
  font-weight: 800;
}

.launch-button {
  background: linear-gradient(135deg, #3279a8, #d36b46);
}

.ghost-button:hover,
.primary-button:hover,
.secondary-button:hover,
.tertiary-button:hover,
.launch-button:hover,
.dropzone:hover {
  transform: translateY(-1px);
}

button:disabled {
  cursor: not-allowed;
  opacity: 0.52;
  transform: none;
}

.button-icon {
  display: inline-grid;
  place-items: center;
  width: 24px;
  height: 24px;
  border-radius: 50%;
  background: rgba(255, 255, 255, 0.14);
  font-weight: 900;
}

.notice {
  margin-top: 18px;
  padding: 14px 16px;
  color: #d7f8e9;
  background: rgba(58, 214, 159, 0.1);
}

.notice.error {
  color: #ffd2cb;
  background: rgba(211, 107, 70, 0.12);
}

.driver-guide {
  margin-top: 18px;
  padding: 18px;
}

.driver-guide-grid {
  display: grid;
  grid-template-columns: minmax(260px, 0.8fr) minmax(0, 1.2fr);
  gap: 16px;
}

.driver-summary {
  display: flex;
  min-height: 152px;
  flex-direction: column;
  justify-content: center;
  gap: 10px;
  padding: 16px;
  border: 1px solid rgba(89, 195, 195, 0.2);
  border-radius: 8px;
  background: rgba(89, 195, 195, 0.06);
}

.driver-summary small {
  color: #59c3c3;
  font-weight: 800;
  text-transform: uppercase;
}

.driver-summary strong {
  font-size: 1.15rem;
}

.driver-summary code {
  display: block;
  padding: 10px 12px;
  border-radius: 6px;
  color: #ffd28a;
  background: rgba(0, 0, 0, 0.26);
  font-family: ui-monospace, SFMono-Regular, Menlo, Monaco, Consolas, monospace;
  overflow-wrap: anywhere;
}

.driver-steps {
  display: grid;
  gap: 10px;
  margin: 0;
  padding: 0;
  list-style-position: inside;
}

.driver-steps li {
  min-height: 32px;
  padding: 10px 12px;
  border: 1px solid rgba(255, 255, 255, 0.08);
  border-radius: 8px;
  color: #c6d0dc;
  background: rgba(255, 255, 255, 0.035);
  line-height: 1.45;
}

.content-grid {
  align-items: stretch;
  display: grid;
  grid-template-columns: minmax(0, 1fr) minmax(320px, 420px);
  gap: 18px;
  margin-top: 18px;
}

.status-board,
.launcher {
  padding: 18px;
}

.section-title {
  justify-content: space-between;
  gap: 12px;
  margin-bottom: 16px;
}

.section-title span {
  color: #8190a0;
  font-size: 0.86rem;
}

.cards {
  display: grid;
  grid-template-columns: repeat(2, minmax(0, 1fr));
  gap: 12px;
}

.status-card {
  min-height: 134px;
  padding: 16px;
  border: 1px solid rgba(255, 255, 255, 0.08);
  border-radius: 8px;
  background: rgba(255, 255, 255, 0.035);
}

.status-head {
  justify-content: space-between;
  gap: 10px;
  min-height: 28px;
}

.status-head strong {
  flex: 1;
}

.status-head em {
  padding: 4px 8px;
  border-radius: 999px;
  color: #ffd28a;
  background: rgba(255, 181, 71, 0.09);
  font-size: 0.74rem;
  font-style: normal;
  font-weight: 900;
}

.status-head em.ok {
  color: #70e6bb;
  background: rgba(58, 214, 159, 0.09);
}

.status-card p {
  margin: 12px 0 0;
  color: #aeb9c5;
  line-height: 1.5;
}

.status-card small {
  display: block;
  margin-top: 12px;
  color: #ffd28a;
  line-height: 1.45;
}

.technical-details {
  margin-top: 14px;
  border-top: 1px solid rgba(255, 255, 255, 0.08);
  padding-top: 14px;
}

.technical-details summary {
  color: #8f9dad;
  cursor: pointer;
  font-weight: 850;
}

.mini-cards {
  display: grid;
  grid-template-columns: repeat(3, minmax(0, 1fr));
  gap: 10px;
  margin-top: 12px;
}

.mini-card {
  display: grid;
  grid-template-columns: auto 1fr;
  gap: 6px 8px;
  align-items: center;
  min-height: 78px;
  padding: 12px;
  border: 1px solid rgba(255, 255, 255, 0.08);
  border-radius: 8px;
  background: rgba(255, 255, 255, 0.03);
}

.mini-card small {
  grid-column: 2;
  color: #9aa8b6;
}

.dropzone {
  display: flex;
  width: 100%;
  min-height: 260px;
  flex-direction: column;
  align-items: center;
  justify-content: center;
  gap: 10px;
  padding: 22px;
  border: 1px dashed rgba(89, 195, 195, 0.52);
  border-radius: 8px;
  color: #edf2f4;
  background: rgba(89, 195, 195, 0.06);
  cursor: pointer;
  text-align: center;
  transition:
    border-color 150ms ease,
    background 150ms ease,
    transform 150ms ease;
}

.dropzone.dragging,
.dropzone.loaded {
  border-color: rgba(58, 214, 159, 0.9);
  background: rgba(58, 214, 159, 0.09);
}

.dropzone strong {
  max-width: 100%;
  overflow-wrap: anywhere;
  font-size: 1.05rem;
}

.dropzone small {
  max-width: 100%;
  color: #93a2b1;
  overflow-wrap: anywhere;
  line-height: 1.45;
}

.drop-icon {
  display: grid;
  place-items: center;
  width: 58px;
  height: 58px;
  border-radius: 50%;
  color: #090d12;
  background: #59c3c3;
  font-size: 2rem;
  font-weight: 900;
}

.loading-block {
  display: flex;
  min-height: 280px;
  align-items: center;
  justify-content: center;
  gap: 12px;
  color: #aeb9c5;
}

.spinner {
  width: 20px;
  height: 20px;
  border: 3px solid rgba(255, 255, 255, 0.28);
  border-top-color: #ffffff;
  border-radius: 50%;
  animation: spin 800ms linear infinite;
}

@keyframes spin {
  to {
    transform: rotate(360deg);
  }
}

@media (max-width: 860px) {
  .shell {
    width: min(100% - 24px, 680px);
    padding-top: 18px;
  }

  .hero,
  .topbar {
    align-items: stretch;
    flex-direction: column;
  }

  .action-panel {
    width: 100%;
  }

  .content-grid,
  .cards,
  .driver-guide-grid,
  .mini-cards {
    grid-template-columns: 1fr;
  }

  h2 {
    font-size: 2.4rem;
  }
}
</style>
