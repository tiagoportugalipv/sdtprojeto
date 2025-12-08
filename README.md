# Projeto de SDT

Este projeto contêm a implementação do sistema distribuido descrito nos requesitos do trabalho prático de SDT.

## Como correr

Quem não quiser instalar a biblioteca do faiss nativamente, faça isto :

export LD_LIBRARY_PATH="$(pwd)/local/lib:${LD_LIBRARY_PATH}"
go run main.go -r caminho/.ipfs -l
