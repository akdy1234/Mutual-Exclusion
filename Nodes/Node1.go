package main

import (
	proto "Mutual_Exclusion/gRPC"
	"context"
	"fmt"
	"log"
	"net"
	"time"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

type Server struct {
	proto.UnimplementedMutual_ExclusionServer
	Id      string
	time2Id map[int64]string
	lTime   int64
}

func main() {

	var port int
	var Id string
	fmt.Println("Enter port:")
	fmt.Scanln(&port)
	fmt.Println("Enter id:")
	fmt.Scanln(&Id)

	array := []int{8000, 8001, 8002}
	var arrayTwo [2]int = [2]int{}
	var num int = 0

	go StartServer(Id, port)
	time.Sleep(2 * time.Second)

	fmt.Println("Type Y when all 3 nodes are running...")
	fmt.Scanln()

	for _, v := range array {
		if !(v == port) {
			arrayTwo[num] = v
			num++
			fmt.Println(v)
		}
	}

	address := fmt.Sprintf("localhost:%d", port)
	address2 := fmt.Sprintf("localhost:%d", arrayTwo[0])
	address3 := fmt.Sprintf("localhost:%d", arrayTwo[1])

	conn, err := grpc.NewClient(address, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		log.Fatalf("could not connect: %v", err)
	}

	conn2, err := grpc.NewClient(address2, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		log.Fatalf("could not connect: %v", err)
	}

	conn3, err := grpc.NewClient(address3, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		log.Fatalf("could not connect: %v", err)
	}

	client := proto.NewMutual_ExclusionClient(conn)
	client2 := proto.NewMutual_ExclusionClient(conn2)
	client3 := proto.NewMutual_ExclusionClient(conn3)

	var lTime int64

	time.Sleep(2 * time.Second)
	for {
		var message string
		var requestString *proto.Reply
		var releaseString *proto.Reply

		fmt.Scanln(&message)

		if message == "request" {
			requestString, _ = client.Request(context.Background(), &proto.AdminRequest{Id: Id, LogicalTime: lTime})
			requestString, _ = client2.Request(context.Background(), &proto.AdminRequest{Id: Id, LogicalTime: lTime})
			requestString, _ = client3.Request(context.Background(), &proto.AdminRequest{Id: Id, LogicalTime: lTime})
			log.Printf("1Response from server (REQUEST): %v", requestString)

		}
		if message == "release" {
			lTime++
			releaseString, _ = client.Release(context.Background(), &proto.AdminRelease{Id: Id, LogicalTime: lTime})
			releaseString, _ = client2.Release(context.Background(), &proto.AdminRelease{Id: Id, LogicalTime: lTime})
			releaseString, _ = client3.Release(context.Background(), &proto.AdminRelease{Id: Id, LogicalTime: lTime})
			log.Printf("1Response from server (RELAESE): %v", releaseString)
		}

	}

}

func (s *Server) Request(ctx context.Context, req *proto.AdminRequest) (*proto.Reply, error) {
	log.Printf("Node %s received access request from Node %s", s.Id, req.Id)

	s.lTime++

	first := true
	s.time2Id[int64(req.LogicalTime)] = string(s.Id)
	var lowest int64 = 1000000
	for num := range s.time2Id {
		if first || num < lowest {
			first = false
			lowest = num
		} else if num == lowest {
			log.Printf("åh nejjjjjjjjj ")
			log.Printf("%d", num)
		}
		value := s.time2Id[num]
		log.Printf("time: %d id: %s", num, value)
	}
	log.Printf("lowest: %d", lowest)

	// Implement your access request handling logic here

	return &proto.Reply{Id: s.Id, Reply: true, LogicalTime: time.Now().Unix()}, nil
}

func (s *Server) Release(ctx context.Context, req *proto.AdminRelease) (*proto.Reply, error) {
	log.Printf("Node %s received release access from Node %s", s.Id, req.Id)
	// Implement your release access handling logic here

	return &proto.Reply{Id: s.Id, Reply: true, LogicalTime: time.Now().Unix()}, nil
}

func sendRequest(target string, fromID string) {
	conn, err := grpc.Dial(target, grpc.WithInsecure())
	if err != nil {
		log.Fatalf("Did not connect: %v", err)
	}
	defer conn.Close()

	c := proto.NewMutual_ExclusionClient(conn)
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()

	req := &proto.AdminRequest{Id: fromID, LogicalTime: time.Now().Unix()}

	reply, err := c.Request(ctx, req)
	if err != nil {
		log.Printf("Could not send request to %s: %v", target, err)
		return
	}

	log.Printf("[%s] Got reply from %s: ok=%v", fromID, target, reply.Reply)
}

func StartServer(Id string, port int) {
	lis, err := net.Listen("tcp", fmt.Sprintf(":%d", port))
	if err != nil {
		log.Fatalf("Failed to listen on port %d: %v", port, err)
	}

	grpcServer := grpc.NewServer()
	proto.RegisterMutual_ExclusionServer(grpcServer, &Server{Id: Id})

	log.Printf("Node: %s listening on port: %d", Id, port)
	if err := grpcServer.Serve(lis); err != nil {
		log.Fatalf("Failed to serve: %v", err)
	}
}
