# Elo

Elo conecta Linux ao PJe/Projudi com uma interface simples. Automatiza Java 8 isolado, diagnóstico de certificados digitais e configuração de tokens A3 via PC/SC, PKCS#11 e navegador, evitando terminal e ajustes manuais.

## Objetivo

O projeto evita que pessoas usuárias precisem configurar Java legado, smartcard e certificados digitais pelo terminal em distribuições Linux modernas, incluindo Arch Linux, Fedora e seus derivados.

## Recursos

- Diagnóstico visual de Java, token, certificado e navegador.
- Instalação automática das dependências livres necessárias.
- Registro do driver PKCS#11 no banco NSS do navegador.
- Instalação assistida de driver proprietário fornecido pela pessoa usuária.
- Execução isolada de arquivos `.jnlp` e `.jar` com Java 8.
- Detecção automática da distribuição Linux para escolher `pacman`, `apt` ou `dnf`.

## Distribuições

O Elo tem automação inicial para:

- Arch Linux, CachyOS, Manjaro, EndeavourOS e derivados com `pacman`.
- Debian, Ubuntu, Deepin, Linux Mint, Pop!_OS, Zorin OS e derivados com `apt`.
- Fedora e derivados com `dnf`.

Em outras distribuições, o app ainda pode diagnosticar parte do ambiente, mas bloqueia a correção automática até existir uma estratégia segura para o gerenciador de pacotes.

## Desenvolvimento

```bash
wails dev
```

## Build

```bash
wails build
```

O binário Linux é gerado em `build/bin/elo`.

## Pacote Arch/AUR

Este repositório inclui uma receita inicial de pacote Arch:

```bash
makepkg -si
```

O pacote compilado publicado no AUR pode ser instalado com:

```bash
yay -S elo-bin
```

## Pacote Fedora/RPM

O instalador gráfico para Fedora x86_64 é gerado no próprio GitHub, sem exigir
Go, Node, Wails ou ferramentas RPM no computador Windows:

1. Abra a aba **Actions** do repositório no GitHub.
2. Escolha **Gerar instalador Fedora RPM** e clique em **Run workflow**.
3. Informe a versão, aguarde o job terminar e baixe o artefato
   `elo-fedora-rpm-<versão>`.
4. Extraia o `.zip`, copie o arquivo `.rpm` para o pendrive e leve-o ao Fedora.
5. No Fedora, dê dois cliques no `.rpm`, escolha **Instalar** e informe a senha.

O pacote inclui o frontend, o executável e um Java 8 Temurin privado em
`/opt/elo/jre8`. GTK/WebKit, PC/SC, OpenSC, NSS e as demais integrações do
sistema são declaradas no RPM e instaladas automaticamente pelo gerenciador de
software. A primeira instalação precisa de internet caso esses componentes ainda
não estejam no Fedora; o `.rpm` isolado não é um instalador totalmente offline.

Para compilar diretamente em uma máquina Fedora, instale as ferramentas:

```bash
sudo dnf install -y curl gcc gcc-c++ git golang gzip gtk3-devel nodejs npm \
  pkgconf-pkg-config rpm tar webkit2gtk4.1-devel
```

Depois gere o pacote:

```bash
bash packaging/fedora/build-rpm.sh 0.1.0
```

O resultado fica em `dist/`.

Se a interface não abrir, execute `elo` no terminal. Erros fatais também são
gravados em `~/.local/state/elo/elo.log` (ou em `$XDG_STATE_HOME/elo/elo.log`).
