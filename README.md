# Elo

Elo conecta Linux ao PJe/Projudi com uma interface simples. Automatiza Java 8 isolado, diagnóstico de certificados digitais e configuração de tokens A3 via PC/SC, PKCS#11 e navegador, evitando terminal e ajustes manuais.

## Objetivo

O projeto evita que pessoas usuárias precisem configurar Java legado, smartcard e certificados digitais pelo terminal em distribuições modernas baseadas em Arch Linux, como CachyOS.

## Recursos

- Diagnóstico visual de Java, token, certificado e navegador.
- Instalação automática das dependências livres necessárias.
- Registro do driver PKCS#11 no banco NSS do navegador.
- Instalação assistida de driver proprietário fornecido pela pessoa usuária.
- Execução isolada de arquivos `.jnlp` e `.jar` com Java 8.
- Detecção automática da distribuição Linux para escolher `pacman` ou `apt`.

## Distribuições

O Elo tem automação inicial para:

- Arch Linux, CachyOS, Manjaro, EndeavourOS e derivados com `pacman`.
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

Para publicar no AUR com o nome `elo`, copie `PKGBUILD` e `.SRCINFO` para o repositório `ssh://aur@aur.archlinux.org/elo.git` e faça o push. Depois disso, usuários poderão instalar com:

```bash
yay -S elo
```
