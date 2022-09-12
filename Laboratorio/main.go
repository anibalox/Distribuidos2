package main

import (
	"context"
	"fmt"
	"log"
	"math/rand"
	"net"
	"os"
	"time"

	pb "github.com/Kendovvul/Ejemplo/Proto"
	amqp "github.com/rabbitmq/amqp091-go"
	"google.golang.org/grpc"
)

type server struct {
	pb.UnimplementedLaboratorioServer
}

var ipCentral string
var resuelto bool

func myIP() string {
	conn, error := net.Dial("udp", "8.8.8.8:80")
	if error != nil {
		fmt.Println(error)
	}
	defer conn.Close()
	ipAddress := conn.LocalAddr().(*net.UDPAddr).IP.String()
	return ipAddress
}

func centralIPValue() string {
	ipAddress := myIP()
	if ipAddress == "10.6.46.48" || ipAddress == "10.6.46.49" || ipAddress == "10.6.46.50" {
		return "10.6.46.47"
	}
	return "localhost"
}

func CalcularEstallido() string {
	var resultado string
	if rand.Float64() <= 0.8 {
		resultado = "ESTALLIDO"
	} else {
		resultado = "OK"
	}

	return resultado
}

func CalcularResolucion() string {
	var resultado string
	if rand.Float64() <= 0.6 {
		resultado = "LISTO"
	} else {
		resultado = "NO LISTO"
	}

	return resultado
}

func (s *server) Intercambio(stream pb.Laboratorio_IntercambioServer) error {

	var resolucion string
	situacion, _ := stream.Recv()
	nro_escuadron := situacion.Body
	fmt.Println("Llega escuadron " + nro_escuadron + ", conteniendo estallido")
	// Comienza batalla
	for resolucion = CalcularResolucion(); resolucion == "NO LISTO"; resolucion = CalcularResolucion() {
		fmt.Println("Revisando Estado Escuadron: [" + resolucion + "]")
		stream.Send(&pb.MessageInter{Body: resolucion})
		_, _ = stream.Recv()
	}

	//Termina batalla. Devolviendo equipo
	fmt.Println("Revisando Estado Escuadron: [" + resolucion + "]")
	stream.Send(&pb.MessageInter{Body: resolucion})
	fmt.Println("Estallido contenido. Escuadron " + nro_escuadron + " Retornando")

	resuelto = true
	return nil
}

func (s *server) Finalizar(ctx context.Context, msg *pb.MessageFin) (*pb.MessageFin, error) {
	for !resuelto {
		time.Sleep(1 * time.Second)
	}
	defer os.Exit(1)
	return &pb.MessageFin{Body: "1"}, nil
}

func main() {
	var estallido string

	ipLab := myIP()        //nombre del laboratorio. Hay que cambiarlo
	qName := "Emergencias" //nombre de la cola
	ipCentral = centralIPValue()
	connQ, err := amqp.Dial("amqp://test:test@" + ipCentral + ":5670/")

	if err != nil {
		log.Fatal(err)
	}
	defer connQ.Close()

	ch, err := connQ.Channel()
	if err != nil {
		log.Fatal(err)
	}
	defer ch.Close()

	listener, err := net.Listen("tcp", ":50051") //conexion sincrona
	if err != nil {
		panic("La conexion no se pudo crear" + err.Error())
	}

	serv := grpc.NewServer()
	pb.RegisterLaboratorioServer(serv, &server{})

	go func() {
		for {

			time.Sleep(5 * time.Second)
			for estallido = CalcularEstallido(); estallido == "OK"; estallido = CalcularEstallido() {
				fmt.Println("Analizando estado Laboratorio [" + estallido + "]")
				time.Sleep(5 * time.Second)
			}
			fmt.Println("Analizando estado Laboratorio [" + estallido + "]")
			fmt.Println("SOS Enviado a Central. Esperando respuesta...")

			err = ch.Publish("", qName, false, false,
				amqp.Publishing{
					Headers:     nil,
					ContentType: "text/plain",
					Body:        []byte(ipLab),
				})

			if err != nil {
				log.Fatal(err)
			}
			resuelto = false
			for !resuelto {
				time.Sleep(1 * time.Second)
			}
		}
	}()

	if err = serv.Serve(listener); err != nil {
		panic("El server no se pudo iniciar" + err.Error())
	}

}
