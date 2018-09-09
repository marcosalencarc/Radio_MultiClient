package main

import (
	"fmt"
	"net"
	"os"
	"log"
	"sync"
	"reflect"
	"strings"
	"bufio"
	"io"
	"time"
	"strconv"
)
type Estacao struct {
	clients []*Cliente
	name	string
	musica 	string
	musicaEnvio chan []byte
	canalAddCliente chan *Cliente
	canalRemoveCliente chan *Cliente

}
type Cliente struct{
	conTCP net.Conn
	conUDP net.Conn
	portaUDP string
	envio          chan []byte
	est 	string
}

var (
	estacoes map[string] Estacao
	mutex sync.RWMutex
	encerrarServer chan string
)

func listarEstacoes() string {

	lista := "Lista de Radios \n"

	mutex.RLock()
	for _,nomeRadio := range reflect.ValueOf(estacoes).MapKeys() {
		lista += ("Estação "+nomeRadio.Interface().(string) + "\n")
	}
	mutex.RUnlock()
	//fmt.Println(lista)
	return lista
}
func listarClientesEstacoes() string {

	lista := "Lista de Radios \n"
	mutex.RLock()

	for _,nomeRadio := range reflect.ValueOf(estacoes).MapKeys() {
		lista += ("Estação "+nomeRadio.Interface().(string)+"{")
		est := estacoes[nomeRadio.Interface().(string)].clients
		//fmt.Println(est)
		for i,client := range est {
			lista+= "Cliente "+ strconv.Itoa(i) + ", PortaUDP:"+ client.portaUDP+"|"

		}
		lista+= "}\n"
	}
	mutex.RUnlock()
	//fmt.Println(lista)
	return lista
}

func acceptConect(listener net.Listener) {
	for {
		select {
		case msg:= <-encerrarServer:
			fmt.Println(msg)
			listener.Close()
			return
		default:
			conn, err := listener.Accept()
			if err != nil {
				log.Print(err)
				continue
			}
			port, _ := bufio.NewReader(conn).ReadString('\n')
			portUDP := (strings.Split(port, "\n")[0])
			client := &Cliente{
				conTCP:conn,
				envio: make(chan []byte,2),
				portaUDP:portUDP,
				est: "",
			}
			lo,err2 := net.Dial("udp", "127.0.0.1:"+client.portaUDP)
			if err2!=nil{
				fmt.Println("erro")
			}
			client.conUDP = lo
			go handleConect(client)
		}
	}
}

func (estacao *Estacao) lerMusica() {
	data := make([]byte, 1024*16)

	for {
		f, err := os.Open(estacao.musica)
		if err != nil {
			fmt.Println(err)
			return
		}
		defer f.Close()
		for {
			_, err = f.Read(data)
			if err != nil {
				if err == io.EOF {
					fmt.Println("Iniciar Novamente")
					break
				}
			}
			time.Sleep(1 * time.Second)
			estacao.musicaEnvio <- data
		}
	}
}


func handleEstacoes(estacao Estacao){
	go estacao.lerMusica()
	for {
		//fmt.Println("Handle estacoes")
		select {
        case fecha := <-encerrarServer:
            for _, cliente := range estacao.clients {
				_, err := cliente.conTCP.Write([]byte(fecha))
				if err != nil{
					fmt.Println(err)
				}
			}
		case client := <-estacao.canalAddCliente:
			clientes := estacao.clients
			estacao.clients = append(clientes, client)
			estacoes[estacao.name] = estacao

		case client := <-estacao.canalRemoveCliente:
			var i int
			for i,_ = range estacao.clients {
				if estacao.clients[i].conTCP == client.conTCP {
					break
				}
			}
			estacao.clients = append(estacao.clients[:i], estacao.clients[i+1:]...)
			estacoes[estacao.name] = estacao
			//fmt.Println(estacao.clients)
		case buff := <-estacao.musicaEnvio:
			for _, cliente := range estacao.clients {
				_, err := cliente.conUDP.Write(buff)
				if err != nil{
					//fmt.Println(err)
				}
			}
		}
	}

}

func iniciarEstacoes(estaco []string){

	for i:=0; i< len(estaco);i++{
		nameEst := strconv.Itoa(i+1)
		EstacaoTemp := Estacao{
			clients: make([]*Cliente,0),
			name: nameEst,
			musica: estaco[i],
			musicaEnvio: make(chan [] byte,1),
			canalAddCliente: make(chan *Cliente,1),
			canalRemoveCliente: make(chan *Cliente,1),
		}
		estacoes[EstacaoTemp.name] = EstacaoTemp
		go handleEstacoes(estacoes[EstacaoTemp.name])


	}
}


func main() {
	estacoes = make(map[string]Estacao)

	encerrarServer = make(chan string,1)

	fmt.Println("Servidor aguardando conexões TCP...")


	listener, erro1 := net.Listen("tcp","127.0.0.1:"+os.Args[1])
	if erro1 != nil {
		fmt.Println(erro1)
		os.Exit(3)
	}
	defer listener.Close()

	iniciarEstacoes(os.Args[2:])
	go receberTerminal()
	acceptConect(listener)


}

func lerShell(toControl chan<- string) {
	leitor := bufio.NewReader(os.Stdin)
	fmt.Println("P - Listar Estações\n Q - Sair")
	for {
		text, _ := leitor.ReadString('\n')
		toControl <- text
	}
	close(toControl)
}

func receberTerminal(){
	entrada := make(chan string)
	go lerShell(entrada)
	for {
		select {
		case msg := <-entrada:
			msg = msg[:len(msg)-1]

			if msg == "p" {
				fmt.Println(listarClientesEstacoes())
			}
			if msg == "q" {
                
				for _,nomeRadio := range reflect.ValueOf(estacoes).MapKeys() {
                    
                    est := estacoes[nomeRadio.Interface().(string)].clients
                    //fmt.Println(est)
                    for _,client := range est {
                        client.conTCP.Write([]byte("f\n"))
                        //fmt.Println(msg)
                    }		
                }
				os.Exit(0)
			}
		}
	}
}
func handleConect(client *Cliente) {

	client.conTCP.Write([]byte(listarEstacoes()))
	reader := bufio.NewReader(client.conTCP)
	//fmt.Println(client.envio)
	for {

		command, err := reader.ReadBytes(byte('\n'))
		//fmt.Println(command)
		//
		if err == nil {
			command = command[:len(command)-1]
			if string(command) == ("alive"){
				client.conTCP.Write([]byte("alive\n"))
				continue
			}else{

				e, ok:= estacoes[string(command)]
				if(ok == true){
					if(client.est!="") {
						eOld := estacoes[client.est]
						eOld.canalRemoveCliente <-client
					}
					client.est = string(command)
					fmt.Println(string(command))
					e.canalAddCliente <- client
				}else{
					client.envio <-([]byte("Estação Inválida"))
				}

			}
			if err != nil {
				fmt.Println("Conexão Falha")
			}
		}else{
			e := estacoes[client.est]
			e.canalRemoveCliente <-client
			client.conUDP.Close()
			return
		}

	}
}


