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

## Desenvolvimento

```bash
wails dev
```

## Build

```bash
wails build
```

O binário Linux é gerado em `build/bin/elo`.
