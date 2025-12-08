package node

import (

	// bibs padrão

	"context"
	"errors"
	"fmt"
	"io"
	"math/rand"

	// "log"
	"os"
	"strings"
	"time"

	// bibs internas
	"projeto/services/embedding"
	"projeto/services/messaging"
	"projeto/types"

	// bibs externas

	"github.com/ipfs/boxo/files"
	"github.com/ipfs/boxo/path"
	"github.com/ipfs/kubo/config"
	"github.com/ipfs/kubo/core"
	"github.com/ipfs/kubo/core/coreapi"
	"github.com/ipfs/kubo/core/node/libp2p"
	"github.com/ipfs/kubo/plugin/loader"
	"github.com/ipfs/kubo/repo"
	"github.com/ipfs/kubo/repo/fsrepo"
	"github.com/libp2p/go-libp2p/core/peer"

	faiss "github.com/DataIntelligenceCrew/go-faiss"
	iface "github.com/ipfs/kubo/core/coreiface"
	ifaceOptions "github.com/ipfs/kubo/core/coreiface/options"
)

// Estruturas e Setup //
var Npeers int = 0 // Npeers na rede

type Vector = types.Vector


// Enum de estado

type NodeState int

// Estados Raft
const (
	FOLLOWER NodeState = iota // 0
	CANDIDATE
	LEADER
)


// No

type Node struct {
	IpfsCore *core.IpfsNode 
	IpfsApi  iface.CoreAPI
	CidVector Vector
	CidVectorEmbs map[string]([]float32)
	SearchIndex	*faiss.IndexFlat 
	VectorCache map[int](Vector)
	EmbsStaging map[string]([]float32)
	State NodeState // Estado do nó

	// Lider Stuff
	StagingAKCs map[int]int
	CommitDone chan struct{} // Para depois sinalizar a api para responder ao cliente depois dos COMMITACK
}


// Novo repositótio IPFS
func newIpfsRepo(repoPath string) (error) {

	_, err := os.Stat(repoPath)

	// err == nil -> retornou info sobre o caminho -> caminho já existente	
	if(err == nil){
		err = errors.New("Dirétoria/Repositório já existente\n") 
		return err
	}

	// Se existe outro tipo de erro para além do caminho não existir retornamos o erro
	if(!os.IsNotExist(err)){
		return err
	}


	// Criar a diretória
	err = os.Mkdir(repoPath, 0750)

	if(err != nil){
		err = fmt.Errorf("Falha ao criar diretoria: %v\n",err)
		return err
	}
	// Criar a configuração do repositório.
	cfg, err := config.Init(io.Discard, 2048)
	cfg.Pubsub.Router="gossipsub"
	cfg.Pubsub.Enabled=1
	cfg.Ipns.UsePubsub=1

	if err != nil {
		err = fmt.Errorf("Falha ao criar configuração: %v\n",err)
		return err
	}


	pluginInjection(repoPath)


	// Inicializar repositório
	err = fsrepo.Init(repoPath, cfg)

	if err != nil {
		err = fmt.Errorf("Falha ao iniciar: %v\n",err)
		return err
	}

	return nil

}

// Novo nó ipfs
func newIpfsNode(ctx context.Context, repoPath string) (*core.IpfsNode, error) {

	var repo repo.Repo	

	err := pluginInjection(repoPath)

	if(err == nil){
		repo, err = fsrepo.Open(repoPath)
	}

	// Erro
	if err != nil {

		//Criado novo repositório
		err = newIpfsRepo(repoPath)
		if err != nil {
			return nil, fmt.Errorf("Falha ao criar novo repositório : %v\n",err)
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
func connectToPeers(ipfs iface.CoreAPI, peers []string, self string) (int,error) {

	// Array para gerir nós desconectados	
	unconnectedPeers := make(map[string]error)
	npeers := 0


	if(len(peers) > 0){

		for _, peerIdString := range peers {

			if(peerIdString == self){
				continue
			}

			// Descodificar o CID
			peerId, err := peer.Decode(peerIdString)
			addr := peer.AddrInfo{ID:peerId}

			if(err != nil){
				err = fmt.Errorf("Peer String Invalida: %v\n",err)
				unconnectedPeers[peerIdString] = err
				continue
			}

			timeoutCtx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
			defer cancel() 


			// Conectar com 1 peer
			err = ipfs.Swarm().Connect(timeoutCtx, addr)


			if(err != nil){
				err = fmt.Errorf("Conexão não foi possivel: %v\n",err)
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

		return npeers,fmt.Errorf("%s",sb.String())
	}

	return npeers,nil


}


// Injeção de plugins (sem isto dá erro: unknown datastore type: flatfs)
// https://discuss.ipfs.tech/t/fixed-unknown-datastore-type-flatfs/15805/5
func pluginInjection(repoPath string)(error){

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


func Create(repoPath string,  peers []string) (*Node,error){

	var npeers int

	ipfsCore, err := newIpfsNode(context.Background(),repoPath);

	if(err!=nil){
		return nil,err
	}

	ipfsCoreApi,err := coreapi.NewCoreAPI(ipfsCore)


	if(err!=nil){
		return nil,err
	}

	if(peers != nil && cap(peers) > 0){
		npeers,err = connectToPeers(ipfsCoreApi,peers,ipfsCore.Identity.String())
	}

	state := FOLLOWER

	if(LeaderFlag){
		state = LEADER
	}

	emptyVector := 
		Vector {
			Ver: 0,
			Content: []string{},
		}

	n := Node {

		IpfsCore : ipfsCore,
		IpfsApi : ipfsCoreApi,
		VectorCache: make(map[int]Vector),
		EmbsStaging: make(map[string]([]float32)),
		CidVector: emptyVector,
		CidVectorEmbs: make(map[string]([]float32)),
		State: state,

	}

	if(LeaderFlag){


		n.CommitDone = make(chan struct{})
		n.StagingAKCs = make(map[int]int)
		

	}


	var faissErr error

	n.SearchIndex,faissErr = faiss.NewIndexFlatL2(embedding.VECTORDIMS) 

	if(faissErr != nil){
		faissErr = fmt.Errorf("Erro ao criar index para o lider : \n%v", err)
		return nil,faissErr
	}

	Npeers = npeers

	return &n,err

}

// Metodos //

// Upload de ficheiros
func (nd *Node) AddFile(fileBytes []byte) (path.ImmutablePath,error){


	uploadCtx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	fileCid, err := nd.IpfsApi.Unixfs().Add(uploadCtx,files.NewBytesFile(fileBytes))

	if(err != nil){
		err = fmt.Errorf("Erro ao dar upload de ficheiro: %v\n",err)
	}


	return fileCid,err

}

func (nd *Node) Run(){

	ctx,cancel := context.WithCancel(context.Background()) 
	defer cancel()

	//
	// go messaging.ListenTo(nd.IpfsApi.PubSub(),messaging.RBLQ,func(sender peer.ID, msg any, stop *bool) { 
	//
	// 	fmt.Printf("Recebi mensagem para dar rebuild ao peer %v\n\n",sender)
	//
	// 	rblqmsg,ok := msg.(messaging.RebuildQueryMessage)
	//     if(!ok){
	// 		fmt.Printf("Esperado HearBeatMessage, obtido %T", msg)
	//     }
	//
	// 	if(rblqmsg.Dest == nd.IpfsCore.Identity && sender != nd.IpfsCore.Identity){
	//
	//
	// 		fmt.Printf("Vou ajudar a dar rebuild ao peer %v\n\n",sender)
	//
	// 		response := make(map[string][]float32)
	//
	// 		if(len(rblqmsg.Info) == 0){
	//
	// 			go messaging.PublishTo(nd.IpfsApi.PubSub(),messaging.RBLR,messaging.RebuildResponseMessage{Response:nd.CidVectorEmbs, Dest: sender, Total:true})
	// 			return
	//
	// 		}
	//
	// 		for _,cid := range rblqmsg.Info {
	//
	// 			emb,exists := nd.CidVectorEmbs[cid]
	//
	// 			if(exists){
	// 				response[cid] = emb
	// 			} 
	// 		}
	//
	//
	// 		go messaging.PublishTo(nd.IpfsApi.PubSub(),messaging.RBLR,messaging.RebuildResponseMessage{Response:response, Dest: sender, Total:false})
	// 		return
	//
	// 	}
	//
	// })
	//

	switch nd.State {
	case LEADER:
	     liderRoutine(nd,ctx)
	default:
	     followerRoutine(nd,ctx)
	}

}

// Follower

func receiveNewVector(nd *Node,v Vector, embs []float32){

	if(nd.CidVector.Ver < v.Ver && isSubset(nd.CidVector,v)){
		fmt.Printf("Vetor recebido:\n%v\n",v.String())
		fmt.Printf("Hash do vetor: %s\n",v.Hash())
		messaging.PublishTo(nd.IpfsApi.PubSub(),messaging.ACK,messaging.AckMessage{Version: v.Ver,Hash: nd.CidVector.Hash()})
	}

	nd.VectorCache[v.Ver] = v
	nd.EmbsStaging[v.Content[len(v.Content)-1]] = embs

}

func receiveCommit(nd *Node, version int){

	nd.CidVector = nd.VectorCache[version]

	for k := range nd.VectorCache {

		if k <= version {
			delete(nd.VectorCache,k)
		}

	}

	// for _,cid := range nd.CidVector.Content {
	//
	// 	emb,emb_exists := nd.EmbsStaging[cid]
	//
	// 	if(emb_exists){
	//
	// 		nd.SearchIndex.Add(emb)
	// 		nd.CidVectorEmbs[cid] = emb
	// 		delete(nd.EmbsStaging,cid)
	//
	// 	} else {
	//
	// 		//TODO ask a random peer for it
	//
	// 	}
	//
	// }
}


//func heartBeatCheck(nd *Node){
func heartBeatCheck(lastLiderHeartBeat *time.Time){


	for {

		if(time.Since(*lastLiderHeartBeat) >= (30*time.Second)){
			fmt.Println("O lider falhou vamos a eleição")
			//TODO passar para rotina de eleicao
		}

		time.Sleep(15 * time.Second)

	}

}


// func joinSender(nd *Node, ctx context.Context) {
//
//     for {
//         select {
// 			case <-ctx.Done():
// 				fmt.Printf("A parar envio de joins\n")
// 				return
// 			case <-time.After(15 * time.Second):
// 				messaging.PublishTo(
// 					nd.IpfsApi.PubSub(),
// 					messaging.JOIN,
// 					messaging.JoinMessage{},
// 				)
//         }
//     }
// }


func (nd *Node) getPubSubRandomPeer(ctx context.Context, topic string) (peer.ID, error) {
    ps := nd.IpfsApi.PubSub()

    peers, err := ps.Peers(ctx, ifaceOptions.PubSub.Topic(topic))
    if err != nil {
		err = fmt.Errorf("Erro ao procurar peers do pubsub para o tópico %s: %v\n", topic, err)
        return nd.IpfsCore.Identity,err 
    }

	idx := rand.Intn(len(peers)) 

    return peers[idx],nil
}

func followerRoutine(nd *Node,ctx context.Context){

	// Assumimos o primeiro beat na rotina follower
	lastLiderHeartBeat := time.Now().Add(15 * time.Second)

    fmt.Printf("A iniciar routina follower\n")

	// TODO Alterar para depois acomodar o processo de eleicao
	go messaging.ListenTo(nd.IpfsApi.PubSub(),messaging.HTB,func(sender peer.ID, msg any, stop *bool) { 

		htbmsg,ok := msg.(messaging.HeartBeatMessage)
	    if(!ok){
			fmt.Printf("Esperado HearBeatMessage, obtido %T", msg)
	    }

		lastLiderHeartBeat = time.Now()

		Npeers = htbmsg.Npeers 
	})

	go heartBeatCheck(&lastLiderHeartBeat)


 //    rpeerctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	// randomPeer,error := nd.getPubSubRandomPeer(rpeerctx,string(messaging.RBLQ))
 //    cancel()
	//
	// if(error == nil){
	//
	//
	// 	fmt.Printf("Entrei agora na rede vou pedir rebuild ao peer %v\n\n",randomPeer)
	//
	// 	go messaging.PublishTo(nd.IpfsApi.PubSub(),messaging.RBLQ,messaging.RebuildQueryMessage{
	// 		Dest: randomPeer,
	// 		Info: []string{},
	// 	})
	//
	// }
	//
	//
	// go messaging.ListenTo(nd.IpfsApi.PubSub(),messaging.RBLR,func(sender peer.ID, msg any, stop *bool) { 
	//
	//
	// 	fmt.Printf("Recebi resposta de rebuild do peer %v\n\n",sender)
	//
	// 	rblrmsg,ok := msg.(messaging.RebuildResponseMessage)
	//     if(!ok){
	// 		fmt.Printf("Esperado rebuildResponseMessage, obtido %T", rblrmsg)
	//     }
	//
	// 	if(rblrmsg.Dest == nd.IpfsCore.Identity){
	//
	//
	// 		fmt.Printf("O destinatario sou eu vou dar rebuild\n\n")
	//
	// 		for cid,emb := range rblrmsg.Response {
	//
	//
	// 			if(rblrmsg.Total){
	//
	// 				nd.CidVector.Content = append(nd.CidVector.Content, cid)
	// 			}
	//
	// 			nd.CidVectorEmbs[cid] = emb
	// 			nd.SearchIndex.Add(emb)
	//
	// 		}
	//
	//
	// 		fmt.Printf("Vetor atualizado :\n%v\n",nd.CidVector.String())
	//
	//
	// 	}
	//
	// })


	go messaging.ListenTo(nd.IpfsApi.PubSub(),messaging.AEM,func(sender peer.ID, msg any, stop *bool) {
	    aemMsg,ok := msg.(messaging.AppendEntryMessage)
	    if(!ok){
	        fmt.Printf("Esperado AppendEntryMessage, obtido %T", msg)
	    }
	    fmt.Printf("Recebi AE message com vetor\n%v\n",aemMsg.Vector.String())
	    receiveNewVector(nd,aemMsg.Vector,aemMsg.Embeddings)
	})

	go messaging.ListenTo(nd.IpfsApi.PubSub(),messaging.COMM,func(sender peer.ID, msg any, stop *bool) { 
		fmt.Printf("Mensagem de commit recebida\n")
		fmt.Printf("Vetor atual :\n%v\n",nd.CidVector.String())
	    commMgs,ok := msg.(messaging.CommitMessage)
	    if(!ok){
			fmt.Printf("Esperado CommitMessage, obtido %T", msg)
	    }
		receiveCommit(nd,commMgs.Version) 
		fmt.Printf("Vetor atualizado :\n%v\n",nd.CidVector.String())
	})


	// Espera bloqueante (equanto o context não for cancelado)
	<-ctx.Done()
        fmt.Printf("A terminar rotina follower\n")

}

// Lider

func receiveAck(nd *Node,hash string,version int) (bool){

	valid := nd.CidVector.Hash() == hash;

	if(valid){
		nd.StagingAKCs[version] = nd.StagingAKCs[version] + 1
		fmt.Printf("Numeros de ACKs para a versao %v: %v/%v\n",version,nd.StagingAKCs[version],Npeers)  
	}

	fmt.Printf("Hash recebida %s\n",hash)
	fmt.Printf("Hash do vetor atual %s\n",nd.CidVector.Hash())


	if(nd.StagingAKCs[version] >= int(Npeers/2)){

	    nd.CidVector = nd.VectorCache[version]
	    messaging.PublishTo(nd.IpfsApi.PubSub(),messaging.COMM,messaging.CommitMessage{ Version: version })

		for k := range nd.VectorCache {

			if k <= version {
				delete(nd.StagingAKCs,k)
				delete(nd.VectorCache,k)
			}

		}
		//
		// for _,cid := range nd.CidVector.Content {
		//
		// 	emb,emb_exists := nd.EmbsStaging[cid]
		//
		// 	if(emb_exists){
		//
		// 		nd.SearchIndex.Add(emb)
		// 		nd.CidVectorEmbs[cid] = emb
		// 		delete(nd.EmbsStaging,cid)
		//
		// 	}
		// }
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


func heartBeatSender(nd *Node) {

    topic := string(messaging.AEM)

    for {
        ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
        n := nd.getPubSubPeersCount(ctx, topic)
        cancel()

        if n > 0 && Npeers != n {
            Npeers = n
            fmt.Printf("Npeers atualizado a partir do pubsub: %v\n", Npeers)
        }

        messaging.PublishTo(
            nd.IpfsApi.PubSub(),
            messaging.HTB,
            messaging.HeartBeatMessage{Npeers: Npeers},
        )

        time.Sleep(15 * time.Second)
    }
}

func liderRoutine(nd *Node,ctx context.Context){


    fmt.Printf("A iniciar routina lider\n")

	go heartBeatSender(nd)

	go  messaging.ListenTo(nd.IpfsApi.PubSub(),messaging.ACK,func(sender peer.ID, msg any, stop *bool) {
	    ackmsg,ok := msg.(messaging.AckMessage)
	    if(!ok){
			fmt.Printf("Esperado AppendEntryMessage, obtido %T", msg)
	    }

	    fmt.Printf("Recebi ACK de %v, com hash %v\n",sender,ackmsg.Hash)

	    valid := receiveAck(nd,ackmsg.Hash,ackmsg.Version)

	    if(!valid){
			fmt.Printf("Vetor atual hash:\n%v\n",nd.CidVector.String())
			fmt.Printf("ACK hash não valida\n")
	    }
		    
	})

	// Espera bloqueante (equanto o context não for cancelado)
	<-ctx.Done()
        fmt.Printf("A terminar rotina follower\n")


}


// Funções Auxiliares // 

func isSubset(sub Vector, super Vector) bool {

	var subArray []string
	var superArray []string

	subArray = sub.Content
	superArray = super.Content

	if(len(subArray) == 0){
		return true
	}

	if(len(subArray) > len(superArray)){
		return false
	}

	for i:=0;i<len(subArray);i++{

		if(superArray[i]!=subArray[i]){
			return false
		}

	}

	return true
}














