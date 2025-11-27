package node

import (

	// bibs padrão

	"context"
	"errors"
	"fmt"
	"io"

	// "log"
	"os"
	"strings"
	"time"

	// bibs internas
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

	iface "github.com/ipfs/kubo/core/coreiface"
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
	VectorCache map[int]Vector
	EmbsStaging [][]float32
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
		EmbsStaging: [][]float32{},
		CidVector: emptyVector,
		State: state,

	}

	if(LeaderFlag){

		n.CommitDone = make(chan struct{})
		n.StagingAKCs = make(map[int]int)

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

	switch nd.State {
	case LEADER:
	     liderRoutine(nd,ctx)
	default:
	     followerRoutine(nd,ctx)
	}

}

// Follower

func receiveNewVector(nd *Node,v Vector, embs [][]float32){

	if(nd.CidVector.Ver < v.Ver && isSubset(nd.CidVector,v)){
		fmt.Printf("Vetor recebido:\n%v\n",v.String())
		fmt.Printf("Hash do vetor: %s\n",v.Hash())
		messaging.PublishTo(nd.IpfsApi.PubSub(),messaging.ACK,messaging.AckMessage{Version: v.Ver,Hash: nd.CidVector.Hash()})
	}

	nd.VectorCache[v.Ver] = v
	nd.EmbsStaging = embs

}

func receiveCommit(nd *Node, version int){

	nd.CidVector = nd.VectorCache[version]

	for k := range nd.VectorCache {

		if k <= version {
			delete(nd.VectorCache,k)
		}

	}
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

func followerRoutine(nd *Node,ctx context.Context){

	// Assumimos o primeiro beat na rotina follower
	lastLiderHeartBeat := time.Now().Add(15 * time.Second)

    fmt.Printf("A iniciar routina follower\n")

	go messaging.ListenTo(nd.IpfsApi.PubSub(),messaging.AEM,func(sender peer.ID, msg any) {
	    aemMsg,ok := msg.(messaging.AppendEntryMessage)
	    if(!ok){
	        fmt.Printf("Esperado AppendEntryMessage, obtido %T", msg)
	    }
	    fmt.Printf("Recebi AE message com vetor\n%v\n",aemMsg.Vector.String())
	    receiveNewVector(nd,aemMsg.Vector,aemMsg.Embeddings)
	})

	go messaging.ListenTo(nd.IpfsApi.PubSub(),messaging.COMM,func(sender peer.ID, msg any) { 
		fmt.Printf("Mensagem de commit recebida\n")
		fmt.Printf("Vetor atual :\n%v\n",nd.CidVector.String())
	    commMgs,ok := msg.(messaging.CommitMessage)
	    if(!ok){
			fmt.Printf("Esperado CommitMessage, obtido %T", msg)
	    }
		receiveCommit(nd,commMgs.Version) 
		fmt.Printf("Vetor atualizado :\n%v\n",nd.CidVector.String())
	})

	// TODO Alterar para depois acomodar o processo de eleicao
	go messaging.ListenTo(nd.IpfsApi.PubSub(),messaging.HTB,func(sender peer.ID, msg any) { 

		htbmsg,ok := msg.(messaging.HeartBeatMessage)
	    if(!ok){
			fmt.Printf("Esperado HearBeatMessage, obtido %T", msg)
	    }

		lastLiderHeartBeat = time.Now()

		Npeers = htbmsg.Npeers 
		messaging.PublishTo(nd.IpfsApi.PubSub(),messaging.IALV,messaging.IAmAliveMessage{})
	})

	go heartBeatCheck(&lastLiderHeartBeat)

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

	// Encontrar numero de peers com swarm (Dá numeros estupidos)


	// findPeersContext, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	// defer cancel()
	//
	// sw := nd.IpfsApi.Swarm()
	// n,err := sw.Peers(findPeersContext)
	//
	// if(err != nil){
	// 	fmt.Printf("Um erro ocurreu á procura dos peers : %v \n",err)
	// }
	//
	// fmt.Printf("Swarm diz que estamos conectados com %v\n", n)

	if(nd.StagingAKCs[version] >= int(Npeers/2)){

	    nd.CidVector = nd.VectorCache[version]
	    messaging.PublishTo(nd.IpfsApi.PubSub(),messaging.COMM,messaging.CommitMessage{ Version: version })

		for k := range nd.VectorCache {

			if k <= version {
				delete(nd.StagingAKCs,k)
				delete(nd.VectorCache,k)
			}

		}
	}

	return valid
}

func heartBeatSender(nd *Node, AlivePeers map[peer.ID]time.Time){

	// A primeira vez é corrida sem atualizar o numero de peers de modo a dar tempo para os peers preencherem o 
	// mapa

	messaging.PublishTo(nd.IpfsApi.PubSub(),messaging.HTB,messaging.HeartBeatMessage{Npeers : Npeers})
	time.Sleep(15 * time.Second)

	for {

		messaging.PublishTo(nd.IpfsApi.PubSub(),messaging.HTB,messaging.HeartBeatMessage{Npeers : Npeers})
		time.Sleep(15 * time.Second)

		for peer, tms := range AlivePeers {

			currentTime := time.Now()
			timeSinceLastIamAlive := currentTime.Sub(tms)

			if(timeSinceLastIamAlive >= (45 * time.Second)){
				delete(AlivePeers,peer)
			}

		}

		Npeers = len(AlivePeers)
		fmt.Printf("Npeers Atualizado para %v\n",Npeers)
	}

}

func liderRoutine(nd *Node,ctx context.Context){


    fmt.Printf("A iniciar routina lider\n")

	AlivePeers := make(map[peer.ID]time.Time) // Timestamp of last IamAliveMsg

	go heartBeatSender(nd,AlivePeers)

	go  messaging.ListenTo(nd.IpfsApi.PubSub(),messaging.ACK,func(sender peer.ID, msg any) {
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


	go  messaging.ListenTo(nd.IpfsApi.PubSub(),messaging.IALV,func(sender peer.ID, msg any) {

	    _,ok := msg.(messaging.IAmAliveMessage)

	    if(!ok){
			fmt.Printf("Esperado IamAliveMessage, obtido %T", msg)
	    }

		AlivePeers[sender] = time.Now()
		    
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














