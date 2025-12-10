package node

import (
	// bibs padrão
	"context"
	"errors"
	"fmt"
	"io"
	"math/rand"
	"slices"
	"sync"

	// "log"
	"os"
	"strings"
	"time"

	// bibs internas
	"projeto/services/embedding"
	"projeto/services/messaging"
	"projeto/types"

	// bibs externas
	faiss "github.com/DataIntelligenceCrew/go-faiss"
	"github.com/ipfs/boxo/files"
	"github.com/ipfs/boxo/path"
	"github.com/ipfs/kubo/config"
	"github.com/ipfs/kubo/core"
	"github.com/ipfs/kubo/core/coreapi"
	iface "github.com/ipfs/kubo/core/coreiface"
	ifaceOptions "github.com/ipfs/kubo/core/coreiface/options"
	"github.com/ipfs/kubo/core/node/libp2p"
	"github.com/ipfs/kubo/plugin/loader"
	"github.com/ipfs/kubo/repo"
	"github.com/ipfs/kubo/repo/fsrepo"
	"github.com/libp2p/go-libp2p/core/peer"
)

// Estruturas e Setup //
var Npeers int = 0 // Npeers na rede
type Vector = types.Vector



// Enum de estado
type NodeState int

// Estados Raft
const (
	FOLLOWER  NodeState = iota // 0
	CANDIDATE
	LEADER
)

type State struct {

	NdState NodeState
	CurrentTerm int
	StateMux sync.Mutex

}

// No
type Node struct {
	IpfsCore      *core.IpfsNode
	IpfsApi       iface.CoreAPI
	CidVector     Vector
	CidVectorEmbs map[string]([]float32)
	SearchIndex   *faiss.IndexFlat
	CidsInIndex   []string
	VectorCache   map[int](Vector)
	EmbsStaging   map[string]([]float32)

	// State         NodeState // Estado do nó
	State State
	// Lider Stuff
	StagingAKCs map[int]int
	API APIInterface
}


type APIInterface interface {
    Initialize(ctx context.Context,nd *Node,port int) (error)
}

func (nd *Node) SetAPI(api APIInterface) {
    nd.API = api
}

// Novo repositótio IPFS
func newIpfsRepo(repoPath string) error {
	_, err := os.Stat(repoPath)
	// err == nil -> retornou info sobre o caminho -> caminho já existente
	if err == nil {
		err = errors.New("Dirétoria/Repositório já existente\n")
		return err
	}

	// Se existe outro tipo de erro para além do caminho não existir retornamos o erro
	if !os.IsNotExist(err) {
		return err
	}

	// Criar a diretória
	err = os.Mkdir(repoPath, 0750)
	if err != nil {
		err = fmt.Errorf("Falha ao criar diretoria: %v\n", err)
		return err
	}

	// Criar a configuração do repositório.
	cfg, err := config.Init(io.Discard, 2048)
	cfg.Pubsub.Router = "gossipsub"
	cfg.Pubsub.Enabled = 1
	cfg.Ipns.UsePubsub = 1
	if err != nil {
		err = fmt.Errorf("Falha ao criar configuração: %v\n", err)
		return err
	}

	pluginInjection(repoPath)

	// Inicializar repositório
	err = fsrepo.Init(repoPath, cfg)
	if err != nil {
		err = fmt.Errorf("Falha ao iniciar: %v\n", err)
		return err
	}

	return nil
}

// Novo nó ipfs
func newIpfsNode(ctx context.Context, repoPath string) (*core.IpfsNode, error) {
	var repo repo.Repo
	err := pluginInjection(repoPath)
	if err == nil {
		repo, err = fsrepo.Open(repoPath)
	}

	// Erro
	if err != nil {
		//Criado novo repositório
		err = newIpfsRepo(repoPath)
		if err != nil {
			return nil, fmt.Errorf("Falha ao criar novo repositório : %v\n", err)
		}

		//Tenta abrir o repositório que esta na pasta repoPath
		repo, err = fsrepo.Open(repoPath)
		if err != nil {
			return nil, fmt.Errorf("Falha ao abrir novo repositótio : %v\n", err)
		}
	}

	// Defenições especificas do nó
	nodeOptions := &core.BuildCfg{
		Online:    true,
		Routing:   libp2p.DHTOption,
		Repo:      repo,
		Permanent: true,
		ExtraOpts: map[string]bool{
			"pubsub": true,
		},
	}

	// Criação do nó
	ipfsCore, err := core.NewNode(ctx, nodeOptions)
	if err != nil {
		return nil, fmt.Errorf("Falha ao criar nó IPFS: %v\n", err)
	}

	return ipfsCore, nil
}

// Connectar nó a peers
func connectToPeers(ipfs iface.CoreAPI, peers []string, self string) (int, error) {
	// Array para gerir nós desconectados
	unconnectedPeers := make(map[string]error)
	npeers := 0
	if len(peers) > 0 {
		for _, peerIdString := range peers {
			if peerIdString == self {
				continue
			}

			// Descodificar o CID
			peerId, err := peer.Decode(peerIdString)
			addr := peer.AddrInfo{ID: peerId}
			if err != nil {
				err = fmt.Errorf("Peer String Invalida: %v\n", err)
				unconnectedPeers[peerIdString] = err
				continue
			}

			timeoutCtx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
			defer cancel()

			// Conectar com 1 peer
			err = ipfs.Swarm().Connect(timeoutCtx, addr)
			if err != nil {
				err = fmt.Errorf("Conexão não foi possivel: %v\n", err)
				unconnectedPeers[peerIdString] = err
			} else {
				npeers = npeers + 1
			}
		}
	}

	// 1 ou + peers não conectados com sucesso -> erro
	if len(unconnectedPeers) > 0 {
		var sb strings.Builder
		sb.WriteString(fmt.Sprintf("%d peers não conectados:\n\n", len(unconnectedPeers)))
		for peerID, err := range unconnectedPeers {
			sb.WriteString(fmt.Sprintf(" - %s → %v\n", peerID, err))
		}

		return npeers, fmt.Errorf("%s", sb.String())
	}

	return npeers, nil
}

// Injeção de plugins (sem isto dá erro: unknown datastore type: flatfs)
// https://discuss.ipfs.tech/t/fixed-unknown-datastore-type-flatfs/15805/5
func pluginInjection(repoPath string) error {
	plugins, err := loader.NewPluginLoader(repoPath)
	if err != nil {
		err = fmt.Errorf("Erro ao carregar plugins: %s", err)
		return err
	}

	if err := plugins.Initialize(); err != nil {
		err = fmt.Errorf("Erro ao inicializar plugins: %s", err)
		return err
	}

	if err := plugins.Inject(); err != nil {
		err = fmt.Errorf("Erro ao injetar plugins: %s", err)
		return err
	}

	return nil
}

// Criar novo nó
// Para colocar o nó a leader basta ativar esta flag
var LeaderFlag bool

func Create(repoPath string, peers []string) (*Node, error) {
	var npeers int
	ipfsCore, err := newIpfsNode(context.Background(), repoPath)
	if err != nil {
		return nil, err
	}

	ipfsCoreApi, err := coreapi.NewCoreAPI(ipfsCore)
	if err != nil {
		return nil, err
	}

	if peers != nil && cap(peers) > 0 {
		npeers, err = connectToPeers(ipfsCoreApi, peers, ipfsCore.Identity.String())
	}

	ndstate := FOLLOWER
	if LeaderFlag {
		ndstate = LEADER
	}

	emptyVector :=
		Vector{
			Ver:     0,
			Content: []string{},
		}

		

	n := Node{
		IpfsCore:      ipfsCore,
		IpfsApi:       ipfsCoreApi,
		VectorCache:   make(map[int]Vector),
		EmbsStaging:   make(map[string]([]float32)),
		CidVector:     emptyVector,
		CidVectorEmbs: make(map[string]([]float32)),
		CidsInIndex:   []string{},
		State:         State{
						NdState: ndstate,
						CurrentTerm: 0,
						StateMux: sync.Mutex{},
					   },
	}

	var faissErr error
	n.SearchIndex, faissErr = faiss.NewIndexFlatL2(embedding.VECTORDIMS)
	if faissErr != nil {
		faissErr = fmt.Errorf("Erro ao criar index para o lider : \n%v", err)
		return nil, faissErr
	}

	Npeers = npeers
	return &n, err
}

// Metodos //
// Upload de ficheiros
func (nd *Node) AddFile(fileBytes []byte) (path.ImmutablePath, error) {
	uploadCtx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	fileCid, err := nd.IpfsApi.Unixfs().Add(uploadCtx, files.NewBytesFile(fileBytes))
	if err != nil {
		err = fmt.Errorf("Erro ao dar upload de ficheiro: %v\n", err)
	}

	return fileCid, err
}

func (nd *Node) Run() {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	go messaging.ListenTo(ctx,nd.IpfsApi.PubSub(), messaging.RBLQ, func(sender peer.ID, msg any, stop *bool) {
		fmt.Printf("[REBUILD-REQ] Recebida solicitação de rebuild do peer: %v\n", sender)
		rblqmsg, ok := msg.(messaging.RebuildQueryMessage)
		if !ok {
			fmt.Printf("Esperado RebuildQueryMessage, obtido %T", msg)
		}

		if rblqmsg.Dest == nd.IpfsCore.Identity && sender != nd.IpfsCore.Identity {
			fmt.Printf("[REBUILD-REQ] A processar rebuild para peer: %v\n", sender)
			response := make(map[string][]float32)
			if len(rblqmsg.Info) == 0 {
				go messaging.PublishTo(nd.IpfsApi.PubSub(), messaging.RBLR, messaging.RebuildResponseMessage{Response: nd.CidVectorEmbs, Dest: sender})
				return
			}

			for _, cid := range rblqmsg.Info {
				emb, exists := nd.CidVectorEmbs[cid]
				if exists {
					response[cid] = emb
				}
			}
			go messaging.PublishTo(nd.IpfsApi.PubSub(), messaging.RBLR, messaging.RebuildResponseMessage{Response: response, Dest: sender})
			return
		}

	})

    for {

		nd.State.StateMux.Lock()
        state := nd.State.NdState
		nd.State.StateMux.Unlock()
        
        switch state {
        case FOLLOWER:
			nd.StagingAKCs = nil
			followerRoutine(nd, ctx)
        case CANDIDATE:
			candidateRoutine(nd, ctx)
        case LEADER:
			nd.StagingAKCs = make(map[int]int)
			liderRoutine(nd, ctx)
        }
    }

}

// Follower
func receiveNewVector(nd *Node, v Vector, embs []float32) {
	if nd.CidVector.Ver <= v.Ver && isSubset(nd.CidVector, v) {
		fmt.Printf("[VETOR] Recebido (versão %d):\n%v\n", v.Ver, v.String())
		fmt.Printf("[VETOR] Hash: %s\n", v.Hash())
		messaging.PublishTo(nd.IpfsApi.PubSub(), messaging.ACK, messaging.AckMessage{Version: v.Ver, Hash: nd.CidVector.Hash()})
	}

	nd.VectorCache[v.Ver] = v
	nd.EmbsStaging[v.Content[len(v.Content)-1]] = embs
}

func receiveCommit(nd *Node, version int) {

	fmt.Printf("\n[COMMIT] Processando versão %d\n", version)
	fmt.Printf("[COMMIT] Vetor na cache (v%d): %v\n", version, nd.VectorCache[version].Content)
	fmt.Printf("[COMMIT] Vetor CID atual: %v\n", nd.CidVector.Content)

	// Atualizar o vetor
	nd.CidVector = nd.VectorCache[version]

	// Limpar cache de versões antigas
	for k := range nd.VectorCache {
		if k <= version {
			delete(nd.VectorCache, k)
		}
	}

	// Mover TODOS os embs de EmbsStaging para CidVectorEmbs
	for _, cid := range nd.CidVector.Content {
		emb, ok := nd.EmbsStaging[cid]
		if ok {
			nd.CidVectorEmbs[cid] = emb
			delete(nd.EmbsStaging, cid)
			fmt.Printf("[COMMIT] Embedding movido de staging para CidVectorEmbs: %s\n", cid)
		}
	}

	idx := 0
	// Adicionar ao índice Faiss todos os embs que temos e ainda não estão no índice
	for ; idx < len(nd.CidVector.Content); idx++ {
		cid := nd.CidVector.Content[idx]
		emb := nd.CidVectorEmbs[cid]
		if emb == nil {
			fmt.Printf("[COMMIT] Embedding não encontrado para CID %s (posição %d)\n", cid, idx)
			break
		}

		if !slices.Contains(nd.CidsInIndex, cid) {
			nd.SearchIndex.Add(emb)
			nd.CidsInIndex = append(nd.CidsInIndex, cid)
			fmt.Printf("[COMMIT] Adicionado ao índice Faiss: %s (Total: %d)\n", cid, len(nd.CidsInIndex))
		}
	}

	// Calcular missing: apenas CIDs que estão no vetor mas não em CidVectorEmbs
	missing := []string{}
	for ; idx < len(nd.CidVector.Content); idx++ {
		cid := nd.CidVector.Content[idx]
		emb := nd.CidVectorEmbs[cid]
		if emb == nil {
			fmt.Printf("[COMMIT] Embedding em falta detectado: %s\n", cid)
			missing = append(missing, cid)
		}
	}

	fmt.Printf("[COMMIT] Lista de embeddings em falta: %v\n", missing)

	// Pedir embeddings em falta se houver
	if len(missing) > 0 {
		rpeerctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		randomPeer, error := nd.getPubSubRandomPeer(rpeerctx, string(messaging.RBLQ))
		cancel()
		if error == nil {
			fmt.Printf("[COMMIT] A solicitar rebuild ao peer %v para %d CIDs\n", randomPeer, len(missing))
			go messaging.PublishTo(
				nd.IpfsApi.PubSub(),
				messaging.RBLQ,
				messaging.RebuildQueryMessage{Dest: randomPeer, Info: missing},
			)

		} else {
			fmt.Printf("[COMMIT] Erro ao encontrar peer para rebuild: %v\n", error)
		}
	}
	fmt.Printf("[COMMIT] Vetor atualizado (versão %d): %v\n", nd.CidVector.Ver, nd.CidVector.String())
}

//func heartBeatCheck(nd *Node){
func heartBeatCheck(lastLiderHeartBeat *time.Time, electionTimeout time.Duration, timeoutChan chan struct{}, ctx context.Context) {
    ticker := time.NewTicker(10 * time.Millisecond)
    defer ticker.Stop()
    
    for {
        select {
        case <-ticker.C:
            if time.Since(*lastLiderHeartBeat) >= electionTimeout {
                fmt.Printf("[HEARTBEAT] Timeout do líder detectado - iniciando eleição\n")
                close(timeoutChan)
                return
            }
        case <-ctx.Done():
            return
        }
    }
}

func (nd *Node) getPubSubRandomPeer(ctx context.Context, topic string) (peer.ID, error) {
	ps := nd.IpfsApi.PubSub()
	peers, err := ps.Peers(ctx, ifaceOptions.PubSub.Topic(topic))
	if err != nil {
		err = fmt.Errorf("Erro ao procurar peers do pubsub para o tópico %s: %v\n", topic, err)
		return nd.IpfsCore.Identity, err
	}

	idx := rand.Intn(len(peers))
	return peers[idx], nil
}

func followerRoutine(nd *Node, ctx context.Context) {
	// Assumimos o primeiro beat na rotina follower
	lastLiderHeartBeat := time.Now()
	electionTimeout := time.Duration(15 + rand.Intn(15)) * time.Second
	votes := make(map[int]peer.ID)
	fmt.Printf("\n[ESTADO] Iniciando rotina FOLLOWER\n")

    localCtx, cancel := context.WithCancel(context.Background())
    defer cancel()

	// TODO Alterar para depois acomodar o processo de eleicao
	go messaging.ListenTo(localCtx,nd.IpfsApi.PubSub(), messaging.HTB, func(sender peer.ID, msg any, stop *bool) {
		htbmsg, ok := msg.(messaging.HeartBeatMessage)
		if !ok {
			fmt.Printf("Esperado HearBeatMessage, obtido %T", msg)
		}

		lastLiderHeartBeat = time.Now()
		Npeers = htbmsg.Npeers
		nd.State.CurrentTerm = htbmsg.Term
	})

    // Canal para sinalizar timeout de eleição
    timeoutChan := make(chan struct{})
    

    go heartBeatCheck(&lastLiderHeartBeat, electionTimeout, timeoutChan, localCtx)

	go messaging.ListenTo(localCtx,nd.IpfsApi.PubSub(), messaging.RBLR, func(sender peer.ID, msg any, stop *bool) {
		rblrmsg, ok := msg.(messaging.RebuildResponseMessage)
		if !ok {
			fmt.Printf("Esperado rebuildResponseMessage, obtido %T", rblrmsg)
		}

		if rblrmsg.Dest == nd.IpfsCore.Identity {
			fmt.Printf("[REBUILD-RES] Recebida resposta de rebuild do peer: %v\n", sender)
			for cid, emb := range rblrmsg.Response {
				fmt.Printf("[REBUILD-RES] CID %s recebido com embedding (dim: %d)\n", cid, len(emb))
				nd.EmbsStaging[cid] = emb
			}
		}
	})


	go messaging.ListenTo(localCtx,nd.IpfsApi.PubSub(), messaging.CDTP, func(sender peer.ID, msg any, stop *bool) {
		cdtpmsg, ok := msg.(messaging.CandidatePorposalMessage)
		if !ok {
			fmt.Printf("Esperado CandidatePorposalMessage, obtido %T", cdtpmsg)
		}

		myTermVote,exists := votes[cdtpmsg.Term]

		if((exists && myTermVote == sender) || !exists && sender != nd.IpfsCore.Identity){
			go messaging.PublishTo(nd.IpfsApi.PubSub(),messaging.VTP,messaging.VoteMessage{Term: cdtpmsg.Term, Candidate: sender})
			votes[cdtpmsg.Term] = sender
		}

	})

	go messaging.ListenTo(localCtx,nd.IpfsApi.PubSub(), messaging.AEM, func(sender peer.ID, msg any, stop *bool) {
		aemMsg, ok := msg.(messaging.AppendEntryMessage)
		if !ok {
			fmt.Printf("Esperado AppendEntryMessage, obtido %T", msg)
		}

		fmt.Printf("[APPEND-ENTRY] Recebido com vetor (versão %d):\n%v\n", aemMsg.Vector.Ver, aemMsg.Vector.String())
		receiveNewVector(nd, aemMsg.Vector, aemMsg.Embeddings)
	})

	go messaging.ListenTo(localCtx,nd.IpfsApi.PubSub(), messaging.COMM, func(sender peer.ID, msg any, stop *bool) {
		fmt.Printf("[COMMIT-MSG] Mensagem de commit recebida\n")
		fmt.Printf("[COMMIT-MSG] Vetor atual (v%d):\n%v\n", nd.CidVector.Ver, nd.CidVector.String())
		commMgs, ok := msg.(messaging.CommitMessage)
		if !ok {
			fmt.Printf("Esperado CommitMessage, obtido %T", msg)
		}

		receiveCommit(nd, commMgs.Version)
		fmt.Printf("[COMMIT-MSG] Vetor atualizado (v%d):\n%v\n", nd.CidVector.Ver, nd.CidVector.String())
	})

    select {
		case <-timeoutChan:
			fmt.Printf("[FOLLOWER] Timeout detectado - transição para CANDIDATE\n")
			nd.State.StateMux.Lock()
			nd.State.NdState = CANDIDATE
			nd.State.StateMux.Unlock()
			cancel()
			return
		case <-localCtx.Done():
			fmt.Printf("\n[ESTADO] Terminando rotina FOLLOWER\n")
			return
    }
}

// Lider
func receiveAck(nd *Node, hash string, version int) bool {

    if version <= nd.CidVector.Ver {
        fmt.Println("Versão antiga vou passar check")
        return true
    }

    valid := nd.CidVector.Hash() == hash
    if valid {
        fmt.Println("Valido a adicionar ack")
        nd.StagingAKCs[version] = nd.StagingAKCs[version] + 1
    }

    fmt.Printf("Hash recebida %s\n", hash)
    fmt.Printf("Hash do vetor atual %s\n", nd.CidVector.Hash())
    fmt.Printf("Numeros de ACKs para a versao %v: %v/%v\n", version, nd.StagingAKCs[version], Npeers)

    if nd.StagingAKCs[version] >= int(Npeers/2) {
        fmt.Println("Vou dar commit")
        nd.CidVector = nd.VectorCache[version]

        // Mover embeddings de EmbsStaging para CidVectorEmbs (igual ao follower)
        for _, cid := range nd.CidVector.Content {
            emb, ok := nd.EmbsStaging[cid]
            if ok {
                nd.CidVectorEmbs[cid] = emb
                delete(nd.EmbsStaging, cid)
                fmt.Printf("Líder: movido emb de staging para CidVectorEmbs: %s\n", cid)
            }
        }

        // Adicionar ao índice Faiss todos os embs que temos e ainda não estão no índice
        for _, cid := range nd.CidVector.Content {
            emb := nd.CidVectorEmbs[cid]
            if emb == nil {
                fmt.Printf("Líder: embedding não encontrado para CID %s\n", cid)
                continue
            }

            if !slices.Contains(nd.CidsInIndex, cid) {
                nd.SearchIndex.Add(emb)
                nd.CidsInIndex = append(nd.CidsInIndex, cid)
                fmt.Printf("Líder: adicionado ao índice Faiss: %s\n", cid)
            }
        }

        // Enviar commit
        messaging.PublishTo(nd.IpfsApi.PubSub(), messaging.COMM,
            messaging.CommitMessage{Version: version})

        // Limpar caches antigas
        for k := range nd.VectorCache {
            if k <= version {
                delete(nd.StagingAKCs, k)
                delete(nd.VectorCache, k)
            }
        }
    } else {
        fmt.Println("Não vou dar commit")
    }

    return valid
}

func (nd *Node) getPubSubPeersCount(ctx context.Context, topic string) int {
	ps := nd.IpfsApi.PubSub()
	peers, err := ps.Peers(ctx, ifaceOptions.PubSub.Topic(topic))
	if err != nil {
		fmt.Printf("Erro ao procurar peers do pubsub para o tópico %s: %v\n", topic, err)
		return Npeers
	}

	return len(peers)
}

func heartBeatSender(nd *Node, ctx context.Context) {
	topic := string(messaging.AEM)
	for {

        select {
        case <-ctx.Done():
            fmt.Printf("[HEARTBEAT] Cancelando heartbeat sender\n")
            
        default:
			ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
			n := nd.getPubSubPeersCount(ctx, topic)
			cancel()
			if n > 0 && Npeers != n {
				Npeers = n
				fmt.Printf("[HEARTBEAT] Peers ativos atualizados via PubSub: %d\n", Npeers)
			}

			messaging.PublishTo(
				nd.IpfsApi.PubSub(),
				messaging.HTB,
				messaging.HeartBeatMessage{Npeers: Npeers, Term: nd.State.CurrentTerm},
			)

			time.Sleep(1500 * time.Millisecond) // 1.5 s
		}
	}
}

func liderRoutine(nd *Node, ctx context.Context) {
	fmt.Printf("\n[ESTADO] Iniciando rotina LEADER\n")


    localCtx, cancel := context.WithCancel(context.Background())
    defer cancel()

	go nd.API.Initialize(localCtx,nd,9000)

	go heartBeatSender(nd,localCtx)
	go messaging.ListenTo(localCtx,nd.IpfsApi.PubSub(), messaging.ACK, func(sender peer.ID, msg any, stop *bool) {
		ackmsg, ok := msg.(messaging.AckMessage)
		if !ok {
			fmt.Printf("Esperado AppendEntryMessage, obtido %T", msg)
		}

		fmt.Printf("[ACK] Recebido de peer %v | Hash: %s\n", sender, ackmsg.Hash)
		valid := receiveAck(nd, ackmsg.Hash, ackmsg.Version)
		if !valid {
			fmt.Printf("[ACK] Hash inválida para vetor:\n%v\n", nd.CidVector.String())
			fmt.Printf("[ACK] Validação falhou\n")
		}

	})

	// Espera bloqueante (equanto o context não for cancelado)
	<-ctx.Done()
	fmt.Printf("\n[ESTADO] Terminando rotina LEADER\n")
}


func candidatePorposalSender(nd *Node, ctx context.Context) {

	for {

        select {
        case <-ctx.Done():
            fmt.Printf("[CDTP] Cancelando candidate porposal sender\n")
            
        default:

			messaging.PublishTo(
				nd.IpfsApi.PubSub(),
				messaging.CDTP,
				messaging.CandidatePorposalMessage{Term: nd.State.CurrentTerm},
			)

			time.Sleep(15 * time.Second)
		}
	}

}




func candidateRoutine(nd *Node, ctx context.Context) {

	fmt.Printf("\n[ESTADO] Iniciando rotina CANDIDATE\n")

	newTerm := nd.State.CurrentTerm + 1


	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	n := nd.getPubSubPeersCount(ctx, string(messaging.RBLQ))
	cancel()


	Npeers = n
	fmt.Printf("[CANDIDATE PEERCOUNT] Peers ativos atualizados via PubSub: %d\n", Npeers)

	if(Npeers <= 1){

		nd.State.StateMux.Lock()
		nd.State.NdState = LEADER
		nd.State.StateMux.Unlock()
		nd.State.CurrentTerm = newTerm
		return

	}

	// Golang não tem sets então pelos vistos é comum utilizar mapas com valores vazios

	votes := map[peer.ID]struct{}{}
    electedChan := make(chan struct{})


    localCtx, cancel := context.WithCancel(context.Background())
    defer cancel()


	go messaging.ListenTo(localCtx,nd.IpfsApi.PubSub(), messaging.HTB, func(sender peer.ID, msg any, stop *bool) {

		htbmsg, ok := msg.(messaging.HeartBeatMessage)
		if !ok {
			fmt.Printf("Esperado HearBeatMessage, obtido %T", msg)
		}

		if(htbmsg.Term >= newTerm){

			nd.State.StateMux.Lock()
			nd.State.NdState = FOLLOWER
			nd.State.StateMux.Unlock()
			nd.State.CurrentTerm = htbmsg.Term
			close(electedChan)


		}



	})


	go messaging.PublishTo(
		nd.IpfsApi.PubSub(),
		messaging.CDTP,
		messaging.CandidatePorposalMessage{Term: newTerm},
	)

	
	go messaging.ListenTo(localCtx,nd.IpfsApi.PubSub(), messaging.VTP, func(sender peer.ID, msg any, stop *bool) {

		voteMsg, ok := msg.(messaging.VoteMessage)
		if !ok {
			fmt.Printf("Esperado VoteMessage, obtido %T", msg)
		}

		if(voteMsg.Term == newTerm && voteMsg.Candidate == nd.IpfsCore.Identity){
			votes[sender] = struct{}{}
		}

		if(len(votes)+1 >= (Npeers/2)){
			close(electedChan)

		}

	})


    select {
		case <-electedChan:
			fmt.Printf("[FOLLOWER] Timeout detectado - transição para LEADER\n")
			nd.State.StateMux.Lock()
			nd.State.NdState = LEADER
			nd.State.StateMux.Unlock()
			cancel()
			return
		case <-localCtx.Done():
			fmt.Printf("\n[ESTADO] Terminando rotina CANDIDATE\n")
			return
    }

}

// Funções Auxiliares //
func isSubset(sub Vector, super Vector) bool {
	var subArray []string
	var superArray []string
	subArray = sub.Content
	superArray = super.Content

	if len(subArray) == 0 {
		return true
	}

	if len(subArray) > len(superArray) {
		return false
	}

	for i := 0; i < len(subArray); i++ {
		if subArray[i] != superArray[i] {
			return false
		}
	}
	return true
}
