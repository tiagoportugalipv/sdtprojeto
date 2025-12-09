# Sistemas distribuídos

**Travbalho elaborado por:**

- Guilherme Bento

- Ricardo Matos

- Tiago Portugal

- Vasco Aparício

## Introdução

    Este trabalho foi constituído por sete sprints, nos quais dois foram de recuperação. O projeto consiste na implementação de um sistema distribuído para o armanezamento e recuperação de ficheiros.

    O grupo implementou a solução em go, de modo a utilizar a biblioteca `kubo (go-ipfs)`, na qual é baseada o `ipfs desktop` e `ipfs cli` (basicamente uma abstração da bibilioteca), para criar um projeto mais integrado sem recurso a *wrappers* ou comandos shell.

    Todos os elementos do grupo interpretaram a implementação do projeto em `go` com entusiasmo, para poder ter *feedback* da linguagem, que tem vindo a ganhar popularidade.

## Arquitetura da solução UML

**Diagrama de sequência:** 

(In progress...)

```mermaid
sequenceDiagram
    participant Cliente
    participant Lider
    participant Peer1
    participant Peer2
    participant Peer3

    Cliente-->>Lider: ficheiro

    Lider-->>Lider: Guarda ficheiro no ipfs, Gera embeedings
```

## Implementação

-

## Conclusão

-
