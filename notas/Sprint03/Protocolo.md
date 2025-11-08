# Protocolo de comunicação:

## Protocolos:

O protocolo que será implementado como solução é o `Raft`, para determinar o consenso entre os vários peers..

- Neste momento, ainda não está a ser implementado do candidato a líder.

- Quando um cliente faz upload do ficheiro em determinado momento existirá várias versões, dessa forma os peers enviam um `hash` e se for igual em todos os peers, o lider faz commit da versão mais recente.

**Diagrama de sequência:**

```mermaid
sequenceDiagram
    participant Cliente
    participant Lider
    participant Peer1
    participant Peer2

    Cliente-->>Lider: ficheiro
    Lider-->>Cliente: CID

    Lider-->>Peer1: newVectorMessage
    Lider-->>Peer2: newVectorMessage
    Peer1-->>Lider: hashACK
    Peer2-->>Lider: hashACK

    activate Lider
    Note left of Lider: Se os hash forem iguais em maioria de todos os peers
    Lider->>Peer1: commitNewVectorVersion
    Lider->>Peer2: commitNewVectorVersion
    deactivate Lider
```
