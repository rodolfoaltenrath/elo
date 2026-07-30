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
- Detecção automática da distribuição Linux para escolher `pacman`, `dnf` ou `apt`.

## Distribuições

O Elo tem automação inicial para:

- Arch Linux, CachyOS, Manjaro, EndeavourOS e derivados com `pacman`.
- Fedora e derivados com `dnf`.
- Debian, Ubuntu, Deepin, Linux Mint, Pop!_OS, Zorin OS e derivados com `apt`.

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

No Fedora, instale primeiro as ferramentas de compilação:

```bash
sudo dnf install -y golang nodejs npm rpm-build gcc gtk3-devel webkit2gtk4.1-devel
```

Gere o binário e o pacote RPM com:

```bash
./packaging/fedora/build-rpm.sh
```

O RPM será gravado em `dist/` e pode ser instalado com:

```bash
sudo dnf install ./dist/*/elo-*.rpm
```

Ao executar **Corrigir Problemas**, o Elo usa o `dnf` para instalar `pcsc-lite`,
`pcsc-lite-ccid`, `opensc`, `pcsc-tools`, `nss-tools` e IcedTea-Web. Para o
Java 8, ele tenta primeiro o OpenJDK dos repositórios do Fedora. Nas versões em
que esse pacote não está mais disponível, configura o repositório RPM oficial
do Eclipse Adoptium, com verificação GPG, e instala o Temurin 8 JRE.

Se a interface não abrir, execute `elo` no terminal. Erros fatais também são
gravados em `~/.local/state/elo/elo.log` (ou em `$XDG_STATE_HOME/elo/elo.log`).
