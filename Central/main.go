package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"os/signal"
	"strconv"
	"sync"
	"syscall"
	"time"

	pb "github.com/anibalox/Distribuidos2/Proto"
	amqp "github.com/rabbitmq/amqp091-go"
	"google.golang.org/grpc"
)

var mu_archivo sync.Mutex
var f *os.File
var ipToNumber = map[string]string{
	"10.6.46.47": "1",
	"10.6.46.48": "2",
	"10.6.46.49": "3",
	"10.6.46.50": "4",
	"localhost" : "10",
}
var EquiposDisponibles = make([]string, 0)

/*
func enqueue

param: ColaEspera []string, element string

return: []string

Esta funcion agrega un elemento a una copia de una cola de espera y devuelve la nueva cola con el elemento
agregado
*/

func enqueue(ColaEspera []string, element string) []string {
	ColaEspera = append(ColaEspera, element)
	return ColaEspera
}

/*
func dequeue

param: ColaEspera []string

return: string, []string

Esta funcion elimina el primer elemento de una copia decola y devuelve el elemento
junto con la nueva cola cambiada
*/

func dequeue(ColaEspera []string) (string, []string) {
	element := ColaEspera[0]
	if len(ColaEspera) == 1 {
		var tmp = []string{}
		return element, tmp

	}
	return element, ColaEspera[1:]
}

func ComunicarseConLaboratorio(client pb.LaboratorioClient, nro_lab string, nro_escuadron string) {

	var situacion *pb.MessageInter
	var cantidadMensajes int

	stream, _ := client.Intercambio(context.Background())

	//Enviar Ayuda
	fmt.Println("Se envia escuadron " + nro_escuadron + " a laboratorio " + nro_lab)
	stream.Send(&pb.MessageInter{Body: nro_escuadron})

	//Realizando battalla. Esperar respuesta de situacion de lab
	cantidadMensajes = 0

	for situacion, _ = stream.Recv(); situacion.Body == "NO LISTO"; situacion, _ = stream.Recv() {
		fmt.Println("Estatus Escuadra " + nro_escuadron + " : [" + situacion.Body + "]")
		cantidadMensajes += 1
		time.Sleep(5 * time.Second)

		if err := stream.Send(&pb.MessageInter{Body: nro_escuadron}); err != nil {
			break
		}
	}

	mu_archivo.Lock()
	f.WriteString("Lab" + nro_lab + ";" + strconv.Itoa(cantidadMensajes) + "\n")
	mu_archivo.Unlock()

	//Equipo listo. Recibiendo al equipo
	fmt.Println("Estatus Escuadra " + nro_escuadron + " : [" + situacion.Body + "]")
	fmt.Println("Retorno a Central Escuadra " + nro_escuadron + ", Conexion Laboratorio " + nro_lab + " Cerrada")

}

func main() {
	qName := "Emergencias"                                      //Nombre de la cola                                           //Host de RabbitMQ 172.17.0.1
	connQ, err := amqp.Dial("amqp://test:test@localhost:5670/") //Conexion con RabbitMQ
	EquiposDisponibles = append(EquiposDisponibles, "1")
	EquiposDisponibles = append(EquiposDisponibles, "2")

	if err != nil {
		log.Fatal(err)
	}
	defer connQ.Close()

	ch, err := connQ.Channel()
	if err != nil {
		log.Fatal(err)
	}
	defer ch.Close()

	q, err := ch.QueueDeclare(qName, false, false, false, false, nil) //Se crea la cola en RabbitMQ
	if err != nil {
		log.Fatal(err)
	}

	fmt.Println(q)

	fmt.Println("Esperando Emergencias")
	chDelivery, err := ch.Consume(qName, "", true, false, false, false, nil) //obtiene la cola de RabbitMQ
	if err != nil {
		log.Fatal(err)
	}

	f, _ = os.Create("SOLICITUDES")
	c := make(chan os.Signal)
	signal.Notify(c, os.Interrupt, syscall.SIGTERM)
	go func() {
		<-c
		fmt.Println("Enviando senales de termino")
		/*
			for k := range ipToNumber {
				hostS := k
				port := ":50051"
				connS, err := grpc.Dial(hostS+port, grpc.WithInsecure())

				if err != nil {
					panic("No se pudo conectar con el servidor" + err.Error())
				}

				serviceCliente := pb.NewLaboratorioClient(connS)
				serviceCliente.Finalizar(context.Background(), &pb.MessageFin{Body: "1"})
				fmt.Println("Se finalizo laboratorio " + ipToNumber[k])
				connS.Close()
			}
		*/
		hostS := "localhost"
		port := ":50051"
		connS, err := grpc.Dial(hostS+port, grpc.WithInsecure())

		if err != nil {
			panic("No se pudo conectar con el servidor" + err.Error())
		}

		serviceCliente := pb.NewLaboratorioClient(connS)
		serviceCliente.Finalizar(context.Background(), &pb.MessageFin{Body: "1"})
		fmt.Println("Se finalizo laboratorio conectado")
		connS.Close()

		os.Exit(1)
	}()

	for delivery := range chDelivery {

		for len(EquiposDisponibles) == 0 {
			time.Sleep(1 * time.Second)
		}

		go func(delivery amqp.Delivery) {
			var equipo string

			hostS := string(delivery.Body)
			port := ":50051"
			fmt.Println("Pedido de ayuda de " + ipToNumber[string(delivery.Body)])
			connS, err := grpc.Dial(hostS+port, grpc.WithInsecure())

			if err != nil {
				panic("No se pudo conectar con el servidor" + err.Error())
			}

			defer connS.Close()

			serviceCliente := pb.NewLaboratorioClient(connS)

			equipo, EquiposDisponibles = dequeue(EquiposDisponibles)
			ComunicarseConLaboratorio(serviceCliente, ipToNumber[string(delivery.Body)], equipo)
			EquiposDisponibles = enqueue(EquiposDisponibles, equipo)
		}(delivery)

	}

}

