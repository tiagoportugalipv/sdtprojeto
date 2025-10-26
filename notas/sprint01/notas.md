# Notas Sprint 01 | Instalação e configuração inicial

IPFS é simplemente um **protocolo**, pelo que para utilizarmos-o temos que escolher uma implementação.
Na documentação dão bastante ênfase a uma em específico, **[kubo](https://github.com/ipfs/kubo)**, pelo que a selecionamos de forma a ter uma base para nosso projeto bem suportada e documentada. 

Para desenvolvimento e testes por agora vamos utilizar VMs baseadas em debian 13.0 ligadas por uma rede interna NAT, futuramente podemos mover este workflow para contentores. O utilizador que estamos a utilizar nas VMs para fazer todas as operações abaixo é o `root`.

## Instalação do IPFS

Para instalar [kubo](https://github.com/ipfs/kubo), simplemente fazemos download dos binários para linux, e corremos os script de instalação (que simplesmente copia o binário para os locais adequados).

```bash
wget https://dist.ipfs.tech/kubo/v0.38.1/kubo_vXXXX_linux-amd64.tar.gz
tar xzf kubo_vXXXX_linux-amd64.tar.gz
cd kubo
sudo ./install.sh
```

Esta implementação é escrita em [go](https://go.dev), e pode dar jeito ter instalado no sistema operativo.
Também estamos a ponderar em implementar o projeto em go sendo que existe algums exemplos e documentação nesta linguagem de mecanismos que queremos replicar.
Para instalar go/golang basta correr a seguinte instrução em debian

```bash
apt-get install go
```

E com isto temos a implementação do protocolo instalada, apartir daqui, seguindo este [gist](https://github.com/Darkrove/Building-Private-IPFS-Network) foi feito o necessário para criar uma rede interna IPFS.

Em todos os nós começamos com os passos aqui descritos :

1. Criar um serviço (systemd), no caminho `/usr/lib/systemd/system/ipfsd.service`, de modo a que o *daemon* do ipfs seja iniciado em boot

```bash
[Unit]
Description=ipfs daemon

[Service]
Environment="LIBP2P_FORCE_PNET=1"
ExecStart=/usr/local/bin/ipfs daemon
Restart=always
User=root
Group=root

[Install]
WantedBy=multi-user.target
```

2. Gerar uma configuração inicial

```bash
ipfs init
```

3. Eliminar todos os bootstrap para redes externas 

```bash
ipfs bootstrap rm --all
```

(Bootstrap nodes em IPFS são nós apartir dos quais vamos obter informações sobre outros nós)

4. Mudar a configuração do ipfs de modo a publicar os vários serviços na rede

```bash
ipfs config Addresses.Gateway /ip4/0.0.0.0/tcp/8080
ipfs config Addresses.API /ip4/0.0.0.0/tcp/5001
ipfs config show
```

5. (Provavelmete opcional) Publicar os endereços em que os nós estão disponiveis

```bash
ipfs config --json Addresses.Announce '["/ip4/$IPDONÓ/tcp/4001"]'
ipfs config --json AutoConf.Enabled false
```

onde IPDONÓ é o ipv4 que pode ser obtido com `ip addr`

6. Ativar e começar o serviço criado (`ipfsd`)

```bash
systemctl enable ipfsd
systemctl start ipfsd
```

Tendo isto feito, vamos ao nosso lider node00, (neste sprint estático), e obtemos o seu endereço (pelo seguro o que está a correr por cima de TCP ou UDP).

```bash
ipfs id

    "{
        "ID": "12D3KooWPiB8x7kfANA9zCokc1MCdAzPgkibmaLnoeVbadovfWCD",
        "PublicKey": "CAESIM5tqM9Vq2sKOM7WYSR1CX4WSnjVIYgz9Gpikm4BhQIe",
        "Addresses": [
            "/ip4/10.10.1.1/tcp/4001/p2p/12D3KooWPiB8x7kfANA9zCokc1MCdAzPgkibmaLnoeVbadovfWCD"
        ],
        "AgentVersion": "kubo/0.38.1/",
        "Protocols": [
            "/ipfs/bitswap",
            "/ipfs/bitswap/1.0.0",
            "/ipfs/bitswap/1.1.0",
            "/ipfs/bitswap/1.2.0",
            "/ipfs/id/1.0.0",
            "/ipfs/id/push/1.0.0",
            "/ipfs/lan/kad/1.0.0",
            "/ipfs/ping/1.0.0",
            "/libp2p/autonat/1.0.0",
            "/libp2p/autonat/2/dial-back",
            "/libp2p/autonat/2/dial-request",
            "/libp2p/circuit/relay/0.2.0/stop",
            "/x/"
        ]
    }"
```

Com o endereço, no exemplo acima especificamente `/ip4/10.10.1.1/tcp/4001/p2p/12D3KooWPiB8x7kfANA9zCokc1MCdAzPgkibmaLnoeVbadovfWCD`, adicionamos o lider como bootstrap nos outros nós

```bash
ipfs bootstrap add /ip4/10.10.1.1/tcp/4001/p2p/12D3KooWPiB8x7kfANA9zCokc1MCdAzPgkibmaLnoeVbadovfWCD
```

Para verificar se realmente os nós foram registados e reconhecidos na rede podemos ir ao nó lider e executar a instrução abaixo

```bash
ipfs swarm peers
```

e obtemos todos os endereços dos peers conhecidos na rede.

## Utilização basica

Para utilizar o ipfs podemos adicionar um ficheiro ao sistema num peer

```bash
ipfs add randomFile.md
```

e depois noutro peer ir obter os conteudos desse ficheiro

```bash
ipfs cat CID
```

![exemplo](./assets/ipfsTest.png)




