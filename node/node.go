package main

import (
    "context"
    "fmt"
    "io"
    "log"
    "os"
    "time"

    "github.com/ipfs/kubo/config"
    "github.com/ipfs/kubo/core"
    "github.com/ipfs/kubo/core/coreapi"
    "github.com/ipfs/kubo/core/node/libp2p"
    "github.com/ipfs/kubo/plugin/loader"
    "github.com/ipfs/kubo/repo/fsrepo"
    "github.com/libp2p/go-libp2p/core/peer"
    "github.com/multiformats/go-multiaddr"
    "sdt/node/api"
    "sdt/node/services/messaging"
)

func createRepo(repoPath string) bool {
    _, err := os.Stat(repoPath)
    if err == nil {
        return true
    }

    if !os.IsNotExist(err) {
        log.Printf("Failed to check repo directory: %v", err)
        return false
    }

    err = os.Mkdir(repoPath, 0750)
    if err != nil {
        log.Printf("Failed to create repo directory: %v", err)
        return false
    }

    cfg, err := config.Init(io.Discard, 2048)
    if err != nil {
        log.Printf("Failed to create repo config: %v", err)
        return false
    }

    err = fsrepo.Init(repoPath, cfg)
    if err != nil {
        log.Printf("Failed to initialize repo: %v", err)
        return false
    }

    return true
}

func createNode(ctx context.Context, repoPath string) (*core.IpfsNode, error) {
    repo, err := fsrepo.Open(repoPath)

    if err != nil {
        if !createRepo(repoPath) {
            return nil, fmt.Errorf("failed to create repo")
        }
        repo, err = fsrepo.Open(repoPath)
        if err != nil {
            return nil, fmt.Errorf("failed to open repo after creation: %v", err)
        }
    }

    nodeOptions := &core.BuildCfg{
        Online:  true,
        Routing: libp2p.DHTOption,
        Repo:    repo,
    }

    node, err := core.NewNode(ctx, nodeOptions)
    if err != nil {
        return nil, fmt.Errorf("failed to create IPFS node: %v", err)
    }

    return node, nil
}

func main() {
    repoPath := "C:\\Users\\bento\\.ipfs"

    plugins, err := loader.NewPluginLoader(repoPath)
    if err != nil {
        panic(fmt.Errorf("error loading plugins: %s", err))
    }

    if err := plugins.Initialize(); err != nil {
        panic(fmt.Errorf("error initializing plugins: %s", err))
    }

    if err := plugins.Inject(); err != nil {
        panic(fmt.Errorf("error initializing plugins: %s", err))
    }

    ctx := context.Background()
    
    node, err := createNode(ctx, repoPath)
    if err != nil {
        log.Fatalf("Error creating IPFS node: %v", err)
    }

    go func() {
        time.Sleep(2 * time.Second)
        
        bootstrapPeers := []string{
            "/dnsaddr/bootstrap.libp2p.io/p2p/QmNnooDu7bfjPFoTZYxMNLWUQJyrVwtbZg5gBMjTezGAJN",
            "/dnsaddr/bootstrap.libp2p.io/p2p/QmQCU2EcMqAqQPR2i9bChDtGNJchTbq5TbXJJ16u19uLTa",
        }
        
        for _, addr := range bootstrapPeers {
            maddr, _ := multiaddr.NewMultiaddr(addr)
            peerInfo, _ := peer.AddrInfoFromP2pAddr(maddr)
            err := node.PeerHost.Connect(ctx, *peerInfo)
            if err == nil {
                log.Printf("Connected to bootstrap peer")
            }
        }
    }()

    ipfsService, err := coreapi.NewCoreAPI(node)
    if err != nil {
        log.Fatalf("Error creating IPFS CoreAPI: %v", err)
    }

    fmt.Println("IPFS Node created successfully: " + node.Identity.String())

    pubSubService, err := messaging.NewPubSubService(ctx, node, "Uploaded-Files-Topic")
    if err != nil {
        log.Fatalf("Error creating PubSub service: %v", err)
    }
    defer pubSubService.Close()

    log.Println("PubSub service started - waiting for peers...")

    api.Initialize(ctx, ipfsService, pubSubService)
}
