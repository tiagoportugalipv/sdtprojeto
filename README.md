# Projeto de SDT

Este projeto contêm a implementação do sistema distribuido descrito nos requesitos do trabalho prático de SDT.

Estrutura do projeto:
````
SDTPROJETO-MAIN/
|- /node/
     |- api/ #
         |- controllers/ # Controladores das rotas
               |- main.go #
         |- routers/ # Rotas da api
               |- main.go #
         |- app.go/ # Inicializar a api
     |- services/ # Implementação pubSubcribe
               |- pubsub.go #
     |- go.mod # Define o nome do módulo e das depências
     |- go.sum # Guarda os Hashes de verificação das depencdências
     |- node.go # Ficheiro principal, ponto de partida, que inicializa o nó
```
