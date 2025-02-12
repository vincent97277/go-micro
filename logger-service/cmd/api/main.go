package main

import (
	"context"
	"fmt"
	"log"
	"log-service/data"
	"net"
	"net/http"
	"net/rpc"
	"time"

	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

const (
	webPort = "80"
	rpcPort = "5001"
	// mongoURL = "mongodb://mongo:27017"
	mongoEndpoint = "mongo:27017"
	gRpcPort      = "50001"
)

var client *mongo.Client

type Config struct {
	Models data.Models
}

func main() {
	// connect to mongo
	mongoClient, err := connectToMongo()
	if err != nil {
		log.Panic(err)
	}
	client = mongoClient

	// create a context in order to disconnect
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	// close connection
	defer func() {
		if err = client.Disconnect(ctx); err != nil {
			panic(err)
		}
	}()

	app := Config{
		Models: data.New(client),
	}

	// Register the RPC Server
	err = rpc.Register(new(RPCServer))
	go app.rpcListen()

	// start web server
	log.Println("Starting service on port", webPort)
	app.serve()
}

func (app *Config) serve() {
	srv := &http.Server{
		Addr:    fmt.Sprintf(":%s", webPort),
		Handler: app.routes(),
	}

	err := srv.ListenAndServe()
	if err != nil {
		log.Panic(err)
	}
}

func (app *Config) rpcListen() error {
	log.Println("Starting RPC server on port ", rpcPort)
	listen, err := net.Listen("tcp", fmt.Sprintf("0.0.0.0:%s", rpcPort))
	if err != nil {
		return err
	}
	defer listen.Close()

	for {
		rpcConn, err := listen.Accept()
		if err != nil {
			continue
		}
		go rpc.ServeConn(rpcConn)
	}
}

func connectToMongo() (*mongo.Client, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	// define full of mongourl
	mongoURL := fmt.Sprintf("mongodb://admin:password@%s/logs?authSource=admin", mongoEndpoint)
	log.Print("mongoURL: ", mongoURL)

	// setup connection options
	clientOptions := options.Client().ApplyURI(mongoURL)
	clientOptions.SetServerSelectionTimeout(5 * time.Second)

	// connect to MongoDB
	c, err := mongo.Connect(ctx, clientOptions)
	if err != nil {
		log.Println("Error connecting:", err)
		return nil, err
	}

	// test connection
	err = c.Ping(ctx, nil)
	if err != nil {
		log.Println("Error pinging MongoDB:", err)
		return nil, err
	}

	log.Println("Successfully connected to MongoDB!")

	return c, nil
}
