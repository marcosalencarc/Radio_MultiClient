package main

import "net"
import "fmt"
import "bufio"
import (
	"os"

	"strconv"
	"strings"
	//"bytes"
	//"encoding/binary"
)

func lerShell(toControl chan<- string) {
	leitor := bufio.NewReader(os.Stdin)
	fmt.Println("Shell\n Selecione uma estação ou Q para sair\n")
	for {
		text, _ := leitor.ReadString('\n')
		toControl <- text
	}
	close(toControl)
}

func lerDaConexao(fromControl chan []byte, conn net.Conn) {
	leitor := bufio.NewReader(conn)
	for {
		text, err := leitor.ReadString('\n')

		if err == nil {
			fromControl <- []byte(strings.Split(text,"\n")[0])
            //fmt.Println(text)
		}
	}
}

func main() {
	entrada := make(chan string)
	canalServidor := make(chan []byte)

	// conectando na porta 6666 via protocolo tcp/ip na máquina local
	conexao, erro1 := net.Dial("tcp", "127.0.0.1:"+os.Args[1])
	if erro1 != nil {
		fmt.Println(erro1)
		os.Exit(3)
	}
	porta := os.Args[2]
	conexao.Write([]byte(porta+"\n"))

	go lerShell(entrada)
	go lerDaConexao(canalServidor, conexao)
    
    
	for {
        
        _,erroAlive := conexao.Write([]byte("alive"+"\n"))
        if(erroAlive!=nil){
            conexao.Close()
            fmt.Println("Servidor Não responde")
            return
        }
		select {
        
        case msg := <-entrada:
			msg = msg[:len(msg)-1]
			//fmt.Println(msg)
			if msg == "q" {
				conexao.Write(append([]byte(msg),byte('\n')))
				conexao.Close()
				fmt.Println("Até logo")
				return
			}
			_, err := strconv.Atoi(msg)
			if err == nil {
				_,erro5 := conexao.Write(append([]byte(msg),byte('\n')))
                if erro5!=nil{
                    conexao.Close()
                    fmt.Println("Servidor Parou")
                }
			} else {
				fmt.Println("Estação Inválida")
				continue
			}
        case msg := <-canalServidor:
            if(string(msg)=="f"){
                fmt.Println("Servidor encerrou")
                os.Exit(0)
            }else if(string(msg)!="alive"){
            fmt.Println(string(msg))
            }
        }

    }
}
