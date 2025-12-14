# Sistemas distribuídos

**Trabalho elaborado por:**

- Guilherme Bento

- Ricardo Matos

- Tiago Portugal

- Vasco Aparício

**Disciplina:** Sistemas Distribuídos

## Introdução

  O projeto consiste na implementação de um sistema distribuído para o armanezamento e recuperação de ficheiros. Este trabalho foi constituído por sete sprints, dois dos quais de recuperação.

    O grupo optou por implementar a solução com a linguagem de programação go(golang), de modo a utilizar a biblioteca `kubo (go-ipfs)`, na qual é baseada o `ipfs desktop` e `ipfs cli` (basicamente uma abstração da bibilioteca), para criar um projeto mais integrado sem recurso a *wrappers* ou comandos shell.

    Todos os elementos do grupo interpretaram a implementação do projeto em `go` com entusiasmo, para poder ter *feedback* da linguagem, que tem vindo a ganhar popularidade.

## Arquitetura da solução UML

### | Add File - Diagrama de sequência

```mermaid
sequenceDiagram
 participant Cliente
 participant Lider
 participant Peer1
 participant Peer2 
Cliente-->>Lider: ficheiro
Lider-->>Peer1: AppendEntryMessage
 Lider-->>Peer2: AppendEntryMessage
 Peer1-->>Lider: AckMessage
 Peer2-->>Lider: AckMessage 
 Note left of Lider: Se os hash forem iguais em maioria de todos os peers
 Lider->>Peer1: CommitMessage
 Lider->>Peer2: CommitMessage
 Lider-->>Cliente: CID 
```

### | Get File - Diagrama de Sequência

```mermaid
sequenceDiagram
 participant Cliente
 participant Lider
 participant Peer1
 participant Peer2 
Cliente-->>Lider: CID
 Note left of Lider: Lider envia um peer aleatório
Lider-->>Peer2: ClientRequest (GetFile)
Peer2-->>Lider: ClientResponse (GetFile)
Lider-->>Cliente: FileBytes
```

### |Prompt - Diagrama de sequência:

```mermaid
sequenceDiagram
    participant Cliente
    participant Lider
    participant Peer1
    participant Peer2
    participant Peer3

    Cliente-->>Lider: query

     Note left of Lider: Lider envia um peer aleatório

    Lider-->>Peer2: ClientRequest (Prompt)

    alt Not Found
        Peer2-->>Peer3: ClientRequest (Prompt)
    else Found
        Peer2-->>Lider: ClientResponse (Prompt)
    end

    Lider-->>Cliente: CID
```

### |Transição de rotinas - Diagrama de estados

```mermaid



stateDiagram-v2
    direction LR
    
    [*]-->Follower
    Follower --> Candidate:heartbeat timeout
    Candidate --> Follower:heartbeat discovery
    Candidate--> Leader:vote majority
   


```

## Implementação

### Canais

**aem**

```go
 AEM Topico = "aem" // Topico AppendEntryMessage 
 
 type AppendEntryMessage struct {
    Vector Vector 
    Embeddings []float32 
}
```

**ack**

```go
 ACK Topico = "ack" // Topico Ack


type AckMessage struct {
    Version int
    Hash string 
}
```

**commit**

```go
COMM Topico = "commit" // Topico Commit


type CommitMessage struct {
    Version int 
}
```

**heartbeat**

```go
HTB Topico = "heartbeat" // Topico Heartbeat


type HeartBeatMessage struct {
    Npeers int
    Term int
}
```

**rebuildquery**

```go
 RBLQ Topico = "rebuildquery" // Topico RebuildQuery
 
 type RebuildQueryMessage struct {
    Info []string 
    Dest peer.ID
}
```



## Conclusão

...
